package main

import (
	"flag"

	"github.com/PaperMan11/goim/im-rpc/msg/internal/config"
	"github.com/PaperMan11/goim/im-rpc/msg/internal/server"
	"github.com/PaperMan11/goim/im-rpc/msg/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/rpcinterceptors/serverinterceptors"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/msg.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)
	defer ctx.Close()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		msg.RegisterMsgServer(grpcServer, server.NewMsgServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()
	s.AddUnaryInterceptors(serverinterceptors.ServerContextInterceptor())

	logx.Infof("Starting rpc server at %s...", c.ListenOn)
	s.Start()
}
