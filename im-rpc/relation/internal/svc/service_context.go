package svc

import (
	"github.com/PaperMan11/goim/im-rpc/relation/internal/config"
	"github.com/PaperMan11/goim/im-rpc/relation/internal/notification"
	"github.com/PaperMan11/goim/pkg/authverify"
	_ "github.com/PaperMan11/goim/pkg/lb/iphash"
	"github.com/PaperMan11/goim/pkg/localcache"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/PaperMan11/goim/pkg/rpcinterceptors/clientinterceptors"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/PaperMan11/goim/pkg/storage/model"
	friendModel "github.com/PaperMan11/goim/pkg/storage/mongo/friend"
	requestModel "github.com/PaperMan11/goim/pkg/storage/mongo/request"
	versionLogModel "github.com/PaperMan11/goim/pkg/storage/mongo/versionlog"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/core/syncx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	AuthVerifier authverify.AuthVerifyService
	LocalCache   localcache.LocalCache
	RedisCli     redis.UniversalClient
	SingleFlight syncx.SingleFlight

	NotificationSender *notification.NotificationSender
	// mongo models
	FriendModel     friendModel.FriendModel
	VersionLogModel versionLogModel.VersionLogModel
	RequestModel    requestModel.RequestModel

	// rpc clients
	UserService userServiceCache.UserServiceWrapperCache
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisCli := sredis.MustNewRedis(c.Redis.RedisConf)
	localCache := localcache.MustNewLocalCache(c.LocalCacheConf, redisCli)
	localCache.Start()
	singleFlight := syncx.NewSingleFlight()

	friendMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionFriend)
	blackMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionBlack)
	friendVersionMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionFriendVersion)
	friendCacheModel := friendModel.NewFriendModel(friendMongo, blackMongo, singleFlight, redisCli, c.Mongo.EnableCache)
	versionLogModel := versionLogModel.NewVersionLogModel(friendVersionMongo, singleFlight, redisCli, c.Mongo.EnableCache)

	friendReqMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionFriendRequest)
	groupReqMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionGroupRequest)
	reqCacheModel := requestModel.NewRequestModel(friendReqMongo, groupReqMongo, singleFlight, redisCli, c.Mongo.EnableCache)

	sc := &ServiceContext{
		Config:          c,
		FriendModel:     friendCacheModel,
		VersionLogModel: versionLogModel,
		RequestModel:    reqCacheModel,
		LocalCache:      localCache,
		RedisCli:        redisCli,
		SingleFlight:    singleFlight,
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
		userService userservice.UserService
		msgService  msgservice.MsgService
	)
	if sc.Config.UserRpc.Stub {
		userService = userservice.NewStubUserService()
	} else {
		userService = userservice.NewUserService(rpcclient.MustNewClient(sc.Config.UserRpc.RpcClientConf, clientOpts...))
	}
	if sc.Config.MsgRpc.Stub {
		msgService = msgservice.NewStubMsgService()
	} else {
		msgService = msgservice.NewMsgService(rpcclient.MustNewClient(sc.Config.MsgRpc.RpcClientConf, clientOpts...))
	}
	sc.UserService = userServiceCache.NewUserServiceWrapperCache(userService, sc.LocalCache, sc.Config.UserRpc.EnableCache)

	sc.AuthVerifier = authverify.NewAuthVerify(sc.UserService)
	sc.NotificationSender = notification.NewNotificationSender(msgService, sc.UserService, sc.RequestModel, sc.FriendModel)
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
