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

	UserRpc        rpcclient.RpcConf
	RelationRpc    rpcclient.RpcConf
	GroupRpc       rpcclient.RpcConf
	LocalCacheConf localcache.CacheConfig
}

type MongoConf struct {
	Uri      string
	Database string
}
