package logic

import (
	"context"
	"crypto/subtle"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/storage/token"
	"github.com/PaperMan11/goim/pkg/utils/jwtx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/PaperMan11/goim/pkg/utils/uuidx"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetAdminTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAdminTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAdminTokenLogic {
	return &GetAdminTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAdminTokenLogic) GetAdminToken(req *auth.GetAdminTokenReq) (*auth.GetAdminTokenResp, error) {
	// 1. 校验共享密钥（常量时间比较，防止时序攻击）
	if subtle.ConstantTimeCompare([]byte(req.Secret), []byte(l.svcCtx.Config.AdminSecret)) != 1 {
		l.Errorf("get admin token: invalid secret, userID: %s", req.UserID)
		return nil, errx.NoPermissionError
	}

	// 2. 校验 userID 是否是 IM 管理员
	if err := requireUserIsAdmin(l.ctx, l.svcCtx, l, req.UserID); err != nil {
		return nil, err
	}

	// if err := requireUserIsValid(l.ctx, l.svcCtx, l, req.UserID); err != nil {
	// 	return nil, err
	// }

	// generate admin token
	adminTokenInfo, err := l.generateAdminToken(l.ctx, req.UserID)
	if err != nil {
		l.Errorf("generate admin token failed, userID: %s, err: %v", req.UserID, err)
		return nil, errx.InternalError.WrapWithError(err)
	}

	if err := l.svcCtx.TokenStore.StoreToken(l.ctx, adminTokenInfo); err != nil {
		l.Errorf("store admin token failed, userID: %s, err: %v", req.UserID, err)
		return nil, errx.InternalError.WrapWithError(err)
	}

	return &auth.GetAdminTokenResp{
		Token:             adminTokenInfo.Token,
		ExpireTimeSeconds: adminTokenInfo.ExpireAt - timex.Unix(),
	}, nil
}

func (l *GetAdminTokenLogic) generateAdminToken(_ context.Context, userID string) (*token.TokenInfo, error) {
	tokenUUID := uuidx.New()
	expireAt := timex.Unix() + l.svcCtx.Config.JwtAuth.AccessExpire

	jwtToken, err := jwtx.GenerateAdminToken(
		tokenUUID,
		l.svcCtx.Config.JwtAuth.Issuer,
		userID,
		constant.AdminPlatformID,
		"",
		[]string{"admin"},
		l.svcCtx.Config.JwtAuth.AccessSecret,
		l.svcCtx.Config.JwtAuth.AccessExpire,
	)
	if err != nil {
		return nil, err
	}

	return &token.TokenInfo{
		UUID:       tokenUUID,
		Token:      jwtToken,
		UserID:     userID,
		PlatformID: constant.AdminPlatformID,
		ExpireAt:   expireAt,
		Roles:      "admin",
	}, nil
}
