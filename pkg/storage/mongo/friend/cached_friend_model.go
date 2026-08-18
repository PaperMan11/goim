package friend

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"sort"
	"strings"

	"github.com/PaperMan11/goim/pkg/storage/model"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/syncx"
)

type cachedFriendModel struct {
	FriendModel
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

func NewCachedFriendModel(inner FriendModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) FriendModel {
	return &cachedFriendModel{
		FriendModel: inner,
		redis:       rdb,
		barrier:     barrier,
	}
}

func (m *cachedFriendModel) friendCacheKeys(owner, friendID string) []string {
	return []string{
		GetFriendInfoKey(owner, friendID),
		GetFriendExistsKey(owner, friendID),
	}
}

func (m *cachedFriendModel) blackCacheKeys(owner, blackID string) []string {
	return []string{
		GetBlackInfoKey(owner, blackID),
	}
}

func (m *cachedFriendModel) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (m *cachedFriendModel) InsertFriend(ctx context.Context, friend *model.Friend) error {
	err := m.FriendModel.InsertFriend(ctx, friend)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.friendCacheKeys(friend.OwnerUserID, friend.FriendUserID)...)
	return nil
}

func (m *cachedFriendModel) InsertFriends(ctx context.Context, friends []*model.Friend) error {
	err := m.FriendModel.InsertFriends(ctx, friends)
	if err != nil {
		return err
	}
	for _, f := range friends {
		sredis.CacheDelDouble(ctx, m.redis, m.friendCacheKeys(f.OwnerUserID, f.FriendUserID)...)
		sredis.CacheDelDouble(ctx, m.redis, m.friendCacheKeys(f.FriendUserID, f.OwnerUserID)...)
	}
	return nil
}

func (m *cachedFriendModel) UpdateFriend(ctx context.Context, owner, friendUserID string, updates map[string]any) error {
	err := m.FriendModel.UpdateFriend(ctx, owner, friendUserID, updates)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.friendCacheKeys(owner, friendUserID)...)
	return nil
}

func (m *cachedFriendModel) DeleteFriend(ctx context.Context, owner, friendUserID string) error {
	err := m.FriendModel.DeleteFriend(ctx, owner, friendUserID)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.friendCacheKeys(owner, friendUserID)...)
	return nil
}

func (m *cachedFriendModel) DeleteFriendBoth(ctx context.Context, userA, userB string) error {
	err := m.FriendModel.DeleteFriendBoth(ctx, userA, userB)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.friendCacheKeys(userA, userB)...)
	sredis.CacheDelDouble(ctx, m.redis, m.friendCacheKeys(userB, userA)...)
	return nil
}

func (m *cachedFriendModel) FindFriend(ctx context.Context, owner, friendUserID string) (*model.Friend, error) {
	if m.redis == nil {
		return m.FriendModel.FindFriend(ctx, owner, friendUserID)
	}

	var friend model.Friend
	key := GetFriendInfoKey(owner, friendUserID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &friend)
	if err != nil {
		return nil, err
	}
	if found {
		if friend.FriendUserID == "" {
			return nil, ErrFriendNotFound
		}
		return &friend, nil
	}

	sfKey := sfKeyPrefixFriend + owner + ":" + friendUserID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerFriend model.Friend
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerFriend)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerFriend.FriendUserID == "" {
				return nil, ErrFriendNotFound
			}
			return &innerFriend, nil
		}

		dbFriend, err2 := m.FriendModel.FindFriend(ctx, owner, friendUserID)
		if err2 != nil {
			if errors.Is(err2, ErrFriendNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(friendNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbFriend.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbFriend, version, m.jitterTTL(friendDefaultExpireSeconds))
		return dbFriend, nil
	})
	if err != nil {
		if errors.Is(err, ErrFriendNotFound) {
			return nil, ErrFriendNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrFriendNotFound
	}
	return v.(*model.Friend), nil
}

func (m *cachedFriendModel) FindFriendsByOwner(ctx context.Context, owner string) ([]*model.Friend, error) {
	return m.FriendModel.FindFriendsByOwner(ctx, owner)
}

