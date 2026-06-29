package logic

import (
	"context"
	"errors"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/PaperMan11/goim/pkg/storage/token"
	"github.com/zeromicro/go-zero/core/logx"
)

type InvalidateTokenLogic struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logx.Logger
}

func NewInvalidateTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InvalidateTokenLogic {
	return &InvalidateTokenLogic{
		svcCtx: svcCtx,
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *InvalidateTokenLogic) InvalidateToken(req *auth.InvalidateTokenReq) (*auth.InvalidateTokenResp, error) {
	tokens, err := l.svcCtx.TokenStore.GetUserTokensByPlatform(l.ctx, req.UserID, req.PlatformID)
	if err != nil && !errors.Is(err, token.ErrTokenNotFound) {
		l.Errorf("get user tokens by platform failed: %v", err)
		return nil, err
	}

	if len(tokens) > 0 {
		deleteTokens := make([]string, 0)
		for _, token := range tokens {
			if token.Token != req.PreservedToken {
				deleteTokens = append(deleteTokens, token.UUID)
			}
		}
		if len(deleteTokens) > 0 {
			err = l.svcCtx.TokenStore.DeleteTokens(l.ctx, deleteTokens)
			if err != nil {
				l.Errorf("delete tokens failed: %v", err)
				return nil, err
			}
		}
	}

	return &auth.InvalidateTokenResp{}, nil
}
