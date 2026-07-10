package authverify

import (
	"context"

	"github.com/PaperMan11/goim/pkg/mcontext"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/rpccache/userservice"
)

type AuthVerifyService interface {
	IsIMAdmin(ctx context.Context, userID string) (bool, error)
	IsValidUser(ctx context.Context, userID string) (bool, error)
	CheckAccess(ctx context.Context, userID string) (bool, error)
}

type AuthVerify struct {
	userService userservice.UserServiceWrapperCache
}

func NewAuthVerify(userService userservice.UserServiceWrapperCache) *AuthVerify {
	return &AuthVerify{
		userService: userService,
	}
}

// 查询用户是否是IM管理员
func (a *AuthVerify) IsIMAdmin(ctx context.Context, userID string) (bool, error) {
	resp, err := a.userService.IsIMAdmin(ctx, &pbuser.IsIMAdminReq{
		UserID: userID,
	})
	if err != nil {
		return false, err
	}
	return resp.IsIMAdmin, nil
}

// 查询用户是否有效
func (a *AuthVerify) IsValidUser(ctx context.Context, userID string) (bool, error) {
	resp, err := a.userService.AccountCheck(ctx, &pbuser.AccountCheckReq{
		CheckUserIDs: []string{userID},
	})
	if err != nil {
		return false, err
	}

	for _, result := range resp.Results {
		if result.UserID == userID {
			return result.AccountStatus == 0, nil
		}
	}
	return false, nil
}

// 检查用户是否有访问权限
func (a *AuthVerify) CheckAccess(ctx context.Context, userID string) (bool, error) {
	valid, err := a.IsValidUser(ctx, userID)
	if err != nil || !valid {
		return false, err
	}
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	if opUserID == userID {
		return true, nil
	}
	return a.IsIMAdmin(ctx, opUserID)
}
