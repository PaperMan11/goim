package model

import "time"

// Friend 好友关系，存储用户之间的好友关联
type Friend struct {
	OwnerUserID    string    `bson:"owner_user_id"`    // 所有者ID
	FriendUserID   string    `bson:"friend_user_id"`   // 好友ID
	Remark         string    `bson:"remark"`           // 备注名
	CreateTime     time.Time `bson:"create_time"`      // 创建时间
	AddSource      int       `bson:"add_source"`       // 添加来源
	OperatorUserID string    `bson:"operator_user_id"` // 操作人ID
	Extra          string    `bson:"extra"`            // 扩展字段（JSON格式）
	IsPinned       bool      `bson:"is_pinned"`        // 是否置顶
	UpdatedAt      time.Time `bson:"updated_at"`       // 更新时间
}

func (f *Friend) CollectionName() string {
	return CollectionFriend
}

// Black 黑名单，存储用户拉黑的其他用户
type Black struct {
	OwnerUserID    string    `bson:"owner_user_id"`    // 所有者ID
	BlackUserID    string    `bson:"black_user_id"`    // 被拉黑用户ID
	CreateTime     time.Time `bson:"create_time"`      // 创建时间
	AddSource      int       `bson:"add_source"`       // 添加来源
	OperatorUserID string    `bson:"operator_user_id"` // 操作人ID
	Extra          string    `bson:"extra"`            // 扩展字段（JSON格式）
	UpdatedAt      time.Time `bson:"updated_at"`       // 更新时间
}

func (b *Black) CollectionName() string {
	return CollectionBlack
}
