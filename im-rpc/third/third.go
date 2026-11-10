package main

import (
	"flag"

	"github.com/PaperMan11/goim/im-rpc/third/internal/config"
	"github.com/PaperMan11/goim/im-rpc/third/internal/server"
	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
	"github.com/PaperMan11/goim/pkg/rpcinterceptors/serverinterceptors"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/third.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	// 环境变量优先于配置文件，便于把 RustFS 密钥从配置文件里剥离
	c.RustFS.ApplyEnvOverrides()
	ctx := svc.NewServiceContext(c)
	defer ctx.Close()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		third.RegisterThirdServer(grpcServer, server.NewThirdServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()
	s.AddUnaryInterceptors(serverinterceptors.ServerContextInterceptor())

	logx.Infof("Starting third RPC server at %s...", c.ListenOn)
	s.Start()
}
