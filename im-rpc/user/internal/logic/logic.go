package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/user/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	sdkws "github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/zeromicro/go-zero/core/logx"
)

type Logic struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logx.Logger
}

func NewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *Logic {
	return &Logic{
		svcCtx: svcCtx,
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
	}
}

// requireSelfOrAdmin 校验：操作人必须是 targetUserID 本人，或是 IM 管理员。
// 校验失败时自动打 error log 并返回统一错误。
func (l *Logic) requireSelfOrAdmin(targetUserID string) error {
	opUserID := mcontext.GetOpUserIDFromContext(l.ctx)
	ok, err := l.svcCtx.AuthVerifier.CheckAccess(l.ctx, targetUserID)
	if err != nil {
		l.Errorf("check access failed, opUserID=%s targetUserID=%s err=%v", opUserID, targetUserID, err)
		return errx.InternalError.WrapWithError(err)
	}
	if !ok {
		l.Errorf("access denied, opUserID=%s targetUserID=%s", opUserID, targetUserID)
		return errx.NoPermissionError
	}
	return nil
}

// requireAdmin 校验：操作人必须是 IM 管理员。
// 校验失败时自动打 error log 并返回统一错误。
func (l *Logic) requireAdmin() error {
	opUserID := mcontext.GetOpUserIDFromContext(l.ctx)
	ok, err := l.svcCtx.AuthVerifier.IsIMAdmin(l.ctx, opUserID)
	if err != nil {
		l.Errorf("check admin failed, opUserID=%s err=%v", opUserID, err)
		return errx.InternalError.WrapWithError(err)
	}
	if !ok {
		l.Errorf("not admin, opUserID=%s", opUserID)
		return errx.NoPermissionError
	}
	return nil
}

// requireValidUser 校验：targetUserID 是真实存在的有效用户（未注销）。
// 校验失败时自动打 error log 并返回统一错误。
func (l *Logic) requireValidUser(targetUserID string) error {
	ok, err := l.svcCtx.AuthVerifier.IsValidUser(l.ctx, targetUserID)
	if err != nil {
		l.Errorf("check valid user failed, targetUserID=%s err=%v", targetUserID, err)
		return errx.InternalError.WrapWithError(err)
	}
	if !ok {
		l.Errorf("invalid user, targetUserID=%s", targetUserID)
		return errx.UserIDNotFoundError
	}
	return nil
}

func modelToUserInfo(user *model.User) *sdkws.UserInfo {
	return &sdkws.UserInfo{
		UserID:           user.UserID,
		Nickname:         user.Nickname,
		FaceURL:          user.FaceURL,
		Ex:               user.Extra,
		CreateTime:       user.CreatedAt.Unix(),
		AppMangerLevel:   int32(user.AppManagerLevel),
		GlobalRecvMsgOpt: int32(user.GlobalRecvMsgOpt),
	}
}

func userInfoToModel(userInfo *sdkws.UserInfo) *model.User {
	return &model.User{
		UserID:           userInfo.GetUserID(),
		Nickname:         userInfo.GetNickname(),
		FaceURL:          userInfo.GetFaceURL(),
		Extra:            userInfo.GetEx(),
		AppManagerLevel:  int(userInfo.GetAppMangerLevel()),
		GlobalRecvMsgOpt: int(userInfo.GetGlobalRecvMsgOpt()),
	}
}

func boolToStatus(b bool) int32 {
	if b {
		return constant.Registered
	}
	return constant.UnRegistered
}
