package svc

import (
	"github.com/PaperMan11/goim/im-api/internal/config"
	"github.com/PaperMan11/goim/pkg/localcache"
	authserviceCache "github.com/PaperMan11/goim/pkg/rpccache/authservice"
	convServiceCache "github.com/PaperMan11/goim/pkg/rpccache/conversationservice"
	groupServiceCache "github.com/PaperMan11/goim/pkg/rpccache/groupservice"
	msgServiceCache "github.com/PaperMan11/goim/pkg/rpccache/msgservice"
	relationServiceCache "github.com/PaperMan11/goim/pkg/rpccache/relationservice"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/authservice"
	convservice "github.com/PaperMan11/goim/pkg/rpcclient/conversationservice"
	groupservice "github.com/PaperMan11/goim/pkg/rpcclient/groupservice"
	msgservice "github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	relationservice "github.com/PaperMan11/goim/pkg/rpcclient/relationservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/zeromicro/go-zero/zrpc"

	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Config     config.Config
	RedisCli   redis.UniversalClient
	LocalCache localcache.LocalCache

	// rpc clients
	UserService     userservice.UserService
	ConvService     convservice.ConversationService
	RelationService relationservice.RelationService
	GroupService    groupservice.GroupService
	MsgService      msgservice.MsgService
	AuthService     authservice.AuthService
}

func NewServiceContext(c config.Config) *ServiceContext {
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
		msgServiceWrapperCache      msgServiceCache.MsgServiceWrapperCache
		authService                 authservice.AuthService
		authServiceWrapperCache     authserviceCache.AuthServiceWrapperCache
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
	if c.AuthRpc.Stub {
		authService = authservice.NewStubAuthService()
	} else {
		authService = authservice.NewAuthService(zrpc.MustNewClient(c.AuthRpc.RpcClientConf))
	}
	redisCli := sredis.MustNewRedis(c.Redis)
	localCache := localcache.MustNewLocalCache(c.LocalCacheConf, redisCli)
	localCache.Start()

	userServiceWrapperCache = userServiceCache.NewUserServiceWrapperCache(userService, localCache)
	convServiceWrapperCache = convServiceCache.NewConversationServiceWrapperCache(convService, localCache)
	relationServiceWrapperCache = relationServiceCache.NewRelationServiceWrapperCache(relationService, localCache)
	groupServiceWrapperCache = groupServiceCache.NewGroupServiceWrapperCache(groupService, localCache)
	msgServiceWrapperCache = msgServiceCache.NewMsgServiceWrapperCache(msgService, localCache)
	authServiceWrapperCache = authserviceCache.NewAuthServiceWrapperCache(authService, localCache)
	// authVerifier := authverify.NewAuthVerify(userServiceWrapperCache)

	return &ServiceContext{
		Config:     c,
		RedisCli:   redisCli,
		LocalCache: localCache,

		// rpc clients
		UserService:     userServiceWrapperCache,
		ConvService:     convServiceWrapperCache,
		RelationService: relationServiceWrapperCache,
		GroupService:    groupServiceWrapperCache,
		MsgService:      msgServiceWrapperCache,
		AuthService:     authServiceWrapperCache,
	}
}

func (sc *ServiceContext) Close() error {
	if sc.LocalCache != nil {
		sc.LocalCache.Close()
	}
	return nil
}
