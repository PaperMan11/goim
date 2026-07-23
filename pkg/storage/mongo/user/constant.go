package user

import (
	"fmt"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
)

// -------------------------- 用户信息 Redis Key 常量 --------------------------

const (
	KeyUserInfo          = "mongo:user:info:%s"
	KeyUserGlobalRecvOpt = "mongo:user:global_recv_opt:%s"
	KeyUserExists        = "mongo:user:exists:%s"
	KeyIMAdmin           = "mongo:user:im_admin:%s"
)

// -------------------------- 用户在线状态 Redis Key 常量 --------------------------

const (
	KeyUserStatusZSet = "mongo:user:status:z:%s"
	// 对"查过 DB 确认该 user 无任何在线平台"的用户，打一个短 TTL 的 STRING 标记，
	// 防止 ZRANGE 返回空（语义上既可能是没上线过，也可能是刚从全离线变过来）导致每次都打 DB。
	KeyUserStatusZNil = "mongo:user:status:zn:%s"
)

// -------------------------- 用户信息 Key 生成函数 --------------------------

func GetUserInfoKey(userID string) string {
	return fmt.Sprintf(KeyUserInfo, userID)
}

func GetUserGlobalRecvOptKey(userID string) string {
	return fmt.Sprintf(KeyUserGlobalRecvOpt, userID)
}

func GetUserExistsKey(userID string) string {
	return fmt.Sprintf(KeyUserExists, userID)
}

func GetIMAdminKey(userID string) string {
	return fmt.Sprintf(KeyIMAdmin, userID)
}

// -------------------------- 用户在线状态 Key 生成函数 --------------------------

// GetUserStatusZKey 返回用户在线状态 ZSET 的缓存 key。
// 同一 user 的不同平台在 ZSET 里以 member=PlatformID 形式存在，
// score 为最后心跳时间戳（UnixMilli）。
func GetUserStatusZKey(userID string) string {
	return fmt.Sprintf(KeyUserStatusZSet, userID)
}

// GetUserStatusZNilKey 返回 ZSET 空集合的 Nil Marker 哨兵 key。
// 查过 DB 确认"当前所有平台全离线"时写入，短 TTL，下一次查到直接跳过，防击穿。
func GetUserStatusZNilKey(userID string) string {
	return fmt.Sprintf(KeyUserStatusZNil, userID)
}

// PlatformZMember 将平台 ID 转换为 ZSET member 名称。
func PlatformZMember(platformID int) string {
	return constant.PlatformID2Name[platformID]
}

// ParsePlatformZMember 将 ZSET member 名称解析为平台 ID。
func ParsePlatformZMember(member string) int {
	return constant.PlatformName2ID[member]
}

// -------------------------- TTL 常量 --------------------------

const (
	// 用户信息相关 TTL
	userDefaultExpireSeconds = 5 * 60
	userNilExpireSeconds     = 60

	// 用户在线状态相关 TTL
	userStatusZDefaultExpireSeconds = 90
	// "查过无在线平台"的 Nil Marker 短 TTL：短一些，保证用户重新上线后不会被 Nil 误挡住太久。
	userStatusZNilExpireSeconds = 15

	// 缓存 TTL 抖动比例（±10%），防止缓存雪崩
	ttlJitterRatioPct = 10

	// 超过 zombieThresholdMs 没收到该平台心跳更新（score < now - thresholdMs），
	// 则逻辑上视为"该平台已离线"，GetUserStatus 不会把这类 member 当在线行返回，
	// 并且读路径会顺手调用 ZREMRANGEBYSCORE 清掉（懒清理，不在后台做 worker）。
	// 默认 120s：心跳 30s 一次，允许连续 4 次漏心跳，足够包容短暂网络抖动。
	ZombieThresholdMs = 120 * 1000
)

// -------------------------- SingleFlight Key 前缀 --------------------------

const (
	// 用户信息相关 SingleFlight 前缀
	sfKeyPrefixUserInfo  = "uf:ui:"
	sfKeyPrefixRecvOpt   = "uf:ro:"
	sfKeyPrefixIMAdmin   = "uf:ad:"
	sfKeyPrefixBatchUser = "uf:bu:"
	sfKeyPrefixExists    = "uf:ex:"

	// 用户在线状态相关 SingleFlight 前缀
	sfKeyPrefixUserStatus  = "uf:us:"
	sfKeyPrefixBatchStatus = "uf:bs:"
)
