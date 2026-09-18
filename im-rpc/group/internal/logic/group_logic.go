package logic

import (
	"context"
	"errors"
	"time"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/mconvert"
	"github.com/PaperMan11/goim/pkg/msgprocessor"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbgroup "github.com/PaperMan11/goim/pkg/protocol/group"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	sdkws "github.com/PaperMan11/goim/pkg/protocol/sdkws"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/storage/model"
	groupModel "github.com/PaperMan11/goim/pkg/storage/mongo/group"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/PaperMan11/goim/pkg/utils/hash"
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
		GroupInfo: mconvert.ModelToPbGroupInfo(group),
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
		groupInfos = append(groupInfos, mconvert.ModelToPbGroupInfo(group))
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
			GroupInfo:        mconvert.ModelToPbGroupInfo(group),
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
			GroupInfo:        mconvert.ModelToPbGroupInfo(group),
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
		memberInfos = append(memberInfos, mconvert.ModelToPbGroupMemberInfo(member))
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
			memberInfos = append(memberInfos, mconvert.ModelToPbGroupMemberInfo(member))
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
		groupInfos = append(groupInfos, mconvert.ModelToPbGroupInfo(group))
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
			memberInfos = append(memberInfos, mconvert.ModelToPbGroupMemberInfo(member))
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

	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}

	memberID := req.GetInviterUserID()
	userResp, err := l.svcCtx.UserService.GetDesignateUsers(ctx, &pbuser.GetDesignateUsersReq{
		UserIDs: []string{memberID},
	})
	if err != nil {
		logx.Errorf("get designate users failed, userID: %s, err: %v", memberID, err)
		return nil, err
	}
	if len(userResp.GetUsersInfo()) == 0 {
		logx.Errorf("get designate users failed, userID: %s, err: user not found", memberID)
		return nil, errx.UserIDNotFoundError
	}
	joinUser := userResp.GetUsersInfo()[0]

	isMember, err := l.svcCtx.GroupModel.IsMember(ctx, groupID, memberID)
	if err != nil {
		l.Errorf("check is member failed, groupID: %s, userID: %s, err: %v", groupID, memberID, err)
		return nil, err
	}
	if isMember {
		return nil, errx.ArgsError.Wrap("already a member")
	}

	now := timex.Now()
	if group.NeedVerification == constant.Directly {
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

		// set conversation user seq
		conversationID := msgprocessor.GetConversationIDBySessionType(constant.ReadGroupChatType, groupID)
		_, err = l.svcCtx.MsgService.SetUserConversationMaxSeq(ctx, &pbmsg.SetUserConversationMaxSeqReq{
			OwnerUserID:    []string{memberID},
			ConversationID: conversationID,
			MaxSeq:         0,
		})
		if err != nil {
			l.Errorf("set user conversation max seq failed, groupID: %s, userID: %s, err: %v", groupID, memberID, err)
			return nil, err
		}

		// version
		groupVersionLog, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, memberID, model.VersionStateInsert)
		if err != nil {
			l.Errorf("incr version log for member insert failed, groupID: %s, userID: %s, err: %v", groupID, memberID, err)
		}
		_, err = l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.JoinGroupDID(memberID), groupID, model.VersionStateInsert)
		if err != nil {
			l.Errorf("incr version log for member insert failed, groupID: %s, userID: %s, err: %v", groupID, memberID, err)
		}
		// send notification
		l.svcCtx.NotificationSender.MemberEnterNotification(ctx, group, member, uint64(groupVersionLog.Version), groupVersionLog.ID.String())
	} else {
		req := &model.GroupRequest{
			UserID:     memberID,
			ReqMsg:     req.ReqMessage,
			GroupID:    req.GroupID,
			JoinSource: int(req.GetJoinSource()),
			ReqTime:    time.Now(),
			HandleTime: now,
			Extra:      req.GetEx(),
		}
		err = l.svcCtx.RequestModel.InsertGroupRequest(ctx, req)
		if err != nil {
			l.Errorf("insert group request failed, groupID: %s, userID: %s, err: %v", groupID, memberID, err)
			return nil, err
		}
		// send notification
		l.svcCtx.NotificationSender.ApplyJoinGroupNotification(ctx, group, joinUser, req)
	}

	return &pbgroup.JoinGroupResp{}, nil
}

