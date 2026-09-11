package imcron

import (
	"github.com/PaperMan11/goim/pkg/rpcclient"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	service.ServiceConf
	CronTask CronTask

	Redis   redis.RedisConf
	ConvRpc rpcclient.RpcConf
	MsgRpc  rpcclient.RpcConf
}

type CronTask struct {
	// 消息保留时间，单位天，默认7天
	MsgRetentionDays int `json:",default=7"`

	// 会话保留时间，单位天，默认7天
	ConvRetentionDays int `json:",default=7"`
}
