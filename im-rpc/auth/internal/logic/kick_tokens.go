package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/zeromicro/go-zero/core/logx"
)

type KickTokensLogic struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logx.Logger
}

func NewKickTokensLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickTokensLogic {
	return &KickTokensLogic{
		svcCtx: svcCtx,
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *KickTokensLogic) KickTokens(req *auth.KickTokensReq) (*auth.KickTokensResp, error) {
	if len(req.Tokens) == 0 {
		return &auth.KickTokensResp{}, nil
	}

	err := l.svcCtx.TokenStore.DeleteTokens(l.ctx, req.Tokens)
	if err != nil {
		l.Errorf("delete tokens failed: %v", err)
		return nil, err
	}
	return &auth.KickTokensResp{}, nil
}
