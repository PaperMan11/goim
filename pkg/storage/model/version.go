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

// IncrementalChange 增量变更分类结果，由 ClassifyIncrementalLogs 输出。
// 各 RPC 的增量同步方法（GetIncrementalGroupMember / GetIncrementalFriends 等）共用此结构。
type IncrementalChange struct {
	InsertIDs    []string // State==Insert 的 EID 列表
	UpdateIDs    []string // State==Update 的 EID 列表
	DeleteIDs    []string // State==Delete 的 EID 列表
	SortChanged  bool     // 是否存在排序变更（VersionSortChangeID）
	SortVersion  uint64   // 排序变更日志中的最大版本号
	GroupChanged bool     // 是否存在群组信息变更（VersionGroupChangeID）
}

// ClassifyIncrementalLogs 对版本日志数组进行分类处理，返回增量变更结果。
// 同一 EID 多次操作只保留最后一次（日志数组中靠后的条目覆盖靠前的）。
//
// 分类规则：
//   - EID == VersionGroupChangeID → GroupChanged = true
//   - EID == VersionSortChangeID  → SortChanged = true, SortVersion = max(version)
//   - 普通 EID 按 State 归入 InsertIDs / UpdateIDs / DeleteIDs
//
// 由于 IncrVersionLogBatch 的 $filter 去重已保证每个 EID 在 logs 中只保留最新一条，
// 正常情况下同一 EID 不会同时出现在多个列表中。
func ClassifyIncrementalLogs(logs []VersionLogElem) IncrementalChange {
	var c IncrementalChange
	seen := make(map[string]int32) // EID → 最后一次 State（后者覆盖前者）
	for _, log := range logs {
		switch log.EID {
		case VersionGroupChangeID:
			c.GroupChanged = true
			continue
		case VersionSortChangeID:
			c.SortChanged = true
			if uint64(log.Version) > c.SortVersion {
				c.SortVersion = uint64(log.Version)
			}
			continue
		}
		seen[log.EID] = log.State
	}
	for eid, state := range seen {
		switch state {
		case VersionStateInsert:
			c.InsertIDs = append(c.InsertIDs, eid)
		case VersionStateDelete:
			c.DeleteIDs = append(c.DeleteIDs, eid)
		case VersionStateUpdate:
			c.UpdateIDs = append(c.UpdateIDs, eid)
		}
	}
	return c
}
