package svc

import (
	"github.com/PaperMan11/goim/im-rpc/auth/internal/config"
	"github.com/PaperMan11/goim/pkg/authverify"
	_ "github.com/PaperMan11/goim/pkg/lb/iphash"
	"github.com/PaperMan11/goim/pkg/localcache"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/authservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/msggatewayservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/PaperMan11/goim/pkg/rpcinterceptors/clientinterceptors"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/PaperMan11/goim/pkg/storage/token"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServiceContext struct {
	Config     config.Config
	TokenStore token.TokenStore

	AuthVerify authverify.AuthVerifyService

	UserService       userServiceCache.UserServiceWrapperCache
	AuthService       authservice.AuthService
	MsgGatewayService msggatewayservice.MsgGatewayService

	LocalCache localcache.LocalCache
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisCli := sredis.MustNewRedis(c.Redis)

	clientOpts := []zrpc.ClientOption{
		zrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		// zrpc.WithDialOption(grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`)),
		zrpc.WithDialOption(grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"iphash"}`)),
		zrpc.WithUnaryClientInterceptor(clientinterceptors.ClientContextInterceptor()),
	}
	userClient := zrpc.MustNewClient(c.UserRpc.RpcClientConf, clientOpts...)
	authClient := zrpc.MustNewClient(c.AuthRpc.RpcClientConf, clientOpts...)
	msgGatewayClient := zrpc.MustNewClient(c.MsgGatewayRpc.RpcClientConf, clientOpts...)

	localCache := localcache.MustNewLocalCache(c.LocalCacheConf, redisCli)
	localCache.Start()

	// rpc client
	var (
		userService             userservice.UserService
		userServiceWrapperCache userServiceCache.UserServiceWrapperCache
	)
	userService = userservice.NewUserService(userClient)
	userServiceWrapperCache = userServiceCache.NewUserServiceWrapperCache(userService, localCache)

	return &ServiceContext{
		Config:            c,
		TokenStore:        token.NewRedisStore(redisCli),
		AuthVerify:        authverify.NewAuthVerify(userServiceWrapperCache),
		UserService:       userServiceWrapperCache,
		AuthService:       authservice.NewAuthService(authClient),
		MsgGatewayService: msggatewayservice.NewMsgGatewayService(msgGatewayClient),
	}
}

func (sc *ServiceContext) Close() error {
	if sc.TokenStore != nil {
		sc.TokenStore.Close()
	}

	if sc.LocalCache != nil {
		sc.LocalCache.Close()
	}
	return nil
}
