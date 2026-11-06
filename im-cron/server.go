package imcron

import (
	"context"
	"fmt"

	"github.com/PaperMan11/goim/pkg/lock"
	redLocker "github.com/PaperMan11/goim/pkg/lock/redis"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	convservice "github.com/PaperMan11/goim/pkg/rpcclient/conversationservice"
	msgservice "github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	userservice "github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/PaperMan11/goim/pkg/rpcinterceptors/clientinterceptors"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/robfig/cron/v3"
)

type CronServer struct {
	cfg         *Config
	locker      lock.Locker
	cron        *cron.Cron
	redisClient redis.UniversalClient
	convService convservice.ConversationService
	msgService  msgservice.MsgService
	userService userservice.UserService
}

func NewCronServer(cfg *Config) *CronServer {
	redisClient := sredis.MustNewRedis(cfg.Redis)
	locker := redLocker.NewRedisLocker(redisClient)

	clientOpts := []zrpc.ClientOption{
		zrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		// zrpc.WithDialOption(grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`)),
		zrpc.WithDialOption(grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"iphash"}`)),
		zrpc.WithUnaryClientInterceptor(clientinterceptors.ClientContextInterceptor()),
	}
	var (
		convService convservice.ConversationService
		msgService  msgservice.MsgService
		userService userservice.UserService
	)
	if !cfg.ConvRpc.Stub {
		convService = convservice.NewConversationService(rpcclient.MustNewClient(cfg.ConvRpc.RpcClientConf, clientOpts...))
	} else {
		convService = convservice.NewStubConversationService()
	}
	if !cfg.MsgRpc.Stub {
		msgService = msgservice.NewMsgService(rpcclient.MustNewClient(cfg.MsgRpc.RpcClientConf, clientOpts...))
	} else {
		msgService = msgservice.NewStubMsgService()
	}
	if !cfg.UserRpc.Stub {
		userService = userservice.NewUserService(rpcclient.MustNewClient(cfg.UserRpc.RpcClientConf, clientOpts...))
	} else {
		userService = userservice.NewStubUserService()
	}

	return &CronServer{
		cfg:         cfg,
		locker:      locker,
		cron:        cron.New(),
		redisClient: redisClient,
		convService: convService,
		msgService:  msgService,
		userService: userService,
	}
}

func (s *CronServer) Start() {
	if s.cron != nil {
		s.registerClearUserMessageJob()
		s.registerDeleteMessageJob()
		s.registerDestructMsgJob()
		s.registerOnlineReconcileJob()
		s.cron.Start()
	}
}

func (s *CronServer) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
	if s.redisClient != nil {
		s.redisClient.Close()
	}
}

func (s *CronServer) registerClearUserMessageJob() {
	if s.cfg.CronTask.ConvRetentionDays <= 0 {
		logx.Errorf("ConvRetentionDays is 0, skip clear user message job")
		return
	}
	// "@every 1h"
	// "@daily"
	s.cron.AddFunc("@daily", func() {
		s.locker.ExecWithLock(context.Background(), "cron:clear-user-message", func() error {
			s.ClearUserMessage()
			return nil
		})
	})
}

func (s *CronServer) registerDeleteMessageJob() {
	if s.cfg.CronTask.MsgRetentionDays <= 0 {
		logx.Errorf("MsgRetentionDays is 0, skip delete message job")
		return
	}
	s.cron.AddFunc("@daily", func() {
		s.locker.ExecWithLock(context.Background(), "cron:delete-message", func() error {
			s.DeleteMessage()
			return nil
		})
	})
}

// registerDestructMsgJob 注册自毁消息清理任务：按会话的 msg_destruct_time 物理删除已过期消息。
func (s *CronServer) registerDestructMsgJob() {
	interval := s.cfg.CronTask.DestructMsgIntervalSeconds
	if interval <= 0 {
		logx.Errorf("DestructMsgIntervalSeconds is 0, skip destruct message job")
		return
	}
	spec := fmt.Sprintf("@every %ds", interval)
	if _, err := s.cron.AddFunc(spec, func() {
		s.locker.ExecWithLock(context.Background(), "cron:destruct-message", func() error {
			s.ClearDestructMsg()
			return nil
		})
	}); err != nil {
		logx.Errorf("register destruct message job failed, spec: %s, err: %v", spec, err)
	}
}

// registerOnlineReconcileJob 注册在线用户集合对账任务：清理 mongo:user:online:set 中的僵尸成员。
// user RPC 为 stub 时无法校验在线状态，直接跳过，避免把全部在线用户误删。
func (s *CronServer) registerOnlineReconcileJob() {
	interval := s.cfg.CronTask.OnlineReconcileIntervalSeconds
	if interval <= 0 {
		logx.Errorf("OnlineReconcileIntervalSeconds is 0, skip online reconcile job")
		return
	}
	if s.cfg.UserRpc.Stub {
		logx.Errorf("UserRpc is stub, skip online reconcile job")
		return
	}
	spec := fmt.Sprintf("@every %ds", interval)
	if _, err := s.cron.AddFunc(spec, func() {
		s.locker.ExecWithLock(context.Background(), "cron:reconcile-online-users", func() error {
			s.ReconcileOnlineUsers()
			return nil
		})
	}); err != nil {
		logx.Errorf("register online reconcile job failed, spec: %s, err: %v", spec, err)
	}
}
