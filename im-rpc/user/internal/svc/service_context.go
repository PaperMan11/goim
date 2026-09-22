package svc

import (
	"github.com/PaperMan11/goim/im-rpc/user/internal/config"
	"github.com/PaperMan11/goim/im-rpc/user/internal/notification"
	"github.com/PaperMan11/goim/pkg/authverify"
	_ "github.com/PaperMan11/goim/pkg/lb/iphash"
	"github.com/PaperMan11/goim/pkg/localcache"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"

	"github.com/PaperMan11/goim/pkg/rpcinterceptors/clientinterceptors"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/PaperMan11/goim/pkg/storage/model"
	userModel "github.com/PaperMan11/goim/pkg/storage/mongo/user"
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
	UserModel userModel.UserModel

	// rpc clients
	UserService userServiceCache.UserServiceWrapperCache
}

func NewServiceContext(c config.Config) *ServiceContext {
	// local cache
	redisCli := sredis.MustNewRedis(c.Redis.RedisConf)
	localCache := localcache.MustNewLocalCache(c.LocalCacheConf, redisCli)
	localCache.Start()
	singleFlight := syncx.NewSingleFlight()

	userMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionUser)
	statusMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionUserStatus)
	cmdMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionUserCommand)
	clientConfigMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionUserClientConfig)
	userInnerModel := userModel.NewUserModel(userMongo, statusMongo, cmdMongo, clientConfigMongo, singleFlight)
	userCachedModel := userModel.NewCachedUserModel(userInnerModel, redisCli, singleFlight)

	sc := &ServiceContext{
		Config:       c,
		UserModel:    userCachedModel,
		LocalCache:   localCache,
		RedisCli:     redisCli,
		SingleFlight: singleFlight,
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
		userService = userservice.NewUserService(zrpc.MustNewClient(sc.Config.UserRpc.RpcClientConf, clientOpts...))
	}
	if sc.Config.MsgRpc.Stub {
		msgService = msgservice.NewStubMsgService()
	} else {
		msgService = msgservice.NewMsgService(zrpc.MustNewClient(sc.Config.MsgRpc.RpcClientConf, clientOpts...))
	}

	sc.UserService = userServiceCache.NewUserServiceWrapperCache(userService, sc.LocalCache)

	// auth verifier
	sc.AuthVerifier = authverify.NewAuthVerify(sc.UserService)
	// notification sender
	sc.NotificationSender = notification.NewNotificationSender(msgService, userService)
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
