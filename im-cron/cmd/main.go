package main

import (
	"flag"

	imcron "github.com/PaperMan11/goim/im-cron"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/proc"
)

var configFile = flag.String("f", "etc/cron.yaml", "config file")

func main() {
	flag.Parse()
	var c imcron.Config
	conf.MustLoad(*configFile, &c)
	c.MustSetUp()

	cronServer := imcron.NewCronServer(&c)
	cronServer.Start()
	defer cronServer.Stop()

	<-proc.Done()
}
