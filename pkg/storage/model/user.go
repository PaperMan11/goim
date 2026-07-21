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

// UserStatus 用户在线状态，记录用户的在线/离线状态及平台/设备信息
//
// P1 升级：唯一键粒度从 {user_id, platform_id} 升级到 {user_id, platform_id, device_id}，
// 支持同一用户同一平台多设备（如 PC 端开多个浏览器标签）各自独立的在线状态。
type UserStatus struct {
	UserID         string    `bson:"user_id"`          // 用户唯一标识
	Status         int       `bson:"status"`           // 在线状态（1-在线，0-离线）
	PlatformID     int       `bson:"platform_id"`      // 平台ID
	DeviceID       string    `bson:"device_id"`        // 设备ID（P1新增，区分同平台多设备）
	TokenUUID      string    `bson:"token_uuid"`       // token 的 UUID（用于精准踢下线）
	ConnID         string    `bson:"conn_id"`          // WebSocket 连接ID（网关分配）
	LastOnlineTime time.Time `bson:"last_online_time"` // 最后在线时间
	LastSeenAt     time.Time `bson:"last_seen_at"`     // 最后心跳时间（用于僵尸连接检测）
	ExpireAt       time.Time `bson:"expire_at"`        // 过期时间（TTL索引自动清理僵尸记录）
	DeviceName     string    `bson:"device_name"`      // 设备名称/型号（如 "iPhone 15 Pro"）
	LoginIP        string    `bson:"login_ip"`         // 登录时的客户端IP
	Extra          string    `bson:"extra"`            // 扩展字段（JSON格式）
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
