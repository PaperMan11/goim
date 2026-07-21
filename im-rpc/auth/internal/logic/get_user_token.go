package logic

import (
	"context"
	"sort"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/loginstrategy"
	"github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/storage/token"
	"github.com/PaperMan11/goim/pkg/utils/jwtx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserTokenLogic struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logx.Logger
}

func NewGetUserTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserTokenLogic {
	return &GetUserTokenLogic{
		svcCtx: svcCtx,
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserTokenLogic) GetUserToken(req *auth.GetUserTokenReq) (*auth.GetUserTokenResp, error) {
	if err := requireUserIsAdmin(l.ctx, l.svcCtx, l, req.UserID); err != nil {
		return nil, err
	}

	if req.PlatformID == constant.AdminPlatformID {
		return nil, errx.NoPermissionError.Wrap("admin platform id is not allowed")
	}

	var (
		needInvalidateTokens []*token.TokenInfo
		err                  error
	)
	switch l.svcCtx.Config.LoginStrategy.LoginStrategy {
	case loginstrategy.LoginStrategySingle:
		needInvalidateTokens, err = l.handleSingleLogin(l.ctx, req.UserID)
	case loginstrategy.LoginStrategyReplace:
		needInvalidateTokens, err = l.handleReplaceLogin(l.ctx, req.UserID)
	case loginstrategy.LoginStrategyReplaceSamePlatform:
		needInvalidateTokens, err = l.handleReplaceSamePlatformLogin(l.ctx, req.UserID, req.PlatformID)
	case loginstrategy.LoginStrategyAllowMulti:
		needInvalidateTokens, err = l.handleAllowMultiLogin(l.ctx, req.UserID, req.PlatformID)
	default:
		needInvalidateTokens, err = l.handleAllowMultiLogin(l.ctx, req.UserID, req.PlatformID)
	}

	if err != nil {
		l.Errorf("login strategy check failed: %v", err)
		return nil, err
	}

	newTokenInfo, err := l.generateToken(l.ctx, req.UserID, req.PlatformID)
	if err != nil {
		l.Errorf("generate token failed: %v", err)
		return nil, errx.InternalError.WrapWithError(err)
	}

	for _, t := range needInvalidateTokens {
		_ = l.svcCtx.TokenStore.DeleteToken(l.ctx, t.UUID)
	}

	if err := l.svcCtx.TokenStore.StoreToken(l.ctx, newTokenInfo); err != nil {
		l.Errorf("store token failed: %v", err)
		return nil, errx.InternalError.WrapWithError(err)
	}

	return &auth.GetUserTokenResp{
		Token:             newTokenInfo.Token,
		ExpireTimeSeconds: newTokenInfo.ExpireAt - timex.Unix(),
	}, nil
}

func (l *GetUserTokenLogic) generateToken(_ context.Context, userID string, platformID int32) (*token.TokenInfo, error) {
	tokenUUID := uuid.New().String()
	expireAt := timex.Unix() + l.svcCtx.Config.Auth.AccessExpire

	jwtToken, err := jwtx.GenerateAccessToken(
		tokenUUID,
		l.svcCtx.Config.Auth.Issuer,
		userID,
		platformID,
		"",
		nil,
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
		PlatformID: platformID,
		ExpireAt:   expireAt,
	}, nil
}

func (l *GetUserTokenLogic) handleSingleLogin(ctx context.Context, userID string) ([]*token.TokenInfo, error) {
	userTokens, err := l.svcCtx.TokenStore.GetUserTokens(ctx, userID)
	if err != nil && err != token.ErrTokenNotFound {
		return nil, err
	}

	if len(userTokens) > 0 {
		return nil, token.ErrUserExists
	}

	return nil, nil
}

func (l *GetUserTokenLogic) handleReplaceLogin(ctx context.Context, userID string) ([]*token.TokenInfo, error) {
	userTokens, err := l.svcCtx.TokenStore.GetUserTokens(ctx, userID)
	if err != nil && err != token.ErrTokenNotFound {
		return nil, err
	}

	return userTokens, nil
}

func (l *GetUserTokenLogic) handleReplaceSamePlatformLogin(ctx context.Context, userID string, platformID int32) ([]*token.TokenInfo, error) {
	userTokens, err := l.svcCtx.TokenStore.GetUserTokens(ctx, userID)
	if err != nil && err != token.ErrTokenNotFound {
		return nil, err
	}

	var platformTokens []*token.TokenInfo
	for _, t := range userTokens {
		if t.PlatformID == platformID {
			platformTokens = append(platformTokens, t)
		}
	}

	return platformTokens, nil
}

func (l *GetUserTokenLogic) handleAllowMultiLogin(ctx context.Context, userID string, platformID int32) ([]*token.TokenInfo, error) {
	maxConnPerUser := l.svcCtx.Config.LoginStrategy.MaxConnPerUser
	maxConnPerPlatform := l.svcCtx.Config.LoginStrategy.MaxConnPerUserPerPlatform

	userTokens, err := l.svcCtx.TokenStore.GetUserTokens(ctx, userID)
	if err != nil && err != token.ErrTokenNotFound {
		return nil, err
	}

	var platformTokens []*token.TokenInfo
	for _, t := range userTokens {
		if t.PlatformID == platformID {
			platformTokens = append(platformTokens, t)
		}
	}

	var toInvalidate []*token.TokenInfo

	if maxConnPerPlatform > 0 && int64(len(platformTokens)) >= maxConnPerPlatform {
		sort.Slice(platformTokens, func(i, j int) bool {
			return platformTokens[i].ExpireAt < platformTokens[j].ExpireAt
		})
		toInvalidate = append(toInvalidate, platformTokens[:len(platformTokens)-int(maxConnPerPlatform)+1]...)
	}

	if maxConnPerUser > 0 {
		remainingTokens := len(userTokens) - len(toInvalidate) + 1
		if int64(remainingTokens) > maxConnPerUser {
			sort.Slice(userTokens, func(i, j int) bool {
				return userTokens[i].ExpireAt < userTokens[j].ExpireAt
			})
			extra := remainingTokens - int(maxConnPerUser)
			for i := 0; i < extra && i < len(userTokens); i++ {
				isInvalidated := false
				for _, t := range toInvalidate {
					if t.UUID == userTokens[i].UUID {
						isInvalidated = true
						break
					}
				}
				if !isInvalidated {
					toInvalidate = append(toInvalidate, userTokens[i])
				}
			}
		}
	}

	return toInvalidate, nil
}
