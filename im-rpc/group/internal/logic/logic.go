package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/group/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	SyncLimit = 200
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

func (l *Logic) getOpUserRole(ctx context.Context, groupID string) (int, error) {
	opUserID := mcontext.GetOpUserIDFromContext(l.ctx)
	roleLevel, err := l.svcCtx.GroupModel.GetMemberRole(ctx, groupID, opUserID)
	if err != nil {
		l.Errorf("get member role failed, groupID=%s userID=%s err=%v", groupID, opUserID, err)
		return 0, err
	}
	return roleLevel, nil
}

func (l *Logic) requireGroupRole(ctx context.Context, groupID string, minRoleLevel int) (string, int, error) {
	opUserID := mcontext.GetOpUserIDFromContext(l.ctx)
	roleLevel, err := l.svcCtx.GroupModel.GetMemberRole(ctx, groupID, opUserID)
	if err != nil {
		l.Errorf("get member role failed, groupID=%s userID=%s err=%v", groupID, opUserID, err)
		return opUserID, 0, err
	}
	if roleLevel < minRoleLevel {
		l.Errorf("insufficient permission, opUserID=%s roleLevel=%d required=%d", opUserID, roleLevel, minRoleLevel)
		return opUserID, roleLevel, errx.NoPermissionError
	}
	return opUserID, roleLevel, nil
}

func (l *Logic) requireGroupOwner(ctx context.Context, groupID string) (string, int, error) {
	return l.requireGroupRole(ctx, groupID, constant.GroupOwner)
}

func (l *Logic) requireGroupAdmin(ctx context.Context, groupID string) (string, int, error) {
	return l.requireGroupRole(ctx, groupID, constant.GroupAdmin)
}

// 检查群是否已解散
func (l *Logic) requireGroupNotDismissed(ctx context.Context, groupID string) (*model.Group, error) {
	group, err := l.svcCtx.GroupModel.FindGroup(ctx, groupID)
	if err != nil {
		l.Errorf("find group failed, groupID=%s err=%v", groupID, err)
		return nil, errx.InternalError.WrapWithError(err)
	}
	if group.Status == constant.GroupStatusDismissed {
		l.Errorf("group %s is dismissed", groupID)
		return nil, errx.GroupDismissedError
	}
	return group, nil
}

// 检测是否有权限
func (l *Logic) requireGroupPermission(ctx context.Context, admin, user *model.GroupMember) bool {
	switch admin.RoleLevel {
	case constant.GroupOwner:
		return admin.UserID != user.UserID
	case constant.GroupAdmin:
		return user.RoleLevel < constant.GroupAdmin && user.UserID != admin.UserID
	case constant.GroupOrdinaryUsers:
		return false
	default:
		return false
	}
}
