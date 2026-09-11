package svc

import (
	"github.com/PaperMan11/goim/im-rpc/user/internal/config"
	"github.com/PaperMan11/goim/pkg/authverify"
	_ "github.com/PaperMan11/goim/pkg/lb/iphash"
	"github.com/PaperMan11/goim/pkg/localcache"
	groupServiceCache "github.com/PaperMan11/goim/pkg/rpccache/groupservice"
	relationServiceCache "github.com/PaperMan11/goim/pkg/rpccache/relationservice"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/groupservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/relationservice"
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
	Config       config.Config
	AuthVerifier authverify.AuthVerifyService
	LocalCache   localcache.LocalCache
	RedisCli     redis.UniversalClient
	SingleFlight syncx.SingleFlight

	// mongo models
	UserModel userModel.UserModel

	// rpc clients
	UserService     userServiceCache.UserServiceWrapperCache
	RelationService relationServiceCache.RelationServiceWrapperCache
	GroupService    groupServiceCache.GroupServiceWrapperCache
}

func NewServiceContext(c config.Config) *ServiceContext {
	// local cache
	redisCli := sredis.MustNewRedis(c.Redis)
	localCache := localcache.MustNewLocalCache(c.LocalCacheConf, redisCli)
	localCache.Start()
	singleFlight := syncx.NewSingleFlight()

	userMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionUser)
	statusMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionUserStatus)
	cmdMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionUserCommand)
	clientConfigMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionUserClientConfig)
	userInnerModel := userModel.NewUserModel(userMongo, statusMongo, cmdMongo, clientConfigMongo, singleFlight)
	userCacheModel := userModel.NewCachedUserModel(userInnerModel, redisCli, singleFlight)

	sc := &ServiceContext{
		Config:       c,
		UserModel:    userCacheModel,
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
		userService     userservice.UserService
		relationService relationservice.RelationService
		groupService    groupservice.GroupService
	)
	if sc.Config.UserRpc.Stub {
		userService = userservice.NewStubUserService()
	} else {
		userService = userservice.NewUserService(zrpc.MustNewClient(sc.Config.UserRpc.RpcClientConf, clientOpts...))
	}
	if sc.Config.RelationRpc.Stub {
		relationService = relationservice.NewStubRelationService()
	} else {
		relationService = relationservice.NewRelationService(zrpc.MustNewClient(sc.Config.RelationRpc.RpcClientConf, clientOpts...))
	}
	if sc.Config.GroupRpc.Stub {
		groupService = groupservice.NewStubGroupService()
	} else {
		groupService = groupservice.NewGroupService(zrpc.MustNewClient(sc.Config.GroupRpc.RpcClientConf, clientOpts...))
	}
	sc.UserService = userServiceCache.NewUserServiceWrapperCache(userService, sc.LocalCache)
	sc.RelationService = relationServiceCache.NewRelationServiceWrapperCache(relationService, sc.LocalCache)
	sc.GroupService = groupServiceCache.NewGroupServiceWrapperCache(groupService, sc.LocalCache)

	// auth verifier
	sc.AuthVerifier = authverify.NewAuthVerify(sc.UserService)
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
