package svc

import (
	"github.com/PaperMan11/goim/im-rpc/msg/internal/config"
	"github.com/PaperMan11/goim/pkg/authverify"
	_ "github.com/PaperMan11/goim/pkg/lb/iphash"
	"github.com/PaperMan11/goim/pkg/localcache"
	queuex "github.com/PaperMan11/goim/pkg/queue"
	kafkax "github.com/PaperMan11/goim/pkg/queue/kafka"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"

	"github.com/PaperMan11/goim/pkg/storage/model"
	msgModel "github.com/PaperMan11/goim/pkg/storage/mongo/msg"
	seqConversationModel "github.com/PaperMan11/goim/pkg/storage/mongo/seqconversation"
	seqUserModel "github.com/PaperMan11/goim/pkg/storage/mongo/sequser"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	AuthVerifier        authverify.AuthVerifyService
	MsgTransferProducer queuex.Producer
	LocalCache          localcache.LocalCache

	// mongo models
	MsgModel             msgModel.MsgModel
	SeqUserModel         seqUserModel.SeqUserModel
	SeqConversationModel seqConversationModel.SeqConversationModel

	// rpc clients
	UserService userservice.UserService
}

func NewServiceContext(c config.Config) *ServiceContext {
	// mongo
	msgMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionMessage)
	convSeqMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionSeqConversation)
	userSeqMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionSeqUser)
	msgModel := msgModel.NewMsgModel(msgMongo)
	seqUserModel := seqUserModel.NewSeqUserModel(userSeqMongo)
	seqConversationModel := seqConversationModel.NewSeqConversationModel(convSeqMongo)

	// auth verifier
	var (
		userService             userservice.UserService
		userServiceWrapperCache userServiceCache.UserServiceWrapperCache
	)
	if c.UserRpc.Stub {
		userService = userservice.NewStubUserService()
	} else {
		userService = userservice.NewUserService(zrpc.MustNewClient(c.UserRpc.RpcClientConf))
	}

	redisCli := sredis.MustNewRedis(c.Redis)
	localCache := localcache.MustNewLocalCache(c.LocalCacheConf, redisCli)
	localCache.Start()

	userServiceWrapperCache = userServiceCache.NewUserServiceWrapperCache(userService, localCache)
	authVerifier := authverify.NewAuthVerify(userServiceWrapperCache)

	// msg transfer producer
	msgTransferProducer := kafkax.MustNewProducer(c.MsgTransferProducer)

	return &ServiceContext{
		Config:               c,
		AuthVerifier:         authVerifier,
		MsgTransferProducer:  msgTransferProducer,
		MsgModel:             msgModel,
		SeqUserModel:         seqUserModel,
		SeqConversationModel: seqConversationModel,

		UserService: userService,
	}
}

func (sc *ServiceContext) Close() error {
	if sc.LocalCache != nil {
		sc.LocalCache.Close()
	}
	return nil
}
