package notification

import (
	"context"
	"time"

	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/mconvert"
	"github.com/PaperMan11/goim/pkg/msgdispatcher"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/PaperMan11/goim/pkg/storage/model"
	groupModel "github.com/PaperMan11/goim/pkg/storage/mongo/group"
	requestModel "github.com/PaperMan11/goim/pkg/storage/mongo/request"
	versionModel "github.com/PaperMan11/goim/pkg/storage/mongo/versionlog"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/proto"
)

type NotificationSender struct {
	MsgDispatcher     msgdispatcher.MsgDispatcher
	MsgService        msgservice.MsgService
	UserService       userservice.UserService
	RequestModel      requestModel.RequestModel
	GroupModel        groupModel.GroupModel
	GroupVersionModel versionModel.VersionLogModel
}

func NewNotificationSender(
	msgService msgservice.MsgService,
	userService userservice.UserService,
	requestModel requestModel.RequestModel,
	groupModel groupModel.GroupModel,
	versionModel versionModel.VersionLogModel,
) *NotificationSender {
	return &NotificationSender{
		MsgDispatcher:     msgdispatcher.NewMsgDispatcher(msgService),
		MsgService:        msgService,
		UserService:       userService,
		RequestModel:      requestModel,
		GroupModel:        groupModel,
		GroupVersionModel: versionModel,
	}
}

func (s *NotificationSender) getGroupAdminIDs(ctx context.Context, groupID string) ([]string, error) {
	resp, err := s.GroupModel.FindMembersByRoleLevels(ctx, groupID, []int32{constant.GroupAdmin, constant.GroupOwner})
	if err != nil {
		logx.Errorf("find group members failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}
	adminIDs := make([]string, 0)
	for _, member := range resp {
		adminIDs = append(adminIDs, member.UserID)
	}
	return adminIDs, nil
}

func (s *NotificationSender) getGroupInfo(ctx context.Context, groupID string) (*sdkws.GroupInfo, error) {
	group, err := s.GroupModel.FindGroup(ctx, groupID)
	if err != nil {
		logx.Errorf("find group failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}
	return mconvert.ModelToPbGroupInfo(group), nil
}

func (s *NotificationSender) getGroupApplyRequest(ctx context.Context, groupID string, userID string) (*sdkws.GroupRequest, error) {
	applyReq, err := s.RequestModel.FindGroupRequest(ctx, userID, groupID)
	if err != nil {
		logx.Errorf("find group request failed, groupID: %s, userID: %s, err: %v", groupID, userID, err)
		return nil, err
	}
	resp, err := s.UserService.GetDesignateUsers(ctx, &pbuser.GetDesignateUsersReq{UserIDs: []string{userID}})
	if err != nil {
		logx.Errorf("get designate users failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	if len(resp.UsersInfo) == 0 {
		logx.Errorf("get designate users failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	applyMember := resp.UsersInfo[0]
	return mconvert.ModelToPbGroupRequest(applyReq, applyMember, nil), nil
}

// sendNotificationToUser 向单个用户投递通知
func (s *NotificationSender) sendNotificationToUser(ctx context.Context, recvID string, contentType int32, notification proto.Message) error {
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	return s.MsgDispatcher.SendNotification(ctx, opUserID, recvID, "", contentType, msgdispatcher.SessionTypeMap[contentType], notification)
}

func (s *NotificationSender) sendNotificationToAdmins(ctx context.Context, groupID string, contentType int32, notification proto.Message) error {
	adminIDs, err := s.getGroupAdminIDs(ctx, groupID)
	if err != nil {
		return err
	}
	for _, recvID := range adminIDs {
		if err := s.sendNotificationToUser(ctx, recvID, contentType, notification); err != nil {
			logx.Errorf("send notification failed, contentType: %d, recvID: %s, err: %v", contentType, recvID, err)
			return err
		}
	}
	return nil
}

func (s *NotificationSender) sendNotificationToGroup(ctx context.Context, groupID string, contentType int32, notification proto.Message) error {
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	return s.MsgDispatcher.SendNotification(ctx, opUserID, groupID, groupID, contentType, msgdispatcher.SessionTypeMap[contentType], notification)
}

// JoinGroupApplyNotification 发送加入群聊申请通知
func (s *NotificationSender) ApplyJoinGroupNotification(ctx context.Context, group *model.Group, joinMember *sdkws.UserInfo, applyReq *model.GroupRequest) error {
	// send notification
	return s.sendNotificationToAdmins(ctx, group.GroupID, int32(constant.JoinGroupApplicationNotification),
		mconvert.ModelToPbGroupRequest(applyReq, joinMember, mconvert.ModelToPbGroupInfo(group)))
}

// GroupCreatedNotification 发送群聊创建通知（携带成员列表与群成员版本，供客户端建群后增量同步）
func (s *NotificationSender) GroupCreatedNotification(ctx context.Context, group *model.Group, owner *model.GroupMember, members []*model.GroupMember, version uint64, versionID string) error {
	memberList := make([]*sdkws.GroupMemberFullInfo, 0, len(members))
	for _, member := range members {
		memberList = append(memberList, mconvert.ModelToPbGroupMemberInfo(member))
	}
	tips := &sdkws.GroupCreatedTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(owner), // 建群操作人即群主
		MemberList:           memberList,
		OperationTime:        time.Now().UnixMilli(),
		GroupOwnerUser:       mconvert.ModelToPbGroupMemberInfo(owner),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupCreatedNotification), tips)
}

// GroupInfoSetNotification 发送群资料变更通知（1502），载荷携带变更后的完整群信息
func (s *NotificationSender) GroupInfoSetNotification(ctx context.Context, group *model.Group, opUser *model.GroupMember, version uint64, versionID string) error {
	tips := &sdkws.GroupInfoSetTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(opUser),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupInfoSetNotification), tips)
}

// MemberQuitNotification 发送成员退出群聊通知。
// 退群成员发送通知前已被移出群成员列表，群维度推送无法触达其本人（含其他登录端），
// 因此额外按单聊通知补发给退群者。
func (s *NotificationSender) MemberQuitNotification(ctx context.Context, group *model.Group, quitMember *model.GroupMember, version uint64, versionID string) error {
	tips := &sdkws.MemberQuitTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		QuitUser:             mconvert.ModelToPbGroupMemberInfo(quitMember),
		OperationTime:        time.Now().UnixMilli(),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}

	groupErr := s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.MemberQuitNotification), tips)

	// 自投递到退群者本人，携带 GroupID 为空，覆盖其多端同步
	quitErr := s.MsgDispatcher.SendNotification(ctx, quitMember.UserID, quitMember.UserID, "", int32(constant.MemberQuitNotification), msgdispatcher.SessionTypeSingle, tips)

	if groupErr != nil {
		return groupErr
	}
	return quitErr
}

