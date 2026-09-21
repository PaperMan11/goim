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

func (s *NotificationSender) sendNotificationToAdmins(ctx context.Context, groupID string, contentType int32, notification proto.Message) error {
	adminIDs, err := s.getGroupAdminIDs(ctx, groupID)
	if err != nil {
		return err
	}
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	for _, recvID := range adminIDs {
		err = s.MsgDispatcher.SendNotification(ctx, opUserID, recvID, "", contentType, msgdispatcher.SessionTypeMap[contentType], notification)
		if err != nil {
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

// MemberQuitNotification 发送成员退出群聊通知
func (s *NotificationSender) MemberQuitNotification(ctx context.Context, group *model.Group, quitMember *model.GroupMember, version uint64, versionID string) error {
	tips := &sdkws.MemberQuitTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		QuitUser:             mconvert.ModelToPbGroupMemberInfo(quitMember),
		OperationTime:        time.Now().UnixMilli(),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.MemberQuitNotification), tips)
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

// MemberKickedNotification 发送成员被踢出群聊通知
func (s *NotificationSender) MemberKickedNotification(ctx context.Context, group *model.Group, kickedMember []*model.GroupMember, version uint64, versionID string) error {
	kickedUserList := make([]*sdkws.GroupMemberFullInfo, 0)
	for _, member := range kickedMember {
		kickedUserList = append(kickedUserList, mconvert.ModelToPbGroupMemberInfo(member))
	}
	tips := &sdkws.MemberKickedTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		KickedUserList:       kickedUserList,
		OperationTime:        time.Now().UnixMilli(),
		GroupMemberVersion:   version,
		GroupMemberVersionID: versionID,
	}
	return s.sendNotificationToGroup(ctx, group.GroupID, int32(constant.MemberKickedNotification), tips)
}

// AcceptGroupApplicationNotification 发送群聊申请接受通知
func (s *NotificationSender) AcceptGroupApplicationNotification(ctx context.Context, group *model.Group, opMember *model.GroupMember, userID, handleMsg string) error {
	applyReq, err := s.getGroupApplyRequest(ctx, group.GroupID, userID)
	if err != nil {
		return err
	}
	groupInfo := mconvert.ModelToPbGroupInfo(group)
	applyReq.GroupInfo = groupInfo
	tips := &sdkws.GroupApplicationAcceptedTips{
		Group:      groupInfo,
		OpUser:     mconvert.ModelToPbGroupMemberInfo(opMember),
		HandleMsg:  handleMsg,
		ReceiverAs: 0,
		Uuid:       group.GroupID,
		Request:    applyReq, // 申请详情
	}
	return s.sendNotificationToAdmins(ctx, group.GroupID, int32(constant.GroupApplicationAcceptedNotification), tips)
}

// RejectGroupApplicationNotification 发送群聊申请拒绝通知
func (s *NotificationSender) RejectGroupApplicationNotification(ctx context.Context, group *model.Group, opMember *model.GroupMember, userID, handleMsg string) error {
	applyReq, err := s.getGroupApplyRequest(ctx, group.GroupID, userID)
	if err != nil {
		return err
	}

	groupInfo := mconvert.ModelToPbGroupInfo(group)
	applyReq.GroupInfo = groupInfo
	tips := &sdkws.GroupApplicationRejectedTips{
		Group:      groupInfo,
		OpUser:     mconvert.ModelToPbGroupMemberInfo(opMember),
		HandleMsg:  handleMsg,
		ReceiverAs: 0,             // 接收者身份(1-管理员, 0-申请人)
		Uuid:       group.GroupID, // 申请唯一标识
		Request:    applyReq,      // 申请详情
	}
	return s.sendNotificationToAdmins(ctx, group.GroupID, int32(constant.GroupApplicationRejectedNotification), tips)
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

// GroupDismissedNotification 发送群聊解散通知
func (s *NotificationSender) GroupDismissedNotification(ctx context.Context, group *model.Group, groupOwner *model.GroupMember) error {
	tips := &sdkws.GroupDismissedTips{
		Group:         mconvert.ModelToPbGroupInfo(group),
		OpUser:        mconvert.ModelToPbGroupMemberInfo(groupOwner),
		OperationTime: time.Now().UnixMilli(),
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
func (s *NotificationSender) GroupMemberMutedNotification(ctx context.Context, group *model.Group, opUser, mutedUser *model.GroupMember, version uint64, versionID string) error {
	tips := &sdkws.GroupMemberMutedTips{
		Group:                mconvert.ModelToPbGroupInfo(group),
		OpUser:               mconvert.ModelToPbGroupMemberInfo(opUser),
		MutedUser:            mconvert.ModelToPbGroupMemberInfo(mutedUser),
		OperationTime:        time.Now().UnixMilli(),
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
