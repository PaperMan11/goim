package svc

import (
	"github.com/PaperMan11/goim/im-rpc/relation/internal/config"
	"github.com/PaperMan11/goim/pkg/authverify"
	_ "github.com/PaperMan11/goim/pkg/lb/iphash"
	"github.com/PaperMan11/goim/pkg/localcache"
	groupServiceCache "github.com/PaperMan11/goim/pkg/rpccache/groupservice"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/groupservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/redis/go-redis/v9"

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

	// mongo models
	FriendModel      friendModel.FriendModel
	VersionLogModel  versionLogModel.VersionLogModel
	RequestModel     requestModel.RequestModel

	// rpc clients
	UserService  userServiceCache.UserServiceWrapperCache
	GroupService groupServiceCache.GroupServiceWrapperCache
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisCli := sredis.MustNewRedis(c.Redis)
	localCache := localcache.MustNewLocalCache(c.LocalCacheConf, redisCli)
	localCache.Start()

	friendMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionFriend)
	blackMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionBlack)
	friendInnerModel := friendModel.NewFriendModel(friendMongo, blackMongo)
	friendCacheModel := friendModel.NewCachedFriendModel(friendInnerModel, redisCli, syncx.NewSingleFlight())

	versionMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionGroupVersion)
	versionLogModel := versionLogModel.NewCachedVersionLogModelFromMongo(versionMongo, redisCli, syncx.NewSingleFlight())

	friendReqMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionFriendRequest)
	groupReqMongo := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, model.CollectionGroupRequest)
	reqInnerModel := requestModel.NewRequestModel(friendReqMongo, groupReqMongo)
	reqCacheModel := requestModel.NewCachedRequestModel(reqInnerModel, redisCli, syncx.NewSingleFlight())

	sc := &ServiceContext{
		Config:          c,
		FriendModel:     friendCacheModel,
		VersionLogModel: versionLogModel,
		RequestModel:    reqCacheModel,
		LocalCache:      localCache,
		RedisCli:        redisCli,
	}
	sc.initRpcClient()
	return sc
}

func (sc *ServiceContext) initRpcClient() {
	var (
		userService  userservice.UserService
		groupService groupservice.GroupService
	)
	if sc.Config.UserRpc.Stub {
		userService = userservice.NewStubUserService()
	} else {
		userService = userservice.NewUserService(zrpc.MustNewClient(sc.Config.UserRpc.RpcClientConf))
	}
	if sc.Config.GroupRpc.Stub {
		groupService = groupservice.NewStubGroupService()
	} else {
		groupService = groupservice.NewGroupService(zrpc.MustNewClient(sc.Config.GroupRpc.RpcClientConf))
	}
	sc.UserService = userServiceCache.NewUserServiceWrapperCache(userService, sc.LocalCache)
	sc.GroupService = groupServiceCache.NewGroupServiceWrapperCache(groupService, sc.LocalCache)

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
