package webhooks

import "time"

// EventType 定义 webhook 事件类型
type EventType string

const (
	// 消息相关事件
	EventMessageSent            EventType = "message.sent"             // 消息发送成功
	EventMessageReceived        EventType = "message.received"         // 消息接收
	EventMessageRevoked         EventType = "message.revoked"          // 消息撤回
	EventMessageDeleted         EventType = "message.deleted"          // 消息删除
	EventMessageRead            EventType = "message.read"             // 消息已读
	EventMessageReactionAdded   EventType = "message.reaction_added"   // 消息表情回应添加
	EventMessageReactionRemoved EventType = "message.reaction_removed" // 消息表情回应删除

	// 用户相关事件
	EventUserOnline        EventType = "user.online"         // 用户上线
	EventUserOffline       EventType = "user.offline"        // 用户下线
	EventUserKicked        EventType = "user.kicked"         // 用户被踢
	EventUserInfoUpdated   EventType = "user.info_updated"   // 用户信息更新
	EventUserStatusChanged EventType = "user.status_changed" // 用户状态变更

	// 好友相关事件
	EventFriendApplicationReceived EventType = "friend.application_received" // 收到好友申请
	EventFriendApplicationApproved EventType = "friend.application_approved" // 好友申请已同意
	EventFriendApplicationRejected EventType = "friend.application_rejected" // 好友申请已拒绝
	EventFriendAdded               EventType = "friend.added"                // 已添加好友
	EventFriendDeleted             EventType = "friend.deleted"              // 已删除好友
	EventFriendBlackAdded          EventType = "friend.black_added"          // 已加入黑名单
	EventFriendBlackDeleted        EventType = "friend.black_deleted"        // 已移出黑名单

	// 群组相关事件
	EventGroupCreated           EventType = "group.created"             // 群组创建
	EventGroupInfoUpdated       EventType = "group.info_updated"        // 群组信息更新
	EventGroupMemberJoined      EventType = "group.member_joined"       // 成员加入群
	EventGroupMemberLeft        EventType = "group.member_left"         // 成员退出群
	EventGroupMemberKicked      EventType = "group.member_kicked"       // 成员被踢
	EventGroupOwnerTransferred  EventType = "group.owner_transferred"   // 群主转让
	EventGroupDismissed         EventType = "group.dismissed"           // 群组解散
	EventGroupMuted             EventType = "group.muted"               // 群组禁言
	EventGroupUnmuted           EventType = "group.unmuted"             // 群组取消禁言
	EventGroupMemberMuted       EventType = "group.member_muted"        // 成员禁言
	EventGroupMemberUnmuted     EventType = "group.member_unmuted"      // 成员取消禁言
	EventGroupMemberRoleChanged EventType = "group.member_role_changed" // 成员角色变更

	// 会话相关事件
	EventConversationCreated       EventType = "conversation.created"        // 会话创建
	EventConversationUpdated       EventType = "conversation.updated"        // 会话更新
	EventConversationDeleted       EventType = "conversation.deleted"        // 会话删除
	EventConversationUnreadChanged EventType = "conversation.unread_changed" // 会话未读数变更

	// 离线推送事件
	EventOfflinePush EventType = "push.offline" // 离线推送
	EventOnlinePush  EventType = "push.online"  // 在线推送
)

// WebhookEvent 定义 webhook 事件数据结构
type WebhookEvent struct {
	EventType   EventType              `json:"eventType"`   // 事件类型
	EventID     string                 `json:"eventId"`     // 事件唯一ID
	Timestamp   int64                  `json:"timestamp"`   // 事件时间戳(毫秒)
	OperationID string                 `json:"operationId"` // 操作ID，用于追踪
	Data        map[string]interface{} `json:"data"`        // 事件数据
	RetryCount  int                    `json:"retryCount"`  // 重试次数
	IsRetry     bool                   `json:"isRetry"`     // 是否为重试事件
}

