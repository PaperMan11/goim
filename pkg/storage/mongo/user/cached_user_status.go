package user

import (
	"context"
	"fmt"
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
)

type userStatusCache struct {
	redis goredis.UniversalClient
}

func newUserStatusCache(redis goredis.UniversalClient) *userStatusCache {
	return &userStatusCache{redis: redis}
}

func (c *userStatusCache) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (c *userStatusCache) cacheKeys(userID string) []string {
	return []string{GetUserStatusZKey(userID), GetUserStatusZNilKey(userID)}
}

func (c *userStatusCache) Del(ctx context.Context, userID string) {
	if userID == "" || c.redis == nil {
		return
	}
	sredis.CacheDelDouble(ctx, c.redis, c.cacheKeys(userID)...)
}

func (c *userStatusCache) zRowsForUser(ctx context.Context, userID string) (rows []*model.UserStatus, needDB bool, err error) {
	rdb := c.redis
	if userID == "" || rdb == nil {
		return nil, true, nil
	}

	var dummy []byte
	foundNil, errN := sredis.CacheGet(ctx, rdb, GetUserStatusZNilKey(userID), &dummy)
	if errN != nil {
		return nil, false, errN
	}
	if foundNil {
		return nil, false, nil
	}

	zs, zFound, errZ := sredis.CacheZRangeWithScores(ctx, rdb, GetUserStatusZKey(userID))
	if errZ != nil {
		return nil, false, errZ
	}
	if !zFound {
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
			Status:     1,
			UpdatedAt:  time.UnixMilli(scoreMs),
		})
	}

	if hasZombie {
		zKey := GetUserStatusZKey(userID)
		nilKey := GetUserStatusZNilKey(userID)
		maxScore := fmt.Sprintf("(%d", cutoff)
		_, _ = sredis.CacheZRemRangeByScore(ctx, rdb, zKey, "-inf", maxScore)
		if !hasAlive {
			card, _ := rdb.ZCard(ctx, zKey).Result()
			if card == 0 {
				_ = rdb.Del(ctx, zKey).Err()
			}
		} else {
			_, _ = sredis.CacheDelOnEmptyZ(ctx, rdb, zKey, nilKey, c.jitterTTL(userStatusZNilExpireSeconds))
		}
	}

	needDB = !hasAlive
	return rows, needDB, nil
}

func (c *userStatusCache) GetUserStatus(ctx context.Context, userIDs []string) ([]*model.UserStatus, []string, error) {
	if c.redis == nil {
		return nil, userIDs, nil
	}
	if len(userIDs) == 0 {
		return []*model.UserStatus{}, nil, nil
	}

	result := make([]*model.UserStatus, 0, len(userIDs))
	missIDs := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		rows, needDB, err := c.zRowsForUser(ctx, userID)
		if err != nil {
			return nil, nil, err
		}
		if !needDB {
			result = append(result, rows...)
			continue
		}
		missIDs = append(missIDs, userID)
	}

	return result, missIDs, nil
}

func (c *userStatusCache) SetOnline(ctx context.Context, userID string, platformID int, scoreMs int64) {
	if userID == "" || c.redis == nil {
		return
	}
	zKey := GetUserStatusZKey(userID)
	nilKey := GetUserStatusZNilKey(userID)
	expire := c.jitterTTL(userStatusZDefaultExpireSeconds)
	member := PlatformZMember(platformID)
	if scoreMs <= 0 {
		scoreMs = timex.UnixMilli()
	}
	_, _ = sredis.CacheZAddGT(ctx, c.redis, zKey, float64(scoreMs), member, expire)
	_ = c.redis.Del(ctx, nilKey).Err()
	c.AddToOnlineSet(ctx, userID)
}

func (c *userStatusCache) SetOffline(ctx context.Context, userID string, platformID int) {
	if userID == "" || c.redis == nil {
		return
	}
	zKey := GetUserStatusZKey(userID)
	nilKey := GetUserStatusZNilKey(userID)
	member := PlatformZMember(platformID)
	_, _ = sredis.CacheZRem(ctx, c.redis, zKey, member)
	deleted, _ := sredis.CacheDelOnEmptyZ(ctx, c.redis, zKey, nilKey, c.jitterTTL(userStatusZNilExpireSeconds))
	if deleted {
		c.RemoveFromOnlineSet(ctx, userID)
	}
}

func (c *userStatusCache) SetNilMarker(ctx context.Context, userID string) {
	if userID == "" || c.redis == nil {
		return
	}
	zKey := GetUserStatusZKey(userID)
	nilKey := GetUserStatusZNilKey(userID)
	_ = c.redis.Del(ctx, zKey).Err()
	_, _ = sredis.CacheSetCAS(ctx, c.redis, nilKey, nil, 0, c.jitterTTL(userStatusZNilExpireSeconds))
	c.RemoveFromOnlineSet(ctx, userID)
}

func (c *userStatusCache) AddToOnlineSet(ctx context.Context, userID string) {
	if userID == "" || c.redis == nil {
		return
	}
	_ = c.redis.SAdd(ctx, GetUserOnlineSetKey(), userID).Err()
}

func (c *userStatusCache) RemoveFromOnlineSet(ctx context.Context, userID string) {
	if userID == "" || c.redis == nil {
		return
	}
	_ = c.redis.SRem(ctx, GetUserOnlineSetKey(), userID).Err()
}

func (c *userStatusCache) GetAllOnlineUsers(ctx context.Context) ([]string, error) {
	if c.redis == nil {
		return nil, nil
	}
	return c.redis.SMembers(ctx, GetUserOnlineSetKey()).Result()
}

func (c *userStatusCache) SetOnlineBatch(ctx context.Context, statuses []*model.UserStatus) {
	if len(statuses) == 0 || c.redis == nil {
		return
	}

	type perUser struct {
		online  map[string]int64
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

	for userID, pu := range perUserMap {
		zKey := GetUserStatusZKey(userID)
		nilKey := GetUserStatusZNilKey(userID)

		for member, ms := range pu.online {
			_, _ = sredis.CacheZAddGT(ctx, c.redis, zKey, float64(ms), member, expire)
		}

		for _, member := range pu.offline {
			if _, ok := pu.online[member]; ok {
				continue
			}
			_, _ = sredis.CacheZRem(ctx, c.redis, zKey, member)
		}

		if len(pu.online) > 0 {
			_ = c.redis.Del(ctx, nilKey).Err()
			c.AddToOnlineSet(ctx, userID)
		} else {
			deleted, _ := sredis.CacheDelOnEmptyZ(ctx, c.redis, zKey, nilKey, c.jitterTTL(userStatusZNilExpireSeconds))
			if deleted {
				c.RemoveFromOnlineSet(ctx, userID)
			}
		}
	}
}
