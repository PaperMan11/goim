package internal

import (
	"errors"
	"runtime"

	offlnepush "github.com/PaperMan11/goim/im-push/internal/offlnepush"
	queuex "github.com/PaperMan11/goim/pkg/queue"
	kafkax "github.com/PaperMan11/goim/pkg/queue/kafka"
	"github.com/PaperMan11/goim/pkg/rpcclient/conversationservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/groupservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/msggatewayservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/pushservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	webhookStore "github.com/PaperMan11/goim/pkg/storage/webhook"
	"github.com/PaperMan11/goim/pkg/webhooks"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/zrpc"
)

type Pusher struct {
	cfg                 *Config
	pushConsumer        queuex.Consumer
	offlinePushConsumer queuex.Consumer
	offlinePushProducer queuex.Producer
	offlinePusher       offlnepush.OfflinePusher
	webhookManager      *webhooks.Manager
	msgService          msgservice.MsgService
	pushService         pushservice.PushService
	msgGatewayService   msggatewayservice.MsgGatewayService
	groupService        groupservice.GroupService
	conversationService conversationservice.ConversationService
	userService         userservice.UserService
}

func NewPusher(cfg *Config) (*Pusher, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	pushConsumer := kafkax.MustNewConsumer(cfg.PushConsumer)
	offlinePushConsumer := kafkax.MustNewConsumer(cfg.OfflinePushTopic)
	offlinePushProducer := kafkax.MustNewProducer(cfg.OfflinePushTopic)

	monClient := mon.MustNewModel(cfg.Mongo.Uri, cfg.Mongo.Database, "webhook")
	redisClient := sredis.MustNewRedis(cfg.Redis)
	webhookStore := webhookStore.NewWebhookMongoStore(monClient, redisClient)
	webhookManager := webhooks.NewManager(webhookStore, runtime.NumCPU())

	var (
		msgRpcClient          zrpc.Client
		pushRpcClient         zrpc.Client
		gatewayRpcClient      zrpc.Client
		groupRpcClient        zrpc.Client
		conversationRpcClient zrpc.Client
		userRpcClient         zrpc.Client
		msgService            msgservice.MsgService
		pushService           pushservice.PushService
		msgGatewayService     msggatewayservice.MsgGatewayService
		groupService          groupservice.GroupService
		conversationService   conversationservice.ConversationService
		userService           userservice.UserService
	)

	if !cfg.MsgRpc.Stub {
		msgRpcClient = zrpc.MustNewClient(cfg.MsgRpc.RpcClientConf)
		msgService = msgservice.NewMsgService(msgRpcClient)
	} else {
		msgService = msgservice.NewStubMsgService()
	}

	if !cfg.PushRpc.Stub {
		pushRpcClient = zrpc.MustNewClient(cfg.PushRpc.RpcClientConf)
		pushService = pushservice.NewPushService(pushRpcClient)
	} else {
		pushService = pushservice.NewStubPushService()
	}

	if !cfg.GatewayRpc.Stub {
		gatewayRpcClient = zrpc.MustNewClient(cfg.GatewayRpc.RpcClientConf)
		msgGatewayService = msggatewayservice.NewMsgGatewayService(gatewayRpcClient)
	} else {
		msgGatewayService = msggatewayservice.NewStubMsgGatewayService()
	}

	if !cfg.GroupRpc.Stub {
		groupRpcClient = zrpc.MustNewClient(cfg.GroupRpc.RpcClientConf)
		groupService = groupservice.NewGroupService(groupRpcClient)
	} else {
		groupService = groupservice.NewStubGroupService()
	}

	if !cfg.ConversationRpc.Stub {
		conversationRpcClient = zrpc.MustNewClient(cfg.ConversationRpc.RpcClientConf)
		conversationService = conversationservice.NewConversationService(conversationRpcClient)
	} else {
		conversationService = conversationservice.NewStubConversationService()
	}

	if !cfg.UserRpc.Stub {
		userRpcClient = zrpc.MustNewClient(cfg.UserRpc.RpcClientConf)
		userService = userservice.NewUserService(userRpcClient)
	} else {
		userService = userservice.NewStubUserService()
	}

	p := &Pusher{
		cfg:                 cfg,
		pushConsumer:        pushConsumer,
		offlinePushConsumer: offlinePushConsumer,
		offlinePushProducer: offlinePushProducer,
		offlinePusher:       offlnepush.NewOfflinePusher(&cfg.OfflinePush),
		webhookManager:      webhookManager,
		msgService:          msgService,
		pushService:         pushService,
		msgGatewayService:   msgGatewayService,
		groupService:        groupService,
		conversationService: conversationService,
		userService:         userService,
	}

	if err := p.pushConsumer.Subscribe(p.consumePushMsg); err != nil {
		return nil, err
	}

	if err := p.offlinePushConsumer.Subscribe(p.consumeOfflinePushMsg); err != nil {
		return nil, err
	}

	return p, nil
}

func validateConfig(cfg *Config) error {
	if len(cfg.OfflinePushTopic.Brokers) == 0 {
		return errors.New("offline push topic brokers cannot be empty")
	}
	if len(cfg.OfflinePushTopic.Topic) == 0 {
		return errors.New("offline push topic topic cannot be empty")
	}
	if len(cfg.Redis.Host) == 0 {
		return errors.New("redis host cannot be empty")
	}
	if len(cfg.Mongo.Uri) == 0 {
		return errors.New("mongo uri cannot be empty")
	}
	if len(cfg.Mongo.Database) == 0 {
		return errors.New("mongo database cannot be empty")
	}
	return nil
}

func (p *Pusher) Start() {
	if err := p.pushConsumer.Start(); err != nil {
		logx.Errorf("Failed to start push consumer: %v", err)
		return
	}

	if err := p.offlinePushConsumer.Start(); err != nil {
		logx.Errorf("Failed to start offline push consumer: %v", err)
		return
	}

	p.webhookManager.Start()

	logx.Infof("OfflinePush Started, consumer topic: %s", p.offlinePushConsumer.Name())
}

func (p *Pusher) Stop() {
	if err := p.pushConsumer.Stop(); err != nil {
		logx.Errorf("Failed to stop push consumer: %v", err)
	}

	if err := p.offlinePushConsumer.Stop(); err != nil {
		logx.Errorf("Failed to stop offline push consumer: %v", err)
	}

	p.webhookManager.Stop()

	logx.Infof("OfflinePush Stopped")
}