// MessageEventData 消息事件数据
type MessageEventData struct {
	MessageID      string            `json:"messageId"`      // 消息ID
	ServerMsgID    string            `json:"serverMsgId"`    // 服务端消息ID
	ClientMsgID    string            `json:"clientMsgId"`    // 客户端消息ID
	SenderID       string            `json:"senderId"`       // 发送者ID
	SenderNickname string            `json:"senderNickname"` // 发送者昵称
	ReceiverID     string            `json:"receiverId"`     // 接收者ID
	GroupID        string            `json:"groupId"`        // 群组ID(群消息时)
	ContentType    int               `json:"contentType"`    // 消息内容类型
	Content        string            `json:"content"`        // 消息内容
	SessionType    int               `json:"sessionType"`    // 会话类型(1:单聊 2:群聊)
	SendTime       int64             `json:"sendTime"`       // 发送时间
	Seq            int64             `json:"seq"`            // 消息序列号
	PlatformID     int               `json:"platformId"`     // 发送者平台ID
	Ex             string            `json:"ex"`             // 扩展字段
	AtUserList     []string          `json:"atUserList"`     // 用户列表
	OfflinePush    map[string]string `json:"offlinePush"`    // 离线推送信息
}

// UserEventData 用户事件数据
type UserEventData struct {
	UserID       string            `json:"userId"`       // 用户ID
	Nickname     string            `json:"nickname"`     // 昵称
	FaceURL      string            `json:"faceUrl"`      // 头像URL
	PlatformID   int               `json:"platformId"`   // 平台ID
	OnlineStatus int               `json:"onlineStatus"` // 在线状态(0:离线 1:在线)
	DeviceID     string            `json:"deviceId"`     // 设备ID
	Extra        map[string]string `json:"extra"`        // 扩展信息
}

// FriendEventData 好友事件数据
type FriendEventData struct {
	OwnerUserID    string            `json:"ownerUserId"`    // 所有者用户ID
	FriendUserID   string            `json:"friendUserId"`   // 好友用户ID
	FriendNickname string            `json:"friendNickname"` // 好友昵称
	FriendFaceURL  string            `json:"friendFaceUrl"`  // 好友头像
	Remark         string            `json:"remark"`         // 备注名
	Source         int               `json:"source"`         // 来源(1:导入 2:申请)
	OperatorUserID string            `json:"operatorUserId"` // 操作者用户ID
	HandleResult   int               `json:"handleResult"`   // 处理结果(1:同意 -1:拒绝)
	ReqMsg         string            `json:"reqMsg"`         // 申请消息
	Extra          map[string]string `json:"extra"`          // 扩展信息
}

// GroupEventData 群组事件数据
type GroupEventData struct {
	GroupID         string            `json:"groupId"`         // 群组ID
	GroupName       string            `json:"groupName"`       // 群组名称
	GroupFaceURL    string            `json:"groupFaceUrl"`    // 群组头像
	GroupType       int               `json:"groupType"`       // 群组类型
	OwnerUserID     string            `json:"ownerUserId"`     // 群主用户ID
	MemberCount     int               `json:"memberCount"`     // 成员数量
	Introduction    string            `json:"introduction"`    // 群介绍
	Notification    string            `json:"notification"`    // 群公告
	MemberUserID    string            `json:"memberUserId"`    // 成员用户ID(成员相关事件)
	MemberNickname  string            `json:"memberNickname"`  // 成员昵称
	MemberRoleLevel int               `json:"memberRoleLevel"` // 成员角色等级
	OperatorUserID  string            `json:"operatorUserId"`  // 操作者用户ID
	Reason          string            `json:"reason"`          // 原因
	Extra           map[string]string `json:"extra"`           // 扩展信息
}

// ConversationEventData 会话事件数据
type ConversationEventData struct {
	ConversationID   string            `json:"conversationId"`   // 会话ID
	ConversationType int               `json:"conversationType"` // 会话类型
	UserID           string            `json:"userId"`           // 用户ID
	GroupID          string            `json:"groupId"`          // 群组ID
	ShowName         string            `json:"showName"`         // 显示名称
	FaceURL          string            `json:"faceUrl"`          // 头像URL
	RecvMsgOpt       int               `json:"recvMsgOpt"`       // 接收消息选项
	UnreadCount      int               `json:"unreadCount"`      // 未读数
	LatestMsg        string            `json:"latestMsg"`        // 最新消息
	LatestMsgTime    int64             `json:"latestMsgTime"`    // 最新消息时间
	IsPinned         bool              `json:"isPinned"`         // 是否置顶
	IsPrivateChat    bool              `json:"isPrivateChat"`    // 是否私聊
	Extra            map[string]string `json:"extra"`            // 扩展信息
}

