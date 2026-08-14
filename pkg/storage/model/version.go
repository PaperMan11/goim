package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 版本日志状态
const (
	VersionStateInsert = iota + 1 // 1 新增
	VersionStateDelete            // 2 删除
	VersionStateUpdate            // 3 更新
)

// 版本日志EID常量
// VersionGroupChangeID 表示群组信息变更（非成员变更）
// VersionSortChangeID 表示排序变更
const (
	VersionGroupChangeID = "__GROUP_CHANGE__"
	VersionSortChangeID  = "__SORT_CHANGE__"
)

const (
	FirstVersion         = 1
	DefaultDeleteVersion = 0
)

// VersionLogElem 单条版本日志，记录一次变更
type VersionLogElem struct {
	EID        string    `bson:"e_id"`        // 实体ID（成员变更=用户ID，群组信息变更=VersionGroupChangeID）
	State      int32     `bson:"state"`       // 变更状态（1-新增，2-删除，3-更新）
	Version    uint      `bson:"version"`     // 该条变更对应的版本号
	LastUpdate time.Time `bson:"last_update"` // 该条变更的时间
}

// VersionLogTable 版本日志表，存储某个DID（群ID或用户ID）的完整变更链
type VersionLogTable struct {
	DID        string           `bson:"d_id"`        // 群成员同步=groupID，加入群同步=userID
	Logs       []VersionLogElem `bson:"logs"`        // 变更日志链
	Version    uint             `bson:"version"`     // 当前最新版本号
	Deleted    uint             `bson:"deleted"`     // 累计删除数量
	LastUpdate time.Time        `bson:"last_update"` // 最后更新时间
}

// VersionLog 返回带 LogLen 字段的版本日志视图
func (v *VersionLogTable) VersionLog() *VersionLog {
	if v == nil {
		return nil
	}
	return &VersionLog{
		DID:        v.DID,
		Logs:       v.Logs,
		Version:    v.Version,
		Deleted:    v.Deleted,
		LastUpdate: v.LastUpdate,
		LogLen:     len(v.Logs),
	}
}

// VersionLog 与 VersionLogTable 相同，额外提供 LogLen 字段方便消费端读取
type VersionLog struct {
	ID         primitive.ObjectID `bson:"_id"`
	DID        string             `bson:"d_id"`
	Logs       []VersionLogElem   `bson:"logs"`
	Version    uint               `bson:"version"`
	Deleted    uint               `bson:"deleted"`
	LastUpdate time.Time          `bson:"last_update"`
	LogLen     int                `bson:"log_len"`
}

func (v *VersionLogTable) CollectionName() string {
	return CollectionGroupVersion
}
