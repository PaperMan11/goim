package main

import (
	"errors"
	"flag"
	"net/http"

	"github.com/PaperMan11/goim/im-msggateway/internal"
	authservice "github.com/PaperMan11/goim/pkg/rpcclient/authservice"
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
	)
	if c.AuthRpc.Stub {
		authService = authservice.NewStubAuthService()
	} else {
		authRpcClient := zrpc.MustNewClient(c.AuthRpc.RpcClientConf)
		authService = authservice.NewAuthService(authRpcClient)
	}

	handler := internal.MessageHandlerFunc(func(conn internal.Connection, req *internal.Request) error {
		return internal.BusinessHandler().Handle(conn, req)
	})

	pipeline := internal.NewMessagePipeline(handler)
	// 创建webhook manager
	webhookManager := webhooks.NewManager(webhooks.NewMemoryDeliveryRepository(), 5)
	webhookManager.Start()
	defer webhookManager.Stop()

	return internal.NewWsServer(&c.WsServer, authService, pipeline, webhookManager)
}
