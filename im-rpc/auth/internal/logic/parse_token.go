package logic

import (
	"context"
	"errors"
	"time"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/PaperMan11/goim/pkg/utils/jwtx"
	"github.com/zeromicro/go-zero/core/logx"
)

type ParseTokenLogic struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logx.Logger
}

func NewParseTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ParseTokenLogic {
	return &ParseTokenLogic{
		svcCtx: svcCtx,
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ParseTokenLogic) ParseToken(req *auth.ParseTokenReq) (*auth.ParseTokenResp, error) {
	if req.Token == "" {
		return nil, errx.TokenUnknownError
	}

	token, err := jwtx.ParseToken(req.Token, l.svcCtx.Config.Auth.AccessSecret)
	switch {
	case errors.Is(err, jwtx.ErrTokenExpired):
		return nil, errx.TokenExpiredError
	case errors.Is(err, jwtx.ErrTokenInvalid):
		return nil, errx.TokenInvalidError
	case err != nil:
		return nil, err
	default:
		return &auth.ParseTokenResp{
			UserID:            token.UserID,
			PlatformID:        token.PlatformID,
			ExpireTimeSeconds: token.ExpiresAt.Unix() - time.Now().Unix(),
		}, nil
	}
}
