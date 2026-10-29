package logic

import (
	"context"
	"errors"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/PaperMan11/goim/pkg/utils/jwtx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
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
		l.Errorf("parse token: token is empty")
		return nil, errx.TokenUnknownError
	}

	token, err := jwtx.ParseToken(req.Token, l.svcCtx.Config.JwtAuth.AccessSecret)
	switch {
	case errors.Is(err, jwtx.ErrTokenExpired):
		l.Infof("parse token: expired, userID: %s", token.UserID)
		return nil, errx.TokenExpiredError
	case errors.Is(err, jwtx.ErrTokenInvalid):
		l.Errorf("parse token: invalid, err: %v", err)
		return nil, errx.TokenInvalidError
	case err != nil:
		l.Errorf("parse token: unexpected err: %v", err)
		return nil, err
	default:
		l.Infof("parse token: ok, userID: %s, platformID: %d", token.UserID, token.PlatformID)
		return &auth.ParseTokenResp{
			UserID:            token.UserID,
			PlatformID:        token.PlatformID,
			ExpireTimeSeconds: token.ExpiresAt.Unix() - timex.Unix(),
		}, nil
	}
}
