package svc

import (
	"github.com/PaperMan11/goim/im-rpc/auth/internal/config"
	"github.com/PaperMan11/goim/pkg/authverify"
	_ "github.com/PaperMan11/goim/pkg/lb/iphash"
	"github.com/PaperMan11/goim/pkg/rpcclient/authservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/msggatewayservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/PaperMan11/goim/pkg/rpcinterceptors/clientinterceptors"
	"github.com/PaperMan11/goim/pkg/storage/cache/token"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServiceContext struct {
	Config     config.Config
	TokenStore token.TokenStore

	AuthVerify authverify.AuthVerifyService

	UserService       userservice.UserService
	AuthService       authservice.AuthService
	MsgGatewayService msggatewayservice.MsgGatewayService
}

func NewServiceContext(c config.Config) *ServiceContext {
	rs := redis.MustNewRedis(c.Redis)
	clientOpts := []zrpc.ClientOption{
		zrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		// zrpc.WithDialOption(grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`)),
		zrpc.WithDialOption(grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"iphash"}`)),
		zrpc.WithUnaryClientInterceptor(clientinterceptors.ClientContextInterceptor()),
	}
	userClient := zrpc.MustNewClient(c.UserRpc.RpcClientConf, clientOpts...)
	authClient := zrpc.MustNewClient(c.AuthRpc.RpcClientConf, clientOpts...)
	msgGatewayClient := zrpc.MustNewClient(c.MsgGatewayRpc.RpcClientConf, clientOpts...)
	return &ServiceContext{
		Config:            c,
		TokenStore:        token.NewRedisStore(rs),
		AuthVerify:        authverify.NewAuthVerify(userservice.NewUserService(userClient)),
		UserService:       userservice.NewUserService(userClient),
		AuthService:       authservice.NewAuthService(authClient),
		MsgGatewayService: msggatewayservice.NewMsgGatewayService(msgGatewayClient),
	}
}

func (sc *ServiceContext) Close() error {
	if sc.TokenStore != nil {
		sc.TokenStore.Close()
	}
	return nil
}