func (l *Logic) QuitGroup(ctx context.Context, req *pbgroup.QuitGroupReq) (*pbgroup.QuitGroupResp, error) {
	groupID := req.GetGroupID()
	userID := req.GetUserID()

	if groupID == "" || userID == "" {
		return nil, errx.ArgsError.Wrap("groupID and userID are required")
	}

	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}

	member, err := l.svcCtx.GroupModel.FindMember(ctx, groupID, userID)
	if err != nil {
		l.Errorf("find member failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
		return nil, err
	}
	if member.UserID == group.OwnerUserID {
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

	groupVersionLog, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, userID, model.VersionStateDelete)
	if err != nil {
		l.Errorf("incr version log for member delete failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
	}
	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.JoinGroupDID(userID), groupID, model.VersionStateDelete); err != nil {
		l.Errorf("incr version log for member delete failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
	}

	l.svcCtx.NotificationSender.MemberQuitNotification(ctx, group, member, uint64(groupVersionLog.Version), groupVersionLog.ID.String())

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

	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}

	inviterUser, err := l.svcCtx.GroupModel.FindMember(ctx, groupID, opUserID)
	if err != nil {
		if errors.Is(err, groupModel.ErrGroupMemberNotFound) {
			return nil, errx.NotInGroupYetError.Wrap("inviter user not found")
		}
		l.Errorf("find member failed, groupID: %s, userID: %s, err: %v", groupID, opUserID, err)
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

		userIDs := make([]string, 0, len(members))
		for _, member := range members {
			userIDs = append(userIDs, member.UserID)
		}
		groupVersionLog, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, groupID, userIDs, model.VersionStateInsert)
		if err != nil {
			l.Errorf("incr version log batch for member insert failed, groupID: %s, err: %v", groupID, err)
			return nil, err
		}
		l.svcCtx.NotificationSender.InviteGroupMemberNotification(ctx, group, inviterUser, members, uint64(groupVersionLog.Version), groupVersionLog.ID.String())
	}

	if len(requests) > 0 {
		for _, req := range requests {
			if err := l.svcCtx.RequestModel.InsertGroupRequest(ctx, req); err != nil {
				l.Errorf("insert group request failed, groupID: %s, userID: %s, err: %v", groupID, req.UserID, err)
				return nil, err
			}
		}
		getUsersResp, _ := l.svcCtx.UserService.GetDesignateUsers(ctx, &pbuser.GetDesignateUsersReq{UserIDs: invitedUserIDs})
		for _, userInfo := range getUsersResp.GetUsersInfo() {
			l.svcCtx.NotificationSender.ApplyJoinGroupNotification(ctx, group, userInfo, &model.GroupRequest{
				UserID:        userInfo.UserID,
				GroupID:       groupID,
				GroupName:     group.GroupName,
				GroupFaceURL:  group.FaceURL,
				HandleResult:  0,
				ReqMsg:        req.GetReason(),
				ReqTime:       now,
				JoinSource:    constant.JoinByInvitation,
				InviterUserID: inviterUser.UserID,
			})
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
	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}

	opUserID, roleLevel, err := l.requireGroupAdmin(ctx, groupID)
	if err != nil {
		return nil, err
	}

	var kickedMembers []*model.GroupMember
	switch roleLevel {
	case constant.GroupOwner:
		for _, userID := range kickedUserIDs {
			if userID == opUserID {
				return nil, errx.ArgsError.Wrap("owner cannot be kicked")
			}
		}
	case constant.GroupAdmin:
		// 检测是否有成员是群主或群管理员
		kickedMembers, err = l.svcCtx.GroupModel.FindMembersByIDs(ctx, groupID, kickedUserIDs)
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
		groupVersionLog, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, groupID, kickedUserIDs, model.VersionStateDelete)
		if err != nil {
			l.Errorf("incr version log batch for member delete failed, groupID: %s, err: %v", groupID, err)
			return nil, err
		}
		for _, userID := range kickedUserIDs {
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.JoinGroupDID(userID), groupID, model.VersionStateDelete); err != nil {
				l.Errorf("incr version log for member delete failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
			}
		}
		l.svcCtx.NotificationSender.MemberKickedNotification(ctx, group, kickedMembers, uint64(groupVersionLog.Version), groupVersionLog.ID.String())
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

	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// members
	members, err := l.svcCtx.GroupModel.FindMembersByIDs(ctx, groupID, []string{oldOwnerUserID, newOwnerUserID})
	if err != nil || len(members) != 2 {
		l.Errorf("find members failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}
	oldGroupOwner := members[0]
	newGroupOwner := members[1]
	if oldGroupOwner.UserID != oldOwnerUserID || newGroupOwner.UserID != newOwnerUserID {
		oldGroupOwner, newGroupOwner = newGroupOwner, oldGroupOwner
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
	groupVersionLog, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, groupID, sortEIDs, model.VersionStateUpdate)
	if err != nil {
		l.Errorf("incr version log batch for transfer owner failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	l.svcCtx.NotificationSender.GroupOwnerTransferredNotification(ctx, group, oldGroupOwner, newGroupOwner, uint64(groupVersionLog.Version), groupVersionLog.ID.String())

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

	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}

	groupOwner, err := l.svcCtx.GroupModel.FindMember(ctx, groupID, group.OwnerUserID)
	if err != nil {
		l.Errorf("find group owner failed, groupID: %s, err: %v", groupID, err)
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
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.JoinGroupDID(userID), groupID, model.VersionStateDelete); err != nil {
				l.Errorf("incr version log for member delete failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
			}
		}
	}

	_, err = l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, model.VersionGroupChangeID, model.VersionStateUpdate)
	if err != nil {
		l.Errorf("incr version log for group change failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	l.svcCtx.NotificationSender.GroupDismissedNotification(ctx, group, groupOwner)
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

	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// 检查用户是否在群中
	members, err := l.svcCtx.GroupModel.FindMembersByIDs(ctx, groupID, []string{userID, opUserID})
	if err != nil {
		l.Errorf("find members failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}
	if len(members) != 2 {
		return nil, errx.NotInGroupYetError
	}
	opUser := members[0]
	mutedUser := members[1]
	if opUser.UserID != opUserID {
		opUser, mutedUser = mutedUser, opUser
	}

	switch roleLevel {
	case constant.GroupOwner:
		if mutedUser.UserID == group.OwnerUserID {
			return nil, errx.ArgsError.Wrap("owner cannot be muted")
		}
	case constant.GroupAdmin:
		if mutedUser.RoleLevel != constant.GroupOrdinaryUsers {
			return nil, errx.ArgsError.Wrap("only ordinary users cannot be muted")
		}
	default:
		return nil, errx.NoPermissionError
	}

	now := timex.Now()
	muteEndTime := timex.AddSeconds(now, int(mutedSeconds))
	if err := l.svcCtx.GroupModel.UpdateMember(ctx, groupID, mutedUser.UserID, map[string]any{
		"mute_end_time": muteEndTime,
		"updated_at":    now,
	}); err != nil {
		l.Errorf("mute group member failed, groupID: %s, userID: %s, err: %v", groupID, mutedUser.UserID, err)
		return nil, err
	}

	groupVersionLog, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, mutedUser.UserID, model.VersionStateUpdate)
	if err != nil {
		l.Errorf("incr version log for member update failed, groupID: %s, userID: %s, err: %v", groupID, mutedUser.UserID, err)
		return nil, err
	}
	l.svcCtx.NotificationSender.GroupMemberMutedNotification(ctx, group, mutedUser, mutedUser, uint64(groupVersionLog.Version), groupVersionLog.ID.Hex())

	return &pbgroup.MuteGroupMemberResp{}, nil
}

func (l *Logic) CancelMuteGroupMember(ctx context.Context, req *pbgroup.CancelMuteGroupMemberReq) (*pbgroup.CancelMuteGroupMemberResp, error) {
	groupID := req.GetGroupID()
	userID := req.GetUserID()
	if groupID == "" || userID == "" {
		return nil, errx.ArgsError.Wrap("groupID and userID are required")
	}

	opUserID, _, err := l.requireGroupAdmin(ctx, groupID)
	if err != nil {
		return nil, err
	}

	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// 检查用户是否在群中
	members, err := l.svcCtx.GroupModel.FindMembersByIDs(ctx, groupID, []string{userID, opUserID})
	if err != nil {
		l.Errorf("find members failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}
	if len(members) != 2 {
		return nil, errx.NotInGroupYetError
	}
	opUser := members[0]
	mutedUser := members[1]
	if opUser.UserID != opUserID {
		opUser, mutedUser = mutedUser, opUser
	}

	if err := l.svcCtx.GroupModel.UpdateMember(ctx, groupID, mutedUser.UserID, map[string]any{
		"mute_end_time": time.Unix(0, 0),
		"updated_at":    timex.Now(),
	}); err != nil {
		l.Errorf("cancel mute group member failed, groupID: %s, userID: %s, err: %v", groupID, mutedUser.UserID, err)
		return nil, err
	}

	groupVersionLog, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, mutedUser.UserID, model.VersionStateUpdate)
	if err != nil {
		l.Errorf("incr version log for member update failed, groupID: %s, userID: %s, err: %v", groupID, mutedUser.UserID, err)
		return nil, err
	}
	l.svcCtx.NotificationSender.GroupMemberCancelMutedNotification(ctx, group, opUser, mutedUser, uint64(groupVersionLog.Version), groupVersionLog.ID.Hex())

	return &pbgroup.CancelMuteGroupMemberResp{}, nil
}

func (l *Logic) SetGroupMemberInfo(ctx context.Context, req *pbgroup.SetGroupMemberInfoReq) (*pbgroup.SetGroupMemberInfoResp, error) {
	members := req.GetMembers()
	if len(members) == 0 {
		return &pbgroup.SetGroupMemberInfoResp{}, nil
	}

	// group cache
	groupIDs := make([]string, 0, len(members))
	groupCache := make(map[string]*model.Group)
	for _, member := range members {
		groupIDs = append(groupIDs, member.GetGroupID())
	}
	groups, err := l.svcCtx.GroupModel.FindGroupsByIDs(ctx, groupIDs)
	if err != nil {
		l.Errorf("find groups failed, groupIDs: %v, err: %v", groupIDs, err)
		return nil, err
	}
	for _, group := range groups {
		if group.Status == constant.GroupStatusDismissed {
			continue
		}
		groupCache[group.GroupID] = group
	}

	// group member cache
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	groupMemberIDs := make(map[string][]string)
	groupMemberCache := make(map[string]*model.GroupMember) // key: groupID-userID
	for _, member := range members {
		groupMemberIDs[member.GetGroupID()] = append(groupMemberIDs[member.GetGroupID()], member.GetUserID())
	}
	for groupID, userIDs := range groupMemberIDs {
		groupMembers, err := l.svcCtx.GroupModel.FindMembersByIDs(ctx, groupID, append(userIDs, opUserID))
		if err != nil {
			l.Errorf("find members failed, groupID: %s, err: %v", groupID, err)
			return nil, err
		}
		for _, m := range groupMembers {
			groupMemberCache[groupID+"-"+m.UserID] = m
		}
	}

	for _, member := range members {
		opUser, ok := groupMemberCache[member.GetGroupID()+"-"+opUserID]
		if !ok {
			continue
		}
		groupMember, ok := groupMemberCache[member.GetGroupID()+"-"+member.GetUserID()]
		if !ok {
			continue
		}
		if !l.requireGroupPermission(ctx, opUser, groupMember) {
			continue
		}
		group, ok := groupCache[member.GetGroupID()]
		if !ok {
			continue
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
				_, err = l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, member.GetGroupID(), []string{member.GetUserID(), model.VersionSortChangeID}, model.VersionStateUpdate)
				if err != nil {
					l.Errorf("incr version log batch for member+sort update failed, groupID: %s, userID: %s, err: %v", member.GetGroupID(), member.GetUserID(), err)
					continue
				}
			}
			groupVersionLog, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, member.GetGroupID(), member.GetUserID(), model.VersionStateUpdate)
			if err != nil {
				l.Errorf("incr version log for member update failed, groupID: %s, userID: %s, err: %v", member.GetGroupID(), member.GetUserID(), err)
				continue
			}

			// send notification
			if member.RoleLevel != nil && member.RoleLevel.GetValue() == constant.GroupAdmin {
				l.svcCtx.NotificationSender.SetGroupAdminNotification(ctx, group, opUser, groupMember, uint64(groupVersionLog.GetSortVersion()), uint64(groupVersionLog.Version), groupVersionLog.ID.Hex())
			} else {
				l.svcCtx.NotificationSender.SetToOrdinaryUserNotification(ctx, group, opUser, groupMember, uint64(groupVersionLog.GetSortVersion()), uint64(groupVersionLog.Version), groupVersionLog.ID.Hex())
			}
			l.svcCtx.NotificationSender.UpdateGroupMemberInfoNotification(ctx, group, opUser, groupMember, uint64(groupVersionLog.GetSortVersion()), uint64(groupVersionLog.Version), groupVersionLog.ID.Hex())
		}
	}

	return &pbgroup.SetGroupMemberInfoResp{}, nil
}

func (l *Logic) MuteGroup(ctx context.Context, req *pbgroup.MuteGroupReq) (*pbgroup.MuteGroupResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group.Status == constant.GroupStatusMuted {
		return &pbgroup.MuteGroupResp{}, nil
	}

	opUserID, _, err := l.requireGroupOwner(ctx, groupID)
	if err != nil {
		return nil, err
	}
	opUser, err := l.svcCtx.GroupModel.FindMember(ctx, groupID, opUserID)
	if err != nil {
		l.Errorf("find member failed, groupID: %s, userID: %s, err: %v", groupID, opUserID, err)
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

	groupVersionLog, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, model.VersionGroupChangeID, model.VersionStateUpdate)
	if err != nil {
		l.Errorf("incr version log for group change failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}
	l.svcCtx.NotificationSender.GroupMutedNotification(ctx, group, opUser, uint64(groupVersionLog.Version), groupVersionLog.ID.Hex())

	return &pbgroup.MuteGroupResp{}, nil
}

func (l *Logic) CancelMuteGroup(ctx context.Context, req *pbgroup.CancelMuteGroupReq) (*pbgroup.CancelMuteGroupResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group.Status != constant.GroupStatusMuted {
		return &pbgroup.CancelMuteGroupResp{}, nil
	}

	opUserID, _, err := l.requireGroupOwner(ctx, groupID)
	if err != nil {
		return nil, err
	}
	opUser, err := l.svcCtx.GroupModel.FindMember(ctx, groupID, opUserID)
	if err != nil {
		l.Errorf("find member failed, groupID: %s, userID: %s, err: %v", groupID, opUserID, err)
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

	groupVersionLog, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, model.VersionGroupChangeID, model.VersionStateUpdate)
	if err != nil {
		l.Errorf("incr version log for group change failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}
	l.svcCtx.NotificationSender.GroupCancelMutedNotification(ctx, group, opUser, uint64(groupVersionLog.Version), groupVersionLog.ID.Hex())

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
		memberInfos = append(memberInfos, mconvert.ModelToPbGroupMemberInfo(member))
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

	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
	}

	opUserID, _, err := l.requireGroupAdmin(ctx, groupID)
	if err != nil {
		return nil, err
	}

	opUser, err := l.svcCtx.GroupModel.FindMember(ctx, groupID, opUserID)
	if err != nil {
		l.Errorf("find member failed, groupID: %s, userID: %s, err: %v", groupID, opUserID, err)
		return nil, err
	}

	request, err := l.svcCtx.RequestModel.FindGroupRequest(ctx, fromUserID, groupID)
	if err != nil {
		l.Errorf("find group request failed, groupID: %s, userID: %s, err: %v", groupID, fromUserID, err)
		return nil, err
	}

	modelHandleResult := constant.GroupResponseAgree
	if handleResult == 2 {
		modelHandleResult = constant.GroupResponseRefuse
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

		_, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, groupID, fromUserID, model.VersionStateInsert)
		if err != nil {
			l.Errorf("incr version log for member insert failed, groupID: %s, userID: %s, err: %v", groupID, fromUserID, err)
			return nil, err
		}
		l.svcCtx.NotificationSender.AcceptGroupApplicationNotification(ctx, group, opUser, fromUserID, req.GetHandledMsg())
	} else {
		l.svcCtx.NotificationSender.RejectGroupApplicationNotification(ctx, group, opUser, fromUserID, req.GetHandledMsg())
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
		groupRequests = append(groupRequests, mconvert.ModelToPbGroupRequest(req, nil, nil))
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
		groupRequests = append(groupRequests, mconvert.ModelToPbGroupRequest(req, nil, nil))
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
			groupRequests = append(groupRequests, mconvert.ModelToPbGroupRequest(request, nil, nil))
		}
		total = int64(len(groupRequests))
	} else {
		requests, t, err := l.svcCtx.RequestModel.FindGroupRequestsByGroup(ctx, groupID, 0, 0)
		if err != nil {
			l.Errorf("find group requests by group failed, groupID: %s, err: %v", groupID, err)
			return nil, err
		}
		for _, req := range requests {
			groupRequests = append(groupRequests, mconvert.ModelToPbGroupRequest(req, nil, nil))
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
		GroupRequests: []*sdkws.GroupRequest{mconvert.ModelToPbGroupRequest(request, nil, nil)},
	}, nil
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
		groupMembers = append(groupMembers, mconvert.ModelToPbGroupMemberInfo(member))
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
		GroupInfo: mconvert.ModelToPbGroupInfo(group),
	}, nil
}

func (l *Logic) GetGroupMemberCache(ctx context.Context, req *pbgroup.GetGroupMemberCacheReq) (*pbgroup.GetGroupMemberCacheResp, error) {
	groupMember, err := l.svcCtx.GroupModel.FindMember(ctx, req.GetGroupID(), req.GetGroupMemberID())
	if err != nil {
		l.Errorf("find group member failed, groupID: %s, groupMemberID: %s, err: %v", req.GetGroupID(), req.GetGroupMemberID(), err)
		return nil, err
	}
	return &pbgroup.GetGroupMemberCacheResp{
		Member: mconvert.ModelToPbGroupMemberInfo(groupMember),
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
	group, err := l.requireGroupNotDismissed(ctx, groupID)
	if err != nil {
		return nil, err
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
	c := model.ClassifyIncrementalLogs(verLog.Logs)

	resp := &pbgroup.GetIncrementalGroupMemberResp{
		Version:   uint64(verLog.Version),
		VersionID: groupID,
		Full:      false,
		Delete:    c.DeleteIDs,
	}
	if c.SortChanged {
		resp.SortVersion = c.SortVersion
	}

	// 拉取新增/更新成员的详情
	fetchIDs := append(append([]string{}, c.InsertIDs...), c.UpdateIDs...)
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
		for _, id := range c.InsertIDs {
			if m, ok := memberMap[id]; ok {
				resp.Insert = append(resp.Insert, mconvert.ModelToPbGroupMemberInfo(m))
			}
		}
		for _, id := range c.UpdateIDs {
			if m, ok := memberMap[id]; ok {
				resp.Update = append(resp.Update, mconvert.ModelToPbGroupMemberInfo(m))
			}
		}
	}

	// 群信息变更：附带最新群信息
	if c.GroupChanged {
		resp.Group = mconvert.ModelToPbGroupInfo(group)
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
		inserts = append(inserts, mconvert.ModelToPbGroupMemberInfo(m))
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
	verLog, err := l.svcCtx.VersionLogModel.FindChangeLog(ctx, model.JoinGroupDID(userID), clientVersion, SyncLimit)
	if err != nil {
		l.Errorf("find change log failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	// 空 Logs → 全量同步
	if len(verLog.Logs) == 0 || (clientVersionID != "" && clientVersionID != verLog.ID.Hex()) {
		return l.fullJoinGroupResp(ctx, userID)
	}

	// 增量同步：分类处理变更日志
	c := model.ClassifyIncrementalLogs(verLog.Logs)

	resp := &pbgroup.GetIncrementalJoinGroupResp{
		Version:   uint64(verLog.Version),
		VersionID: userID,
		Full:      false,
		Delete:    c.DeleteIDs,
	}

	// 拉取新增/更新群组详情
	fetchIDs := append(append([]string{}, c.InsertIDs...), c.UpdateIDs...)
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
		for _, id := range c.InsertIDs {
			if g, ok := groupMap[id]; ok {
				resp.Insert = append(resp.Insert, mconvert.ModelToPbGroupInfo(g))
			}
		}
		for _, id := range c.UpdateIDs {
			if g, ok := groupMap[id]; ok {
				resp.Update = append(resp.Update, mconvert.ModelToPbGroupInfo(g))
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
			inserts = append(inserts, mconvert.ModelToPbGroupInfo(g))
		}
	}
	var curVersion uint64
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, model.JoinGroupDID(userID)); err2 == nil && verLog != nil {
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
	curHash := hash.HashStringSet(userIDs)

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
	curHash := hash.HashStringSet(groupIDs)

	resp := &pbgroup.GetFullJoinGroupIDsResp{
		Equal:    req.GetIdHash() != 0 && req.GetIdHash() == curHash,
		GroupIDs: groupIDs,
	}
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, model.JoinGroupDID(userID)); err2 == nil && verLog != nil {
		resp.VersionID = verLog.ID.Hex()
		resp.Version = uint64(verLog.Version)
	} else if err2 != nil {
		l.Errorf("get version log failed, userID: %s, err: %v", userID, err2)
	}
	return resp, nil
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

func (l *Logic) IsGroupMember(ctx context.Context, req *pbgroup.IsGroupMemberReq) (*pbgroup.IsGroupMemberResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}
	isMember, err := l.svcCtx.GroupModel.IsMember(ctx, groupID, userID)
	if err != nil {
		l.Errorf("is member failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
		return nil, err
	}
	return &pbgroup.IsGroupMemberResp{
		IsMember: isMember,
	}, nil
}
