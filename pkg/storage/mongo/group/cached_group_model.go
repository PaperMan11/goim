package group

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

type cachedGroupModel struct {
	GroupModel
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

func NewCachedGroupModel(inner GroupModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) GroupModel {
	return &cachedGroupModel{
		GroupModel: inner,
		redis:      rdb,
		barrier:    barrier,
	}
}

func (m *cachedGroupModel) groupCacheKeys(gid string) []string {
	return []string{
		GetGroupInfoKey(gid),
		GetGroupExistsKey(gid),
		GetGroupMemberCountKey(gid),
	}
}

func (m *cachedGroupModel) memberCacheKeys(gid, uid string) []string {
	return []string{
		GetGroupMemberKey(gid, uid),
	}
}

func (m *cachedGroupModel) versionCacheKey(gid string) string {
	return GetGroupVersionKey(gid)
}

func (m *cachedGroupModel) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (m *cachedGroupModel) InsertGroup(ctx context.Context, group *model.Group) error {
	err := m.GroupModel.InsertGroup(ctx, group)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.groupCacheKeys(group.GroupID)...)
	return nil
}

func (m *cachedGroupModel) FindGroup(ctx context.Context, groupID string) (*model.Group, error) {
	if m.redis == nil {
		return m.GroupModel.FindGroup(ctx, groupID)
	}

	var group model.Group
	key := GetGroupInfoKey(groupID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &group)
	if err != nil {
		return nil, err
	}
	if found {
		if group.GroupName == "" {
			return nil, ErrGroupNotFound
		}
		return &group, nil
	}

	sfKey := sfKeyPrefixGroupInfo + groupID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerGroup model.Group
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerGroup)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerGroup.GroupName == "" {
				return nil, ErrGroupNotFound
			}
			return &innerGroup, nil
		}

		dbGroup, err2 := m.GroupModel.FindGroup(ctx, groupID)
		if err2 != nil {
			if errors.Is(err2, ErrGroupNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(groupNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbGroup.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbGroup, version, m.jitterTTL(groupDefaultExpireSeconds))
		return dbGroup, nil
	})
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrGroupNotFound
	}
	return v.(*model.Group), nil
}

func (m *cachedGroupModel) FindGroupsByIDs(ctx context.Context, groupIDs []string) ([]*model.Group, error) {
	if m.redis == nil {
		return m.GroupModel.FindGroupsByIDs(ctx, groupIDs)
	}

	result := make([]*model.Group, 0, len(groupIDs))
	missIDs := make([]string, 0, len(groupIDs))

	for _, groupID := range groupIDs {
		var group model.Group
		found, err := sredis.CacheGet(ctx, m.redis, GetGroupInfoKey(groupID), &group)
		if err != nil {
			return nil, err
		}
		if found && group.GroupName != "" {
			result = append(result, &group)
			continue
		}
		missIDs = append(missIDs, groupID)
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	sort.Strings(missIDs)
	sum := sha1.Sum([]byte(strings.Join(missIDs, ",")))
	sfKey := sfKeyPrefixMemberBatch + hex.EncodeToString(sum[:])

	v, err := m.barrier.Do(sfKey, func() (any, error) {
		for i, gid := range missIDs {
			g, errLoad := m.FindGroup(ctx, gid)
			if errLoad != nil && !errors.Is(errLoad, ErrGroupNotFound) {
				return nil, errLoad
			}
			if g != nil {
				missIDs[i] = ""
				result = append(result, g)
			}
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}
	_ = v

	for _, gid := range missIDs {
		if gid == "" {
			continue
		}
		var group model.Group
		found, err := sredis.CacheGet(ctx, m.redis, GetGroupInfoKey(gid), &group)
		if err != nil {
			return nil, err
		}
		if found && group.GroupName != "" {
			result = append(result, &group)
		}
	}

	return result, nil
}

func (m *cachedGroupModel) UpdateGroup(ctx context.Context, group *model.Group) error {
	err := m.GroupModel.UpdateGroup(ctx, group)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.groupCacheKeys(group.GroupID)...)
	return nil
}

func (m *cachedGroupModel) UpdateGroupEx(ctx context.Context, groupID string, updates map[string]any) error {
	err := m.GroupModel.UpdateGroupEx(ctx, groupID, updates)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.groupCacheKeys(groupID)...)
	return nil
}

func (m *cachedGroupModel) DeleteGroup(ctx context.Context, groupID string) error {
	err := m.GroupModel.DeleteGroup(ctx, groupID)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.groupCacheKeys(groupID)...)
	return nil
}

