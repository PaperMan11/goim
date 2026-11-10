package logic

import (
	"context"
	"fmt"

	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
	"github.com/PaperMan11/goim/pkg/storage/s3"
	"github.com/zeromicro/go-zero/core/logx"
)

// logicBase 是所有 third Logic 的通用基础：持有 ctx、svcCtx 和 logx.Logger。
// 同时提供一些把 s3 适配层类型转 proto 类型的辅助方法，避免每个 Logic 重复写。
type logicBase struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func newLogicBase(ctx context.Context, svcCtx *svc.ServiceContext) logicBase {
	return logicBase{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// toProtoHeaders 把 s3.HeaderKV 列表转成 proto 的 []*third.KeyValues。
func toProtoHeaders(kvs []s3.HeaderKV) []*third.KeyValues {
	if len(kvs) == 0 {
		return nil
	}
	out := make([]*third.KeyValues, 0, len(kvs))
	for _, kv := range kvs {
		out = append(out, &third.KeyValues{Key: kv.Key, Values: kv.Values})
	}
	return out
}

// errxArgs / errxInternal 是 third.rpc 逻辑包内统一的错误包装器。
func errxArgs(msg string) error      { return fmt.Errorf("[third.rpc] args error: %s", msg) }
func errxInternal(err error) error   { return fmt.Errorf("[third.rpc] s3 error: %w", err) }
