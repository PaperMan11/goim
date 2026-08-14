// Package constant 定义了 OpenIM 系统中使用的各种常量
package constant

// 消息内容类型常量
// 用于标识不同类型的消息内容
const (

	// 内容类型起始值
	ContentTypeBegin = 100

	// 基础消息类型
	Text     = 101 // 文本消息
	Picture  = 102 // 图片消息
	Voice    = 103 // 语音消息
	Video    = 104 // 视频消息
	File     = 105 // 文件消息
	AtText   = 106 // @消息
	Merger   = 107 // 合并转发消息
	Card     = 108 // 名片消息
	Location = 109 // 位置消息
	Custom   = 110 // 自定义消息
	Revoke   = 111 // 撤回消息
	Typing   = 113 // 正在输入
	Quote    = 114 // 引用消息

	// 高级消息类型
	AdvancedText                 = 117 // 高级文本
	MarkdownText                 = 118 // Markdown 文本
	CustomNotTriggerConversation = 119 // 不触发会话的自定义消息
	CustomOnlineOnly             = 120 // 仅在线可见的自定义消息
	ReactionMessageModifier      = 121 // 表情回应修改
	ReactionMessageDeleter       = 122 // 表情回应删除

	// 通用消息类型
	Common             = 200 // 普通消息
	GroupMsg           = 201 // 群消息
	SignalMsg          = 202 // 信号消息
	CustomNotification = 203 // 自定义通知

	// 系统通知类型起始值
	NotificationBegin = 1000

	// 好友相关通知
	FriendApplicationApprovedNotification = 1201 // 好友申请已同意
	FriendApplicationRejectedNotification = 1202 // 好友申请已拒绝
	FriendApplicationNotification         = 1203 // 收到好友申请
	FriendAddedNotification               = 1204 // 已添加好友
	FriendDeletedNotification             = 1205 // 已删除好友
	FriendRemarkSetNotification           = 1206 // 好友备注已设置
	BlackAddedNotification                = 1207 // 已加入黑名单
	BlackDeletedNotification              = 1208 // 已移出黑名单
	FriendInfoUpdatedNotification         = 1209 // 好友信息已更新
	FriendsInfoUpdateNotification         = 1210 // 好友信息更新

	// 会话相关通知
	ConversationChangeNotification = 1300 // 会话变更

	// 用户相关通知起始值
	UserNotificationBegin = 1301

	UserInfoUpdatedNotification           = 1303 // 用户信息已更新
	UserStatusChangeNotification          = 1304 // 用户状态变更
	UserCommandAddNotification            = 1305 // 用户命令添加
	UserCommandDeleteNotification         = 1306 // 用户命令删除
	UserCommandUpdateNotification         = 1307 // 用户命令更新
	UserSubscribeOnlineStatusNotification = 1308 // 订阅用户在线状态

	// 用户相关通知结束值
	UserNotificationEnd = 1399
	OANotification      = 1400 // OA 通知

	// 群组相关通知起始值
	GroupNotificationBegin = 1500

	GroupCreatedNotification                 = 1501 // 群组已创建
	GroupInfoSetNotification                 = 1502 // 群组信息已设置
	JoinGroupApplicationNotification         = 1503 // 加群申请
	MemberQuitNotification                   = 1504 // 成员退出
	GroupApplicationAcceptedNotification     = 1505 // 群申请已接受
	GroupApplicationRejectedNotification     = 1506 // 群申请已拒绝
	GroupOwnerTransferredNotification        = 1507 // 群主已转让
	MemberKickedNotification                 = 1508 // 成员被踢
	MemberInvitedNotification                = 1509 // 成员被邀请
	MemberEnterNotification                  = 1510 // 成员进入群
	GroupDismissedNotification               = 1511 // 群组已解散
	GroupMemberMutedNotification             = 1512 // 群成员被禁言
	GroupMemberCancelMutedNotification       = 1513 // 群成员被取消禁言
	GroupMutedNotification                   = 1514 // 群组被禁言
	GroupCancelMutedNotification             = 1515 // 群组取消禁言
	GroupMemberInfoSetNotification           = 1516 // 群成员信息已设置
	GroupMemberSetToAdminNotification        = 1517 // 群成员设为管理员
	GroupMemberSetToOrdinaryUserNotification = 1518 // 群成员设为普通用户
	GroupInfoSetAnnouncementNotification     = 1519 // 群组公告已设置
	GroupInfoSetNameNotification             = 1520 // 群组名称已设置

	// 超级群相关通知
	SuperGroupNotificationBegin  = 1650 // 超级群通知起始
	SuperGroupUpdateNotification = 1651 // 超级群更新通知
	MsgDeleteNotification        = 1652 // 消息删除通知
	SuperGroupNotificationEnd    = 1699 // 超级群通知结束

	// 会话私有聊天相关通知
	ConversationPrivateChatNotification = 1701 // 会话私有聊天
	ConversationUnreadNotification      = 1702 // 会话未读
	ClearConversationNotification       = 1703 // 清空会话
	ConversationDeleteNotification      = 1704 // 删除会话

	// 业务相关通知
	BusinessNotificationBegin = 2000 // 业务通知起始
	BusinessNotification      = 2001 // 业务通知
	BusinessNotificationEnd   = 2099 // 业务通知结束

	// 消息撤回相关通知
	MsgRevokeNotification  = 2101 // 消息撤回
	DeleteMsgsNotification = 2102 // 删除消息

	HasReadReceipt = 2200 // 已读回执

	// 通知类型结束值
	NotificationEnd = 5000

	// 消息状态
	MsgNormal  = 1 // 正常消息
	MsgDeleted = 4 // 已删除消息

	// 消息来源类型
	UserMsgType = 100 // 用户消息
	SysMsgType  = 200 // 系统消息
)

// 消息内容类型到推送内容的映射表
// 用于将不同类型的消息转换为对应的推送文本提示
var ContentType2PushContent = map[int64]string{
	Picture:   "[PICTURE]",      // 图片消息推送文本
	Voice:     "[VOICE]",        // 语音消息推送文本
	Video:     "[VIDEO]",        // 视频消息推送文本
	File:      "[File]",         // 文件消息推送文本
	Text:      "[TEXT]",         // 文本消息推送文本
	AtText:    "[@TEXT]",        // @消息推送文本
	GroupMsg:  "[GROUPMSG]]",    // 群消息推送文本
	Common:    "[NEWMSG]",       // 普通消息推送文本
	SignalMsg: "[SIGNALINVITE]", // 信号消息推送文本
}
