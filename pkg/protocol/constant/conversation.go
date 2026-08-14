// Package constant 定义了 OpenIM 系统中使用的各种常量
package constant

// 会话类型
const (
	SingleChatType = 1 // 单聊
	// WriteGroupChatType Not enabled temporarily // 写扩散群聊 (暂未启用)
	WriteGroupChatType   = 2 // 写扩散群聊 (消息写入所有成员的收件箱)
	ReadGroupChatType    = 3 // 读扩散群聊 (消息只写入发件人收件箱，成员读取时同步)
	NotificationChatType = 4 // 通知会话
)

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
