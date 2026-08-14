package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/group/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	sdkws "github.com/PaperMan11/goim/pkg/protocol/sdkws"
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

func modelToGroupInfo(group *model.Group) *sdkws.GroupInfo {
	if group == nil {
		return nil
	}
	return &sdkws.GroupInfo{
		GroupID:           group.GroupID,
		GroupName:         group.GroupName,
		Notification:      group.Notification,
		Introduction:      group.Introduction,
		FaceURL:           group.FaceURL,
		OwnerUserID:       group.OwnerUserID,
		CreateTime:        group.CreateTime.Unix(),
		MemberCount:       uint32(group.MemberCount),
		Ex:                group.Extra,
		Status:            int32(group.Status),
		NeedVerification:  int32(group.NeedVerification),
		LookMemberInfo:    int32(group.LookMemberInfo),
		ApplyMemberFriend: int32(group.ApplyMemberFriend),
	}
}

func modelToGroupMemberInfo(member *model.GroupMember) *sdkws.GroupMemberFullInfo {
	if member == nil {
		return nil
	}
	return &sdkws.GroupMemberFullInfo{
		GroupID:        member.GroupID,
		UserID:         member.UserID,
		RoleLevel:      int32(member.RoleLevel),
		JoinTime:       member.JoinTime.Unix(),
		Nickname:       member.Nickname,
		FaceURL:        member.FaceURL,
		AppMangerLevel: int32(member.AppManagerLevel),
		JoinSource:     int32(member.JoinSource),
		OperatorUserID: member.OperatorUserID,
		Ex:             member.Extra,
		MuteEndTime:    member.MuteEndTime.Unix(),
		InviterUserID:  member.InviterUserID,
	}
}
