package user

import (
	"context"

	"github.com/PaperMan11/goim/pkg/storage/model"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/PaperMan11/goim/pkg/utils/convert"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
)

type userInfoCache struct {
	redis goredis.UniversalClient
}

func newUserInfoCache(redis goredis.UniversalClient) *userInfoCache {
	return &userInfoCache{redis: redis}
}

func (c *userInfoCache) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (c *userInfoCache) cacheKeys(userID string) []string {
	return []string{
		GetUserInfoKey(userID),
		GetUserGlobalRecvOptKey(userID),
		GetUserExistsKey(userID),
		GetIMAdminKey(userID),
	}
}

func (c *userInfoCache) Del(ctx context.Context, userID string) {
	if userID == "" || c.redis == nil {
		return
	}
	sredis.CacheDelDouble(ctx, c.redis, c.cacheKeys(userID)...)
}

func (c *userInfoCache) DelBatch(ctx context.Context, userIDs []string) {
	for _, uid := range userIDs {
		c.Del(ctx, uid)
	}
}

func (c *userInfoCache) GetUserInfo(ctx context.Context, userID string) (*model.User, bool, error) {
	if c.redis == nil {
		return nil, false, nil
	}
	var user model.User
	found, err := sredis.CacheGet(ctx, c.redis, GetUserInfoKey(userID), &user)
	if err != nil {
		return nil, false, err
	}
	if found && user.UserID != "" {
		return &user, true, nil
	}
	return nil, found, nil
}

func (c *userInfoCache) SetUserInfo(ctx context.Context, userID string, user *model.User, version int64) {
	if c.redis == nil {
		return
	}
	key := GetUserInfoKey(userID)
	if user == nil {
		_, _ = sredis.CacheSetCAS(ctx, c.redis, key, nil, 0, c.jitterTTL(userNilExpireSeconds))
		return
	}
	if version <= 0 {
		version = timex.UnixMilli()
	}
	_, _ = sredis.CacheSetCAS(ctx, c.redis, key, user, version, c.jitterTTL(userDefaultExpireSeconds))
}

func (c *userInfoCache) GetUserExists(ctx context.Context, userID string) (bool, bool, error) {
	if c.redis == nil {
		return false, false, nil
	}
	data, found, err := sredis.CacheGetString(ctx, c.redis, GetUserExistsKey(userID))
	if err != nil {
		return false, false, err
	}
	if found {
		return data == "1", true, nil
	}
	return false, false, nil
}

func (c *userInfoCache) SetUserExists(ctx context.Context, userID string, exists bool) {
	if c.redis == nil {
		return
	}
	val := "0"
	if exists {
		val = "1"
	}
	nowMs := timex.UnixMilli()
	if exists {
		_, _ = sredis.CacheSetCASString(ctx, c.redis, GetUserExistsKey(userID), val, nowMs, c.jitterTTL(userDefaultExpireSeconds))
	} else {
		_, _ = sredis.CacheSetCASString(ctx, c.redis, GetUserExistsKey(userID), val, 0, c.jitterTTL(userNilExpireSeconds))
	}
}

func (c *userInfoCache) GetGlobalRecvMsgOpt(ctx context.Context, userID string) (int, bool, error) {
	if c.redis == nil {
		return 0, false, nil
	}
	key := GetUserGlobalRecvOptKey(userID)
	data, found, err := sredis.CacheGetString(ctx, c.redis, key)
	if err != nil {
		return 0, false, err
	}
	if found {
		val, errConv := convert.ToIntE(data)
		if errConv == nil {
			return val, true, nil
		}
		_ = sredis.CacheDel(ctx, c.redis, key)
	}
	return 0, false, nil
}

func (c *userInfoCache) SetGlobalRecvMsgOpt(ctx context.Context, userID string, opt int) {
	if c.redis == nil {
		return
	}
	key := GetUserGlobalRecvOptKey(userID)
	if opt == 0 {
		_, _ = sredis.CacheSetCASString(ctx, c.redis, key, "0", 0, c.jitterTTL(userNilExpireSeconds))
		return
	}
	_, _ = sredis.CacheSetCASString(ctx, c.redis, key, convert.ToString(opt), timex.UnixMilli(), c.jitterTTL(userDefaultExpireSeconds))
}

func (c *userInfoCache) DelGlobalRecvOpt(ctx context.Context, userID string) {
	if userID == "" || c.redis == nil {
		return
	}
	sredis.CacheDelDouble(ctx, c.redis,
		GetUserGlobalRecvOptKey(userID),
		GetUserInfoKey(userID),
	)
}

func (c *userInfoCache) GetIMAdmin(ctx context.Context, userID string) (bool, bool, error) {
	if c.redis == nil {
		return false, false, nil
	}
	data, found, err := sredis.CacheGetString(ctx, c.redis, GetIMAdminKey(userID))
	if err != nil {
		return false, false, err
	}
	if found {
		return data == "1", true, nil
	}
	return false, false, nil
}

func (c *userInfoCache) SetIMAdmin(ctx context.Context, userID string, isAdmin bool) {
	if c.redis == nil {
		return
	}
	val := "0"
	if isAdmin {
		val = "1"
	}
	if isAdmin {
		_, _ = sredis.CacheSetCASString(ctx, c.redis, GetIMAdminKey(userID), val, timex.UnixMilli(), c.jitterTTL(userDefaultExpireSeconds))
	} else {
		_, _ = sredis.CacheSetCASString(ctx, c.redis, GetIMAdminKey(userID), val, 0, c.jitterTTL(userNilExpireSeconds))
	}
}