// MemberEnterNotification 发送成员加入群聊通知
func (s *NotificationSender) MemberEnterNotification(ctx context.Context, group *model.Group, enterMember *model.GroupMember, version uint64, versionID string) error {
	tips := &sdkws.MemberEnterTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		EntrantUser:          mconvert.ModelToPbGroupMemberInfo(enterMember),
		OperationTime:        time.Now().UnixMilli(),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.MemberEnterNotification), tips)
}

// MemberKickedNotification 发送成员被踢出群聊通知。
// 群维度通知按推送时的群成员列表投递，被踢成员此时已被移出群成员列表，
// 收不到群维度推送，因此额外按单聊通知逐个补发给被踢成员本人。
func (s *NotificationSender) MemberKickedNotification(ctx context.Context, group *model.Group, kickedMember []*model.GroupMember, version uint64, versionID string) error {
	kickedUserList := make([]*sdkws.GroupMemberFullInfo, 0, len(kickedMember))
	kickedUserIDs := make([]string, 0, len(kickedMember))
	for _, member := range kickedMember {
		kickedUserList = append(kickedUserList, mconvert.ModelToPbGroupMemberInfo(member))
		kickedUserIDs = append(kickedUserIDs, member.UserID)
	}
	tips := &sdkws.MemberKickedTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		KickedUserList:       kickedUserList,
		OperationTime:        time.Now().UnixMilli(),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	if opUserID := mcontext.GetOpUserIDFromContext(ctx); opUserID != "" {
		if opMember, err := s.GroupModel.FindMember(ctx, group.GroupID, opUserID); err != nil {
			logx.Errorf("find kick operator failed, groupID: %s, opUserID: %s, err: %v", group.GroupID, opUserID, err)
		} else {
			tips.OpUser = mconvert.ModelToPbGroupMemberInfo(opMember)
		}
	}

	groupErr := s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.MemberKickedNotification), tips)

	// 被踢成员已不在群内，需单独投递才能收到通知
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	kickedErr := s.MsgDispatcher.SendBatchNotification(ctx, opUserID, kickedUserIDs, int32(constant.MemberKickedNotification), msgdispatcher.SessionTypeSingle, tips)

	if groupErr != nil {
		return groupErr
	}
	return kickedErr
}

