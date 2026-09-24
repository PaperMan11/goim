package rpcclient

import (
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

// MustNewClient creates a zrpc client without panicking on error.
// Unlike zrpc.MustNewClient which calls os.Exit on error,
// this function logs the error and returns the client (possibly nil).
//
// zrpc.NewClient is lazy: it doesn't connect to the server immediately.
// The "bad resolver state" logs are non-fatal and will auto-recover
// once the target service registers in Etcd.
// NewClient only fails for configuration errors (empty endpoints, etc.),
// which won't resolve by retrying.
func MustNewClient(c zrpc.RpcClientConf, options ...zrpc.ClientOption) zrpc.Client {
	cli, err := zrpc.NewClient(c, options...)
	if err != nil {
		logx.Errorf("rpc client init failed: %v", err)
	}
	return cli
}
