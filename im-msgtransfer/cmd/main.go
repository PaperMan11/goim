package main

import (
	"flag"

	msgtransfer "github.com/PaperMan11/goim/im-msgtransfer"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/proc"
)

var configFile = flag.String("f", "../etc/msgtransfer.yml", "config file")

func main() {
	flag.Parse()
	var cfg msgtransfer.Config
	conf.MustLoad(*configFile, &cfg)
	cfg.MustSetUp()

	msgTransfer, err := msgtransfer.NewMsgTransfer(&cfg)
	if err != nil {
		panic(err)
	}
	msgTransfer.Start()
	defer msgTransfer.Stop()

	<-proc.Done()
}
