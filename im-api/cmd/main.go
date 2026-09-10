package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/PaperMan11/goim/im-api/internal/config"
	"github.com/PaperMan11/goim/im-api/internal/handler"
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/proc"
)

var configFile = flag.String("f", "etc/msggateway.yaml", "config file")

func main() {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)
	defer ctx.Close()

	e := handler.InitRoutes(ctx)
	c.MustSetUp()

	server := http.Server{
		Addr:    fmt.Sprintf("%s:%d", c.ServerConf.Host, c.ServerConf.Port),
		Handler: e,
	}
	defer server.Close()

	go func() {
		server.ListenAndServe()
	}()
	<-proc.Done()
}
