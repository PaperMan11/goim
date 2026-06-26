package config

import (
	"github.com/PaperMan11/goim/pkg/loginstrategy"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Redis redis.RedisConf
	Mongo MongoConf

	LoginStrategy loginstrategy.LoginStrategyConf

	AuthRpc       RpcConf
	UserRpc       RpcConf
	MsgGatewayRpc RpcConf

	Auth struct {
		AccessSecret string `json:",default=goim-access-secret"`
		AccessExpire int64  `json:",default=86400"`
		Issuer       string `json:",default=goim"`
	}
}

type RpcConf struct {
	zrpc.RpcClientConf
	Stub bool `json:",default=false"`
}

type MongoConf struct {
	Uri      string
	Database string
}
