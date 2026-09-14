package svc

import (
	"github.com/PaperMan11/goim/im-rpc/group/internal/config"
	"github.com/PaperMan11/goim/pkg/authverify"
	_ "github.com/PaperMan11/goim/pkg/lb/iphash"
	"github.com/PaperMan11/goim/pkg/localcache"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/PaperMan11/goim/pkg/rpcinterceptors/clientinterceptors"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/PaperMan11/goim/pkg/storage/model"
	groupModel "github.com/PaperMan11/goim/pkg/storage/mongo/group"
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

	// mongo models
	GroupModel      groupModel.GroupModel
	VersionLogModel versionLogModel.VersionLogModel
	RequestModel    requestModel.RequestModel

	// rpc clients
	UserService userServiceCache.UserServiceWrapperCache
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisCli := sredis.MustNewRedis(c.Redis)
	localCache := localcache.MustNewLocalCache(c.LocalCacheConf, redisCli)
	localCache.Start()
	singleFlight := syncx.NewSingleFlight()

	groupMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionGroup)
	memberMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionGroupMember)
	groupInnerModel := groupModel.NewGroupModel(groupMongo, memberMongo)
	groupCacheModel := groupModel.NewCachedGroupModel(groupInnerModel, redisCli, singleFlight)

	versionMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionGroupVersion)
	versionLogModel := versionLogModel.NewCachedVersionLogModelFromMongo(versionMongo, redisCli, singleFlight)

	friendReqMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionFriendRequest)
	groupReqMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionGroupRequest)
	reqInnerModel := requestModel.NewRequestModel(friendReqMongo, groupReqMongo)
	reqCacheModel := requestModel.NewCachedRequestModel(reqInnerModel, redisCli, singleFlight)

	sc := &ServiceContext{
		Config:          c,
		GroupModel:      groupCacheModel,
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
	)
	if sc.Config.UserRpc.Stub {
		userService = userservice.NewStubUserService()
	} else {
		userService = userservice.NewUserService(zrpc.MustNewClient(sc.Config.UserRpc.RpcClientConf, clientOpts...))
	}
	sc.UserService = userServiceCache.NewUserServiceWrapperCache(userService, sc.LocalCache)

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