func (m *cachedGroupModel) CheckGroupExists(ctx context.Context, groupIDs []string) (map[string]bool, error) {
	if m.redis == nil {
		return m.GroupModel.CheckGroupExists(ctx, groupIDs)
	}

	result := make(map[string]bool, len(groupIDs))
	missIDs := make([]string, 0, len(groupIDs))

	for _, groupID := range groupIDs {
		data, found, err := sredis.CacheGetString(ctx, m.redis, GetGroupExistsKey(groupID))
		if err != nil {
			return nil, err
		}
		if found {
			result[groupID] = data == "1"
			continue
		}
		missIDs = append(missIDs, groupID)
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	sort.Strings(missIDs)
	sum := sha1.Sum([]byte(strings.Join(missIDs, ",")))
	sfKey := sfKeyPrefixExists + hex.EncodeToString(sum[:])

	_, err := m.barrier.Do(sfKey, func() (any, error) {
		dbResult, errDB := m.GroupModel.CheckGroupExists(ctx, missIDs)
		if errDB != nil {
			return nil, errDB
		}
		nowMs := timex.UnixMilli()
		for groupID, exists := range dbResult {
			val := "0"
			if exists {
				val = "1"
			}
			_, _ = sredis.CacheSetCASString(ctx, m.redis, GetGroupExistsKey(groupID), val, nowMs, m.jitterTTL(groupDefaultExpireSeconds))
			if _, ok := result[groupID]; !ok {
				result[groupID] = exists
			}
		}
		for _, groupID := range missIDs {
			if _, ok := result[groupID]; !ok {
				_, _ = sredis.CacheSetCASString(ctx, m.redis, GetGroupExistsKey(groupID), "0", 0, m.jitterTTL(groupNilExpireSeconds))
				result[groupID] = false
			}
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}

	for _, groupID := range missIDs {
		if _, ok := result[groupID]; ok {
			continue
		}
		data, found, errStr := sredis.CacheGetString(ctx, m.redis, GetGroupExistsKey(groupID))
		if errStr != nil {
			return nil, errStr
		}
		if found {
			result[groupID] = data == "1"
		} else {
			result[groupID] = false
		}
	}

	return result, nil
}

func (m *cachedGroupModel) InsertMember(ctx context.Context, member *model.GroupMember) error {
	err := m.GroupModel.InsertMember(ctx, member)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.memberCacheKeys(member.GroupID, member.UserID)...)
	sredis.CacheDelDouble(ctx, m.redis, GetGroupMemberCountKey(member.GroupID))
	return nil
}

func (m *cachedGroupModel) InsertMembers(ctx context.Context, members []*model.GroupMember) error {
	err := m.GroupModel.InsertMembers(ctx, members)
	if err != nil {
		return err
	}
	delKeys := make([]string, 0)
	countKeys := make(map[string]struct{})
	for _, member := range members {
		delKeys = append(delKeys, m.memberCacheKeys(member.GroupID, member.UserID)...)
		countKeys[member.GroupID] = struct{}{}
	}
	sredis.CacheDelDouble(ctx, m.redis, delKeys...)
	for gid := range countKeys {
		sredis.CacheDelDouble(ctx, m.redis, GetGroupMemberCountKey(gid))
	}
	return nil
}

func (m *cachedGroupModel) UpdateMember(ctx context.Context, groupID, userID string, updates map[string]any) error {
	err := m.GroupModel.UpdateMember(ctx, groupID, userID, updates)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.memberCacheKeys(groupID, userID)...)
	return nil
}

func (m *cachedGroupModel) UpsertMember(ctx context.Context, member *model.GroupMember) error {
	err := m.GroupModel.UpsertMember(ctx, member)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.memberCacheKeys(member.GroupID, member.UserID)...)
	sredis.CacheDelDouble(ctx, m.redis, GetGroupMemberCountKey(member.GroupID))
	return nil
}

func (m *cachedGroupModel) DeleteMember(ctx context.Context, groupID, userID string) error {
	err := m.GroupModel.DeleteMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.memberCacheKeys(groupID, userID)...)
	sredis.CacheDelDouble(ctx, m.redis, GetGroupMemberCountKey(groupID))
	return nil
}

func (m *cachedGroupModel) DeleteMembers(ctx context.Context, groupID string, userIDs []string) error {
	err := m.GroupModel.DeleteMembers(ctx, groupID, userIDs)
	if err != nil {
		return err
	}
	delKeys := make([]string, 0, len(userIDs))
	for _, uid := range userIDs {
		delKeys = append(delKeys, m.memberCacheKeys(groupID, uid)...)
	}
	sredis.CacheDelDouble(ctx, m.redis, delKeys...)
	sredis.CacheDelDouble(ctx, m.redis, GetGroupMemberCountKey(groupID))
	return nil
}

