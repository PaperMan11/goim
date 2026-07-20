package user

import (
	"context"
	"errors"
	"strconv"

	"github.com/PaperMan11/goim/pkg/storage/model"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	goredis "github.com/redis/go-redis/v9"
)

const (
	userDefaultExpireSeconds = 5 * 60
	userNilExpireSeconds     = 60
)

type cachedUserModel struct {
	UserModel
	redis goredis.UniversalClient
}

func NewCachedUserModel(inner UserModel, rdb goredis.UniversalClient) UserModel {
	return &cachedUserModel{
		UserModel: inner,
		redis:     rdb,
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

func (m *cachedUserModel) Insert(ctx context.Context, users []*model.User) error {
	err := m.UserModel.Insert(ctx, users)
	if err != nil {
		return err
	}
	if m.redis != nil {
		for _, user := range users {
			_ = sredis.CacheDel(ctx, m.redis, m.userCacheKeys(user.UserID)...)
		}
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

	dbUsers, err := m.UserModel.FindByIDs(ctx, missIDs)
	if err != nil {
		return nil, err
	}

	foundIDs := make(map[string]bool, len(dbUsers))
	for _, user := range dbUsers {
		_ = sredis.CacheSet(ctx, m.redis, GetUserInfoKey(user.UserID), user, userDefaultExpireSeconds)
		foundIDs[user.UserID] = true
		result = append(result, user)
	}

	for _, missID := range missIDs {
		if !foundIDs[missID] {
			_ = sredis.CacheSet(ctx, m.redis, GetUserInfoKey(missID), nil, userNilExpireSeconds)
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

	dbUser, err := m.UserModel.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = sredis.CacheSet(ctx, m.redis, key, nil, userNilExpireSeconds)
		}
		return nil, err
	}
	_ = sredis.CacheSet(ctx, m.redis, key, dbUser, userDefaultExpireSeconds)
	return dbUser, nil
}

func (m *cachedUserModel) Update(ctx context.Context, user *model.User) error {
	err := m.UserModel.Update(ctx, user)
	if err != nil {
		return err
	}
	_ = sredis.CacheDel(ctx, m.redis, m.userCacheKeys(user.UserID)...)
	return nil
}

func (m *cachedUserModel) UpdateEx(ctx context.Context, userID string, updates map[string]any) error {
	err := m.UserModel.UpdateEx(ctx, userID, updates)
	if err != nil {
		return err
	}
	_ = sredis.CacheDel(ctx, m.redis, m.userCacheKeys(userID)...)
	return nil
}

func (m *cachedUserModel) Delete(ctx context.Context, userID string) error {
	err := m.UserModel.Delete(ctx, userID)
	if err != nil {
		return err
	}
	_ = sredis.CacheDel(ctx, m.redis, m.userCacheKeys(userID)...)
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

	dbResult, err := m.UserModel.CheckExists(ctx, missIDs)
	if err != nil {
		return nil, err
	}

	for userID, exists := range dbResult {
		val := "0"
		if exists {
			val = "1"
		}
		_ = sredis.CacheSetString(ctx, m.redis, GetUserExistsKey(userID), val, userDefaultExpireSeconds)
		result[userID] = exists
	}

	return result, nil
}

func (m *cachedUserModel) SetGlobalRecvMsgOpt(ctx context.Context, userID string, opt int) error {
	err := m.UserModel.SetGlobalRecvMsgOpt(ctx, userID, opt)
	if err != nil {
		return err
	}
	if m.redis != nil && userID != "" {
		_ = sredis.CacheDel(ctx, m.redis, GetUserGlobalRecvOptKey(userID), GetUserInfoKey(userID))
	}
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
		val, err := strconv.Atoi(data)
		if err == nil {
			return val, nil
		}
		_ = sredis.CacheDel(ctx, m.redis, key)
	}

	opt, err := m.UserModel.GetGlobalRecvMsgOpt(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = sredis.CacheSetString(ctx, m.redis, key, "0", userNilExpireSeconds)
			return 0, nil
		}
		return 0, err
	}
	_ = sredis.CacheSetString(ctx, m.redis, key, strconv.Itoa(opt), userDefaultExpireSeconds)
	return opt, nil
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

	isAdmin, err := m.UserModel.IsIMAdmin(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = sredis.CacheSetString(ctx, m.redis, key, "0", userNilExpireSeconds)
			return false, nil
		}
		return false, err
	}
	val := "0"
	if isAdmin {
		val = "1"
	}
	_ = sredis.CacheSetString(ctx, m.redis, key, val, userDefaultExpireSeconds)
	return isAdmin, nil
}
