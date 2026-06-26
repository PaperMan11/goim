package authverify

import (
	"context"

	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
)

type AuthVerifyService interface {
	IsIMAdmin(ctx context.Context, userID string) (bool, error)
	IsValidUser(ctx context.Context, userID string) (bool, error)
}

type AuthVerify struct {
	userService userservice.UserService
}

func NewAuthVerify(userService userservice.UserService) *AuthVerify {
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
