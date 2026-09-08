package main

import (
	"flag"

	push "github.com/PaperMan11/goim/im-push/internal"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/proc"
)

var configFile = flag.String("f", "../etc/push.yml", "config file")

func main() {
	flag.Parse()
	var cfg push.Config
	conf.MustLoad(*configFile, &cfg)
	cfg.MustSetUp()

	push, err := push.NewPusher(&cfg)
	if err != nil {
		panic(err)
	}
	push.Start()
	defer push.Stop()

	<-proc.Done()
}
