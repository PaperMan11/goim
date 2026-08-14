package logic

import (
	"context"
	"errors"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbgroup "github.com/PaperMan11/goim/pkg/protocol/group"
	sdkws "github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
	groupModel "github.com/PaperMan11/goim/pkg/storage/mongo/group"

	"github.com/PaperMan11/goim/pkg/utils/timex"
)

// ==================== 群组管理 ====================

func (l *Logic) CreateGroup(ctx context.Context, req *pbgroup.CreateGroupReq) (*pbgroup.CreateGroupResp, error) {
	groupInfo := req.GetGroupInfo()
	if groupInfo == nil {
		return nil, errx.ArgsError.Wrap("group info is required")
	}
	if groupInfo.GetGroupType() != constant.WorkingGroup {
		return nil, errx.ArgsError.Wrap("group type must be working group")
	}

	memberUserIDs := req.GetMemberUserIDs()
	adminUserIDs := req.GetAdminUserIDs()
	ownerUserID := req.GetOwnerUserID()

	now := timex.Now()
	group := &model.Group{
		GroupID:           groupInfo.GetGroupID(),
		GroupName:         groupInfo.GetGroupName(),
		Notification:      groupInfo.GetNotification(),
		Introduction:      groupInfo.GetIntroduction(),
		FaceURL:           groupInfo.GetFaceURL(),
		OwnerUserID:       ownerUserID,
		MemberCount:       len(memberUserIDs) + 1,
		Extra:             groupInfo.GetEx(),
		Status:            int(groupInfo.GetStatus()),
		GroupType:         int(groupInfo.GetGroupType()),
		NeedVerification:  int(groupInfo.GetNeedVerification()),
		LookMemberInfo:    int(groupInfo.GetLookMemberInfo()),
		ApplyMemberFriend: int(groupInfo.GetApplyMemberFriend()),
		CreatorUserID:     ownerUserID,
		CreateTime:        now,
		UpdatedAt:         now,
	}

	if err := l.svcCtx.GroupModel.InsertGroup(ctx, group); err != nil {
		l.Errorf("insert group failed, groupID: %s, err: %v", group.GroupID, err)
		return nil, err
	}

	var members []*model.GroupMember
	for _, userID := range memberUserIDs {
		roleLevel := constant.GroupOrdinaryUsers
		if userID == ownerUserID {
			roleLevel = constant.GroupOwner
		} else {
			for _, adminID := range adminUserIDs {
				if userID == adminID {
					roleLevel = constant.GroupAdmin
					break
				}
			}
		}
		members = append(members, &model.GroupMember{
			GroupID:        group.GroupID,
			UserID:         userID,
			RoleLevel:      roleLevel,
			Nickname:       groupInfo.GetGroupName(),
			JoinTime:       now,
			JoinSource:     1,
			OperatorUserID: ownerUserID,
			UpdatedAt:      now,
		})
	}

	if err := l.svcCtx.GroupModel.InsertMembers(ctx, members); err != nil {
		l.Errorf("insert members failed, groupID: %s, err: %v", group.GroupID, err)
		return nil, err
	}

	// 推进版本日志：群信息变更 + 成员批量新增（一次 version 推进，减少 round trip）
	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, group.GroupID, model.VersionGroupChangeID, model.VersionStateUpdate); err != nil {
		l.Errorf("incr version log for group change failed, groupID: %s, err: %v", group.GroupID, err)
	}
	if len(members) > 0 {
		userIDs := make([]string, 0, len(members))
		for _, member := range members {
			userIDs = append(userIDs, member.UserID)
		}
		if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, group.GroupID, userIDs, model.VersionStateInsert); err != nil {
			l.Errorf("incr version log batch for member insert failed, groupID: %s, err: %v", group.GroupID, err)
		}
	}

	return &pbgroup.CreateGroupResp{
		GroupInfo: modelToGroupInfo(group),
	}, nil
}

func (l *Logic) GetGroupsInfo(ctx context.Context, req *pbgroup.GetGroupsInfoReq) (*pbgroup.GetGroupsInfoResp, error) {
	groupIDs := req.GetGroupIDs()
	if len(groupIDs) == 0 {
		return &pbgroup.GetGroupsInfoResp{}, nil
	}

	groups, err := l.svcCtx.GroupModel.FindGroupsByIDs(ctx, groupIDs)
	if err != nil {
		l.Errorf("find groups by ids failed, groupIDs: %v, err: %v", groupIDs, err)
		return nil, err
	}

	var groupInfos []*sdkws.GroupInfo
	for _, group := range groups {
		groupInfos = append(groupInfos, modelToGroupInfo(group))
	}

	return &pbgroup.GetGroupsInfoResp{
		GroupInfos: groupInfos,
	}, nil
}

func (l *Logic) SetGroupInfo(ctx context.Context, req *pbgroup.SetGroupInfoReq) (*pbgroup.SetGroupInfoResp, error) {
	groupInfo := req.GetGroupInfoForSet()
	if groupInfo == nil {
		return nil, errx.ArgsError.Wrap("group info is required")
	}

	group, err := l.svcCtx.GroupModel.FindGroup(ctx, groupInfo.GetGroupID())
	if err != nil {
		l.Errorf("find group failed, groupID: %s, err: %v", groupInfo.GetGroupID(), err)
		return nil, err
	}

	now := timex.Now()
	group.GroupName = groupInfo.GetGroupName()
	group.FaceURL = groupInfo.GetFaceURL()
	group.Notification = groupInfo.GetNotification()
	group.Introduction = groupInfo.GetIntroduction()
	if ex := groupInfo.GetEx(); ex != nil {
		group.Extra = ex.GetValue()
	}
	if needVerification := groupInfo.GetNeedVerification(); needVerification != nil {
		group.NeedVerification = int(needVerification.GetValue())
	}
	if lookMemberInfo := groupInfo.GetLookMemberInfo(); lookMemberInfo != nil {
		group.LookMemberInfo = int(lookMemberInfo.GetValue())
	}
	if applyMemberFriend := groupInfo.GetApplyMemberFriend(); applyMemberFriend != nil {
		group.ApplyMemberFriend = int(applyMemberFriend.GetValue())
	}
	group.UpdatedAt = now

	if err := l.svcCtx.GroupModel.UpdateGroup(ctx, group); err != nil {
		l.Errorf("update group failed, groupID: %s, err: %v", group.GroupID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, group.GroupID, model.VersionGroupChangeID, model.VersionStateUpdate); err != nil {
		l.Errorf("incr version log for group change failed, groupID: %s, err: %v", group.GroupID, err)
	}

	return &pbgroup.SetGroupInfoResp{}, nil
}

func (l *Logic) SetGroupInfoEx(ctx context.Context, req *pbgroup.SetGroupInfoExReq) (*pbgroup.SetGroupInfoExResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	updates := make(map[string]any)
	if req.GroupName != nil {
		updates["group_name"] = req.GroupName.GetValue()
	}
	if req.Notification != nil {
		updates["notification"] = req.Notification.GetValue()
	}
	if req.Introduction != nil {
		updates["introduction"] = req.Introduction.GetValue()
	}
	if req.FaceURL != nil {
		updates["face_url"] = req.FaceURL.GetValue()
	}
	if req.Ex != nil {
		updates["extra"] = req.Ex.GetValue()
	}
	if req.NeedVerification != nil {
		updates["need_verification"] = req.NeedVerification.GetValue()
	}
	if req.LookMemberInfo != nil {
		updates["look_member_info"] = req.LookMemberInfo.GetValue()
	}
	if req.ApplyMemberFriend != nil {
		updates["apply_member_friend"] = req.ApplyMemberFriend.GetValue()
	}
	if len(updates) > 0 {
		updates["updated_at"] = timex.Now()
	}

	if len(updates) > 0 {
		if err := l.svcCtx.GroupModel.UpdateGroupEx(ctx, groupID, updates); err != nil {
			l.Errorf("update group ex failed, groupID: %s, err: %v", groupID, err)
			return nil, err
		}
		if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, model.VersionGroupChangeID, model.VersionStateUpdate); err != nil {
			l.Errorf("incr version log for group change failed, groupID: %s, err: %v", groupID, err)
		}
	}

	return &pbgroup.SetGroupInfoExResp{}, nil
}

