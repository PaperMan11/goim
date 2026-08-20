package config

import (
	"github.com/PaperMan11/goim/pkg/localcache"
	queuex "github.com/PaperMan11/goim/pkg/queue/kafka"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	storagemongo "github.com/PaperMan11/goim/pkg/storage/mongo"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Redis redis.RedisConf
	Mongo storagemongo.MongoConf

	MsgTransferProducer queuex.KafkaConfig

	UserRpc        rpcclient.RpcConf
	GroupRpc       rpcclient.RpcConf
	MsgRpc         rpcclient.RpcConf
	LocalCacheConf localcache.CacheConfig
}
