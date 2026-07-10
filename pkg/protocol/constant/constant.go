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

	// 信令通知 (暂未启用)
	//SignalingNotificationBegin = 1600
	//SignalingNotification      = 1601
	//SignalingNotificationEnd   = 1649

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

	// 会话类型
	SingleChatType = 1 // 单聊
	// WriteGroupChatType Not enabled temporarily // 写扩散群聊 (暂未启用)
	WriteGroupChatType   = 2 // 写扩散群聊 (消息写入所有成员的收件箱)
	ReadGroupChatType    = 3 // 读扩散群聊 (消息只写入发件人收件箱，成员读取时同步)
	NotificationChatType = 4 // 通知会话

	// Token 状态
	NormalToken  = 0 // 正常 Token
	InValidToken = 1 // 无效 Token
	KickedToken  = 2 // 被踢 Token
	ExpiredToken = 3 // 过期 Token

	// 多端登录策略
	LoginStrategyAllowMulti          = "allow_multi"           // 允许全端登录，但同端互斥
	LoginStrategySingle              = "single"                // 允许单端登录
	LoginStrategyReplace             = "replace"               // 替换登录
	LoginStrategyReplaceSamePlatform = "replace_same_platform" // 替换相同平台登录

	// 在线状态
	Online  = 1 // 在线
	Offline = 0 // 离线

	// 注册状态
	Registered   = 1 // 已注册
	UnRegistered = 0 // 未注册

	// 消息接收选项
	ReceiveMessage          = 0 // 接收消息
	NotReceiveMessage       = 1 // 不收消息
	ReceiveNotNotifyMessage = 2 // 接收但不通知

	// 消息选项键
	IsHistory                  = "history"                  // 是否历史消息
	IsPersistent               = "persistent"               // 是否持久化
	IsOfflinePush              = "offlinePush"              // 是否离线推送
	IsUnreadCount              = "unreadCount"              // 是否计入未读数
	IsConversationUpdate       = "conversationUpdate"       // 是否更新会话
	IsSenderSync               = "senderSync"               // 是否发送端同步
	IsNotPrivate               = "notPrivate"               // 是否非私有
	IsSenderConversationUpdate = "senderConversationUpdate" // 是否发送端会话更新
	IsSenderNotificationPush   = "senderNotificationPush"   // 是否发送端通知推送
	IsReactionFromCache        = "reactionFromCache"        // 是否从缓存获取回应
	IsNotNotification          = "isNotNotification"        // 是否非通知
	IsSendMsg                  = "isSendMsg"                // 是否发送消息

	// 群组状态
	GroupOk              = 0 // 正常
	GroupBanChat         = 1 // 禁止发言
	GroupStatusDismissed = 2 // 已解散
	GroupStatusMuted     = 3 // 已禁言

	// 群组类型
	NormalGroup  = 0 // 普通群
	SuperGroup   = 1 // 超级群
	WorkingGroup = 2 // 工作群

	GroupBaned          = 3 // 已封禁
	GroupBanPrivateChat = 4 // 禁止私聊

	// 用户加入群组的来源
	JoinByAdmin      = 1 // 管理员添加
	JoinByInvitation = 2 // 邀请加入
	JoinBySearch     = 3 // 搜索加入
	JoinByQRCode     = 4 // 扫码加入

	// 对象存储超时时间 (秒)
	MinioDurationTimes = 3600 // Minio 预签名 URL 有效期
	AwsDurationTimes   = 3600 // AWS 预签名 URL 有效期

	// 回调命令
	CallbackBeforeSendSingleMsgCommand                   = "callbackBeforeSendSingleMsgCommand"                   // 发送单聊消息前回调
	CallbackAfterSendSingleMsgCommand                    = "callbackAfterSendSingleMsgCommand"                    // 发送单聊消息后回调
	CallbackBeforeSendGroupMsgCommand                    = "callbackBeforeSendGroupMsgCommand"                    // 发送群消息前回调
	CallbackAfterSendGroupMsgCommand                     = "callbackAfterSendGroupMsgCommand"                     // 发送群消息后回调
	CallbackMsgModifyCommand                             = "callbackMsgModifyCommand"                             // 消息修改回调
	CallbackUserOnlineCommand                            = "callbackUserOnlineCommand"                            // 用户上线回调
	CallbackUserOfflineCommand                           = "callbackUserOfflineCommand"                           // 用户下线回调
	CallbackUserKickOffCommand                           = "callbackUserKickOffCommand"                           // 用户被踢回调
	CallbackOfflinePushCommand                           = "callbackOfflinePushCommand"                           // 离线推送回调
	CallbackOnlinePushCommand                            = "callbackOnlinePushCommand"                            // 在线推送回调
	CallbackSuperGroupOnlinePushCommand                  = "callbackSuperGroupOnlinePushCommand"                  // 超级群在线推送回调
	CallbackBeforeAddFriendCommand                       = "callbackBeforeAddFriendCommand"                       // 添加好友前回调
	CallbackBeforeUpdateUserInfoCommand                  = "callbackBeforeUpdateUserInfoCommand"                  // 更新用户信息前回调
	CallbackBeforeCreateGroupCommand                     = "callbackBeforeCreateGroupCommand"                     // 创建群前回调
	CallbackBeforeMemberJoinGroupCommand                 = "callbackBeforeMemberJoinGroupCommand"                 // 成员加群前回调
	CallbackBeforeSetGroupMemberInfoCommand              = "CallbackBeforeSetGroupMemberInfoCommand"              // 设置群成员信息前回调
	CallbackBeforeSetMessageReactionExtensionCommand     = "callbackBeforeSetMessageReactionExtensionCommand"     // 设置消息表情回应前回调
	CallbackBeforeDeleteMessageReactionExtensionsCommand = "callbackBeforeDeleteMessageReactionExtensionsCommand" // 删除消息表情回应前回调
	CallbackGetMessageListReactionExtensionsCommand      = "callbackGetMessageListReactionExtensionsCommand"      // 获取消息列表表情回应回调
	CallbackAddMessageListReactionExtensionsCommand      = "callbackAddMessageListReactionExtensionsCommand"      // 添加消息列表表情回应回调

	// 回调动作代码
	ActionAllow     = 0 // 允许
	ActionForbidden = 1 // 禁止

	// 回调处理代码
	CallbackHandleSuccess = 0 // 处理成功
	CallbackHandleFailed  = 1 // 处理失败

	// 文件上传类型
	OtherType = 1 // 其他类型
	VideoType = 2 // 视频类型
	ImageType = 3 // 图片类型

	// 消息发送状态
	MsgStatusNotExist = 0 // 消息不存在
	MsgIsSending      = 1 // 发送中
	MsgSendSuccessed  = 2 // 发送成功
	MsgSendFailed     = 3 // 发送失败
)

