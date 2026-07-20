package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/msg/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type Logic struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logx.Logger
}

func NewLogic(ctx context.Context, svc *svc.ServiceContext) *Logic {
	return &Logic{
		svcCtx: svc,
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
	}
}
