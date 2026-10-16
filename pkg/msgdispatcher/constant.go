package msgdispatcher

import "github.com/PaperMan11/goim/pkg/protocol/constant"

const (
	MsgFromUser   int32 = constant.UserMsgType
	MsgFromSystem int32 = constant.SysMsgType

	SessionTypeSingle       int32 = constant.SingleChatType
	SessionTypeGroup        int32 = constant.ReadGroupChatType
	SessionTypeNotification int32 = constant.NotificationChatType

	ContentTypeText               int32 = constant.Text
	ContentTypeImage              int32 = constant.Picture
	ContentTypeVoice              int32 = constant.Voice
	ContentTypeVideo              int32 = constant.Video
	ContentTypeFile               int32 = constant.File
	ContentTypeAtText             int32 = constant.AtText
	ContentTypeMerger             int32 = constant.Merger
	ContentTypeCard               int32 = constant.Card
	ContentTypeLocation           int32 = constant.Location
	ContentTypeCustom             int32 = constant.Custom
	ContentTypeRevoke             int32 = constant.Revoke
	ContentTypeTyping             int32 = constant.Typing
	ContentTypeQuote              int32 = constant.Quote
	ContentTypeAdvancedText       int32 = constant.AdvancedText
	ContentTypeMarkdownText       int32 = constant.MarkdownText
	ContentTypeCommon             int32 = constant.Common
	ContentTypeGroupMsg           int32 = constant.GroupMsg
	ContentTypeSignalMsg          int32 = constant.SignalMsg
	ContentTypeCustomNotification int32 = constant.CustomNotification
	ContentTypeNotification       int32 = constant.CustomNotification
)

var DefaultNotification = Notification{
	GroupCreated:              NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupInfoSet:              NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	JoinGroupApplication:      NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	MemberQuit:                NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupApplicationAccepted:  NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupApplicationRejected:  NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupOwnerTransferred:     NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	MemberKicked:              NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	MemberInvited:             NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	MemberEnter:               NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupDismissed:            NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupMuted:                NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupCancelMuted:          NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupMemberMuted:          NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupMemberCancelMuted:    NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupMemberInfoSet:        NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupMemberSetToAdmin:     NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupMemberSetToOrdinary:  NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupInfoSetAnnouncement:  NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	GroupInfoSetName:          NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	FriendApplicationAdded:    NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	FriendApplicationApproved: NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	FriendApplicationRejected: NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	FriendAdded:               NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	FriendDeleted:             NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	FriendRemarkSet:           NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	BlackAdded:                NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	BlackDeleted:              NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.ReliableNotificationMsg, UnreadCount: true},
	// 以下为"实时信号"类通知：只用于多端同步/缓存刷新，不应进入会话历史，也不应计入未读。
	// 本项目中等价于"不落库"，离线端可通过会话列表、会话详情等接口补齐，无需靠消息流补发。
	FriendInfoUpdated:      NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.UnreliableNotification, UnreadCount: true},
	UserInfoUpdated:        NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.UnreliableNotification, UnreadCount: true},
	UserStatusChanged:      NotificationConfig{IsSendMsg: true, ReliabilityLevel: constant.UnreliableNotification, UnreadCount: true},
	ConversationChanged:    NotificationConfig{IsSendMsg: false, ReliabilityLevel: constant.UnreliableNotification, UnreadCount: false},
	ConversationSetPrivate: NotificationConfig{IsSendMsg: false, ReliabilityLevel: constant.UnreliableNotification, UnreadCount: false},
}

