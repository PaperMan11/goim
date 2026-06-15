package internal

// SDK类型常量定义
const (
	GoSDK = "go" // Go语言SDK
	JsSDK = "js" // JavaScript SDK
)

// WebSocket协议消息类型常量定义
const (
	// WSGetNewestSeq 获取最新消息序列号
	WSGetNewestSeq = 1001
	// WSPullMsgBySeqList 按序列号列表拉取消息
	WSPullMsgBySeqList = 1002
	// WSSendMsg 发送消息
	WSSendMsg = 1003
	// WSSendSignalMsg 发送信号消息
	WSSendSignalMsg = 1004
	// WSPullMsg 拉取消息
	WSPullMsg = 1005
	// WSGetConvMaxReadSeq 获取会话最大已读序列号
	WSGetConvMaxReadSeq = 1006
	// WsPullConvLastMessage 拉取会话最后一条消息
	WsPullConvLastMessage = 1007
	// WSPushMsg 推送消息（服务端主动推送）
	WSPushMsg = 2001
	// WSKickOnlineMsg 踢下线消息
	WSKickOnlineMsg = 2002
	// WSLogoutMsg 登出消息
	WSLogoutMsg = 2003
	// WSSetBackgroundStatus 设置后台状态
	WSSetBackgroundStatus = 2004
	// WSSubUserOnlineStatus 订阅用户在线状态
	WSSubUserOnlineStatus = 2005
	// WSDataError 数据错误消息
	WSDataError = 3001
)
