package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
)

// DeleteOutdatedLogic 过期对象清理。
// 现阶段 S3 bucket 还没配 lifecycle 规则，这个 RPC 留作 cron 手动清理用。
type DeleteOutdatedLogic struct {
	logicBase
}

func NewDeleteOutdatedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteOutdatedLogic {
	return &DeleteOutdatedLogic{logicBase: newLogicBase(ctx, svcCtx)}
}

func (l *DeleteOutdatedLogic) DeleteOutdatedData(req *third.DeleteOutdatedDataReq) (*third.DeleteOutdatedDataResp, error) {
	// 简单实现：如果没有配置 S3 lifecycle，这里做一次 ListObjects + 删除 LastModified < expireTime 的。
	// 先留空，等后续 bucket lifecycle 策略确认后再决定是手动扫还是直接在 bucket 上配规则。
	l.Infof("delete outdated data called, expireTime=%d, objectGroup=%v", req.ExpireTime, req.ObjectGroup)
	return &third.DeleteOutdatedDataResp{}, nil
}
