package svc

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/third/internal/config"
	"github.com/PaperMan11/goim/pkg/storage/s3"
)

// ServiceContext 是 third RPC 服务的运行时依赖容器。
// 只持有一个 S3Client——存储适配层屏蔽 RustFS / MinIO / AWS S3 差异。
type ServiceContext struct {
	Config    config.Config
	S3Client  s3.S3Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	ctx := context.Background()
	cli, err := s3.NewS3Client(ctx, &c.RustFS)
	if err != nil {
		// 启动时硬性失败：S3 是 third 服务存在的唯一理由，连不上就不应该启动。
		panic("third.rpc: failed to init S3 client: " + err.Error())
	}
	return &ServiceContext{
		Config:   c,
		S3Client: cli,
	}
}

func (sc *ServiceContext) Close() error {
	// s3.S3Client 本身无连接需要显式关闭（HTTP client 由 AWS SDK 管理）。
	// 此处保留接口占位，未来如果加 Redis 等依赖再处理。
	return nil
}
