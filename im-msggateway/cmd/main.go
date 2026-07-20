package main

import (
	"errors"
	"flag"
	"net/http"

	"github.com/PaperMan11/goim/im-msggateway/internal"
	"github.com/PaperMan11/goim/pkg/authverify"
	"github.com/PaperMan11/goim/pkg/localcache"
	pbmsggateway "github.com/PaperMan11/goim/pkg/protocol/msggateway"
	userServiceCache "github.com/PaperMan11/goim/pkg/rpccache/userservice"
	authservice "github.com/PaperMan11/goim/pkg/rpcclient/authservice"
	msgservice "github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	pushservice "github.com/PaperMan11/goim/pkg/rpcclient/pushservice"
	userservice "github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	webhookStore "github.com/PaperMan11/goim/pkg/storage/webhook"
	"github.com/PaperMan11/goim/pkg/webhooks"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/msggateway.yaml", "config file")

func main() {
	flag.Parse()
	var c internal.MsgGatewayConfig
	conf.MustLoad(*configFile, &c)
	c.MustSetUp()

	wsServer := newWsServer(&c)
	proc.AddShutdownListener(func() {
		wsServer.Stop()
	})
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		hubServer, close := startHubServer(&c, wsServer)
		pbmsggateway.RegisterMsgGatewayServer(grpcServer, hubServer)
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
		hubServer.Start()
		proc.AddWrapUpListener(func() {
			close()
		})
	})
	go s.Start()
	defer s.Stop()

	if err := wsServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logx.Errorf("Failed to start server: %v", err)
	}
}

func startHubServer(c *internal.MsgGatewayConfig, wsServer internal.WsServer) (hubServer *internal.HubServer, close func()) {
	var (
		userService             userservice.UserService
		userServiceWrapperCache userServiceCache.UserServiceWrapperCache
	)
	if c.UserRpc.Stub {
		userService = userservice.NewStubUserService()
	} else {
		userService = userservice.NewUserService(zrpc.MustNewClient(c.UserRpc.RpcClientConf))
	}

	redisCli := sredis.MustNewRedis(c.Redis.RedisConf)
	localCache := localcache.MustNewLocalCache(c.HubServerConf.LocalCacheConf, redisCli)
	userServiceWrapperCache = userServiceCache.NewUserServiceWrapperCache(userService, localCache)

	authverifier := authverify.NewAuthVerify(userServiceWrapperCache)
	hubServer = internal.NewHubServer(wsServer, authverifier, &c.HubServerConf)
	hubServer.Start()
	localCache.Start()
	return hubServer, func() {
		hubServer.Stop()
		localCache.Close()
	}
}

func newWsServer(c *internal.MsgGatewayConfig) internal.WsServer {
	var (
		authService authservice.AuthService
		userService userservice.UserService
		msgService  msgservice.MsgService
		pushService pushservice.PushService
	)
	if c.AuthRpc.Stub {
		authService = authservice.NewStubAuthService()
	} else {
		authService = authservice.NewAuthService(zrpc.MustNewClient(c.AuthRpc.RpcClientConf))
	}
	if c.UserRpc.Stub {
		userService = userservice.NewStubUserService()
	} else {
		userService = userservice.NewUserService(zrpc.MustNewClient(c.UserRpc.RpcClientConf))
	}
	if c.MsgRpc.Stub {
		msgService = msgservice.NewStubMsgService()
	} else {
		msgService = msgservice.NewMsgService(zrpc.MustNewClient(c.MsgRpc.RpcClientConf))
	}
	if c.PushRpc.Stub {
		pushService = pushservice.NewStubPushService()
	} else {
		pushService = pushservice.NewPushService(zrpc.MustNewClient(c.PushRpc.RpcClientConf))
	}

	// 消息处理器
	pipeline := internal.NewMessagePipeline(internal.NewBusinessHandler(pushService, msgService))

	// 创建webhook manager
	redisCli := sredis.MustNewRedis(c.Redis.RedisConf)
	monCli := mon.MustNewModel(c.Mongo.Uri, c.Mongo.Database, "webhook")
	webhookManager := webhooks.NewManager(webhookStore.NewWebhookMongoStore(monCli, redisCli), 5)

	return internal.NewWsServer(&c.WsServer, pipeline, webhookManager, authService, userService)
}