// 消息扩散模式
const (
	WriteDiffusion = 0 // 写扩散模式：消息写入所有接收者的收件箱
	ReadDiffusion  = 1 // 读扩散模式：消息只写入发件人收件箱，接收者读取时同步
)

// 通知可靠性类型
const (
	UnreliableNotification    = 1 // 不可靠通知：不保证送达
	ReliableNotificationNoMsg = 2 // 可靠通知 (无消息): 保证送达但不携带消息内容
	ReliableNotificationMsg   = 3 // 可靠通知 (有消息): 保证送达并携带消息内容
)

// @提醒相关常量
const (
	AtAllString       = "AtAllTag" // @所有人标记字符串
	AtNormal          = 0          // 普通@
	AtMe              = 1          // @我
	AtAll             = 2          // @所有人
	AtAllAtMe         = 3          // @所有人且@我
	GroupNotification = 4          // 群组通知
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

// 会话字段标识
// 用于标识会话中需要更新的字段类型
const (
	FieldRecvMsgOpt    = 1  // 接收消息选项字段
	FieldIsPinned      = 2  // 置顶状态字段
	FieldAttachedInfo  = 3  // 附加信息字段
	FieldIsPrivateChat = 4  // 私有聊天状态字段
	FieldGroupAtType   = 5  // 群组@类型字段
	FieldEx            = 7  // 扩展字段
	FieldUnread        = 8  // 未读数状态字段
	FieldBurnDuration  = 9  // 阅后即焚时长字段
	FieldHasReadSeq    = 10 // 已读序列号字段
)

// 用户角色和权限常量
const (
	// 用户类型
	IMOrdinaryUser       = 0 // IM 普通用户
	AppOrdinaryUsers     = 1 // 应用普通用户
	AppAdmin             = 2 // 应用管理员
	AppNotificationAdmin = 3 // 应用通知管理员
	AppRobotAdmin        = 4 // 应用机器人管理员

	// 群角色
	GroupOwner         = 100 // 群主
	GroupAdmin         = 60  // 群管理员
	GroupOrdinaryUsers = 20  // 群普通成员

	// 群组申请响应状态
	GroupResponseAgree  = 1  // 同意
	GroupResponseRefuse = -1 // 拒绝

	// 好友申请响应状态
	FriendResponseNotHandle = 0  // 未处理
	FriendResponseAgree     = 1  // 同意
	FriendResponseRefuse    = -1 // 拒绝

	// 性别
	Male   = 1 // 男性
	Female = 2 // 女性
)

// RPC 和 HTTP 请求头键名常量
const (
	OperationID     = "operationID"  // 操作 ID，用于请求追踪
	OpUserID        = "opUserID"     // 操作用户 ID
	ConnID          = "connID"       // 连接 ID
	OpUserPlatform  = "platform"     // 用户平台
	Token           = "token"        // 认证令牌
	RpcCustomHeader = "customHeader" // 自定义 RPC 头 (用于 RPC 中间件)
	CheckKey        = "checkKey"     // 校验键
	TriggerID       = "triggerID"    // 触发器 ID
	ClientIP        = "clientIP"     // 客户端 IP
)

// 好友添加方式
const (
	BecomeFriendByImport = 1 // 通过管理员导入成为好友
	BecomeFriendByApply  = 2 // 通过申请成为好友
)

// 群组加入验证模式
const (
	ApplyNeedVerificationInviteDirectly = 0 // 申请需要验证，邀请直接加入
	AllNeedVerification                 = 1 // 所有人都需要验证 (除非群主或管理员邀请)
	Directly                            = 2 // 直接加入，无需验证
)

// 群组 RPC 消息批量大小
const (
	GroupRPCRecvSize = 30 // 群组 RPC 接收批量大小
	GroupRPCSendSize = 30 // 群组 RPC 发送批量大小
)

// 好友接受提示语
const FriendAcceptTip = "You have successfully become friends, so start chatting"

// GroupIsBanChat 检查群组是否被禁言
// 参数 status: 群组状态
// 返回：如果群组被禁言返回 true，否则返回 false
func GroupIsBanChat(status int32) bool {
	if status != GroupStatusMuted {
		return false
	}
	return true
}

// GroupIsBanPrivateChat 检查是否禁止私聊
// 参数 status: 群组状态
// 返回：如果禁止私聊返回 true，否则返回 false
func GroupIsBanPrivateChat(status int32) bool {
	if status != GroupBanPrivateChat {
		return false
	}
	return true
}

// 日志文件名
const LogFileName = "OpenIM.log"

// 本地监听地址
const LocalHost = "0.0.0.0"

// 命令行参数标志
// 用于解析启动时的命令行参数
const (
	FlagPort                  = "port"                  // 服务端口
	FlagWsPort                = "ws_port"               // WebSocket 端口
	FlagTransferProgressIndex = "transferProgressIndex" // 传输进度索引
	FlagPrometheusPort        = "prometheus_port"       // Prometheus 监控端口
	FlagConf                  = "config_folder_path"    // 配置文件目录路径
)

// OpenIM 通用配置键
const OpenIMCommonConfigKey = "OpenIMServerConfig"

// 回调命令键
const CallbackCommand = "command"

// 批量操作数量
const BatchNum = 100

// 用户订阅相关常量
const (
	SubscriberUser = 1 // 订阅用户
	Unsubscribe    = 2 // 取消订阅
)

// 群组搜索位置
const (
	GroupSearchPositionHead = 1 // 搜索位置在头部
	GroupSearchPositionAny  = 2 // 搜索位置在任意位置
)

// 分页相关常量
const (
	FirstPageNumber   = 1   // 首页页码
	MaxSyncPullNumber = 500 // 最大同步拉取数量
)

// 消息状态常量
const (
	MsgStatusSending     = 1 // 发送中
	MsgStatusSendSuccess = 2 // 发送成功
	MsgStatusSendFailed  = 3 // 发送失败
	MsgStatusHasDeleted  = 4 // 已删除
	MsgStatusFiltered    = 5 // 已过滤
)
