package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/zeromicro/go-zero/core/logx"
)

// requireUserIsAdmin 校验：userID 本人必须是 IM 管理员。
// 适用场景：取管理员 token、管理员自身操作等——原语义不变。
func requireUserIsAdmin(ctx context.Context, svcCtx *svc.ServiceContext, log logx.Logger, userID string) error {
	isAdmin, err := svcCtx.AuthVerify.IsIMAdmin(ctx, userID)
	if err != nil {
		log.Errorf("failed to check admin, userID: %s, err: %v", userID, err)
		return errx.InternalError.WrapWithError(err)
	}
	if !isAdmin {
		log.Errorf("user %s is not admin", userID)
		return errx.NoPermissionError
	}
	return nil
}

// requireUserIsValid 校验：userID 是真实存在的有效用户（未注销）。
func requireUserIsValid(ctx context.Context, svcCtx *svc.ServiceContext, log logx.Logger, userID string) error {
	isValid, err := svcCtx.AuthVerify.IsValidUser(ctx, userID)
	if err != nil {
		log.Errorf("failed to check user validity, userID: %s, err: %v", userID, err)
		return errx.InternalError.WrapWithError(err)
	}
	if !isValid {
		log.Errorf("user %s is not valid", userID)
		return errx.UserIDNotFoundError
	}
	return nil
}
