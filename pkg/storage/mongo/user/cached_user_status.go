package user

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/syncx"
)

type userStatusCache struct {
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

func newUserStatusCache(redis goredis.UniversalClient, barrier syncx.SingleFlight) *userStatusCache {
	return &userStatusCache{redis: redis, barrier: barrier}
}

func (c *userStatusCache) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (c *userStatusCache) cacheKeys(userID string) []string {
	return []string{GetUserStatusZKey(userID), GetUserStatusZNilKey(userID)}
}

func (c *userStatusCache) invalidate(ctx context.Context, userID string) {
	if userID == "" || c.redis == nil {
		return
	}
	sredis.CacheDelDouble(ctx, c.redis, c.cacheKeys(userID)...)
}

// zRowsForUser 从 ZSET 读一个用户的在线平台并过滤僵尸；检测到僵尸时顺带懒清理（读时写消）。
// 过滤规则：member 存在且 score >= nowMs - ZombieThresholdMs 视为在线。
// 懒清理：
//
//	· 如果检测到 member.score < cutoff → ZREMRANGEBYSCORE 顺手剔除
//	· 清理后若整个 ZSET 空 → 清 key + 写 Nil Marker，防止后续重复查空 zset
//
// 返回：在线 rows（Status=1）、是否所有结果都不是僵尸、错误。
// needDB = false  ：缓存可信，直接用 rows
// needDB = true   ：结果全是僵尸或 key 没命中，需要回落 DB 确认该用户是否真的全离线
func (c *userStatusCache) zRowsForUser(ctx context.Context, userID string) (rows []*model.UserStatus, needDB bool, err error) {
	rdb := c.redis
	if userID == "" || rdb == nil {
		return nil, true, nil
	}

	// 1) 读 nil marker：确认该用户此前已被判定为「全部离线」，跳过 zset 读 + DB 查询
	var dummy []byte
	foundNil, errN := sredis.CacheGet(ctx, rdb, GetUserStatusZNilKey(userID), &dummy)
	if errN != nil {
		return nil, false, errN
	}
	if foundNil {
		// 全离线：直接返回空 rows，不需要 DB
		return nil, false, nil
	}

	// 2) 读 ZSET
	zs, zFound, errZ := sredis.CacheZRangeWithScores(ctx, rdb, GetUserStatusZKey(userID))
	if errZ != nil {
		return nil, false, errZ
	}
	if !zFound {
		// 没命中缓存且没 nil marker → 回源 DB
		return nil, true, nil
	}

	nowMs := timex.UnixMilli()
	cutoff := nowMs - ZombieThresholdMs

	hasAlive := false
	hasZombie := false
	rows = make([]*model.UserStatus, 0, len(zs))
	for _, z := range zs {
		scoreMs := int64(z.Score)
		if scoreMs < cutoff {
			hasZombie = true
			continue
		}
		member, ok := z.Member.(string)
		if !ok {
			continue
		}
		plat := ParsePlatformZMember(member)
		if plat < 0 {
			continue
		}
		hasAlive = true
		rows = append(rows, &model.UserStatus{
			UserID:     userID,
			PlatformID: plat,
			Status:     1, // ZSET 中存在的 member 必为在线
			UpdatedAt:  time.UnixMilli(scoreMs),
		})
	}

	// 3) 懒清理：检测到僵尸时，顺手 ZREMRANGEBYSCORE 清掉 score < cutoff 的所有 member
	//    - 若仍有存活 member（!needDB）：清理后空 zset 可安全补 Nil Marker
	//    - 若全部是僵尸（needDB=true）：**不要**在此写 Nil Marker，必须等 DB 回源确认全离线后再写
	if hasZombie {
		zKey := GetUserStatusZKey(userID)
		nilKey := GetUserStatusZNilKey(userID)
		maxScore := fmt.Sprintf("(%d", cutoff)
		_, _ = sredis.CacheZRemRangeByScore(ctx, rdb, zKey, "-inf", maxScore)
		if !hasAlive {
			// 全僵尸场景：只清 + 可能删空 key，不写 Nil Marker（等 DB 确认）
			card, _ := rdb.ZCard(ctx, zKey).Result()
			if card == 0 {
				_ = rdb.Del(ctx, zKey).Err()
			}
		} else {
			_ = sredis.CacheDelOnEmptyZ(ctx, rdb, zKey, nilKey, c.jitterTTL(userStatusZNilExpireSeconds))
		}
	}

	// 4) 若 ZSET 有内容但全部是僵尸 → 缓存可信度低，需要回源 DB 再确认一次
	needDB = !hasAlive
	return rows, needDB, nil
}

func (c *userStatusCache) GetUserStatus(ctx context.Context, inner UserModel, userIDs []string) ([]*model.UserStatus, error) {
	if c.redis == nil {
		return inner.GetUserStatus(ctx, userIDs)
	}
	if len(userIDs) == 0 {
		return []*model.UserStatus{}, nil
	}

	result := make([]*model.UserStatus, 0, len(userIDs))
	missIDs := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		rows, needDB, err := c.zRowsForUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !needDB {
			result = append(result, rows...)
			continue
		}
		missIDs = append(missIDs, userID)
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	sort.Strings(missIDs)
	sum := sha1.Sum([]byte(strings.Join(missIDs, ",")))
	sfKey := sfKeyPrefixBatchStatus + hex.EncodeToString(sum[:])

	sfRowsAny, err := c.barrier.Do(sfKey, func() (any, error) {
		sfRows := make(map[string][]*model.UserStatus, len(missIDs))
		realMiss := make([]string, 0, len(missIDs))
		for _, uid := range missIDs {
			rows, needDB, err2 := c.zRowsForUser(ctx, uid)
			if err2 != nil {
				return nil, err2
			}
			if !needDB {
				sfRows[uid] = rows
				continue
			}
			realMiss = append(realMiss, uid)
		}
		if len(realMiss) == 0 {
			return sfRows, nil
		}

		dbRows, errDB := inner.GetUserStatus(ctx, realMiss)
		if errDB != nil {
			return nil, errDB
		}

		groupMap := make(map[string][]*model.UserStatus, len(realMiss))
		for _, s := range dbRows {
			if s == nil {
				continue
			}
			groupMap[s.UserID] = append(groupMap[s.UserID], s)
		}

		nowMs := timex.UnixMilli()
		expire := c.jitterTTL(userStatusZDefaultExpireSeconds)
		nilTTL := c.jitterTTL(userStatusZNilExpireSeconds)

		for _, uid := range realMiss {
			zKey := GetUserStatusZKey(uid)
			nilKey := GetUserStatusZNilKey(uid)
			usrRows := groupMap[uid]

			// 过滤出在线平台（Status=1）并按 platform 聚合最大更新时间
			onlineMax := make(map[string]int64, len(usrRows))
			for _, r := range usrRows {
				if r == nil || r.Status != 1 {
					continue
				}
				ms := r.UpdatedAt.UnixMilli()
				if ms <= 0 {
					ms = nowMs
				}
				member := PlatformZMember(r.PlatformID)
				if prev, ok := onlineMax[member]; !ok || ms > prev {
					onlineMax[member] = ms
				}
			}

			// 按 platform 展开 Status=1 的 rows 写入当前 sf 返回结果（与 ZSET 形态对齐）
			// Status=0 的 DB 行不写入 ZSET 也不返回，逻辑层自行按 user 预填兜底。
			built := make([]*model.UserStatus, 0, len(onlineMax))
			for member, ms := range onlineMax {
				plat := ParsePlatformZMember(member)
				if plat <= 0 {
					continue
				}
				built = append(built, &model.UserStatus{
					UserID:     uid,
					PlatformID: plat,
					Status:     1,
					UpdatedAt:  time.UnixMilli(ms),
				})
			}
			sfRows[uid] = built

			if len(onlineMax) > 0 {
				_, _ = sredis.CacheZAddGTBatch(ctx, c.redis, zKey, onlineMax, expire)
				_ = c.redis.Del(ctx, nilKey).Err()
			} else {
				// 确认全离线 → 写 Nil Marker，删空 ZSET key
				_ = c.redis.Del(ctx, zKey).Err()
				_, _ = sredis.CacheSetCAS(ctx, c.redis, nilKey, nil, 0, nilTTL)
			}
		}
		return sfRows, nil
	})
	if err != nil {
		return nil, err
	}
	sfRows, _ := sfRowsAny.(map[string][]*model.UserStatus)
	for _, uid := range missIDs {
		result = append(result, sfRows[uid]...)
	}

	return result, nil
}

func (c *userStatusCache) UpdateUserStatus(ctx context.Context, inner UserModel, userID string, platformID int, deviceID string, status int) error {
	err := inner.UpdateUserStatus(ctx, userID, platformID, deviceID, status)
	if err != nil {
		return err
	}
	if userID == "" || c.redis == nil {
		return nil
	}

	nowMs := timex.UnixMilli()
	zKey := GetUserStatusZKey(userID)
	nilKey := GetUserStatusZNilKey(userID)
	expire := c.jitterTTL(userStatusZDefaultExpireSeconds)

	if status == 1 {
		// 在线：ZADD GT + 确保 zset key TTL + 清 Nil Marker
		member := PlatformZMember(platformID)
		_, _ = sredis.CacheZAddGT(ctx, c.redis, zKey, float64(nowMs), member, expire)
		_ = c.redis.Del(ctx, nilKey).Err()
	} else {
		// 离线：ZREM platform member，若 zset 变空则写 Nil Marker
		member := PlatformZMember(platformID)
		_, _ = sredis.CacheZRem(ctx, c.redis, zKey, member)
		_ = sredis.CacheDelOnEmptyZ(ctx, c.redis, zKey, nilKey, c.jitterTTL(userStatusZNilExpireSeconds))
	}
	return nil
}

func (c *userStatusCache) SetUserOnlineStatus(ctx context.Context, inner UserModel, statuses []*model.UserStatus) error {
	err := inner.SetUserOnlineStatus(ctx, statuses)
	if err != nil {
		return err
	}
	if len(statuses) == 0 || c.redis == nil {
		return nil
	}

	type perUser struct {
		online  map[string]int64 // member -> max ms
		offline []string
	}
	perUserMap := make(map[string]*perUser, 16)

	nowMs := timex.UnixMilli()
	for _, s := range statuses {
		if s == nil || s.UserID == "" {
			continue
		}
		pu, ok := perUserMap[s.UserID]
		if !ok {
			pu = &perUser{online: make(map[string]int64, 2)}
			perUserMap[s.UserID] = pu
		}
		ms := s.UpdatedAt.UnixMilli()
		if ms <= 0 {
			ms = nowMs
		}
		member := PlatformZMember(s.PlatformID)
		if s.Status == 1 {
			if prev, ok2 := pu.online[member]; !ok2 || ms > prev {
				pu.online[member] = ms
			}
		} else {
			pu.offline = append(pu.offline, member)
		}
	}

	expire := c.jitterTTL(userStatusZDefaultExpireSeconds)
	nilTTL := c.jitterTTL(userStatusZNilExpireSeconds)

	for userID, pu := range perUserMap {
		zKey := GetUserStatusZKey(userID)
		nilKey := GetUserStatusZNilKey(userID)

		for member := range pu.online {
			_, _ = sredis.CacheZAddGT(ctx, c.redis, zKey, float64(pu.online[member]), member, expire)
		}

		for _, member := range pu.offline {
			if _, ok := pu.online[member]; ok {
				continue
			}
			_, _ = sredis.CacheZRem(ctx, c.redis, zKey, member)
		}

		if len(pu.online) > 0 {
			_ = c.redis.Del(ctx, nilKey).Err()
		} else {
			_ = sredis.CacheDelOnEmptyZ(ctx, c.redis, zKey, nilKey, nilTTL)
		}
	}

	return nil
}
