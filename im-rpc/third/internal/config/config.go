package config

import (
	"github.com/PaperMan11/goim/pkg/storage/s3"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config 是 third RPC 服务的 go-zero 配置结构。
// 包含 zrpc.RpcServerConf（Etcd/ListenOn/Mode 等）和 RustFS 对象存储配置。
type Config struct {
	zrpc.RpcServerConf
	RustFS s3.Config
}
