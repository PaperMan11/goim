package immsgtransfer

import (
	"errors"
	"runtime"

	"github.com/PaperMan11/goim/pkg/msgprocessor"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	queuex "github.com/PaperMan11/goim/pkg/queue"
	kafkax "github.com/PaperMan11/goim/pkg/queue/kafka"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/rpcinterceptors/clientinterceptors"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	webhookStore "github.com/PaperMan11/goim/pkg/storage/webhook"
	"github.com/PaperMan11/goim/pkg/utils/batcher"
	"github.com/PaperMan11/goim/pkg/webhooks"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	msgPersistentConsumer queuex.Consumer
	msgPersistentProducer queuex.Producer
	msgPushProducer       queuex.Producer
	webhookManager        *webhooks.Manager

	msgService msgservice.MsgService
	batcher    *batcher.Batcher[sdkws.MsgData]
}

func NewMsgTransfer(cfg *Config) (*MsgTransfer, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	msgTransferConsumer := kafkax.MustNewConsumer(cfg.MsgTransferConsumer)
	msgPersistentConsumer := kafkax.MustNewConsumer(cfg.MsgPersistentTopic)
	msgPersistentProducer := kafkax.MustNewProducer(cfg.MsgPersistentTopic)
	msgPushProducer := kafkax.MustNewProducer(cfg.MsgPushProducer)

	monClient := mon.MustNewModel(cfg.Mongo.Uri, cfg.Mongo.Database, "webhook")
	redisClient := sredis.MustNewRedis(cfg.Redis)
	webhookStore := webhookStore.NewWebhookMongoStore(monClient, redisClient)
	webhookManager := webhooks.NewManager(webhookStore, runtime.NumCPU())

	clientOpts := []zrpc.ClientOption{
		zrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		// zrpc.WithDialOption(grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`)),
		zrpc.WithDialOption(grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"iphash"}`)),
		zrpc.WithUnaryClientInterceptor(clientinterceptors.ClientContextInterceptor()),
	}
	var (
		msgService msgservice.MsgService
	)

	if !cfg.MsgRpc.Stub {
		msgService = msgservice.NewMsgService(zrpc.MustNewClient(cfg.MsgRpc.RpcClientConf, clientOpts...))
	} else {
		msgService = msgservice.NewStubMsgService()
	}

	// 初始化批量处理器
	batcher := batcher.NewBatcher[sdkws.MsgData]()
	batcher.SetResetFunc(func(md *sdkws.MsgData) {
		md.Reset()
	})

	mt := &MsgTransfer{
		cfg:                   cfg,
		msgTransferConsumer:   msgTransferConsumer,
		msgPersistentConsumer: msgPersistentConsumer,
		msgPersistentProducer: msgPersistentProducer,
		msgPushProducer:       msgPushProducer,
		webhookManager:        webhookManager,
		msgService:            msgService,
		batcher:               batcher,
	}

	if err := mt.msgTransferConsumer.Subscribe(mt.consumeMsg); err != nil {
		return nil, err
	}
	if err := mt.msgPersistentConsumer.Subscribe(mt.consumePersistentMsgs); err != nil {
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
	if len(cfg.MsgPersistentTopic.Brokers) == 0 {
		return errors.New("msg persistent topic brokers cannot be empty")
	}
	if len(cfg.MsgPersistentTopic.Topic) == 0 {
		return errors.New("msg persistent topic topic cannot be empty")
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

	if err := mt.msgPersistentConsumer.Start(); err != nil {
		logx.Errorf("Failed to start msg persistent consumer: %v", err)
		return
	}

	mt.webhookManager.Start()
	mt.batcher.Start(func(msg *sdkws.MsgData) string {
		return msgprocessor.GetConversationIDByMsg(msg)
	}, mt.batchHandleMsg)

	logx.Infof("MsgTransfer Started, consumer topic: %s, persistent producer topic: %s, push producer topic: %s",
		mt.msgTransferConsumer.Name(), mt.msgPersistentProducer.Name(), mt.msgPushProducer.Name())
}

func (mt *MsgTransfer) Stop() {

	if err := mt.msgPersistentProducer.Close(); err != nil {
		logx.Errorf("Failed to close msg persistent producer: %v", err)
	}

	if err := mt.msgPushProducer.Close(); err != nil {
		logx.Errorf("Failed to close msg push producer: %v", err)
	}

	if err := mt.msgTransferConsumer.Stop(); err != nil {
		logx.Errorf("Failed to stop msg transfer consumer: %v", err)
	}

	if err := mt.msgPersistentConsumer.Stop(); err != nil {
		logx.Errorf("Failed to stop msg persistent consumer: %v", err)
	}

	mt.webhookManager.Stop()
	mt.batcher.Stop()

	logx.Infof("MsgTransfer Stopped")
}
