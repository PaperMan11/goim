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

	UserRpc       rpcclient.RpcConf
	MsgGatewayRpc rpcclient.RpcConf

	LocalCacheConf localcache.CacheConfig

	// AdminSecret 是 get_admin_token 接口的共享密钥，调用方必须传入正确值才能换取 Admin Token。
	AdminSecret string `json:",default=goim-admin-secret"`

	JwtAuth struct {
		AccessSecret string `json:",default=goim-access-secret"`
		AccessExpire int64  `json:",default=86400"`
		Issuer       string `json:",default=goim"`
	}
}