func (l *Logic) GetGroups(ctx context.Context, req *pbgroup.GetGroupsReq) (*pbgroup.GetGroupsResp, error) {
	if req.GetGroupID() != "" {
		group, err := l.svcCtx.GroupModel.FindGroup(ctx, req.GetGroupID())
		if err != nil {
			l.Errorf("find group failed, groupID: %s, err: %v", req.GetGroupID(), err)
			return nil, err
		}
		cmsGroups := []*pbgroup.CMSGroup{{
			GroupInfo:        modelToGroupInfo(group),
			GroupOwnerUserID: group.OwnerUserID,
		}}
		return &pbgroup.GetGroupsResp{
			Total:  1,
			Groups: cmsGroups,
		}, nil
	}

	pagination := req.GetPagination()
	page := int64(pagination.GetPageNumber())
	size := int64(pagination.GetShowNumber())

	groups, total, err := l.svcCtx.GroupModel.PageGroups(ctx, page, size, req.GetGroupName())
	if err != nil {
		l.Errorf("page groups failed, err: %v", err)
		return nil, err
	}

	var cmsGroups []*pbgroup.CMSGroup
	for _, group := range groups {
		cmsGroups = append(cmsGroups, &pbgroup.CMSGroup{
			GroupInfo:        modelToGroupInfo(group),
			GroupOwnerUserID: group.OwnerUserID,
		})
	}

	return &pbgroup.GetGroupsResp{
		Total:  uint32(total),
		Groups: cmsGroups,
	}, nil
}

// ==================== 群成员管理 ====================

