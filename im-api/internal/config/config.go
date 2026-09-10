package config

import (
	"github.com/PaperMan11/goim/pkg/localcache"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	storagemongo "github.com/PaperMan11/goim/pkg/storage/mongo"
	"github.com/zeromicro/go-zero/core/service"
	zredis "github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	ServerConf
	Redis          zredis.RedisConf
	Mongo          storagemongo.MongoConf
	LocalCacheConf localcache.CacheConfig

	AuthRpc     rpcclient.RpcConf
	UserRpc     rpcclient.RpcConf
	ConvRpc     rpcclient.RpcConf
	RelationRpc rpcclient.RpcConf
	GroupRpc    rpcclient.RpcConf
	MsgRpc      rpcclient.RpcConf
}

type ServerConf struct {
	service.ServiceConf
	Host string `json:",default=0.0.0.0"`
	Port int    `json:",default=8080"`
}
