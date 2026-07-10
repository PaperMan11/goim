package rpcclient

import "github.com/zeromicro/go-zero/zrpc"

type RpcConf struct {
	zrpc.RpcClientConf
	Stub bool `json:",default=false"`
}
