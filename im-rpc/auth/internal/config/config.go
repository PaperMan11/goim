package config

import (
	"github.com/PaperMan11/goim/pkg/localcache"
	"github.com/PaperMan11/goim/pkg/loginstrategy"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	LoginStrategy loginstrategy.LoginStrategyConf

	AuthRpc       rpcclient.RpcConf
	UserRpc       rpcclient.RpcConf
	MsgGatewayRpc rpcclient.RpcConf

	LocalCacheConf localcache.CacheConfig

	Auth struct {
		AccessSecret string `json:",default=goim-access-secret"`
		AccessExpire int64  `json:",default=86400"`
		Issuer       string `json:",default=goim"`
	}
}
