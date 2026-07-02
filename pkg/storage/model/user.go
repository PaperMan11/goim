package model

import "time"

// User 用户信息，存储用户的基础资料和权限配置
type User struct {
	UserID           string    `bson:"user_id"`             // 用户唯一标识
	Nickname         string    `bson:"nickname"`            // 用户昵称
	FaceURL          string    `bson:"face_url"`            // 用户头像URL
	Extra            string    `bson:"extra"`               // 扩展字段（JSON格式）
	AppManagerLevel  int       `bson:"app_manager_level"`   // 应用管理员等级（0-普通用户，>0-管理员）
	GlobalRecvMsgOpt int       `bson:"global_recv_msg_opt"` // 全局消息接收选项（0-接收，1-不接收，2-接收但不通知）
	CreatedAt        time.Time `bson:"created_at"`          // 创建时间
	UpdatedAt        time.Time `bson:"updated_at"`          // 更新时间
}

func (u *User) CollectionName() string {
	return CollectionUser
}

// UserStatus 用户在线状态，记录用户的在线/离线状态及平台信息
type UserStatus struct {
	UserID         string    `bson:"user_id"`          // 用户唯一标识
	Status         int       `bson:"status"`           // 在线状态（1-在线，0-离线）
	PlatformID     int       `bson:"platform_id"`      // 平台ID
	LastOnlineTime time.Time `bson:"last_online_time"` // 最后在线时间
	CreatedAt      time.Time `bson:"created_at"`       // 创建时间
	UpdatedAt      time.Time `bson:"updated_at"`       // 更新时间
}

func (u *UserStatus) CollectionName() string {
	return CollectionUserStatus
}

// UserCommand 用户命令，用于存储用户自定义的客户端命令数据
// - 客户端发送自定义命令（如"正在输入"状态）
// - 跨设备同步用户状态
// - 扩展业务功能（如设备绑定、快捷操作）
type UserCommand struct {
	UserID     string    `bson:"user_id"`     // 用户唯一标识
	Type       int       `bson:"type"`        // 命令类型
	CreateTime time.Time `bson:"create_time"` // 创建时间
	UUID       string    `bson:"uuid"`        // 命令唯一标识
	Value      string    `bson:"value"`       // 命令值
	UpdatedAt  time.Time `bson:"updated_at"`  // 更新时间
}

func (u *UserCommand) CollectionName() string {
	return CollectionUserCommand
}
