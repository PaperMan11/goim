package config

import (
	"github.com/PaperMan11/goim/pkg/localcache"
	queuex "github.com/PaperMan11/goim/pkg/queue/kafka"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Redis redis.RedisConf
	Mongo MongoConf

	MsgTransferProducer queuex.KafkaConfig

	AuthRpc        rpcclient.RpcConf
	UserRpc        rpcclient.RpcConf
	MsgGatewayRpc  rpcclient.RpcConf
	LocalCacheConf localcache.CacheConfig

	Auth struct {
		AccessSecret string `json:",default=goim-access-secret"`
		AccessExpire int64  `json:",default=86400"`
		Issuer       string `json:",default=goim"`
	}
}

type MongoConf struct {
	Uri      string
	Database string
}