// AcceptGroupApplicationNotification 发送群聊申请接受通知。
// receiverAs: 1-管理员视角，0-申请人视角。两类接收者都必须通知：
// 申请人此时还不是群成员，收不到群维度通知，若只发管理员则申请人无法得知申请结果。
func (s *NotificationSender) AcceptGroupApplicationNotification(ctx context.Context, group *model.Group, opMember *model.GroupMember, userID, handleMsg string) error {
	applyReq, err := s.getGroupApplyRequest(ctx, group.GroupID, userID)
	if err != nil {
		return err
	}
	groupInfo := mconvert.ModelToPbGroupInfo(group)
	applyReq.GroupInfo = groupInfo

	adminIDs, err := s.getGroupAdminIDs(ctx, group.GroupID)
	if err != nil {
		return err
	}

	// 同一用户既是管理员又是申请人时，以申请人视角为准，避免重复投递
	receivers := make(map[string]int32, len(adminIDs)+1)
	for _, adminID := range adminIDs {
		receivers[adminID] = 1
	}
	receivers[userID] = 0

	for recvID, receiverAs := range receivers {
		tips := &sdkws.GroupApplicationAcceptedTips{
			Group:      groupInfo,
			OpUser:     mconvert.ModelToPbGroupMemberInfo(opMember),
			HandleMsg:  handleMsg,
			ReceiverAs: receiverAs,
			Uuid:       group.GroupID,
			Request:    applyReq, // 申请详情
		}
		if err := s.sendNotificationToUser(ctx, recvID, int32(constant.GroupApplicationAcceptedNotification), tips); err != nil {
			return err
		}
	}
	return nil
}

// RejectGroupApplicationNotification 发送群聊申请拒绝通知。
// receiverAs 语义同 AcceptGroupApplicationNotification：管理员=1、申请人=0，两类接收者都要通知。
func (s *NotificationSender) RejectGroupApplicationNotification(ctx context.Context, group *model.Group, opMember *model.GroupMember, userID, handleMsg string) error {
	applyReq, err := s.getGroupApplyRequest(ctx, group.GroupID, userID)
	if err != nil {
		return err
	}

	groupInfo := mconvert.ModelToPbGroupInfo(group)
	applyReq.GroupInfo = groupInfo

	adminIDs, err := s.getGroupAdminIDs(ctx, group.GroupID)
	if err != nil {
		return err
	}

	receivers := make(map[string]int32, len(adminIDs)+1)
	for _, adminID := range adminIDs {
		receivers[adminID] = 1
	}
	receivers[userID] = 0

	for recvID, receiverAs := range receivers {
		tips := &sdkws.GroupApplicationRejectedTips{
			Group:      groupInfo,
			OpUser:     mconvert.ModelToPbGroupMemberInfo(opMember),
			HandleMsg:  handleMsg,
			ReceiverAs: receiverAs,    // 接收者身份(1-管理员, 0-申请人)
			Uuid:       group.GroupID, // 申请唯一标识
			Request:    applyReq,      // 申请详情
		}
		if err := s.sendNotificationToUser(ctx, recvID, int32(constant.GroupApplicationRejectedNotification), tips); err != nil {
			return err
		}
	}
	return nil
}

// GroupOwnerTransferredNotification 发送群聊主转让通知
func (s *NotificationSender) GroupOwnerTransferredNotification(ctx context.Context, group *model.Group, oldGroupOwner, newGroupOwner *model.GroupMember, version uint64, versionID string) error {
	tips := &sdkws.GroupOwnerTransferredTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(oldGroupOwner),
		NewGroupOwner:        mconvert.ModelToPbGroupMemberInfo(newGroupOwner),
		OldGroupOwnerInfo:    mconvert.ModelToPbGroupMemberInfo(oldGroupOwner),
		OldGroupOwner:        oldGroupOwner.UserID,
		OperationTime:        time.Now().UnixMilli(),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupOwnerTransferredNotification), tips)
}

// GroupDismissedNotification 发送群聊解散通知。
// memberDeleted 为 true 时成员记录已被删除，群维度推送按成员列表扇出会无人可收，
// 此时改为按单聊逐个投递给解散前的全体成员。
func (s *NotificationSender) GroupDismissedNotification(ctx context.Context, group *model.Group, groupOwner *model.GroupMember, memberIDs []string, memberDeleted bool) error {
	tips := &sdkws.GroupDismissedTips{
		Group:         mconvert.ModelToPbGroupInfo(group),
		OpUser:        mconvert.ModelToPbGroupMemberInfo(groupOwner),
		OperationTime: time.Now().UnixMilli(),
	}
	if memberDeleted {
		opUserID := mcontext.GetOpUserIDFromContext(ctx)
		return s.MsgDispatcher.SendBatchNotification(ctx, opUserID, memberIDs, int32(constant.GroupDismissedNotification), msgdispatcher.SessionTypeSingle, tips)
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupDismissedNotification), tips)
}

