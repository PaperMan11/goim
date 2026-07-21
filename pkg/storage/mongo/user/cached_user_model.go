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

type cachedUserModel struct {
	UserModel
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

func NewCachedUserModel(inner UserModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) UserModel {
	return &cachedUserModel{
		UserModel: inner,
		redis:     rdb,
		barrier:   barrier,
	}
}

func (m *cachedUserModel) userCacheKeys(userID string) []string {
	return []string{
		GetUserInfoKey(userID),
		GetUserGlobalRecvOptKey(userID),
		GetUserExistsKey(userID),
		GetIMAdminKey(userID),
	}
}

func (m *cachedUserModel) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (m *cachedUserModel) Insert(ctx context.Context, users []*model.User) error {
	err := m.UserModel.Insert(ctx, users)
	if err != nil {
		return err
	}
	for _, user := range users {
		sredis.CacheDelDouble(ctx, m.redis, m.userCacheKeys(user.UserID)...)
	}
	return nil
}

func (m *cachedUserModel) FindByIDs(ctx context.Context, userIDs []string) ([]*model.User, error) {
	if m.redis == nil {
		return m.UserModel.FindByIDs(ctx, userIDs)
	}

	result := make([]*model.User, 0, len(userIDs))
	missIDs := make([]string, 0, len(userIDs))

	for _, userID := range userIDs {
		var user model.User
		found, err := sredis.CacheGet(ctx, m.redis, GetUserInfoKey(userID), &user)
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

	v, err := m.barrier.Do(sfKey, func() (any, error) {
		for i, uid := range missIDs {
			u, errLoad := m.FindByID(ctx, uid)
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
		found, err := sredis.CacheGet(ctx, m.redis, GetUserInfoKey(uid), &user)
		if err != nil {
			return nil, err
		}
		if found && user.UserID != "" {
			result = append(result, &user)
		}
	}

	return result, nil
}

func (m *cachedUserModel) FindByID(ctx context.Context, userID string) (*model.User, error) {
	if m.redis == nil {
		return m.UserModel.FindByID(ctx, userID)
	}

	var user model.User
	key := GetUserInfoKey(userID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &user)
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
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerUser model.User
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerUser)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerUser.UserID == "" {
				return nil, ErrUserNotFound
			}
			return &innerUser, nil
		}

		dbUser, err2 := m.UserModel.FindByID(ctx, userID)
		if err2 != nil {
			if errors.Is(err2, ErrUserNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(userNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbUser.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbUser, version, m.jitterTTL(userDefaultExpireSeconds))
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

func (m *cachedUserModel) Update(ctx context.Context, user *model.User) error {
	err := m.UserModel.Update(ctx, user)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.userCacheKeys(user.UserID)...)
	return nil
}

func (m *cachedUserModel) UpdateEx(ctx context.Context, userID string, updates map[string]any) error {
	err := m.UserModel.UpdateEx(ctx, userID, updates)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.userCacheKeys(userID)...)
	return nil
}

func (m *cachedUserModel) Delete(ctx context.Context, userID string) error {
	err := m.UserModel.Delete(ctx, userID)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.userCacheKeys(userID)...)
	return nil
}

func (m *cachedUserModel) CheckExists(ctx context.Context, userIDs []string) (map[string]bool, error) {
	if m.redis == nil {
		return m.UserModel.CheckExists(ctx, userIDs)
	}

	result := make(map[string]bool, len(userIDs))
	missIDs := make([]string, 0, len(userIDs))

	for _, userID := range userIDs {
		data, found, err := sredis.CacheGetString(ctx, m.redis, GetUserExistsKey(userID))
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

	_, err := m.barrier.Do(sfKey, func() (any, error) {
		dbResult, errDB := m.UserModel.CheckExists(ctx, missIDs)
		if errDB != nil {
			return nil, errDB
		}
		nowMs := timex.UnixMilli()
		for userID, exists := range dbResult {
			val := "0"
			if exists {
				val = "1"
			}
			_, _ = sredis.CacheSetCASString(ctx, m.redis, GetUserExistsKey(userID), val, nowMs, m.jitterTTL(userDefaultExpireSeconds))
			if _, ok := result[userID]; !ok {
				result[userID] = exists
			}
		}
		for _, userID := range missIDs {
			if _, ok := result[userID]; !ok {
				_, _ = sredis.CacheSetCASString(ctx, m.redis, GetUserExistsKey(userID), "0", 0, m.jitterTTL(userNilExpireSeconds))
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
		data, found, errStr := sredis.CacheGetString(ctx, m.redis, GetUserExistsKey(userID))
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

func (m *cachedUserModel) SetGlobalRecvMsgOpt(ctx context.Context, userID string, opt int) error {
	err := m.UserModel.SetGlobalRecvMsgOpt(ctx, userID, opt)
	if err != nil {
		return err
	}
	if userID == "" {
		return nil
	}
	sredis.CacheDelDouble(ctx, m.redis,
		GetUserGlobalRecvOptKey(userID),
		GetUserInfoKey(userID),
	)
	return nil
}

func (m *cachedUserModel) GetGlobalRecvMsgOpt(ctx context.Context, userID string) (int, error) {
	if m.redis == nil {
		return m.UserModel.GetGlobalRecvMsgOpt(ctx, userID)
	}

	key := GetUserGlobalRecvOptKey(userID)
	data, found, err := sredis.CacheGetString(ctx, m.redis, key)
	if err != nil {
		return 0, err
	}
	if found {
		val, errConv := convert.ToIntE(data)
		if errConv == nil {
			return val, nil
		}
		_ = sredis.CacheDel(ctx, m.redis, key)
	}

	sfKey := sfKeyPrefixRecvOpt + userID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		data2, found2, err2 := sredis.CacheGetString(ctx, m.redis, key)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			val, errConv := convert.ToIntE(data2)
			if errConv == nil {
				return val, nil
			}
			_ = sredis.CacheDel(ctx, m.redis, key)
		}

		opt, err2 := m.UserModel.GetGlobalRecvMsgOpt(ctx, userID)
		if err2 != nil {
			if errors.Is(err2, ErrUserNotFound) {
				_, _ = sredis.CacheSetCASString(ctx, m.redis, key, "0", 0, m.jitterTTL(userNilExpireSeconds))
				return 0, nil
			}
			return nil, err2
		}
		_, _ = sredis.CacheSetCASString(ctx, m.redis, key, convert.ToString(opt), timex.UnixMilli(), m.jitterTTL(userDefaultExpireSeconds))
		return opt, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int), nil
}

func (m *cachedUserModel) IsIMAdmin(ctx context.Context, userID string) (bool, error) {
	if m.redis == nil {
		return m.UserModel.IsIMAdmin(ctx, userID)
	}

	key := GetIMAdminKey(userID)
	data, found, err := sredis.CacheGetString(ctx, m.redis, key)
	if err != nil {
		return false, err
	}
	if found {
		return data == "1", nil
	}

	sfKey := sfKeyPrefixIMAdmin + userID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		data2, found2, err2 := sredis.CacheGetString(ctx, m.redis, key)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			return data2 == "1", nil
		}

		isAdmin, err2 := m.UserModel.IsIMAdmin(ctx, userID)
		if err2 != nil {
			if errors.Is(err2, ErrUserNotFound) {
				_, _ = sredis.CacheSetCASString(ctx, m.redis, key, "0", 0, m.jitterTTL(userNilExpireSeconds))
				return false, nil
			}
			return nil, err2
		}
		val := "0"
		if isAdmin {
			val = "1"
		}
		_, _ = sredis.CacheSetCASString(ctx, m.redis, key, val, timex.UnixMilli(), m.jitterTTL(userDefaultExpireSeconds))
		return isAdmin, nil
	})
	if err != nil {
		return false, err
	}
	return v.(bool), nil
}
