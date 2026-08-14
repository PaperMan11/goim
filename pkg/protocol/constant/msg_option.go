// Package constant 定义了 OpenIM 系统中使用的各种常量
package constant

// 消息接收选项
const (
	ReceiveMessage          = 0 // 接收消息
	NotReceiveMessage       = 1 // 不收消息
	ReceiveNotNotifyMessage = 2 // 接收但不通知
)

// 消息选项键
const (
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

// 文件上传类型
const (
	OtherType = 1 // 其他类型
	VideoType = 2 // 视频类型
	ImageType = 3 // 图片类型
)

// 消息发送状态
const (
	MsgStatusNotExist = 0 // 消息不存在
	MsgIsSending      = 1 // 发送中
	MsgSendSuccessed  = 2 // 发送成功
	MsgSendFailed     = 3 // 发送失败
)

// 消息状态常量
const (
	MsgStatusSending     = 1 // 发送中
	MsgStatusSendSuccess = 2 // 发送成功
	MsgStatusSendFailed  = 3 // 发送失败
	MsgStatusHasDeleted  = 4 // 已删除
	MsgStatusFiltered    = 5 // 已过滤
)
