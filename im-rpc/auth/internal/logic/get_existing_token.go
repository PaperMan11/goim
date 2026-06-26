package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetExistingTokenLogic struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logx.Logger
}

func NewGetExistingTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExistingTokenLogic {
	return &GetExistingTokenLogic{
		svcCtx: svcCtx,
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetExistingTokenLogic) GetExistingToken(req *auth.GetExistingTokenReq) (*auth.GetExistingTokenResp, error) {
	tokens, err := l.svcCtx.TokenStore.GetUserTokensByPlatform(l.ctx, req.UserID, req.PlatformID)
	if err != nil {
		l.Errorf("get user tokens by platform failed: %v", err)
		return nil, err
	}

	respTokens := make(map[string]int32)
	for _, token := range tokens {
		if token.IsExpired() {
			respTokens[token.Token] = constant.ExpiredToken
		} else {
			respTokens[token.Token] = constant.NormalToken
		}
	}
	return &auth.GetExistingTokenResp{
		TokenStates: respTokens,
	}, nil
}