func (m *cachedFriendModel) FindFriendsByIDs(ctx context.Context, owner string, friendIDs []string) ([]*model.Friend, error) {
	if m.redis == nil {
		return m.FriendModel.FindFriendsByIDs(ctx, owner, friendIDs)
	}

	result := make([]*model.Friend, 0, len(friendIDs))
	missIDs := make([]string, 0, len(friendIDs))

	for _, fid := range friendIDs {
		var friend model.Friend
		found, err := sredis.CacheGet(ctx, m.redis, GetFriendInfoKey(owner, fid), &friend)
		if err != nil {
			return nil, err
		}
		if found && friend.FriendUserID != "" {
			result = append(result, &friend)
			continue
		}
		missIDs = append(missIDs, fid)
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	sort.Strings(missIDs)
	sum := sha1.Sum([]byte(owner + "," + strings.Join(missIDs, ",")))
	sfKey := sfKeyPrefixFriendBatch + hex.EncodeToString(sum[:])

	_, err := m.barrier.Do(sfKey, func() (any, error) {
		for i, fid := range missIDs {
			f, errLoad := m.FindFriend(ctx, owner, fid)
			if errLoad != nil && !errors.Is(errLoad, ErrFriendNotFound) {
				return nil, errLoad
			}
			if f != nil {
				missIDs[i] = ""
				result = append(result, f)
			}
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}

	for _, fid := range missIDs {
		if fid == "" {
			continue
		}
		var friend model.Friend
		found, err := sredis.CacheGet(ctx, m.redis, GetFriendInfoKey(owner, fid), &friend)
		if err != nil {
			return nil, err
		}
		if found && friend.FriendUserID != "" {
			result = append(result, &friend)
		}
	}

	return result, nil
}

func (m *cachedFriendModel) IsFriend(ctx context.Context, userA, userB string) (bool, error) {
	if m.redis == nil {
		return m.FriendModel.IsFriend(ctx, userA, userB)
	}

	keyAB := GetFriendExistsKey(userA, userB)
	dataAB, foundAB, err := sredis.CacheGetString(ctx, m.redis, keyAB)
	if err != nil {
		return false, err
	}
	keyBA := GetFriendExistsKey(userB, userA)
	dataBA, foundBA, err := sredis.CacheGetString(ctx, m.redis, keyBA)
	if err != nil {
		return false, err
	}

	if foundAB && foundBA {
		return dataAB == "1" && dataBA == "1", nil
	}

	sfKey := sfKeyPrefixFriendExists + userA + ":" + userB
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		dataAB2, foundAB2, err2 := sredis.CacheGetString(ctx, m.redis, keyAB)
		if err2 != nil {
			return nil, err2
		}
		dataBA2, foundBA2, err2 := sredis.CacheGetString(ctx, m.redis, keyBA)
		if err2 != nil {
			return nil, err2
		}
		if foundAB2 && foundBA2 {
			return dataAB2 == "1" && dataBA2 == "1", nil
		}

		isFriend, err2 := m.FriendModel.IsFriend(ctx, userA, userB)
		if err2 != nil {
			return nil, err2
		}

		nowMs := timex.UnixMilli()
		val := "0"
		if isFriend {
			val = "1"
		}

		_, _ = sredis.CacheSetCASString(ctx, m.redis, keyAB, val, nowMs, m.jitterTTL(friendDefaultExpireSeconds))
		_, _ = sredis.CacheSetCASString(ctx, m.redis, keyBA, val, nowMs, m.jitterTTL(friendDefaultExpireSeconds))

		return isFriend, nil
	})
	if err != nil {
		return false, err
	}
	return v.(bool), nil
}

func (m *cachedFriendModel) CountFriends(ctx context.Context, owner string) (int64, error) {
	return m.FriendModel.CountFriends(ctx, owner)
}

func (m *cachedFriendModel) InsertBlack(ctx context.Context, black *model.Black) error {
	err := m.FriendModel.InsertBlack(ctx, black)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.blackCacheKeys(black.OwnerUserID, black.BlackUserID)...)
	return nil
}

func (m *cachedFriendModel) DeleteBlack(ctx context.Context, owner, blackUserID string) error {
	err := m.FriendModel.DeleteBlack(ctx, owner, blackUserID)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.blackCacheKeys(owner, blackUserID)...)
	return nil
}

func (m *cachedFriendModel) FindBlack(ctx context.Context, owner, blackUserID string) (*model.Black, error) {
	if m.redis == nil {
		return m.FriendModel.FindBlack(ctx, owner, blackUserID)
	}

	var black model.Black
	key := GetBlackInfoKey(owner, blackUserID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &black)
	if err != nil {
		return nil, err
	}
	if found {
		if black.BlackUserID == "" {
			return nil, ErrBlackNotFound
		}
		return &black, nil
	}

	sfKey := sfKeyPrefixBlack + owner + ":" + blackUserID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerBlack model.Black
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerBlack)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerBlack.BlackUserID == "" {
				return nil, ErrBlackNotFound
			}
			return &innerBlack, nil
		}

		dbBlack, err2 := m.FriendModel.FindBlack(ctx, owner, blackUserID)
		if err2 != nil {
			if errors.Is(err2, ErrBlackNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(friendNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbBlack.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbBlack, version, m.jitterTTL(friendDefaultExpireSeconds))
		return dbBlack, nil
	})
	if err != nil {
		if errors.Is(err, ErrBlackNotFound) {
			return nil, ErrBlackNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrBlackNotFound
	}
	return v.(*model.Black), nil
}

func (m *cachedFriendModel) FindBlacksByOwner(ctx context.Context, owner string) ([]*model.Black, error) {
	return m.FriendModel.FindBlacksByOwner(ctx, owner)
}

func (m *cachedFriendModel) IsBlack(ctx context.Context, owner, targetUserID string) (bool, error) {
	return m.FriendModel.IsBlack(ctx, owner, targetUserID)
}
