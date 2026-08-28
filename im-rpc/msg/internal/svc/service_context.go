package svc

import (
	"github.com/PaperMan11/goim/im-rpc/msg/internal/config"
	"github.com/PaperMan11/goim/pkg/authverify"
	_ "github.com/PaperMan11/goim/pkg/lb/iphash"
	"github.com/PaperMan11/goim/pkg/localcache"
	queuex "github.com/PaperMan11/goim/pkg/queue"
	kafkax "github.com/PaperMan11/goim/pkg/queue/kafka"
	convServiceCache "github.com/PaperMan11/goim/pkg/rpccache/conversationservice"
	groupServiceCache "github.com/PaperMan11/goim/pkg/rpccache/groupservice"
	relationServiceCache "github.com/PaperMan11/goim/pkg/rpccache/relationservice"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	convservice "github.com/PaperMan11/goim/pkg/rpcclient/conversationservice"
	groupservice "github.com/PaperMan11/goim/pkg/rpcclient/groupservice"
	msgservice "github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	relationservice "github.com/PaperMan11/goim/pkg/rpcclient/relationservice"
	"github.com/redis/go-redis/v9"

	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"

	"github.com/PaperMan11/goim/pkg/msgdispatcher"
	"github.com/PaperMan11/goim/pkg/storage/model"
	msgModel "github.com/PaperMan11/goim/pkg/storage/mongo/msg"
	seqConversationModel "github.com/PaperMan11/goim/pkg/storage/mongo/seqconversation"
	seqUserModel "github.com/PaperMan11/goim/pkg/storage/mongo/sequser"
	allocator "github.com/PaperMan11/goim/pkg/storage/redis/allocator"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	AuthVerifier        authverify.AuthVerifyService
	MsgTransferProducer queuex.Producer
	LocalCache          localcache.LocalCache
	RedisClient         redis.UniversalClient

	// seq allocator
	SeqAllocator allocator.SeqAllocator
	// notification dispatcher
	NotificationSender msgdispatcher.MsgDispatcher
	// mongo models
	MsgModel             msgModel.MsgModel
	SeqUserModel         seqUserModel.SeqUserModel
	SeqConversationModel seqConversationModel.SeqConversationModel

	// rpc clients
	UserService     userservice.UserService
	ConvService     convservice.ConversationService
	RelationService relationservice.RelationService
	GroupService    groupservice.GroupService
	MsgService      msgservice.MsgService
}

func NewServiceContext(c config.Config) *ServiceContext {
	// mongo
	msgMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionMessage)
	convSeqMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionSeqConversation)
	userSeqMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionSeqUser)
	msgModel := msgModel.NewMsgModel(msgMongo)
	seqUserModel := seqUserModel.NewSeqUserModel(userSeqMongo)
	seqConversationModel := seqConversationModel.NewSeqConversationModel(convSeqMongo)

	// rpc clients
	var (
		userService                 userservice.UserService
		userServiceWrapperCache     userServiceCache.UserServiceWrapperCache
		convService                 convservice.ConversationService
		convServiceWrapperCache     convServiceCache.ConversationServiceWrapperCache
		relationService             relationservice.RelationService
		relationServiceWrapperCache relationServiceCache.RelationServiceWrapperCache
		groupService                groupservice.GroupService
		groupServiceWrapperCache    groupServiceCache.GroupServiceWrapperCache
		msgService                  msgservice.MsgService
		// msgServiceWrapperCache      msgServiceCache.MsgServiceWrapperCache
	)
	if c.UserRpc.Stub {
		userService = userservice.NewStubUserService()
	} else {
		userService = userservice.NewUserService(zrpc.MustNewClient(c.UserRpc.RpcClientConf))
	}
	if c.ConvRpc.Stub {
		convService = convservice.NewStubConversationService()
	} else {
		convService = convservice.NewConversationService(zrpc.MustNewClient(c.ConvRpc.RpcClientConf))
	}
	if c.RelationRpc.Stub {
		relationService = relationservice.NewStubRelationService()
	} else {
		relationService = relationservice.NewRelationService(zrpc.MustNewClient(c.RelationRpc.RpcClientConf))
	}
	if c.GroupRpc.Stub {
		groupService = groupservice.NewStubGroupService()
	} else {
		groupService = groupservice.NewGroupService(zrpc.MustNewClient(c.GroupRpc.RpcClientConf))
	}
	if c.MsgRpc.Stub {
		msgService = msgservice.NewStubMsgService()
	} else {
		msgService = msgservice.NewMsgService(zrpc.MustNewClient(c.MsgRpc.RpcClientConf))
	}

	redisCli := sredis.MustNewRedis(c.Redis)
	localCache := localcache.MustNewLocalCache(c.LocalCacheConf, redisCli)
	localCache.Start()

	userServiceWrapperCache = userServiceCache.NewUserServiceWrapperCache(userService, localCache)
	authVerifier := authverify.NewAuthVerify(userServiceWrapperCache)
	convServiceWrapperCache = convServiceCache.NewConversationServiceWrapperCache(convService, localCache)
	relationServiceWrapperCache = relationServiceCache.NewRelationServiceWrapperCache(relationService, localCache)
	groupServiceWrapperCache = groupServiceCache.NewGroupServiceWrapperCache(groupService, localCache)

	// msg transfer producer
	msgTransferProducer := kafkax.MustNewProducer(c.MsgTransferProducer)

	// seq allocator
	seqAllocator := allocator.MustNewRedisSeqAllocator(redisCli, allocator.WithGetMaxSeqFn(seqConversationModel.GetConversationMaxSeq))

	// notification dispatcher
	notificationSender := msgdispatcher.NewMsgDispatcher(msgService)

	return &ServiceContext{
		Config:               c,
		AuthVerifier:         authVerifier,
		MsgTransferProducer:  msgTransferProducer,
		MsgModel:             msgModel,
		SeqUserModel:         seqUserModel,
		SeqConversationModel: seqConversationModel,
		SeqAllocator:         seqAllocator,
		NotificationSender:   notificationSender,

		UserService:     userServiceWrapperCache,
		ConvService:     convServiceWrapperCache,
		RelationService: relationServiceWrapperCache,
		GroupService:    groupServiceWrapperCache,
	}
}

func (sc *ServiceContext) Close() error {
	if sc.LocalCache != nil {
		sc.LocalCache.Close()
	}
	return nil
}