func BuildContentTypeMap(n *Notification) map[int32]*NotificationConfig {
	if n == nil {
		n = &DefaultNotification
	}
	return map[int32]*NotificationConfig{
		constant.GroupCreatedNotification:                 &n.GroupCreated,
		constant.GroupInfoSetNotification:                 &n.GroupInfoSet,
		constant.JoinGroupApplicationNotification:         &n.JoinGroupApplication,
		constant.MemberQuitNotification:                   &n.MemberQuit,
		constant.GroupApplicationAcceptedNotification:     &n.GroupApplicationAccepted,
		constant.GroupApplicationRejectedNotification:     &n.GroupApplicationRejected,
		constant.GroupOwnerTransferredNotification:        &n.GroupOwnerTransferred,
		constant.MemberKickedNotification:                 &n.MemberKicked,
		constant.MemberInvitedNotification:                &n.MemberInvited,
		constant.MemberEnterNotification:                  &n.MemberEnter,
		constant.GroupDismissedNotification:               &n.GroupDismissed,
		constant.GroupMutedNotification:                   &n.GroupMuted,
		constant.GroupCancelMutedNotification:             &n.GroupCancelMuted,
		constant.GroupMemberMutedNotification:             &n.GroupMemberMuted,
		constant.GroupMemberCancelMutedNotification:       &n.GroupMemberCancelMuted,
		constant.GroupMemberInfoSetNotification:           &n.GroupMemberInfoSet,
		constant.GroupMemberSetToAdminNotification:        &n.GroupMemberSetToAdmin,
		constant.GroupMemberSetToOrdinaryUserNotification: &n.GroupMemberSetToOrdinary,
		constant.GroupInfoSetAnnouncementNotification:     &n.GroupInfoSetAnnouncement,
		constant.GroupInfoSetNameNotification:             &n.GroupInfoSetName,
		constant.UserInfoUpdatedNotification:              &n.UserInfoUpdated,
		constant.UserStatusChangeNotification:             &n.UserStatusChanged,
		constant.FriendApplicationNotification:            &n.FriendApplicationAdded,
		constant.FriendApplicationApprovedNotification:    &n.FriendApplicationApproved,
		constant.FriendApplicationRejectedNotification:    &n.FriendApplicationRejected,
		constant.FriendAddedNotification:                  &n.FriendAdded,
		constant.FriendDeletedNotification:                &n.FriendDeleted,
		constant.FriendRemarkSetNotification:              &n.FriendRemarkSet,
		constant.BlackAddedNotification:                   &n.BlackAdded,
		constant.BlackDeletedNotification:                 &n.BlackDeleted,
		constant.FriendInfoUpdatedNotification:            &n.FriendInfoUpdated,
		constant.FriendsInfoUpdateNotification:            &n.FriendInfoUpdated,
		constant.ConversationChangeNotification:           &n.ConversationChanged,
		constant.ConversationUnreadNotification:           &n.ConversationChanged,
		constant.ConversationPrivateChatNotification:      &n.ConversationSetPrivate,
		// 撤回/删除通知需落库：客户端依赖它更新本地已缓存消息的撤回/删除状态。
		constant.MsgRevokeNotification:  {IsSendMsg: false, ReliabilityLevel: constant.ReliableNotificationNoMsg, UnreadCount: false},
		constant.DeleteMsgsNotification: {IsSendMsg: false, ReliabilityLevel: constant.ReliableNotificationNoMsg, UnreadCount: false},
		// 已读回执不落库：已读位置已记录在 SeqUser 上，客户端可直接查询；
		// 落库只会占用会话 seq、把"对方已读"变成自己的 +1 未读。
		constant.HasReadReceipt: {IsSendMsg: false, ReliabilityLevel: constant.UnreliableNotification, UnreadCount: false},
		// 用户指令为多端自同步信号（sendID == recvID），不进会话历史。
		constant.UserCommandAddNotification:    {IsSendMsg: false, ReliabilityLevel: constant.UnreliableNotification, UnreadCount: false},
		constant.UserCommandDeleteNotification: {IsSendMsg: false, ReliabilityLevel: constant.UnreliableNotification, UnreadCount: false},
		constant.UserCommandUpdateNotification: {IsSendMsg: false, ReliabilityLevel: constant.UnreliableNotification, UnreadCount: false},
	}
}

var ContentTypeMap = BuildContentTypeMap(nil)

var SessionTypeMap = map[int32]int32{
	constant.GroupCreatedNotification:                 constant.ReadGroupChatType,
	constant.GroupInfoSetNotification:                 constant.ReadGroupChatType,
	constant.JoinGroupApplicationNotification:         constant.SingleChatType,
	constant.MemberQuitNotification:                   constant.ReadGroupChatType,
	constant.GroupApplicationAcceptedNotification:     constant.SingleChatType,
	constant.GroupApplicationRejectedNotification:     constant.SingleChatType,
	constant.GroupOwnerTransferredNotification:        constant.ReadGroupChatType,
	constant.MemberKickedNotification:                 constant.ReadGroupChatType,
	constant.MemberInvitedNotification:                constant.ReadGroupChatType,
	constant.MemberEnterNotification:                  constant.ReadGroupChatType,
	constant.GroupDismissedNotification:               constant.ReadGroupChatType,
	constant.GroupMutedNotification:                   constant.ReadGroupChatType,
	constant.GroupCancelMutedNotification:             constant.ReadGroupChatType,
	constant.GroupMemberMutedNotification:             constant.ReadGroupChatType,
	constant.GroupMemberCancelMutedNotification:       constant.ReadGroupChatType,
	constant.GroupMemberInfoSetNotification:           constant.ReadGroupChatType,
	constant.GroupMemberSetToAdminNotification:        constant.ReadGroupChatType,
	constant.GroupMemberSetToOrdinaryUserNotification: constant.ReadGroupChatType,
	constant.GroupInfoSetAnnouncementNotification:     constant.ReadGroupChatType,
	constant.GroupInfoSetNameNotification:             constant.ReadGroupChatType,
	constant.UserInfoUpdatedNotification:              constant.SingleChatType,
	constant.UserStatusChangeNotification:             constant.SingleChatType,
	constant.FriendApplicationNotification:            constant.SingleChatType,
	constant.FriendApplicationApprovedNotification:    constant.SingleChatType,
	constant.FriendApplicationRejectedNotification:    constant.SingleChatType,
	constant.FriendAddedNotification:                  constant.SingleChatType,
	constant.FriendDeletedNotification:                constant.SingleChatType,
	constant.FriendRemarkSetNotification:              constant.SingleChatType,
	constant.BlackAddedNotification:                   constant.SingleChatType,
	constant.BlackDeletedNotification:                 constant.SingleChatType,
	constant.FriendInfoUpdatedNotification:            constant.SingleChatType,
	constant.FriendsInfoUpdateNotification:            constant.SingleChatType,
	constant.ConversationChangeNotification:           constant.SingleChatType,
	constant.ConversationUnreadNotification:           constant.SingleChatType,
	constant.ConversationPrivateChatNotification:      constant.SingleChatType,
	constant.DeleteMsgsNotification:                   constant.SingleChatType,
}
