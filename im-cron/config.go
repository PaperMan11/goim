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
	UserRpc rpcclient.RpcConf
}

type CronTask struct {
	// 消息保留时间，单位天，默认7天
	MsgRetentionDays int `json:",default=7"`

	// 会话保留时间，单位天，默认7天
	ConvRetentionDays int `json:",default=7"`

	// 自毁消息清理间隔，单位秒，默认5分钟；<=0 时关闭该任务
	DestructMsgIntervalSeconds int `json:",default=300"`

	// 在线用户集合对账间隔，单位秒，默认5分钟；<=0 时关闭该任务
	OnlineReconcileIntervalSeconds int `json:",default=300"`
}
