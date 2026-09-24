package imcron

import (
	"context"

	"github.com/PaperMan11/goim/pkg/lock"
	redLocker "github.com/PaperMan11/goim/pkg/lock/redis"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	convservice "github.com/PaperMan11/goim/pkg/rpcclient/conversationservice"
	msgservice "github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
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

	return &CronServer{
		cfg:         cfg,
		locker:      locker,
		cron:        cron.New(),
		convService: convService,
		msgService:  msgService,
	}
}

func (s *CronServer) Start() {
	if s.cron != nil {
		s.registerClearUserMessageJob()
		s.registerDeleteMessageJob()
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
