package immsgtransfer

import (
	"errors"
	"runtime"

	queuex "github.com/PaperMan11/goim/pkg/queue"
	kafkax "github.com/PaperMan11/goim/pkg/queue/kafka"
	"github.com/PaperMan11/goim/pkg/rpcclient/conversationservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/groupservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/msggatewayservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/pushservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	webhookStore "github.com/PaperMan11/goim/pkg/storage/webhook"
	"github.com/PaperMan11/goim/pkg/webhooks"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

var (
	ErrNilConfig          = errors.New("config cannot be nil")
	ErrConsumerNotStarted = errors.New("consumer not started")
	ErrProducerNotStarted = errors.New("producer not started")
	ErrWebhookManagerNil  = errors.New("webhook manager cannot be nil")
)

type MsgTransfer struct {
	cfg                   *Config
	msgTransferConsumer   queuex.Consumer
	msgPersistentProducer queuex.Producer
	msgPushProducer       queuex.Producer
	webhookManager        *webhooks.Manager
	msgService            msgservice.MsgService
	pushService           pushservice.PushService
	msgGatewayService     msggatewayservice.MsgGatewayService
	groupService          groupservice.GroupService
	conversationService   conversationservice.ConversationService
	userService           userservice.UserService
	msgRpcClient          zrpc.Client
	pushRpcClient         zrpc.Client
	gatewayRpcClient      zrpc.Client
	groupRpcClient        zrpc.Client
	conversationRpcClient zrpc.Client
	userRpcClient         zrpc.Client
}

func NewMsgTransfer(cfg *Config) (*MsgTransfer, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	msgTransferConsumer := kafkax.MustNewConsumer(cfg.MsgTransferConsumer)
	msgPersistentProducer := kafkax.MustNewProducer(cfg.MsgPersistentProducer)
	msgPushProducer := kafkax.MustNewProducer(cfg.MsgPushProducer)

	monClient := mon.MustNewModel(cfg.Mongo.Uri, cfg.Mongo.Database, "webhook")
	redisClient := redis.MustNewRedis(cfg.Redis)
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

	mt := &MsgTransfer{
		cfg:                   cfg,
		msgTransferConsumer:   msgTransferConsumer,
		msgPersistentProducer: msgPersistentProducer,
		msgPushProducer:       msgPushProducer,
		webhookManager:        webhookManager,
		msgService:            msgService,
		pushService:           pushService,
		msgGatewayService:     msgGatewayService,
		groupService:          groupService,
		conversationService:   conversationService,
		userService:           userService,
		msgRpcClient:          msgRpcClient,
		pushRpcClient:         pushRpcClient,
		gatewayRpcClient:      gatewayRpcClient,
		groupRpcClient:        groupRpcClient,
		conversationRpcClient: conversationRpcClient,
		userRpcClient:         userRpcClient,
	}

	if err := mt.msgTransferConsumer.Subscribe(mt.handleMsg); err != nil {
		return nil, err
	}

	return mt, nil
}

func validateConfig(cfg *Config) error {
	if len(cfg.MsgTransferConsumer.Brokers) == 0 {
		return errors.New("msg transfer consumer brokers cannot be empty")
	}
	if len(cfg.MsgTransferConsumer.Topic) == 0 {
		return errors.New("msg transfer consumer topic cannot be empty")
	}
	if len(cfg.MsgPersistentProducer.Brokers) == 0 {
		return errors.New("msg persistent producer brokers cannot be empty")
	}
	if len(cfg.MsgPersistentProducer.Topic) == 0 {
		return errors.New("msg persistent producer topic cannot be empty")
	}
	if len(cfg.MsgPushProducer.Brokers) == 0 {
		return errors.New("msg push producer brokers cannot be empty")
	}
	if len(cfg.MsgPushProducer.Topic) == 0 {
		return errors.New("msg push producer topic cannot be empty")
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

func (mt *MsgTransfer) Start() {
	if err := mt.msgTransferConsumer.Start(); err != nil {
		logx.Errorf("Failed to start msg transfer consumer: %v", err)
		return
	}

	mt.webhookManager.Start()

	logx.Infof("MsgTransfer Started, consumer topic: %s, persistent producer topic: %s, push producer topic: %s",
		mt.msgTransferConsumer.Name(), mt.msgPersistentProducer.Name(), mt.msgPushProducer.Name())
}

func (mt *MsgTransfer) Stop() {

	if err := mt.msgPersistentProducer.Close(); err != nil {
		logx.Errorf("Failed to close msg persistent producer: %v", err)
	}

	if err := mt.msgTransferConsumer.Stop(); err != nil {
		logx.Errorf("Failed to stop msg transfer consumer: %v", err)
	}

	if err := mt.msgPushProducer.Close(); err != nil {
		logx.Errorf("Failed to close msg push producer: %v", err)
	}

	mt.webhookManager.Stop()

	logx.Infof("MsgTransfer Stopped")
}
