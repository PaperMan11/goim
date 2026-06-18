package main

import (
	"errors"
	"flag"
	"net/http"

	"github.com/PaperMan11/goim/im-msggateway/internal"
	authservice "github.com/PaperMan11/goim/pkg/rpcclient/authservice"
	userservice "github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/PaperMan11/goim/pkg/webhooks"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/zrpc"
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

	if err := wsServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logx.Errorf("Failed to start server: %v", err)
	}
}

func newWsServer(c *internal.MsgGatewayConfig) internal.Server {
	var (
		authService authservice.AuthService
		userService userservice.UserService
	)
	if c.AuthRpc.Stub {
		authService = authservice.NewStubAuthService()
	} else {
		authRpcClient := zrpc.MustNewClient(c.AuthRpc.RpcClientConf)
		authService = authservice.NewAuthService(authRpcClient)
	}
	if c.UserRpc.Stub {
		userService = userservice.NewStubUserService()
	} else {
		userRpcClient := zrpc.MustNewClient(c.UserRpc.RpcClientConf)
		userService = userservice.NewUserService(userRpcClient)
	}

	// 消息处理器
	pipeline := internal.NewMessagePipeline(internal.NewBusinessHandler())

	// 创建webhook manager
	webhookManager := webhooks.NewManager(webhooks.NewMemoryDeliveryRepository(), 5)
	webhookManager.Start()
	defer webhookManager.Stop()

	return internal.NewWsServer(&c.WsServer, pipeline, webhookManager, authService, userService)
}