func (l *Logic) GetGroupMemberList(ctx context.Context, req *pbgroup.GetGroupMemberListReq) (*pbgroup.GetGroupMemberListResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	members, err := l.svcCtx.GroupModel.FindMembersByGroup(ctx, groupID)
	if err != nil {
		l.Errorf("find members by group failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	filter := req.GetFilter()
	keyword := req.GetKeyword()

	var filteredMembers []*model.GroupMember
	for _, member := range members {
		if filter != 0 {
			if filter == 1 && member.RoleLevel != constant.GroupOwner {
				continue
			}
			if filter == 2 && member.RoleLevel != constant.GroupAdmin {
				continue
			}
		}
		if keyword != "" && !containsKeyword(member.Nickname, keyword) {
			continue
		}
		filteredMembers = append(filteredMembers, member)
	}

	var memberInfos []*sdkws.GroupMemberFullInfo
	for _, member := range filteredMembers {
		memberInfos = append(memberInfos, modelToGroupMemberInfo(member))
	}

	return &pbgroup.GetGroupMemberListResp{
		Total:   uint32(len(memberInfos)),
		Members: memberInfos,
	}, nil
}

func (l *Logic) GetGroupMembersInfo(ctx context.Context, req *pbgroup.GetGroupMembersInfoReq) (*pbgroup.GetGroupMembersInfoResp, error) {
	groupID := req.GetGroupID()
	userIDs := req.GetUserIDs()

	if groupID == "" || len(userIDs) == 0 {
		return &pbgroup.GetGroupMembersInfoResp{}, nil
	}

	var memberInfos []*sdkws.GroupMemberFullInfo
	for _, userID := range userIDs {
		member, err := l.svcCtx.GroupModel.FindMember(ctx, groupID, userID)
		if err != nil {
			l.Errorf("find member failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
			continue
		}
		if member != nil {
			memberInfos = append(memberInfos, modelToGroupMemberInfo(member))
		}
	}

	return &pbgroup.GetGroupMembersInfoResp{
		Members: memberInfos,
	}, nil
}

func (l *Logic) GetJoinedGroupList(ctx context.Context, req *pbgroup.GetJoinedGroupListReq) (*pbgroup.GetJoinedGroupListResp, error) {
	userID := req.GetFromUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	members, err := l.svcCtx.GroupModel.FindMembersByUser(ctx, userID)
	if err != nil {
		l.Errorf("find members by user failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	groupIDs := make([]string, 0, len(members))
	seen := make(map[string]bool)
	for _, member := range members {
		if !seen[member.GroupID] {
			seen[member.GroupID] = true
			groupIDs = append(groupIDs, member.GroupID)
		}
	}

	groups, err := l.svcCtx.GroupModel.FindGroupsByIDs(ctx, groupIDs)
	if err != nil {
		l.Errorf("find groups by ids failed, groupIDs: %v, err: %v", groupIDs, err)
		return nil, err
	}

	var groupInfos []*sdkws.GroupInfo
	for _, group := range groups {
		groupInfos = append(groupInfos, modelToGroupInfo(group))
	}

	return &pbgroup.GetJoinedGroupListResp{
		Total:  uint32(len(groupInfos)),
		Groups: groupInfos,
	}, nil
}

func (l *Logic) GetGroupMemberUserIDs(ctx context.Context, req *pbgroup.GetGroupMemberUserIDsReq) (*pbgroup.GetGroupMemberUserIDsResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	userIDs, err := l.svcCtx.GroupModel.FindMemberIDsByGroup(ctx, groupID)
	if err != nil {
		l.Errorf("find member ids by group failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	return &pbgroup.GetGroupMemberUserIDsResp{
		UserIDs: userIDs,
	}, nil
}

func (l *Logic) GetUserInGroupMembers(ctx context.Context, req *pbgroup.GetUserInGroupMembersReq) (*pbgroup.GetUserInGroupMembersResp, error) {
	userID := req.GetUserID()
	groupIDs := req.GetGroupIDs()

	if userID == "" || len(groupIDs) == 0 {
		return &pbgroup.GetUserInGroupMembersResp{}, nil
	}

	var memberInfos []*sdkws.GroupMemberFullInfo
	for _, groupID := range groupIDs {
		member, err := l.svcCtx.GroupModel.FindMember(ctx, groupID, userID)
		if err != nil {
			l.Errorf("find member failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
			continue
		}
		if member != nil {
			memberInfos = append(memberInfos, modelToGroupMemberInfo(member))
		}
	}

	return &pbgroup.GetUserInGroupMembersResp{
		Members: memberInfos,
	}, nil
}

// ==================== 成员操作 ====================

func (l *Logic) JoinGroup(ctx context.Context, req *pbgroup.JoinGroupReq) (*pbgroup.JoinGroupResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	group, err := l.svcCtx.GroupModel.FindGroup(ctx, groupID)
	if err != nil {
		l.Errorf("find group failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	if group.Status != 0 {
		return nil, errx.ArgsError.Wrap("group is dismissed")
	}

	memberID := req.GetInviterUserID()
	isMember, err := l.svcCtx.GroupModel.IsMember(ctx, groupID, memberID)
	if err != nil {
		l.Errorf("check is member failed, groupID: %s, userID: %s, err: %v", groupID, memberID, err)
		return nil, err
	}
	if isMember {
		return nil, errx.ArgsError.Wrap("already a member")
	}

	now := timex.Now()
	member := &model.GroupMember{
		GroupID:        groupID,
		UserID:         memberID,
		RoleLevel:      constant.GroupOrdinaryUsers,
		JoinTime:       now,
		JoinSource:     int(req.GetJoinSource()),
		OperatorUserID: mcontext.GetOpUserIDFromContext(ctx),
		Extra:          req.GetEx(),
		UpdatedAt:      now,
	}

	if err := l.svcCtx.GroupModel.InsertMember(ctx, member); err != nil {
		l.Errorf("insert member failed, groupID: %s, userID: %s, err: %v", groupID, memberID, err)
		return nil, err
	}

	if err := l.svcCtx.GroupModel.IncrMemberCount(ctx, groupID, 1); err != nil {
		l.Errorf("incr member count failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, memberID, model.VersionStateInsert); err != nil {
		l.Errorf("incr version log for member insert failed, groupID: %s, userID: %s, err: %v", groupID, memberID, err)
	}
	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, memberID, groupID, model.VersionStateInsert); err != nil {
		l.Errorf("incr version log for member insert failed, groupID: %s, userID: %s, err: %v", groupID, memberID, err)
	}

	return &pbgroup.JoinGroupResp{}, nil
}

func (l *Logic) QuitGroup(ctx context.Context, req *pbgroup.QuitGroupReq) (*pbgroup.QuitGroupResp, error) {
	groupID := req.GetGroupID()
	userID := req.GetUserID()

	if groupID == "" || userID == "" {
		return nil, errx.ArgsError.Wrap("groupID and userID are required")
	}

	member, err := l.svcCtx.GroupModel.FindMember(ctx, groupID, userID)
	if err != nil {
		l.Errorf("find member failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
		return nil, err
	}

	if member.RoleLevel == constant.GroupOwner {
		return nil, errx.ArgsError.Wrap("owner cannot quit group")
	}

	if err := l.svcCtx.GroupModel.DeleteMember(ctx, groupID, userID); err != nil {
		l.Errorf("delete member failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
		return nil, err
	}

	if err := l.svcCtx.GroupModel.IncrMemberCount(ctx, groupID, -1); err != nil {
		l.Errorf("incr member count failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, userID, model.VersionStateDelete); err != nil {
		l.Errorf("incr version log for member delete failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
	}
	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, userID, groupID, model.VersionStateDelete); err != nil {
		l.Errorf("incr version log for member delete failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
	}

	return &pbgroup.QuitGroupResp{}, nil
}

func (l *Logic) InviteUserToGroup(ctx context.Context, req *pbgroup.InviteUserToGroupReq) (*pbgroup.InviteUserToGroupResp, error) {
	groupID := req.GetGroupID()
	invitedUserIDs := req.GetInvitedUserIDs()

	if groupID == "" || len(invitedUserIDs) == 0 {
		return nil, errx.ArgsError.Wrap("groupID and invitedUserIDs are required")
	}

	opUserID, _, err := l.requireGroupAdmin(ctx, groupID)
	if err != nil {
		return nil, err
	}

	group, err := l.svcCtx.GroupModel.FindGroup(ctx, groupID)
	if err != nil {
		l.Errorf("find group failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	needVerification := group.NeedVerification == constant.AllNeedVerification
	now := timex.Now()
	var members []*model.GroupMember
	var requests []*model.GroupRequest

	for _, userID := range invitedUserIDs {
		isMember, err2 := l.svcCtx.GroupModel.IsMember(ctx, groupID, userID)
		if err2 != nil {
			l.Errorf("check is member failed, groupID: %s, userID: %s, err: %v", groupID, userID, err2)
			return nil, err2
		}
		if isMember {
			continue
		}

		if needVerification {
			requests = append(requests, &model.GroupRequest{
				UserID:        userID,
				GroupID:       groupID,
				GroupName:     group.GroupName,
				GroupFaceURL:  group.FaceURL,
				HandleResult:  0,
				ReqMsg:        req.GetReason(),
				ReqTime:       now,
				JoinSource:    constant.JoinByInvitation,
				InviterUserID: opUserID,
			})
		} else {
			members = append(members, &model.GroupMember{
				GroupID:        groupID,
				UserID:         userID,
				RoleLevel:      constant.GroupOrdinaryUsers,
				JoinTime:       now,
				JoinSource:     constant.JoinByInvitation,
				OperatorUserID: opUserID,
				InviterUserID:  opUserID,
				UpdatedAt:      now,
			})
		}
	}

	if len(members) > 0 {
		if err := l.svcCtx.GroupModel.InsertMembers(ctx, members); err != nil {
			l.Errorf("insert members failed, groupID: %s, err: %v", groupID, err)
			return nil, err
		}
		if err := l.svcCtx.GroupModel.IncrMemberCount(ctx, groupID, len(members)); err != nil {
			l.Errorf("incr member count failed, groupID: %s, err: %v", groupID, err)
			return nil, err
		}
		if len(members) > 0 {
			userIDs := make([]string, 0, len(members))
			for _, member := range members {
				userIDs = append(userIDs, member.UserID)
			}
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, groupID, userIDs, model.VersionStateInsert); err != nil {
				l.Errorf("incr version log batch for member insert failed, groupID: %s, err: %v", groupID, err)
			}
		}
	}

	if len(requests) > 0 {
		for _, req := range requests {
			if err := l.svcCtx.RequestModel.InsertGroupRequest(ctx, req); err != nil {
				l.Errorf("insert group request failed, groupID: %s, userID: %s, err: %v", groupID, req.UserID, err)
				return nil, err
			}
		}
	}

	return &pbgroup.InviteUserToGroupResp{}, nil
}

func (l *Logic) KickGroupMember(ctx context.Context, req *pbgroup.KickGroupMemberReq) (*pbgroup.KickGroupMemberResp, error) {
	groupID := req.GetGroupID()
	kickedUserIDs := req.GetKickedUserIDs()

	if groupID == "" || len(kickedUserIDs) == 0 {
		return nil, errx.ArgsError.Wrap("groupID and kickedUserIDs are required")
	}

	opUserID, roleLevel, err := l.requireGroupAdmin(ctx, groupID)
	if err != nil {
		return nil, err
	}

	switch roleLevel {
	case constant.GroupOwner:
		for _, userID := range kickedUserIDs {
			if userID == opUserID {
				return nil, errx.ArgsError.Wrap("owner cannot be kicked")
			}
		}
	case constant.GroupAdmin:
		// 检测是否有成员是群主或群管理员
		kickedMembers, err := l.svcCtx.GroupModel.FindMembersByIDs(ctx, groupID, kickedUserIDs)
		if err != nil {
			l.Errorf("find members failed, groupID: %s, kickedUserIDs: %v, err: %v", groupID, kickedUserIDs, err)
			return nil, err
		}
		for _, member := range kickedMembers {
			if member.RoleLevel == constant.GroupOwner || member.RoleLevel == constant.GroupAdmin {
				return nil, errx.ArgsError.Wrap("owner or admin cannot be kicked")
			}
		}
	}

	if err := l.svcCtx.GroupModel.DeleteMembers(ctx, groupID, kickedUserIDs); err != nil {
		l.Errorf("delete members failed, groupID: %s, kickedUserIDs: %v, err: %v", groupID, kickedUserIDs, err)
		return nil, err
	}

	if err := l.svcCtx.GroupModel.IncrMemberCount(ctx, groupID, -len(kickedUserIDs)); err != nil {
		l.Errorf("incr member count failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	if len(kickedUserIDs) > 0 {
		if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, groupID, kickedUserIDs, model.VersionStateDelete); err != nil {
			l.Errorf("incr version log batch for member delete failed, groupID: %s, err: %v", groupID, err)
		}
		for _, userID := range kickedUserIDs {
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, userID, groupID, model.VersionStateDelete); err != nil {
				l.Errorf("incr version log for member delete failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
			}
		}
	}

	return &pbgroup.KickGroupMemberResp{}, nil
}

func (l *Logic) TransferGroupOwner(ctx context.Context, req *pbgroup.TransferGroupOwnerReq) (*pbgroup.TransferGroupOwnerResp, error) {
	groupID := req.GetGroupID()
	oldOwnerUserID := req.GetOldOwnerUserID()
	newOwnerUserID := req.GetNewOwnerUserID()
	now := timex.Now()

	if groupID == "" || oldOwnerUserID == "" || newOwnerUserID == "" {
		return nil, errx.ArgsError.Wrap("groupID, oldOwnerUserID and newOwnerUserID are required")
	}

	opUserID, _, err := l.requireGroupOwner(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if opUserID != oldOwnerUserID {
		return nil, errx.NoPermissionError
	}

	if err := l.svcCtx.GroupModel.UpdateGroupEx(ctx, groupID, map[string]any{
		"owner_user_id": newOwnerUserID,
		"updated_at":    now,
	}); err != nil {
		l.Errorf("update group owner failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	if err := l.svcCtx.GroupModel.UpdateMember(ctx, groupID, oldOwnerUserID, map[string]any{
		"role_level": constant.GroupAdmin,
		"updated_at": now,
	}); err != nil {
		l.Errorf("update old owner role failed, groupID: %s, userID: %s, err: %v", groupID, oldOwnerUserID, err)
		return nil, err
	}

	if err := l.svcCtx.GroupModel.UpdateMember(ctx, groupID, newOwnerUserID, map[string]any{
		"role_level": constant.GroupOwner,
		"updated_at": now,
	}); err != nil {
		l.Errorf("update new owner role failed, groupID: %s, userID: %s, err: %v", groupID, newOwnerUserID, err)
		return nil, err
	}

	sortEIDs := []string{model.VersionGroupChangeID, model.VersionSortChangeID, oldOwnerUserID, newOwnerUserID}
	if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, groupID, sortEIDs, model.VersionStateUpdate); err != nil {
		l.Errorf("incr version log batch for transfer owner failed, groupID: %s, err: %v", groupID, err)
	}

	return &pbgroup.TransferGroupOwnerResp{}, nil
}

// ==================== 群组操作 ====================

func (l *Logic) DismissGroup(ctx context.Context, req *pbgroup.DismissGroupReq) (*pbgroup.DismissGroupResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	_, _, err := l.requireGroupOwner(ctx, groupID)
	if err != nil {
		return nil, err
	}

	if err := l.svcCtx.GroupModel.UpdateGroupEx(ctx, groupID, map[string]any{
		"status":     constant.GroupStatusDismissed,
		"updated_at": timex.Now(),
	}); err != nil {
		l.Errorf("update group status failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	if req.GetDeleteMember() {
		memberIDs, _ := l.svcCtx.GroupModel.FindMemberIDsByGroup(ctx, groupID)
		_ = l.svcCtx.GroupModel.DeleteMembers(ctx, groupID, memberIDs)
		for _, userID := range memberIDs {
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, userID, groupID, model.VersionStateDelete); err != nil {
				l.Errorf("incr version log for member delete failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
			}
		}
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, model.VersionGroupChangeID, model.VersionStateUpdate); err != nil {
		l.Errorf("incr version log for group change failed, groupID: %s, err: %v", groupID, err)
	}

	return &pbgroup.DismissGroupResp{}, nil
}

func (l *Logic) MuteGroupMember(ctx context.Context, req *pbgroup.MuteGroupMemberReq) (*pbgroup.MuteGroupMemberResp, error) {
	groupID := req.GetGroupID()
	userID := req.GetUserID()
	mutedSeconds := req.GetMutedSeconds()

	if groupID == "" || userID == "" {
		return nil, errx.ArgsError.Wrap("groupID and userID are required")
	}

	opUserID, roleLevel, err := l.requireGroupAdmin(ctx, groupID)
	if err != nil {
		return nil, err
	}

	switch roleLevel {
	case constant.GroupOwner:
		if userID == opUserID {
			return nil, errx.ArgsError.Wrap("owner cannot be muted")
		}
	case constant.GroupAdmin:
		user, err := l.svcCtx.GroupModel.FindMember(ctx, groupID, userID)
		if err != nil {
			l.Errorf("find member failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
			return nil, err
		}
		if user.RoleLevel != constant.GroupOrdinaryUsers {
			return nil, errx.ArgsError.Wrap("only ordinary users cannot be muted")
		}
	default:
		return nil, errx.NoPermissionError
	}

	now := timex.Now()
	muteEndTime := timex.AddSeconds(now, int(mutedSeconds))
	if err := l.svcCtx.GroupModel.UpdateMember(ctx, groupID, userID, map[string]any{
		"mute_end_time": muteEndTime,
		"updated_at":    now,
	}); err != nil {
		l.Errorf("mute group member failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, userID, model.VersionStateUpdate); err != nil {
		l.Errorf("incr version log for member update failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
	}

	return &pbgroup.MuteGroupMemberResp{}, nil
}

func (l *Logic) CancelMuteGroupMember(ctx context.Context, req *pbgroup.CancelMuteGroupMemberReq) (*pbgroup.CancelMuteGroupMemberResp, error) {
	groupID := req.GetGroupID()
	userID := req.GetUserID()
	if groupID == "" || userID == "" {
		return nil, errx.ArgsError.Wrap("groupID and userID are required")
	}

	_, _, err := l.requireGroupOwner(ctx, groupID)
	if err != nil {
		return nil, err
	}

	if err := l.svcCtx.GroupModel.UpdateMember(ctx, groupID, userID, map[string]any{
		"mute_end_time": time.Unix(0, 0),
		"updated_at":    timex.Now(),
	}); err != nil {
		l.Errorf("cancel mute group member failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, userID, model.VersionStateUpdate); err != nil {
		l.Errorf("incr version log for member update failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
	}

	return &pbgroup.CancelMuteGroupMemberResp{}, nil
}

func (l *Logic) SetGroupMemberInfo(ctx context.Context, req *pbgroup.SetGroupMemberInfoReq) (*pbgroup.SetGroupMemberInfoResp, error) {
	members := req.GetMembers()
	if len(members) == 0 {
		return &pbgroup.SetGroupMemberInfoResp{}, nil
	}

	for _, member := range members {
		if _, _, err := l.requireGroupAdmin(ctx, member.GetGroupID()); err != nil {
			return nil, err
		}

		updates := make(map[string]any)
		if member.Nickname != nil {
			updates["nickname"] = member.Nickname.GetValue()
		}
		if member.FaceURL != nil {
			updates["face_url"] = member.FaceURL.GetValue()
		}
		if member.RoleLevel != nil {
			updates["role_level"] = member.RoleLevel.GetValue()
		}
		if member.Ex != nil {
			updates["extra"] = member.Ex.GetValue()
		}
		if len(updates) > 0 {
			updates["updated_at"] = timex.Now()
			if err := l.svcCtx.GroupModel.UpdateMember(ctx, member.GetGroupID(), member.GetUserID(), updates); err != nil {
				l.Errorf("update member info failed, groupID: %s, userID: %s, err: %v", member.GetGroupID(), member.GetUserID(), err)
				return nil, err
			}
			// role_level 变更会影响成员列表排序顺序，合并成员更新 + 排序变更为一次 batch（state 均为 Update）
			if member.RoleLevel != nil {
				if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, member.GetGroupID(), []string{member.GetUserID(), model.VersionSortChangeID}, model.VersionStateUpdate); err != nil {
					l.Errorf("incr version log batch for member+sort update failed, groupID: %s, userID: %s, err: %v", member.GetGroupID(), member.GetUserID(), err)
				}
			} else {
				if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, member.GetGroupID(), member.GetUserID(), model.VersionStateUpdate); err != nil {
					l.Errorf("incr version log for member update failed, groupID: %s, userID: %s, err: %v", member.GetGroupID(), member.GetUserID(), err)
				}
			}
		}
	}

	return &pbgroup.SetGroupMemberInfoResp{}, nil
}

func (l *Logic) MuteGroup(ctx context.Context, req *pbgroup.MuteGroupReq) (*pbgroup.MuteGroupResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	_, _, err := l.requireGroupOwner(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// 更新群组状态为禁言
	if err := l.svcCtx.GroupModel.UpdateGroupEx(ctx, groupID, map[string]any{
		"status":     constant.GroupStatusMuted,
		"updated_at": timex.Now(),
	}); err != nil {
		l.Errorf("update group status failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, model.VersionGroupChangeID, model.VersionStateUpdate); err != nil {
		l.Errorf("incr version log for group change failed, groupID: %s, err: %v", groupID, err)
	}

	return &pbgroup.MuteGroupResp{}, nil
}

func (l *Logic) CancelMuteGroup(ctx context.Context, req *pbgroup.CancelMuteGroupReq) (*pbgroup.CancelMuteGroupResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	_, _, err := l.requireGroupOwner(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// 更新群组状态为正常
	if err := l.svcCtx.GroupModel.UpdateGroupEx(ctx, groupID, map[string]any{
		"status":     constant.GroupOk,
		"updated_at": timex.Now(),
	}); err != nil {
		l.Errorf("update group status failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, model.VersionGroupChangeID, model.VersionStateUpdate); err != nil {
		l.Errorf("incr version log for group change failed, groupID: %s, err: %v", groupID, err)
	}

	return &pbgroup.CancelMuteGroupResp{}, nil
}

// ==================== 群组查询 ====================

func (l *Logic) GetGroupAbstractInfo(ctx context.Context, req *pbgroup.GetGroupAbstractInfoReq) (*pbgroup.GetGroupAbstractInfoResp, error) {
	groupIDs := req.GetGroupIDs()
	if len(groupIDs) == 0 {
		return &pbgroup.GetGroupAbstractInfoResp{}, nil
	}

	groups, err := l.svcCtx.GroupModel.FindGroupsByIDs(ctx, groupIDs)
	if err != nil {
		l.Errorf("find groups by ids failed, groupIDs: %v, err: %v", groupIDs, err)
		return nil, err
	}

	var abstractInfos []*pbgroup.GroupAbstractInfo
	for _, group := range groups {
		count, err2 := l.svcCtx.GroupModel.CountMembers(ctx, group.GroupID)
		if err2 != nil {
			l.Errorf("count members failed, groupID: %s, err: %v", group.GroupID, err2)
			count = 0
		}
		abstractInfos = append(abstractInfos, &pbgroup.GroupAbstractInfo{
			GroupID:           group.GroupID,
			GroupMemberNumber: uint32(count),
		})
	}

	return &pbgroup.GetGroupAbstractInfoResp{
		GroupAbstractInfos: abstractInfos,
	}, nil
}

func (l *Logic) GetGroupMembersCMS(ctx context.Context, req *pbgroup.GetGroupMembersCMSReq) (*pbgroup.GetGroupMembersCMSResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	members, err := l.svcCtx.GroupModel.FindMembersByGroup(ctx, groupID)
	if err != nil {
		l.Errorf("find members by group failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	var memberInfos []*sdkws.GroupMemberFullInfo
	for _, member := range members {
		if req.GetUserName() != "" && !containsKeyword(member.Nickname, req.GetUserName()) {
			continue
		}
		memberInfos = append(memberInfos, modelToGroupMemberInfo(member))
	}

	return &pbgroup.GetGroupMembersCMSResp{
		Total:   uint32(len(memberInfos)),
		Members: memberInfos,
	}, nil
}

// ==================== 群组申请 ====================

func (l *Logic) GroupApplicationResponse(ctx context.Context, req *pbgroup.GroupApplicationResponseReq) (*pbgroup.GroupApplicationResponseResp, error) {
	groupID := req.GetGroupID()
	fromUserID := req.GetFromUserID()
	handleResult := req.GetHandleResult()

	if groupID == "" || fromUserID == "" {
		return nil, errx.ArgsError.Wrap("groupID and fromUserID are required")
	}

	opUserID, _, err := l.requireGroupAdmin(ctx, groupID)
	if err != nil {
		return nil, err
	}

	request, err := l.svcCtx.RequestModel.FindGroupRequest(ctx, fromUserID, groupID)
	if err != nil {
		l.Errorf("find group request failed, groupID: %s, userID: %s, err: %v", groupID, fromUserID, err)
		return nil, err
	}

	modelHandleResult := 1
	if handleResult == 2 {
		modelHandleResult = -1
	}

	if err := l.svcCtx.RequestModel.HandleGroupRequest(ctx, fromUserID, groupID, opUserID, modelHandleResult, req.GetHandledMsg()); err != nil {
		l.Errorf("handle group request failed, groupID: %s, userID: %s, err: %v", groupID, fromUserID, err)
		return nil, err
	}

	if handleResult == constant.GroupResponseAgree {
		now := timex.Now()
		member := &model.GroupMember{
			GroupID:        groupID,
			UserID:         fromUserID,
			RoleLevel:      constant.GroupOrdinaryUsers,
			JoinTime:       now,
			JoinSource:     request.JoinSource,
			OperatorUserID: opUserID,
			InviterUserID:  request.InviterUserID,
			UpdatedAt:      now,
		}

		if err := l.svcCtx.GroupModel.InsertMember(ctx, member); err != nil {
			l.Errorf("insert member failed, groupID: %s, userID: %s, err: %v", groupID, fromUserID, err)
			return nil, err
		}

		if err := l.svcCtx.GroupModel.IncrMemberCount(ctx, groupID, 1); err != nil {
			l.Errorf("incr member count failed, groupID: %s, err: %v", groupID, err)
			return nil, err
		}

		if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, fromUserID, model.VersionStateInsert); err != nil {
			l.Errorf("incr version log for member insert failed, groupID: %s, userID: %s, err: %v", groupID, fromUserID, err)
		}

		if err := l.svcCtx.RequestModel.DeleteGroupRequest(ctx, fromUserID, groupID); err != nil {
			l.Errorf("delete group request failed, groupID: %s, userID: %s, err: %v", groupID, fromUserID, err)
		}
	}

	return &pbgroup.GroupApplicationResponseResp{}, nil
}

func (l *Logic) GetGroupApplicationList(ctx context.Context, req *pbgroup.GetGroupApplicationListReq) (*pbgroup.GetGroupApplicationListResp, error) {
	pagination := req.GetPagination()
	page := int64(pagination.GetPageNumber())
	size := int64(pagination.GetShowNumber())

	fromUserID := req.GetFromUserID()
	groupIDs := req.GetGroupIDs()

	if len(groupIDs) == 0 && fromUserID != "" {
		userGroups, err := l.svcCtx.GroupModel.FindMembersByUser(ctx, fromUserID)
		if err != nil {
			l.Errorf("find members by user failed, userID: %s, err: %v", fromUserID, err)
			return nil, err
		}
		for _, member := range userGroups {
			if member.RoleLevel >= constant.GroupAdmin {
				groupIDs = append(groupIDs, member.GroupID)
			}
		}
	}

	handleResults := make([]int, len(req.GetHandleResults()))
	for i, hr := range req.GetHandleResults() {
		handleResults[i] = int(hr)
	}

	total, err := l.svcCtx.RequestModel.CountGroupRequests(ctx, groupIDs, handleResults)
	if err != nil {
		l.Errorf("count group requests failed, err: %v", err)
		return nil, err
	}

	var allRequests []*model.GroupRequest
	for _, groupID := range groupIDs {
		requests, _, err2 := l.svcCtx.RequestModel.FindGroupRequestsByGroup(ctx, groupID, page, size)
		if err2 != nil {
			l.Errorf("find group requests by group failed, groupID: %s, err: %v", groupID, err2)
			continue
		}
		allRequests = append(allRequests, requests...)
	}

	var groupRequests []*sdkws.GroupRequest
	for _, req := range allRequests {
		groupRequests = append(groupRequests, modelToSDKGroupRequest(req))
	}

	return &pbgroup.GetGroupApplicationListResp{
		Total:         uint32(total),
		GroupRequests: groupRequests,
	}, nil
}

func (l *Logic) GetGroupApplicationUnhandledCount(ctx context.Context, req *pbgroup.GetGroupApplicationUnhandledCountReq) (*pbgroup.GetGroupApplicationUnhandledCountResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	groupIDs := make([]string, 0)
	members, err := l.svcCtx.GroupModel.FindMembersByUser(ctx, userID)
	if err != nil {
		l.Errorf("find members by user failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	for _, member := range members {
		if member.RoleLevel >= constant.GroupAdmin {
			groupIDs = append(groupIDs, member.GroupID)
		}
	}

	count, err := l.svcCtx.RequestModel.CountGroupRequests(ctx, groupIDs, []int{0})
	if err != nil {
		l.Errorf("count unhandled group requests failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	return &pbgroup.GetGroupApplicationUnhandledCountResp{
		Count: count,
	}, nil
}

func (l *Logic) GetUserReqApplicationList(ctx context.Context, req *pbgroup.GetUserReqApplicationListReq) (*pbgroup.GetUserReqApplicationListResp, error) {
	pagination := req.GetPagination()
	page := int64(pagination.GetPageNumber())
	size := int64(pagination.GetShowNumber())

	userID := mcontext.GetOpUserIDFromContext(l.ctx)
	requests, total, err := l.svcCtx.RequestModel.FindGroupRequestsByUser(ctx, userID, page, size)
	if err != nil {
		l.Errorf("find group requests by user failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	var groupRequests []*sdkws.GroupRequest
	for _, req := range requests {
		groupRequests = append(groupRequests, modelToSDKGroupRequest(req))
	}

	return &pbgroup.GetUserReqApplicationListResp{
		Total:         uint32(total),
		GroupRequests: groupRequests,
	}, nil
}

func (l *Logic) GetGroupUsersReqApplicationList(ctx context.Context, req *pbgroup.GetGroupUsersReqApplicationListReq) (*pbgroup.GetGroupUsersReqApplicationListResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	_, _, err := l.requireGroupAdmin(ctx, groupID)
	if err != nil {
		return nil, err
	}

	userIDs := req.GetUserIDs()
	var groupRequests []*sdkws.GroupRequest
	var total int64

	if len(userIDs) > 0 {
		for _, userID := range userIDs {
			request, err := l.svcCtx.RequestModel.FindGroupRequest(ctx, userID, groupID)
			if err != nil {
				continue
			}
			groupRequests = append(groupRequests, modelToSDKGroupRequest(request))
		}
		total = int64(len(groupRequests))
	} else {
		requests, t, err := l.svcCtx.RequestModel.FindGroupRequestsByGroup(ctx, groupID, 0, 0)
		if err != nil {
			l.Errorf("find group requests by group failed, groupID: %s, err: %v", groupID, err)
			return nil, err
		}
		for _, req := range requests {
			groupRequests = append(groupRequests, modelToSDKGroupRequest(req))
		}
		total = t
	}

	return &pbgroup.GetGroupUsersReqApplicationListResp{
		Total:         total,
		GroupRequests: groupRequests,
	}, nil
}

func (l *Logic) GetSpecifiedUserGroupRequestInfo(ctx context.Context, req *pbgroup.GetSpecifiedUserGroupRequestInfoReq) (*pbgroup.GetSpecifiedUserGroupRequestInfoResp, error) {
	userID := req.GetUserID()
	groupID := req.GetGroupID()

	if userID == "" || groupID == "" {
		return nil, errx.ArgsError.Wrap("userID and groupID are required")
	}

	request, err := l.svcCtx.RequestModel.FindGroupRequest(ctx, userID, groupID)
	if err != nil {
		l.Errorf("find group request failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
		return nil, err
	}

	return &pbgroup.GetSpecifiedUserGroupRequestInfoResp{
		Total:         1,
		GroupRequests: []*sdkws.GroupRequest{modelToSDKGroupRequest(request)},
	}, nil
}

func modelToSDKGroupRequest(req *model.GroupRequest) *sdkws.GroupRequest {
	if req == nil {
		return nil
	}
	return &sdkws.GroupRequest{
		UserInfo: &sdkws.PublicUserInfo{
			UserID:   req.UserID,
			Nickname: req.Nickname,
			FaceURL:  req.FaceURL,
		},
		GroupInfo: &sdkws.GroupInfo{
			GroupID:   req.GroupID,
			GroupName: req.GroupName,
			FaceURL:   req.GroupFaceURL,
		},
		HandleResult:  int32(req.HandleResult),
		ReqMsg:        req.ReqMsg,
		HandleMsg:     req.HandleMsg,
		ReqTime:       req.ReqTime.Unix(),
		HandleUserID:  req.HandleUserID,
		HandleTime:    req.HandleTime.Unix(),
		Ex:            req.Extra,
		JoinSource:    int32(req.JoinSource),
		InviterUserID: req.InviterUserID,
	}
}

// ==================== 群组设置 ====================

func (l *Logic) GetGroupMemberRoleLevel(ctx context.Context, req *pbgroup.GetGroupMemberRoleLevelReq) (*pbgroup.GetGroupMemberRoleLevelResp, error) {
	if len(req.GetRoleLevels()) == 0 {
		return nil, errx.ArgsError.Wrap("roleLevels is required")
	}
	_, _, err := l.requireGroupRole(ctx, req.GetGroupID(), constant.GroupAdmin)
	if err != nil {
		return nil, errx.NoPermissionError.Wrap("not admin")
	}

	groupMember, err := l.svcCtx.GroupModel.FindMembersByRoleLevels(ctx, req.GetGroupID(), req.GetRoleLevels())
	if err != nil {
		l.Errorf("find group member failed, groupID: %s, roleLevels: %v, err: %v", req.GetGroupID(), req.GetRoleLevels(), err)
		return nil, err
	}

	var groupMembers []*sdkws.GroupMemberFullInfo
	for _, member := range groupMember {
		groupMembers = append(groupMembers, modelToGroupMemberInfo(member))
	}
	return &pbgroup.GetGroupMemberRoleLevelResp{
		Members: groupMembers,
	}, nil
}

// ==================== 缓存相关 ====================

func (l *Logic) GetGroupInfoCache(ctx context.Context, req *pbgroup.GetGroupInfoCacheReq) (*pbgroup.GetGroupInfoCacheResp, error) {
	group, err := l.svcCtx.GroupModel.FindGroup(ctx, req.GetGroupID())
	if err != nil {
		l.Errorf("find group failed, groupID: %s, err: %v", req.GetGroupID(), err)
		return nil, err
	}
	return &pbgroup.GetGroupInfoCacheResp{
		GroupInfo: modelToGroupInfo(group),
	}, nil
}

func (l *Logic) GetGroupMemberCache(ctx context.Context, req *pbgroup.GetGroupMemberCacheReq) (*pbgroup.GetGroupMemberCacheResp, error) {
	groupMember, err := l.svcCtx.GroupModel.FindMember(ctx, req.GetGroupID(), req.GetGroupMemberID())
	if err != nil {
		l.Errorf("find group member failed, groupID: %s, groupMemberID: %s, err: %v", req.GetGroupID(), req.GetGroupMemberID(), err)
		return nil, err
	}
	return &pbgroup.GetGroupMemberCacheResp{
		Member: modelToGroupMemberInfo(groupMember),
	}, nil
}

func (l *Logic) GroupCreateCount(ctx context.Context, req *pbgroup.GroupCreateCountReq) (*pbgroup.GroupCreateCountResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	start := req.GetStart()
	end := req.GetEnd()

	// 统计总群数
	total, err := l.svcCtx.GroupModel.CountGroups(ctx)
	if err != nil {
		l.Errorf("count groups failed, err: %v", err)
		return nil, err
	}

	// 统计开始时间之前的群数
	var before int64
	if start > 0 {
		before, err = l.svcCtx.GroupModel.CountGroupsBefore(ctx, time.Unix(start, 0))
		if err != nil {
			l.Errorf("count groups before failed, err: %v", err)
			return nil, err
		}
	}

	// 统计时间段内每日创建数
	countMap := make(map[string]int64)
	if start > 0 && end > 0 {
		startTime := time.Unix(start, 0)
		endTime := time.Unix(end, 0)
		results, err := l.svcCtx.GroupModel.CountGroupsByTimeRange(ctx, startTime, endTime)
		if err != nil {
			l.Errorf("count groups by time range failed, err: %v", err)
			return nil, err
		}
		for _, r := range results {
			countMap[r.ID] = r.Count
		}
	}

	return &pbgroup.GroupCreateCountResp{
		Total:  total,
		Before: before,
		Count:  countMap,
	}, nil
}

func (l *Logic) NotificationUserInfoUpdate(ctx context.Context, req *pbgroup.NotificationUserInfoUpdateReq) (*pbgroup.NotificationUserInfoUpdateResp, error) {
	return &pbgroup.NotificationUserInfoUpdateResp{}, nil
}

// ==================== 增量同步 ====================
/*
	┌──────────────────────────────────────────────────────────────────────────────┐
	│                          客户端处理阶段                                       │
	└──────────────────────────────────────────────────────────────────────────────┘

	收到响应
					│
					▼
	┌─────────────────────────────────────────────────────┐
	│ 1. 处理 Update 列表                                  │
	│    - 更新本地 user_123 的好友信息                    │
	│                                                     │
	│ 2. 检查 SortVersion                                  │
	│    - if clientSortVersion != resp.SortVersion:      │
	│        重新对好友列表进行排序 (按 is_pinned 等字段)   │
	│        clientSortVersion = resp.SortVersion         │
	└─────────────────────────────────────────────────────┘
*/

// GetIncrementalGroupMember 获取群成员的增量变更。
// DID=groupID。使用 FindChangeLog（全有或全无语义）拉取变更：
//   - 文档不存在 / VersionID 不匹配 / 空 Logs（兼容性校验或超限） → 全量同步
//   - 有 Logs → 分类处理 insert/delete/update + 群信息变更
func (l *Logic) GetIncrementalGroupMember(ctx context.Context, req *pbgroup.GetIncrementalGroupMemberReq) (*pbgroup.GetIncrementalGroupMemberResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}
	clientVersion := uint(req.GetVersion())
	clientVersionID := req.GetVersionID()

	// 群是否存在
	group, err := l.svcCtx.GroupModel.FindGroup(ctx, groupID)
	if err != nil {
		if errors.Is(err, groupModel.ErrGroupNotFound) {
			return nil, errx.GroupNotFoundError
		}
		l.Errorf("find group failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}
	if group.Status == constant.GroupStatusDismissed {
		return nil, errx.DismissedAlreadyError
	}

	// 群组成员才能获取增量变更
	_, _, err = l.requireGroupRole(ctx, groupID, constant.GroupOrdinaryUsers)
	if err != nil {
		return nil, err
	}

	// FindChangeLog：全有或全无（limit=0 不限条数，文档不存在自动初始化并返回空 Logs）
	verLog, err := l.svcCtx.VersionLogModel.FindChangeLog(ctx, groupID, clientVersion, SyncLimit)
	if err != nil {
		l.Errorf("find change log failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	// 空 Logs → 全量同步（文档刚初始化 / 兼容性校验失败 / 变更数超限）
	if len(verLog.Logs) == 0 || clientVersionID != verLog.ID.Hex() {
		return l.fullGroupMemberResp(ctx, groupID)
	}

	// 增量同步：分类处理变更日志（FindChangeLog 已在 DB 端过滤 version > clientVersion）
	var (
		groupChanged bool
		sortChanged  bool
		sortVersion  uint64
		insertIDs    []string
		updateIDs    []string
		deleteIDs    []string
		seenInsert   = make(map[string]struct{})
		seenUpdate   = make(map[string]struct{})
		seenDelete   = make(map[string]struct{})
	)
	for _, log := range verLog.Logs {
		switch {
		case log.EID == model.VersionGroupChangeID:
			groupChanged = true
		case log.EID == model.VersionSortChangeID:
			// 排序变更：记录最新版本号，客户端据此重新排序成员列表
			sortChanged = true
			if uint64(log.Version) > sortVersion {
				sortVersion = uint64(log.Version)
			}
		case log.State == model.VersionStateInsert:
			if _, ok := seenInsert[log.EID]; !ok {
				seenInsert[log.EID] = struct{}{}
				insertIDs = append(insertIDs, log.EID)
			}
		case log.State == model.VersionStateDelete:
			if _, ok := seenDelete[log.EID]; !ok {
				seenDelete[log.EID] = struct{}{}
				deleteIDs = append(deleteIDs, log.EID)
			}
		case log.State == model.VersionStateUpdate:
			if _, ok := seenUpdate[log.EID]; !ok {
				seenUpdate[log.EID] = struct{}{}
				updateIDs = append(updateIDs, log.EID)
			}
		}
	}

	resp := &pbgroup.GetIncrementalGroupMemberResp{
		Version:   uint64(verLog.Version),
		VersionID: groupID,
		Full:      false,
		Delete:    deleteIDs,
	}
	if sortChanged {
		resp.SortVersion = sortVersion
	}

	// 拉取新增/更新成员的详情
	fetchIDs := append(append([]string{}, insertIDs...), updateIDs...)
	if len(fetchIDs) > 0 {
		members, err2 := l.svcCtx.GroupModel.FindMembersByIDs(ctx, groupID, fetchIDs)
		if err2 != nil {
			l.Errorf("find members by ids failed, groupID: %s, ids: %v, err: %v", groupID, fetchIDs, err2)
			return nil, err2
		}
		memberMap := make(map[string]*model.GroupMember, len(members))
		for _, m := range members {
			memberMap[m.UserID] = m
		}
		for _, id := range insertIDs {
			if m, ok := memberMap[id]; ok {
				resp.Insert = append(resp.Insert, modelToGroupMemberInfo(m))
			}
		}
		for _, id := range updateIDs {
			if m, ok := memberMap[id]; ok {
				resp.Update = append(resp.Update, modelToGroupMemberInfo(m))
			}
		}
	}

	// 群信息变更：附带最新群信息
	if groupChanged {
		resp.Group = modelToGroupInfo(group)
		// group, err2 := l.svcCtx.GroupModel.FindGroup(ctx, groupID)
		// if err2 != nil {
		// 	l.Errorf("find group failed, groupID: %s, err: %v", groupID, err2)
		// } else {
		// 	resp.Group = modelToGroupInfo(group)
		// }
	}

	return resp, nil
}

// fullGroupMemberResp 构造群成员全量同步响应
func (l *Logic) fullGroupMemberResp(ctx context.Context, groupID string) (*pbgroup.GetIncrementalGroupMemberResp, error) {
	members, err := l.svcCtx.GroupModel.FindMembersByGroup(ctx, groupID)
	if err != nil {
		l.Errorf("find members by group failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}
	inserts := make([]*sdkws.GroupMemberFullInfo, 0, len(members))
	for _, m := range members {
		inserts = append(inserts, modelToGroupMemberInfo(m))
	}
	var curVersion uint64
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, groupID); err2 == nil && verLog != nil {
		curVersion = uint64(verLog.Version)
	}
	return &pbgroup.GetIncrementalGroupMemberResp{
		Version:   curVersion,
		VersionID: groupID,
		Full:      true,
		Insert:    inserts,
		// 全量同步：SortVersion = 当前 version，客户端后续据此判断是否还需重新排序
		SortVersion: curVersion,
	}, nil
}

// BatchGetIncrementalGroupMember 批量获取多个群的成员增量变更。
func (l *Logic) BatchGetIncrementalGroupMember(ctx context.Context, req *pbgroup.BatchGetIncrementalGroupMemberReq) (*pbgroup.BatchGetIncrementalGroupMemberResp, error) {
	respList := make(map[string]*pbgroup.GetIncrementalGroupMemberResp, len(req.GetReqList()))
	for _, subReq := range req.GetReqList() {
		if subReq == nil || respList[subReq.GetGroupID()] != nil {
			continue
		}
		subResp, err := l.GetIncrementalGroupMember(ctx, subReq)
		if err != nil {
			l.Errorf("get incremental group member failed, groupID: %s, err: %v", subReq.GetGroupID(), err)
			continue
		}
		respList[subReq.GetGroupID()] = subResp
	}
	return &pbgroup.BatchGetIncrementalGroupMemberResp{
		RespList: respList,
	}, nil
}

// GetIncrementalJoinGroup 获取用户加入群组的增量变更。DID=userID。
// 使用 FindChangeLog（全有或全无语义）拉取变更，空 Logs → 全量同步。
func (l *Logic) GetIncrementalJoinGroup(ctx context.Context, req *pbgroup.GetIncrementalJoinGroupReq) (*pbgroup.GetIncrementalJoinGroupResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}
	clientVersion := uint(req.GetVersion())
	clientVersionID := req.GetVersionID()

	// FindChangeLog：全有或全无
	verLog, err := l.svcCtx.VersionLogModel.FindChangeLog(ctx, userID, clientVersion, SyncLimit)
	if err != nil {
		l.Errorf("find change log failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	// 空 Logs → 全量同步
	if len(verLog.Logs) == 0 || (clientVersionID != "" && clientVersionID != verLog.ID.Hex()) {
		return l.fullJoinGroupResp(ctx, userID)
	}

	// 增量同步：分类处理变更日志
	var (
		insertIDs = make([]string, 0)
		updateIDs = make([]string, 0)
		deleteIDs = make([]string, 0)
		seenIns   = make(map[string]struct{})
		seenUpd   = make(map[string]struct{})
		seenDel   = make(map[string]struct{})
	)
	for _, log := range verLog.Logs {
		switch log.State {
		case model.VersionStateInsert:
			if _, ok := seenIns[log.EID]; !ok {
				seenIns[log.EID] = struct{}{}
				insertIDs = append(insertIDs, log.EID)
			}
		case model.VersionStateDelete:
			if _, ok := seenDel[log.EID]; !ok {
				seenDel[log.EID] = struct{}{}
				deleteIDs = append(deleteIDs, log.EID)
			}
		case model.VersionStateUpdate:
			if _, ok := seenUpd[log.EID]; !ok {
				seenUpd[log.EID] = struct{}{}
				updateIDs = append(updateIDs, log.EID)
			}
		}
	}

	resp := &pbgroup.GetIncrementalJoinGroupResp{
		Version:   uint64(verLog.Version),
		VersionID: userID,
		Full:      false,
		Delete:    deleteIDs,
	}

	// 拉取新增/更新群组详情
	fetchIDs := append(append([]string{}, insertIDs...), updateIDs...)
	if len(fetchIDs) > 0 {
		groups, err2 := l.svcCtx.GroupModel.FindGroupsByIDs(ctx, fetchIDs)
		if err2 != nil {
			l.Errorf("find groups by ids failed, ids: %v, err: %v", fetchIDs, err2)
			return nil, err2
		}
		groupMap := make(map[string]*model.Group, len(groups))
		for _, g := range groups {
			groupMap[g.GroupID] = g
		}
		for _, id := range insertIDs {
			if g, ok := groupMap[id]; ok {
				resp.Insert = append(resp.Insert, modelToGroupInfo(g))
			}
		}
		for _, id := range updateIDs {
			if g, ok := groupMap[id]; ok {
				resp.Update = append(resp.Update, modelToGroupInfo(g))
			}
		}
	}

	return resp, nil
}

// fullJoinGroupResp 构造用户加入群的全量同步响应
func (l *Logic) fullJoinGroupResp(ctx context.Context, userID string) (*pbgroup.GetIncrementalJoinGroupResp, error) {
	members, err := l.svcCtx.GroupModel.FindMembersByUser(ctx, userID)
	if err != nil {
		l.Errorf("find members by user failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	groupIDs := make([]string, 0, len(members))
	for _, m := range members {
		groupIDs = append(groupIDs, m.GroupID)
	}
	var inserts []*sdkws.GroupInfo
	if len(groupIDs) > 0 {
		groups, err3 := l.svcCtx.GroupModel.FindGroupsByIDs(ctx, groupIDs)
		if err3 != nil {
			l.Errorf("find groups by ids failed, ids: %v, err: %v", groupIDs, err3)
			return nil, err3
		}
		inserts = make([]*sdkws.GroupInfo, 0, len(groups))
		for _, g := range groups {
			inserts = append(inserts, modelToGroupInfo(g))
		}
	}
	var curVersion uint64
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, userID); err2 == nil && verLog != nil {
		curVersion = uint64(verLog.Version)
	}
	return &pbgroup.GetIncrementalJoinGroupResp{
		Version:   curVersion,
		VersionID: userID,
		Full:      true,
		Insert:    inserts,
	}, nil
}

// GetFullGroupMemberUserIDs 返回群内全量成员ID列表，并通过哈希比对判断客户端是否已同步。
func (l *Logic) GetFullGroupMemberUserIDs(ctx context.Context, req *pbgroup.GetFullGroupMemberUserIDsReq) (*pbgroup.GetFullGroupMemberUserIDsResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	userIDs, err := l.svcCtx.GroupModel.FindMemberIDsByGroup(ctx, groupID)
	if err != nil {
		l.Errorf("find member ids by group failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}
	sort.Strings(userIDs)
	curHash := hashIDs(userIDs)

	resp := &pbgroup.GetFullGroupMemberUserIDsResp{
		Equal:   req.GetIdHash() != 0 && req.GetIdHash() == curHash,
		UserIDs: userIDs,
	}
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, groupID); err2 == nil && verLog != nil {
		resp.VersionID = verLog.ID.Hex()
		resp.Version = uint64(verLog.Version)
	} else if err2 != nil {
		l.Errorf("get version log failed, groupID: %s, err: %v", groupID, err2)
	}
	return resp, nil
}

// GetFullJoinGroupIDs 返回用户加入的全量群组ID列表，并通过哈希比对判断客户端是否已同步。
func (l *Logic) GetFullJoinGroupIDs(ctx context.Context, req *pbgroup.GetFullJoinGroupIDsReq) (*pbgroup.GetFullJoinGroupIDsResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	members, err := l.svcCtx.GroupModel.FindMembersByUser(ctx, userID)
	if err != nil {
		l.Errorf("find members by user failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	groupIDs := make([]string, 0, len(members))
	for _, m := range members {
		groupIDs = append(groupIDs, m.GroupID)
	}
	sort.Strings(groupIDs)
	curHash := hashIDs(groupIDs)

	resp := &pbgroup.GetFullJoinGroupIDsResp{
		Equal:    req.GetIdHash() != 0 && req.GetIdHash() == curHash,
		GroupIDs: groupIDs,
	}
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, userID); err2 == nil && verLog != nil {
		resp.VersionID = verLog.ID.Hex()
		resp.Version = uint64(verLog.Version)
	} else if err2 != nil {
		l.Errorf("get version log failed, userID: %s, err: %v", userID, err2)
	}
	return resp, nil
}

// hashIDs 计算有序ID列表的 FNV-1a 哈希，用于全量同步的等值比较。
func hashIDs(ids []string) uint64 {
	if len(ids) == 0 {
		return 0
	}
	h := fnv.New64a()
	h.Write([]byte(strings.Join(ids, ",")))
	return h.Sum64()
}

// ==================== 辅助函数 ====================

func containsKeyword(str, keyword string) bool {
	if str == "" || keyword == "" {
		return true
	}
	return len(str) >= len(keyword) && (str == keyword || containsIgnoreCase(str, keyword))
}

func containsIgnoreCase(str, keyword string) bool {
	for i := 0; i <= len(str)-len(keyword); i++ {
		if equalsIgnoreCase(str[i:i+len(keyword)], keyword) {
			return true
		}
	}
	return false
}

func equalsIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