// GroupMutedNotification 发送群聊禁言通知
func (s *NotificationSender) GroupMutedNotification(ctx context.Context, group *model.Group, opUser *model.GroupMember, version uint64, versionID string) error {
	tips := &sdkws.GroupMutedTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(opUser),
		OperationTime:        time.Now().UnixMilli(),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupMutedNotification), tips)
}

// GroupCancelMutedNotification 发送群聊取消禁言通知
func (s *NotificationSender) GroupCancelMutedNotification(ctx context.Context, group *model.Group, opUser *model.GroupMember, version uint64, versionID string) error {
	tips := &sdkws.GroupCancelMutedTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(opUser),
		OperationTime:        time.Now().UnixMilli(),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupCancelMutedNotification), tips)
}

// GroupMemberMutedNotification 发送群组成员禁言通知
func (s *NotificationSender) GroupMemberMutedNotification(ctx context.Context, group *model.Group, opUser, mutedUser *model.GroupMember, mutedSeconds uint32, version uint64, versionID string) error {
	tips := &sdkws.GroupMemberMutedTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(opUser),
		MutedUser:            mconvert.ModelToPbGroupMemberInfo(mutedUser),
		OperationTime:        time.Now().UnixMilli(),
		MutedSeconds:         mutedSeconds,
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupMemberMutedNotification), tips)
}

// GroupMemberCancelMutedNotification 发送群组成员取消禁言通知
func (s *NotificationSender) GroupMemberCancelMutedNotification(ctx context.Context, group *model.Group, opUser, mutedUser *model.GroupMember, version uint64, versionID string) error {
	tips := &sdkws.GroupMemberCancelMutedTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(opUser),
		MutedUser:            mconvert.ModelToPbGroupMemberInfo(mutedUser),
		OperationTime:        time.Now().UnixMilli(),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupMemberCancelMutedNotification), tips)
}

// SetGroupAdminNotification 发送群聊设置管理员通知
func (s *NotificationSender) SetGroupAdminNotification(ctx context.Context, group *model.Group, opUser, user *model.GroupMember, sortVersion, version uint64, versionID string) error {
	tips := &sdkws.GroupMemberInfoSetTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(opUser),
		ChangedUser:          mconvert.ModelToPbGroupMemberInfo(user),
		OperationTime:        time.Now().UnixMilli(),
		GroupSortVersion:     sortVersion,
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupMemberSetToAdminNotification), tips)
}

// SetToOrdinaryUserNotification 发送群聊设置为普通用户通知
func (s *NotificationSender) SetToOrdinaryUserNotification(ctx context.Context, group *model.Group, opUser, user *model.GroupMember, sortVersion, version uint64, versionID string) error {
	tips := &sdkws.GroupMemberInfoSetTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(opUser),
		ChangedUser:          mconvert.ModelToPbGroupMemberInfo(user),
		OperationTime:        time.Now().UnixMilli(),
		GroupSortVersion:     sortVersion,
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupMemberSetToOrdinaryUserNotification), tips)
}

// UpdateGroupMemberInfoNotification 发送群聊更新成员信息通知
func (s *NotificationSender) UpdateGroupMemberInfoNotification(ctx context.Context, group *model.Group, opUser, user *model.GroupMember, sortVersion, version uint64, versionID string) error {
	tips := &sdkws.GroupMemberInfoSetTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(opUser),
		ChangedUser:          mconvert.ModelToPbGroupMemberInfo(user),
		OperationTime:        time.Now().UnixMilli(),
		GroupSortVersion:     sortVersion,
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.GroupMemberInfoSetNotification), tips)
}

// InviteGroupMemberNotification 发送群聊邀请成员通知
func (s *NotificationSender) InviteGroupMemberNotification(ctx context.Context, group *model.Group, inviterUser *model.GroupMember, invitedUserList []*model.GroupMember, version uint64, versionID string) error {
	invitedUserListPb := make([]*sdkws.GroupMemberFullInfo, 0, len(invitedUserList))
	for _, user := range invitedUserList {
		invitedUserListPb = append(invitedUserListPb, mconvert.ModelToPbGroupMemberInfo(user))
	}
	tips := &sdkws.MemberInvitedTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(inviterUser),
		InviterUser:          mconvert.ModelToPbGroupMemberInfo(inviterUser),
		InvitedUserList:      invitedUserListPb,
		OperationTime:        time.Now().UnixMilli(),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.MemberInvitedNotification), tips)
}