func (m *cachedGroupModel) FindMember(ctx context.Context, groupID, userID string) (*model.GroupMember, error) {
	if m.redis == nil {
		return m.GroupModel.FindMember(ctx, groupID, userID)
	}

	var member model.GroupMember
	key := GetGroupMemberKey(groupID, userID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &member)
	if err != nil {
		return nil, err
	}
	if found {
		if member.GroupID == "" || member.UserID == "" {
			return nil, ErrGroupMemberNotFound
		}
		return &member, nil
	}

	sfKey := sfKeyPrefixMember + groupID + ":" + userID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerMember model.GroupMember
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerMember)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerMember.GroupID == "" || innerMember.UserID == "" {
				return nil, ErrGroupMemberNotFound
			}
			return &innerMember, nil
		}

		dbMember, err2 := m.GroupModel.FindMember(ctx, groupID, userID)
		if err2 != nil {
			if errors.Is(err2, ErrGroupMemberNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(groupNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbMember.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbMember, version, m.jitterTTL(groupDefaultExpireSeconds))
		return dbMember, nil
	})
	if err != nil {
		if errors.Is(err, ErrGroupMemberNotFound) {
			return nil, ErrGroupMemberNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrGroupMemberNotFound
	}
	return v.(*model.GroupMember), nil
}

func (m *cachedGroupModel) CountMembers(ctx context.Context, groupID string) (int64, error) {
	if m.redis == nil {
		return m.GroupModel.CountMembers(ctx, groupID)
	}

	key := GetGroupMemberCountKey(groupID)
	data, found, err := sredis.CacheGetString(ctx, m.redis, key)
	if err != nil {
		return 0, err
	}
	if found {
		val, errConv := convert.ToInt64E(data)
		if errConv == nil {
			return val, nil
		}
		_ = sredis.CacheDel(ctx, m.redis, key)
	}

	sfKey := sfKeyPrefixMemberCount + groupID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		data2, found2, err2 := sredis.CacheGetString(ctx, m.redis, key)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			val, errConv := convert.ToInt64E(data2)
			if errConv == nil {
				return val, nil
			}
			_ = sredis.CacheDel(ctx, m.redis, key)
		}

		count, err2 := m.GroupModel.CountMembers(ctx, groupID)
		if err2 != nil {
			return nil, err2
		}
		_, _ = sredis.CacheSetCASString(ctx, m.redis, key, convert.ToString(count), timex.UnixMilli(), m.jitterTTL(groupDefaultExpireSeconds))
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}

func (m *cachedGroupModel) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	member, err := m.FindMember(ctx, groupID, userID)
	if err != nil {
		if errors.Is(err, ErrGroupMemberNotFound) {
			return false, nil
		}
		return false, err
	}
	return member != nil, nil
}

func (m *cachedGroupModel) GetMemberRole(ctx context.Context, groupID, userID string) (int, error) {
	member, err := m.FindMember(ctx, groupID, userID)
	if err != nil {
		return 0, err
	}
	return member.RoleLevel, nil
}

func (m *cachedGroupModel) IncrMemberCount(ctx context.Context, groupID string, delta int) error {
	err := m.GroupModel.IncrMemberCount(ctx, groupID, delta)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetGroupMemberCountKey(groupID))
	return nil
}

func (m *cachedGroupModel) UpsertGroupVersion(ctx context.Context, ver *model.GroupVersion) error {
	err := m.GroupModel.UpsertGroupVersion(ctx, ver)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.versionCacheKey(ver.GroupID))
	return nil
}

func (m *cachedGroupModel) GetGroupVersion(ctx context.Context, groupID string) (*model.GroupVersion, error) {
	if m.redis == nil {
		return m.GroupModel.GetGroupVersion(ctx, groupID)
	}

	var ver model.GroupVersion
	key := m.versionCacheKey(groupID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &ver)
	if err != nil {
		return nil, err
	}
	if found {
		if ver.GroupID == "" {
			return nil, ErrGroupVersionNotFound
		}
		return &ver, nil
	}

	sfKey := sfKeyPrefixVersion + groupID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerVer model.GroupVersion
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerVer)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerVer.GroupID == "" {
				return nil, ErrGroupVersionNotFound
			}
			return &innerVer, nil
		}

		dbVer, err2 := m.GroupModel.GetGroupVersion(ctx, groupID)
		if err2 != nil {
			if errors.Is(err2, ErrGroupVersionNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(groupNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbVer.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbVer, version, m.jitterTTL(groupDefaultExpireSeconds))
		return dbVer, nil
	})
	if err != nil {
		if errors.Is(err, ErrGroupVersionNotFound) {
			return nil, ErrGroupVersionNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrGroupVersionNotFound
	}
	return v.(*model.GroupVersion), nil
}

func (m *cachedGroupModel) IncrGroupMemberVersion(ctx context.Context, groupID string) (*model.GroupVersion, error) {
	ver, err := m.GroupModel.IncrGroupMemberVersion(ctx, groupID)
	if err != nil {
		return nil, err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.versionCacheKey(groupID))
	return ver, nil
}
