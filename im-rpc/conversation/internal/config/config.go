package config

import (
	"github.com/PaperMan11/goim/pkg/localcache"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	storagemongo "github.com/PaperMan11/goim/pkg/storage/mongo"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Mongo storagemongo.MongoConf

	UserRpc        rpcclient.RpcConf
	GroupRpc       rpcclient.RpcConf
	MsgRpc         rpcclient.RpcConf
	LocalCacheConf localcache.CacheConfig
}
