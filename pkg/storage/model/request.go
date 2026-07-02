package model

import "time"

// FriendRequest 好友申请，存储用户之间的好友申请记录
type FriendRequest struct {
	FromUserID    string    `bson:"from_user_id"`    // 申请人ID
	FromNickname  string    `bson:"from_nickname"`   // 申请人昵称
	FromFaceURL   string    `bson:"from_face_url"`   // 申请人头像URL
	ToUserID      string    `bson:"to_user_id"`      // 被申请人ID
	ToNickname    string    `bson:"to_nickname"`     // 被申请人昵称
	ToFaceURL     string    `bson:"to_face_url"`     // 被申请人头像URL
	HandleResult  int       `bson:"handle_result"`   // 处理结果（0-未处理，1-已同意，-1-已拒绝）
	ReqMsg        string    `bson:"req_msg"`         // 申请消息
	CreateTime    time.Time `bson:"create_time"`     // 创建时间
	HandlerUserID string    `bson:"handler_user_id"` // 处理人ID
	HandleMsg     string    `bson:"handle_msg"`      // 处理消息
	HandleTime    time.Time `bson:"handle_time"`     // 处理时间
	Extra         string    `bson:"extra"`           // 扩展字段（JSON格式）
}

func (f *FriendRequest) CollectionName() string {
	return CollectionFriendRequest
}

// GroupRequest 群组申请，存储用户加入群组的申请记录
type GroupRequest struct {
	UserID        string    `bson:"user_id"`         // 申请人ID
	Nickname      string    `bson:"nickname"`        // 申请人昵称
	FaceURL       string    `bson:"face_url"`        // 申请人头像URL
	GroupID       string    `bson:"group_id"`        // 群组ID
	GroupName     string    `bson:"group_name"`      // 群组名称
	GroupFaceURL  string    `bson:"group_face_url"`  // 群组头像URL
	HandleResult  int       `bson:"handle_result"`   // 处理结果（0-未处理，1-已同意，-1-已拒绝）
	ReqMsg        string    `bson:"req_msg"`         // 申请消息
	HandleMsg     string    `bson:"handle_msg"`      // 处理消息
	ReqTime       time.Time `bson:"req_time"`        // 申请时间
	HandleUserID  string    `bson:"handle_user_id"`  // 处理人ID
	HandleTime    time.Time `bson:"handle_time"`     // 处理时间
	Extra         string    `bson:"extra"`           // 扩展字段（JSON格式）
	JoinSource    int       `bson:"join_source"`     // 加入来源
	InviterUserID string    `bson:"inviter_user_id"` // 邀请人ID
}

func (g *GroupRequest) CollectionName() string {
	return CollectionGroupRequest
}