// PushEventData 推送事件数据
type PushEventData struct {
	UserID          string            `json:"userId"`          // 用户ID
	PlatformID      int               `json:"platformId"`      // 平台ID
	Title           string            `json:"title"`           // 推送标题
	Content         string            `json:"content"`         // 推送内容
	Ex              string            `json:"ex"`              // 扩展字段
	IOSBadge        int               `json:"iosBadge"`        // iOS角标
	IOSProduction   bool              `json:"iosProduction"`   // iOS生产环境
	Sound           string            `json:"sound"`           // 提示音
	OfflinePushInfo map[string]string `json:"offlinePushInfo"` // 离线推送信息
}

// WebhookConfig webhook 配置
type WebhookConfig struct {
	URL           string            `json:"url"`           // webhook URL
	Secret        string            `json:"secret"`        // 签名密钥
	Timeout       time.Duration     `json:"timeout"`       // 请求超时时间
	MaxRetries    int               `json:"maxRetries"`    // 最大重试次数
	RetryInterval time.Duration     `json:"retryInterval"` // 重试间隔
	Enabled       bool              `json:"enabled"`       // 是否启用
	Events        []EventType       `json:"events"`        // 订阅的事件列表
	Headers       map[string]string `json:"headers"`       // 自定义请求头
}

// WebhookResponse webhook 响应
type WebhookResponse struct {
	StatusCode int               `json:"statusCode"` // HTTP状态码
	Success    bool              `json:"success"`    // 是否成功
	Error      string            `json:"error"`      // 错误信息
	Headers    map[string]string `json:"headers"`    // 响应头
	Body       string            `json:"body"`       // 响应体
	Duration   time.Duration     `json:"duration"`   // 请求耗时
}

// DeliveryStatus 投递状态
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"   // 待投递
	DeliveryStatusSending   DeliveryStatus = "sending"   // 投递中
	DeliveryStatusSuccess   DeliveryStatus = "success"   // 投递成功
	DeliveryStatusFailed    DeliveryStatus = "failed"    // 投递失败
	DeliveryStatusRetrying  DeliveryStatus = "retrying"  // 重试中
	DeliveryStatusAbandoned DeliveryStatus = "abandoned" // 已放弃
)

// DeliveryRecord 投递记录
type DeliveryRecord struct {
	ID           string         `json:"id" bson:"_id"`                     // 投递记录ID
	EventID      string         `json:"eventId" bson:"event_id"`           // 事件ID
	EventType    EventType      `json:"eventType" bson:"event_type"`       // 事件类型
	EventPayload string         `json:"eventPayload" bson:"event_payload"` // 事件数据（JSON序列化）
	WebhookURL   string         `json:"webhookUrl" bson:"webhook_url"`     // webhook URL
	Status       DeliveryStatus `json:"status" bson:"status"`              // 投递状态
	StatusCode   int            `json:"statusCode" bson:"status_code"`     // HTTP状态码
	AttemptCount int            `json:"attemptCount" bson:"attempt_count"` // 尝试次数
	LastAttempt  time.Time      `json:"lastAttempt" bson:"last_attempt"`   // 最后尝试时间
	NextAttempt  time.Time      `json:"nextAttempt" bson:"next_attempt"`   // 下次尝试时间
	ErrorMessage string         `json:"errorMessage" bson:"error_message"` // 错误信息
	Response     string         `json:"response" bson:"response"`          // 响应内容
	Duration     time.Duration  `json:"duration" bson:"duration"`          // 请求耗时
	CreatedAt    time.Time      `json:"createdAt" bson:"created_at"`       // 创建时间
	UpdatedAt    time.Time      `json:"updatedAt" bson:"updated_at"`       // 更新时间
}
