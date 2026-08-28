package model

import "time"

// SeqConversation 会话级序列号管理，记录整个会话的消息序列号范围
type SeqConversation struct {
	ConversationID string    `bson:"conversation_id"` // 会话ID
	MaxSeq         int64     `bson:"max_seq"`         // 最大序列号
	MinSeq         int64     `bson:"min_seq"`         // 最小序列号
	UpdatedAt      time.Time `bson:"updated_at"`      // 更新时间
}

func (s *SeqConversation) CollectionName() string {
	return CollectionSeqConversation
}

// SeqUser 用户级序列号管理，记录每个用户在特定会话中的序列号状态
type SeqUser struct {
	UserID         string    `bson:"user_id"`         // 用户ID
	ConversationID string    `bson:"conversation_id"` // 会话ID
	MinSeq         int64     `bson:"min_seq"`         // 最小序列号
	MaxSeq         int64     `bson:"max_seq"`         // 最大序列号
	ReadSeq        int64     `bson:"read_seq"`        // 已读序列号
	UpdatedAt      time.Time `bson:"updated_at"`      // 更新时间
}

func (s *SeqUser) CollectionName() string {
	return CollectionSeqUser
}
