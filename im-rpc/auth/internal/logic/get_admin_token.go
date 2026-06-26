package logic

import (
	"context"
	"time"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/storage/cache/token"
	"github.com/PaperMan11/goim/pkg/utils/jwtx"
	"github.com/google/uuid"
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
	// todo check admin
	isAdmin, err := l.svcCtx.AuthVerify.IsIMAdmin(l.ctx, req.UserID)
	if err != nil {
		l.Errorf("failed to check admin, userID: %s, err: %v", req.UserID, err)
		return nil, err
	}
	if !isAdmin {
		l.Errorf("user %s is not admin", req.UserID)
		return nil, nil
	}

	// check user
	isValid, err := l.svcCtx.AuthVerify.IsValidUser(l.ctx, req.UserID)
	if err != nil {
		l.Errorf("failed to check user, userID: %s, err: %v", req.UserID, err)
		return nil, err
	}
	if !isValid {
		l.Errorf("user %s is not valid", req.UserID)
		return nil, nil
	}

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
		ExpireTimeSeconds: adminTokenInfo.ExpireAt - time.Now().Unix(),
	}, nil
}

func (l *GetAdminTokenLogic) generateAdminToken(_ context.Context, userID string) (*token.TokenInfo, error) {
	tokenUUID := uuid.New().String()
	expireAt := time.Now().Unix() + l.svcCtx.Config.Auth.AccessExpire

	jwtToken, err := jwtx.GenerateAdminToken(
		tokenUUID,
		l.svcCtx.Config.Auth.Issuer,
		userID,
		constant.AdminPlatformID,
		"",
		[]string{"admin"},
		l.svcCtx.Config.Auth.AccessSecret,
		l.svcCtx.Config.Auth.AccessExpire,
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
