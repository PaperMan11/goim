package svc

import (
	"github.com/PaperMan11/goim/im-rpc/conversation/internal/config"
	"github.com/PaperMan11/goim/im-rpc/conversation/internal/notification"
	"github.com/PaperMan11/goim/pkg/authverify"
	_ "github.com/PaperMan11/goim/pkg/lb/iphash"
	"github.com/PaperMan11/goim/pkg/localcache"
	groupServiceCache "github.com/PaperMan11/goim/pkg/rpccache/groupservice"
	msgServiceCache "github.com/PaperMan11/goim/pkg/rpccache/msgservice"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	"github.com/PaperMan11/goim/pkg/rpcclient/groupservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/PaperMan11/goim/pkg/rpcinterceptors/clientinterceptors"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/PaperMan11/goim/pkg/storage/model"
	conversationModel "github.com/PaperMan11/goim/pkg/storage/mongo/conversation"
	seqConversationModel "github.com/PaperMan11/goim/pkg/storage/mongo/seqconversation"
	seqUserModel "github.com/PaperMan11/goim/pkg/storage/mongo/sequser"
	versionLogModel "github.com/PaperMan11/goim/pkg/storage/mongo/versionlog"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/core/syncx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config             config.Config
	AuthVerifier       authverify.AuthVerifyService
	LocalCache         localcache.LocalCache
	RedisCli           redis.UniversalClient
	SingleFlight       syncx.SingleFlight
	NotificationSender *notification.NotificationSender

	// mongo models
	ConversationModel conversationModel.ConversationModel
	VersionLogModel   versionLogModel.VersionLogModel
	// todo: 移到msg rpc
	SeqUserModel         seqUserModel.SeqUserModel
	SeqConversationModel seqConversationModel.SeqConversationModel

	// rpc clients
	UserService  userServiceCache.UserServiceWrapperCache
	GroupService groupServiceCache.GroupServiceWrapperCache
	MsgService   msgServiceCache.MsgServiceWrapperCache
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisCli := sredis.MustNewRedis(c.Redis.RedisConf)
	localCache := localcache.MustNewLocalCache(c.LocalCacheConf, redisCli)
	localCache.Start()
	singleFlight := syncx.NewSingleFlight()

	convMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionConversation)
	latestMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionConversationLatestMsg)
	convInnerModel := conversationModel.NewConversationModel(convMongo, latestMongo)
	convCacheModel := conversationModel.NewCachedConversationModel(convInnerModel, redisCli, singleFlight)

	versionMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionGroupVersion)
	versionLogModel := versionLogModel.NewCachedVersionLogModelFromMongo(versionMongo, redisCli, singleFlight)

	// seq 表：全局会话级 + 用户级（含 read_seq）
	seqConvMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionSeqConversation)
	seqConvInner := seqConversationModel.NewSeqConversationModel(seqConvMongo)
	seqConvCache := seqConversationModel.NewCachedSeqConversationModel(seqConvInner, redisCli, singleFlight)

	seqUserMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionSeqUser)
	seqUserInner := seqUserModel.NewSeqUserModel(seqUserMongo)
	seqUserCache := seqUserModel.NewCachedSeqUserModel(seqUserInner, redisCli, singleFlight)

	sc := &ServiceContext{
		Config:               c,
		ConversationModel:    convCacheModel,
		VersionLogModel:      versionLogModel,
		SeqUserModel:         seqUserCache,
		SeqConversationModel: seqConvCache,
		LocalCache:           localCache,
		RedisCli:             redisCli,
		SingleFlight:         singleFlight,
	}
	sc.initRpcClient()
	return sc
}

func (sc *ServiceContext) initRpcClient() {
	clientOpts := []zrpc.ClientOption{
		zrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		// zrpc.WithDialOption(grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`)),
		zrpc.WithDialOption(grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"iphash"}`)),
		zrpc.WithUnaryClientInterceptor(clientinterceptors.ClientContextInterceptor()),
	}
	var (
		userService  userservice.UserService
		groupService groupservice.GroupService
		msgService   msgservice.MsgService
	)
	if sc.Config.UserRpc.Stub {
		userService = userservice.NewStubUserService()
	} else {
		userService = userservice.NewUserService(rpcclient.MustNewClient(sc.Config.UserRpc.RpcClientConf, clientOpts...))
	}
	if sc.Config.GroupRpc.Stub {
		groupService = groupservice.NewStubGroupService()
	} else {
		groupService = groupservice.NewGroupService(rpcclient.MustNewClient(sc.Config.GroupRpc.RpcClientConf, clientOpts...))
	}
	if sc.Config.MsgRpc.Stub {
		msgService = msgservice.NewStubMsgService()
	} else {
		msgService = msgservice.NewMsgService(rpcclient.MustNewClient(sc.Config.MsgRpc.RpcClientConf, clientOpts...))
	}
	sc.UserService = userServiceCache.NewUserServiceWrapperCache(userService, sc.LocalCache)
	sc.GroupService = groupServiceCache.NewGroupServiceWrapperCache(groupService, sc.LocalCache)
	sc.MsgService = msgServiceCache.NewMsgServiceWrapperCache(msgService, sc.LocalCache)

	sc.AuthVerifier = authverify.NewAuthVerify(sc.UserService)
	sc.NotificationSender = notification.NewNotificationSender(sc.MsgService, sc.UserService)
}

func (sc *ServiceContext) Close() error {
	if sc.LocalCache != nil {
		sc.LocalCache.Close()
	}
	if sc.RedisCli != nil {
		sc.RedisCli.Close()
	}
	return nil
}
