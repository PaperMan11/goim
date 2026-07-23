package user

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"sort"
	"strings"

	"github.com/PaperMan11/goim/pkg/storage/model"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/PaperMan11/goim/pkg/utils/convert"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/syncx"
)

type userInfoCache struct {
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

func newUserInfoCache(redis goredis.UniversalClient, barrier syncx.SingleFlight) *userInfoCache {
	return &userInfoCache{redis: redis, barrier: barrier}
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

func (c *userInfoCache) invalidate(ctx context.Context, userID string) {
	if userID == "" || c.redis == nil {
		return
	}
	sredis.CacheDelDouble(ctx, c.redis, c.cacheKeys(userID)...)
}

func (c *userInfoCache) invalidateBatch(ctx context.Context, userIDs []string) {
	for _, uid := range userIDs {
		c.invalidate(ctx, uid)
	}
}

func (c *userInfoCache) FindByID(ctx context.Context, inner UserModel, userID string) (*model.User, error) {
	if c.redis == nil {
		return inner.FindByID(ctx, userID)
	}

	var user model.User
	key := GetUserInfoKey(userID)
	found, err := sredis.CacheGet(ctx, c.redis, key, &user)
	if err != nil {
		return nil, err
	}
	if found {
		if user.UserID == "" {
			return nil, ErrUserNotFound
		}
		return &user, nil
	}

	sfKey := sfKeyPrefixUserInfo + userID
	v, err := c.barrier.Do(sfKey, func() (any, error) {
		var innerUser model.User
		found2, err2 := sredis.CacheGet(ctx, c.redis, key, &innerUser)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerUser.UserID == "" {
				return nil, ErrUserNotFound
			}
			return &innerUser, nil
		}

		dbUser, err2 := inner.FindByID(ctx, userID)
		if err2 != nil {
			if errors.Is(err2, ErrUserNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, c.redis, key, nil, 0, c.jitterTTL(userNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbUser.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, c.redis, key, dbUser, version, c.jitterTTL(userDefaultExpireSeconds))
		return dbUser, nil
	})
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrUserNotFound
	}
	return v.(*model.User), nil
}

func (c *userInfoCache) FindByIDs(ctx context.Context, inner UserModel, userIDs []string) ([]*model.User, error) {
	if c.redis == nil {
		return inner.FindByIDs(ctx, userIDs)
	}

	result := make([]*model.User, 0, len(userIDs))
	missIDs := make([]string, 0, len(userIDs))

	for _, userID := range userIDs {
		var user model.User
		found, err := sredis.CacheGet(ctx, c.redis, GetUserInfoKey(userID), &user)
		if err != nil {
			return nil, err
		}
		if found && user.UserID != "" {
			result = append(result, &user)
			continue
		}
		missIDs = append(missIDs, userID)
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	sort.Strings(missIDs)
	sum := sha1.Sum([]byte(strings.Join(missIDs, ",")))
	sfKey := sfKeyPrefixBatchUser + hex.EncodeToString(sum[:])

	v, err := c.barrier.Do(sfKey, func() (any, error) {
		for i, uid := range missIDs {
			u, errLoad := c.FindByID(ctx, inner, uid)
			if errLoad != nil && !errors.Is(errLoad, ErrUserNotFound) {
				return nil, errLoad
			}
			if u != nil {
				missIDs[i] = ""
				result = append(result, u)
			}
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}
	_ = v

	for _, uid := range missIDs {
		if uid == "" {
			continue
		}
		var user model.User
		found, err := sredis.CacheGet(ctx, c.redis, GetUserInfoKey(uid), &user)
		if err != nil {
			return nil, err
		}
		if found && user.UserID != "" {
			result = append(result, &user)
		}
	}

	return result, nil
}

func (c *userInfoCache) CheckExists(ctx context.Context, inner UserModel, userIDs []string) (map[string]bool, error) {
	if c.redis == nil {
		return inner.CheckExists(ctx, userIDs)
	}

	result := make(map[string]bool, len(userIDs))
	missIDs := make([]string, 0, len(userIDs))

	for _, userID := range userIDs {
		data, found, err := sredis.CacheGetString(ctx, c.redis, GetUserExistsKey(userID))
		if err != nil {
			return nil, err
		}
		if found {
			result[userID] = data == "1"
			continue
		}
		missIDs = append(missIDs, userID)
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	sort.Strings(missIDs)
	sum := sha1.Sum([]byte(strings.Join(missIDs, ",")))
	sfKey := sfKeyPrefixExists + hex.EncodeToString(sum[:])

	_, err := c.barrier.Do(sfKey, func() (any, error) {
		dbResult, errDB := inner.CheckExists(ctx, missIDs)
		if errDB != nil {
			return nil, errDB
		}
		nowMs := timex.UnixMilli()
		for userID, exists := range dbResult {
			val := "0"
			if exists {
				val = "1"
			}
			_, _ = sredis.CacheSetCASString(ctx, c.redis, GetUserExistsKey(userID), val, nowMs, c.jitterTTL(userDefaultExpireSeconds))
			if _, ok := result[userID]; !ok {
				result[userID] = exists
			}
		}
		for _, userID := range missIDs {
			if _, ok := result[userID]; !ok {
				_, _ = sredis.CacheSetCASString(ctx, c.redis, GetUserExistsKey(userID), "0", 0, c.jitterTTL(userNilExpireSeconds))
				result[userID] = false
			}
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}

	for _, userID := range missIDs {
		if _, ok := result[userID]; ok {
			continue
		}
		data, found, errStr := sredis.CacheGetString(ctx, c.redis, GetUserExistsKey(userID))
		if errStr != nil {
			return nil, errStr
		}
		if found {
			result[userID] = data == "1"
		} else {
			result[userID] = false
		}
	}

	return result, nil
}

func (c *userInfoCache) GetGlobalRecvMsgOpt(ctx context.Context, inner UserModel, userID string) (int, error) {
	if c.redis == nil {
		return inner.GetGlobalRecvMsgOpt(ctx, userID)
	}

	key := GetUserGlobalRecvOptKey(userID)
	data, found, err := sredis.CacheGetString(ctx, c.redis, key)
	if err != nil {
		return 0, err
	}
	if found {
		val, errConv := convert.ToIntE(data)
		if errConv == nil {
			return val, nil
		}
		_ = sredis.CacheDel(ctx, c.redis, key)
	}

	sfKey := sfKeyPrefixRecvOpt + userID
	v, err := c.barrier.Do(sfKey, func() (any, error) {
		data2, found2, err2 := sredis.CacheGetString(ctx, c.redis, key)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			val, errConv := convert.ToIntE(data2)
			if errConv == nil {
				return val, nil
			}
			_ = sredis.CacheDel(ctx, c.redis, key)
		}

		opt, err2 := inner.GetGlobalRecvMsgOpt(ctx, userID)
		if err2 != nil {
			if errors.Is(err2, ErrUserNotFound) {
				_, _ = sredis.CacheSetCASString(ctx, c.redis, key, "0", 0, c.jitterTTL(userNilExpireSeconds))
				return 0, nil
			}
			return nil, err2
		}
		_, _ = sredis.CacheSetCASString(ctx, c.redis, key, convert.ToString(opt), timex.UnixMilli(), c.jitterTTL(userDefaultExpireSeconds))
		return opt, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int), nil
}

func (c *userInfoCache) invalidateGlobalRecvOpt(ctx context.Context, userID string) {
	if userID == "" || c.redis == nil {
		return
	}
	sredis.CacheDelDouble(ctx, c.redis,
		GetUserGlobalRecvOptKey(userID),
		GetUserInfoKey(userID),
	)
}

func (c *userInfoCache) IsIMAdmin(ctx context.Context, inner UserModel, userID string) (bool, error) {
	if c.redis == nil {
		return inner.IsIMAdmin(ctx, userID)
	}

	key := GetIMAdminKey(userID)
	data, found, err := sredis.CacheGetString(ctx, c.redis, key)
	if err != nil {
		return false, err
	}
	if found {
		return data == "1", nil
	}

	sfKey := sfKeyPrefixIMAdmin + userID
	v, err := c.barrier.Do(sfKey, func() (any, error) {
		data2, found2, err2 := sredis.CacheGetString(ctx, c.redis, key)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			return data2 == "1", nil
		}

		isAdmin, err2 := inner.IsIMAdmin(ctx, userID)
		if err2 != nil {
			if errors.Is(err2, ErrUserNotFound) {
				_, _ = sredis.CacheSetCASString(ctx, c.redis, key, "0", 0, c.jitterTTL(userNilExpireSeconds))
				return false, nil
			}
			return nil, err2
		}
		val := "0"
		if isAdmin {
			val = "1"
		}
		_, _ = sredis.CacheSetCASString(ctx, c.redis, key, val, timex.UnixMilli(), c.jitterTTL(userDefaultExpireSeconds))
		return isAdmin, nil
	})
	if err != nil {
		return false, err
	}
	return v.(bool), nil
}
