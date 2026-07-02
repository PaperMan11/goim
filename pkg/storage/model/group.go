package model

import "time"

// Group 群组信息，存储群组的基础配置和元数据
type Group struct {
	GroupID                string    `bson:"group_id"`                 // 群组唯一标识
	GroupName              string    `bson:"group_name"`               // 群组名称
	Notification           string    `bson:"notification"`             // 群组公告
	Introduction           string    `bson:"introduction"`             // 群组介绍
	FaceURL                string    `bson:"face_url"`                 // 群组头像URL
	OwnerUserID            string    `bson:"owner_user_id"`            // 群主ID
	CreateTime             time.Time `bson:"create_time"`              // 创建时间
	MemberCount            int       `bson:"member_count"`             // 成员数量
	Extra                  string    `bson:"extra"`                    // 扩展字段（JSON格式）
	Status                 int       `bson:"status"`                   // 群组状态（0-正常，1-解散）
	CreatorUserID          string    `bson:"creator_user_id"`          // 创建者ID
	GroupType              int       `bson:"group_type"`               // 群组类型（1-普通群，2-部门群等）
	NeedVerification       int       `bson:"need_verification"`        // 是否需要验证（0-不需要，1-需要）
	LookMemberInfo         int       `bson:"look_member_info"`         // 是否允许查看成员信息（0-不允许，1-允许）
	ApplyMemberFriend      int       `bson:"apply_member_friend"`      // 是否允许成员添加好友（0-不允许，1-允许）
	NotificationUpdateTime time.Time `bson:"notification_update_time"` // 公告更新时间
	NotificationUserID     string    `bson:"notification_user_id"`     // 公告更新者ID
	UpdatedAt              time.Time `bson:"updated_at"`               // 更新时间
}

func (g *Group) CollectionName() string {
	return CollectionGroup
}

// GroupMember 群成员信息，存储群成员的角色和权限
type GroupMember struct {
	GroupID         string    `bson:"group_id"`          // 群组唯一标识
	UserID          string    `bson:"user_id"`           // 用户唯一标识
	RoleLevel       int       `bson:"role_level"`        // 角色等级（100-群主，60-管理员，20-普通成员）
	JoinTime        time.Time `bson:"join_time"`         // 加入时间
	Nickname        string    `bson:"nickname"`          // 群昵称
	FaceURL         string    `bson:"face_url"`          // 头像URL
	AppManagerLevel int       `bson:"app_manager_level"` // 应用管理员等级
	JoinSource      int       `bson:"join_source"`       // 加入来源
	OperatorUserID  string    `bson:"operator_user_id"`  // 操作人ID
	Extra           string    `bson:"extra"`             // 扩展字段（JSON格式）
	MuteEndTime     time.Time `bson:"mute_end_time"`     // 禁言结束时间（0表示未禁言）
	InviterUserID   string    `bson:"inviter_user_id"`   // 邀请人ID
	UpdatedAt       time.Time `bson:"updated_at"`        // 更新时间
}

func (g *GroupMember) CollectionName() string {
	return CollectionGroupMember
}

// GroupVersion 群组版本信息，用于同步群成员和排序的版本控制
type GroupVersion struct {
	GroupID         string    `bson:"group_id"`          // 群组唯一标识
	MemberVersion   int64     `bson:"member_version"`    // 群成员版本号
	MemberVersionID string    `bson:"member_version_id"` // 群成员版本ID
	SortVersion     int64     `bson:"sort_version"`      // 群排序版本号
	UpdatedAt       time.Time `bson:"updated_at"`        // 更新时间
}

func (g *GroupVersion) CollectionName() string {
	return CollectionGroupVersion
}
