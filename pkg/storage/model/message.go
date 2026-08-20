package model

import "time"

// Message 消息数据，存储消息的完整内容和元信息
type Message struct {
	ConversationID   string                 `bson:"conversation_id"`    // 会话ID
	ServerMsgID      string                 `bson:"server_msg_id"`      // 服务端消息ID
	ClientMsgID      string                 `bson:"client_msg_id"`      // 客户端消息ID
	SendID           string                 `bson:"send_id"`            // 发送者ID
	RecvID           string                 `bson:"recv_id"`            // 接收者ID
	GroupID          string                 `bson:"group_id"`           // 群组ID（群消息时使用）
	SenderPlatformID int                    `bson:"sender_platform_id"` // 发送者平台ID
	SenderNickname   string                 `bson:"sender_nickname"`    // 发送者昵称
	SenderFaceURL    string                 `bson:"sender_face_url"`    // 发送者头像URL
	SessionType      int                    `bson:"session_type"`       // 会话类型（1-单聊，2-群聊）
	MsgFrom          int                    `bson:"msg_from"`           // 消息来源（100-用户消息，200-系统消息）
	ContentType      int                    `bson:"content_type"`       // 消息内容类型
	Content          []byte                 `bson:"content"`            // 消息内容（二进制）
	Seq              int64                  `bson:"seq"`                // 消息序列号
	SendTime         time.Time              `bson:"send_time"`          // 发送时间
	CreateTime       time.Time              `bson:"create_time"`        // 创建时间
	Status           int                    `bson:"status"`             // 消息状态
	IsRead           bool                   `bson:"is_read"`            // 是否已读
	Options          map[string]bool        `bson:"options"`            // 消息选项
	OfflinePushInfo  *OfflinePushInfo       `bson:"offline_push_info"`  // 离线推送信息
	AtUserIDList     []string               `bson:"at_user_id_list"`    // @用户ID列表
	AttachedInfo     string                 `bson:"attached_info"`      // 附加信息
	Extra            string                 `bson:"extra"`              // 扩展字段（JSON格式）
	IsRevoked        bool                   `bson:"is_revoked"`         // 是否已撤回
	RevokedContent   *MessageRevokedContent `bson:"revoked_content"`    // 撤回消息内容
}

func (m *Message) CollectionName() string {
	return CollectionMessage
}

// OfflinePushInfo 离线推送信息，用于APNS/FCM等推送服务
type OfflinePushInfo struct {
	Title         string `bson:"title"`           // 推送标题
	Desc          string `bson:"desc"`            // 推送描述
	Extra         string `bson:"extra"`           // 扩展字段（JSON格式）
	IOSPushSound  string `bson:"ios_push_sound"`  // iOS推送声音
	IOSBadgeCount bool   `bson:"ios_badge_count"` // iOS角标计数
	SignalInfo    string `bson:"signal_info"`     // 信令信息
}

// MessageRevokedContent 撤回消息内容，存储撤回操作的详细信息
type MessageRevokedContent struct {
	RevokerID                   string    `bson:"revoker_id"`                     // 撤回者ID
	RevokerRole                 int       `bson:"revoker_role"`                   // 撤回者角色
	ClientMsgID                 string    `bson:"client_msg_id"`                  // 客户端消息ID
	RevokerNickname             string    `bson:"revoker_nickname"`               // 撤回者昵称
	RevokeTime                  time.Time `bson:"revoke_time"`                    // 撤回时间
	SourceMessageSendTime       time.Time `bson:"source_message_send_time"`       // 原消息发送时间
	SourceMessageSendID         string    `bson:"source_message_send_id"`         // 原消息发送者ID
	SourceMessageSenderNickname string    `bson:"source_message_sender_nickname"` // 原消息发送者昵称
	SessionType                 int       `bson:"session_type"`                   // 会话类型
	Seq                         int64     `bson:"seq"`                            // 消息序列号
	Extra                       string    `bson:"extra"`                          // 扩展字段（JSON格式）
}
