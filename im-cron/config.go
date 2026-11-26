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

	// 在线用户集合对账间隔，单位秒，默认5分钟；<=0 时关闭该任务。
	// 只清理 Redis 在线集合中的僵尸成员（缓存索引兜底，不写 DB、不发 1304）。
	OnlineReconcileIntervalSeconds int `json:",default=300"`

	// 僵尸在线状态清扫间隔，单位秒，默认60秒；<=0 时关闭该任务。
	// 扫描超过 ZombieThresholdMs 未心跳的在线快照行，置为离线并补发 1304（业务状态修复）。
	StaleStatusSweepIntervalSeconds int `json:",default=60"`

	// token 索引僵尸数据清理间隔，单位秒，默认1小时；<=0 时关闭该任务
	TokenCleanupIntervalSeconds int `json:",default=3600"`
}
