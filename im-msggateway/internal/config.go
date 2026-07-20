package internal

import (
	"github.com/PaperMan11/goim/pkg/localcache"
	"github.com/PaperMan11/goim/pkg/loginstrategy"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	"github.com/zeromicro/go-zero/zrpc"
)

type MsgGatewayConfig struct {
	HubServerConf
	WsServer WsServerConf
	AuthRpc  rpcclient.RpcConf
	UserRpc  rpcclient.RpcConf
	MsgRpc   rpcclient.RpcConf
	PushRpc  rpcclient.RpcConf
}

type HubServerConf struct {
	zrpc.RpcServerConf
	LocalCacheConf localcache.CacheConfig
	Mongo          struct {
		Uri      string
		Database string
	}
}

type WsServerConf struct {
	Host          string `json:",default=0.0.0.0"`
	Port          int    `json:",default=50003"`
	MaxConns      int64  `json:",default=1024096"`
	MaxMsgSize    int64  `json:",default=1024096"` // 最大消息大小
	WriteWait     int64  `json:",default=10"`      // 写入超时时间
	PongWait      int64  `json:",default=60"`      // 读取超时时间
	PingPeriod    int64  `json:",default=5"`       // 心跳周期
	EnableAuth    bool   `json:",default=true"`    // 是否开启认证，默认开启
	LoginStrategy LoginStrategyConf
}

type LoginStrategyConf = loginstrategy.LoginStrategyConf
