package model

import "time"

// Conversation 会话信息，存储用户与其他用户/群组的会话配置
type Conversation struct {
	OwnerUserID           string    `bson:"owner_user_id"`            // 会话所有者ID
	ConversationID        string    `bson:"conversation_id"`          // 会话ID
	RecvMsgOpt            int       `bson:"recv_msg_opt"`             // 消息接收选项（0-接收，1-不接收，2-接收但不通知）
	ConversationType      int       `bson:"conversation_type"`        // 会话类型（1-单聊，2-群聊）
	UserID                string    `bson:"user_id"`                  // 对方用户ID（单聊）
	GroupID               string    `bson:"group_id"`                 // 群组ID（群聊）
	IsPinned              bool      `bson:"is_pinned"`                // 是否置顶
	AttachedInfo          string    `bson:"attached_info"`            // 附加信息
	IsPrivateChat         bool      `bson:"is_private_chat"`          // 是否私聊
	GroupAtType           int       `bson:"group_at_type"`            // 群@类型（0-普通，1-@我，2-@所有人，3-@我和@所有人）
	Extra                 string    `bson:"extra"`                    // 扩展字段（JSON格式）
	BurnDuration          int       `bson:"burn_duration"`            // 阅后即焚时长（秒）
	MinSeq                int64     `bson:"min_seq"`                  // 最小消息序列号
	MaxSeq                int64     `bson:"max_seq"`                  // 最大消息序列号
	MsgDestructTime       time.Time `bson:"msg_destruct_time"`        // 消息销毁时间
	LatestMsgDestructTime time.Time `bson:"latest_msg_destruct_time"` // 最新消息销毁时间
	IsMsgDestruct         bool      `bson:"is_msg_destruct"`          // 是否开启消息销毁
	UnreadCount           int64     `bson:"unread_count"`             // 未读消息数量
	UpdatedAt             time.Time `bson:"updated_at"`               // 更新时间
}

func (c *Conversation) CollectionName() string {
	return CollectionConversation
}

// ConversationLatestMsg 会话最新消息摘要，用于会话列表显示
type ConversationLatestMsg struct {
	ConversationID    string    `bson:"conversation_id"`      // 会话ID
	OwnerUserID       string    `bson:"owner_user_id"`        // 会话所有者ID
	ServerMsgID       string    `bson:"server_msg_id"`        // 最新消息服务端ID
	ClientMsgID       string    `bson:"client_msg_id"`        // 最新消息客户端ID
	SessionType       int       `bson:"session_type"`         // 会话类型
	SendID            string    `bson:"send_id"`              // 发送者ID
	RecvID            string    `bson:"recv_id"`              // 接收者ID
	SenderName        string    `bson:"sender_name"`          // 发送者名称
	FaceURL           string    `bson:"face_url"`             // 发送者头像URL
	GroupID           string    `bson:"group_id"`             // 群组ID
	GroupName         string    `bson:"group_name"`           // 群组名称
	GroupFaceURL      string    `bson:"group_face_url"`       // 群组头像URL
	GroupType         int       `bson:"group_type"`           // 群组类型
	GroupMemberCount  int       `bson:"group_member_count"`   // 群成员数量
	LatestMsgRecvTime time.Time `bson:"latest_msg_recv_time"` // 最新消息接收时间
	MsgFrom           int       `bson:"msg_from"`             // 消息来源
	ContentType       int       `bson:"content_type"`         // 消息内容类型
	Content           string    `bson:"content"`              // 消息内容摘要
	Extra             string    `bson:"extra"`                // 扩展字段（JSON格式）
	UpdatedAt         time.Time `bson:"updated_at"`           // 更新时间
}

func (c *ConversationLatestMsg) CollectionName() string {
	return CollectionConversationLatestMsg
}
