package internal

import (
	"github.com/zeromicro/go-zero/zrpc"
)

type MsgGatewayConfig struct {
	zrpc.RpcServerConf
	WsServer WsServerConfig
	AuthRpc  RpcConfig
	UserRpc  RpcConfig
	MsgRpc   RpcConfig
	PushRpc  RpcConfig
}

type WsServerConfig struct {
	Host       string `json:",default=0.0.0.0"`
	Port       int    `json:",default=50003"`
	MaxConns   int64  `json:",default=1024096"`
	MaxMsgSize int64  `json:",default=1024096"` // 最大消息大小
	WriteWait  int64  `json:",default=10"`      // 写入超时时间
	PongWait   int64  `json:",default=60"`      // 读取超时时间
	PingPeriod int64  `json:",default=5"`       // 心跳周期
	EnableAuth bool   `json:",default=true"`    // 是否开启认证，默认开启
	LoginStrategyConfig
}

type RpcConfig struct {
	zrpc.RpcClientConf
	Stub bool `json:",default=false"`
}

type LoginStrategyConfig struct {
	LoginStrategy             LoginStrategy `json:",default=allow_multi"` // 多端登录策略
	MaxConnPerUser            int64         `json:",default=10"`          // 每个用户最大连接数（allow_multi策略下生效）
	MaxConnPerUserPerPlatform int64         `json:",default=3"`           // 每个用户每个平台最大连接数
}
