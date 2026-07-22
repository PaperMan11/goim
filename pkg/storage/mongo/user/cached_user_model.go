package user

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

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

// --------------------------------------UserInfo Cache--------------------------------------

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

// --------------------------------------UserStatus Cache--------------------------------------

func (m *cachedUserModel) userStatusCacheKeys(userID string) []string {
	return []string{GetUserStatusZKey(userID), GetUserStatusZNilKey(userID)}
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
func (m *cachedUserModel) zRowsForUser(ctx context.Context, userID string) (rows []*model.UserStatus, needDB bool, err error) {
	rdb := m.redis
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
	//    并在清空后补 Nil Marker + 删空 key
	if hasZombie {
		zKey := GetUserStatusZKey(userID)
		nilKey := GetUserStatusZNilKey(userID)
		maxScore := fmt.Sprintf("(%d", cutoff)
		_, _ = sredis.CacheZRemRangeByScore(ctx, rdb, zKey, "-inf", maxScore)
		_ = sredis.CacheDelOnEmptyZ(ctx, rdb, zKey, nilKey, m.jitterTTL(userStatusZNilExpireSeconds))
	}

	// 4) 若 ZSET 有内容但全部是僵尸 → 缓存可信度低，需要回源 DB 再确认一次
	needDB = !hasAlive
	return rows, needDB, nil
}

func (m *cachedUserModel) GetUserStatus(ctx context.Context, userIDs []string) ([]*model.UserStatus, error) {
	if m.redis == nil {
		return m.UserModel.GetUserStatus(ctx, userIDs)
	}
	if len(userIDs) == 0 {
		return []*model.UserStatus{}, nil
	}

	result := make([]*model.UserStatus, 0, len(userIDs))
	missIDs := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		rows, needDB, err := m.zRowsForUser(ctx, userID)
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

	_, err := m.barrier.Do(sfKey, func() (any, error) {
		realMiss := make([]string, 0, len(missIDs))
		for _, uid := range missIDs {
			rows, needDB, err2 := m.zRowsForUser(ctx, uid)
			if err2 != nil {
				return nil, err2
			}
			if !needDB {
				result = append(result, rows...)
				continue
			}
			realMiss = append(realMiss, uid)
		}
		if len(realMiss) == 0 {
			return struct{}{}, nil
		}

		dbRows, errDB := m.UserModel.GetUserStatus(ctx, realMiss)
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
		expire := m.jitterTTL(userStatusZDefaultExpireSeconds)
		nilTTL := m.jitterTTL(userStatusZNilExpireSeconds)

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

			// 先把结果回填给调用方（按 platform 展开 Status=1 的 rows，与 ZSET 形态对齐）
			// 另外 Status=0 的 DB rows 不写入 ZSET 也不返回，逻辑层会按 user 维度自行预填兜底。
			for member, ms := range onlineMax {
				plat := ParsePlatformZMember(member)
				if plat <= 0 {
					continue
				}
				result = append(result, &model.UserStatus{
					UserID:     uid,
					PlatformID: plat,
					Status:     1,
					UpdatedAt:  time.UnixMilli(ms),
				})
			}

			if len(onlineMax) > 0 {
				_, _ = sredis.CacheZAddGTBatch(ctx, m.redis, zKey, onlineMax, expire)
				_ = m.redis.Del(ctx, nilKey).Err()
			} else {
				// 确认全离线 → 写 Nil Marker，删空 ZSET key
				_ = m.redis.Del(ctx, zKey).Err()
				_, _ = sredis.CacheSetCAS(ctx, m.redis, nilKey, nil, 0, nilTTL)
			}
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}

	for _, uid := range missIDs {
		rows, needDB, err2 := m.zRowsForUser(ctx, uid)
		if err2 != nil {
			return nil, err2
		}
		if needDB {
			continue
		}
		if len(rows) > 0 {
			result = append(result, rows...)
		}
	}

	return result, nil
}

func (m *cachedUserModel) InsertUserStatus(ctx context.Context, status *model.UserStatus) error {
	err := m.UserModel.InsertUserStatus(ctx, status)
	if err != nil {
		return err
	}
	if status != nil && status.UserID != "" && m.redis != nil {
		sredis.CacheDelDouble(ctx, m.redis, m.userStatusCacheKeys(status.UserID)...)
	}
	return nil
}

func (m *cachedUserModel) UpdateUserStatus(ctx context.Context, userID string, platformID int, deviceID string, status int) error {
	err := m.UserModel.UpdateUserStatus(ctx, userID, platformID, deviceID, status)
	if err != nil {
		return err
	}
	if userID == "" || m.redis == nil {
		return nil
	}

	nowMs := timex.UnixMilli()
	zKey := GetUserStatusZKey(userID)
	nilKey := GetUserStatusZNilKey(userID)
	expire := m.jitterTTL(userStatusZDefaultExpireSeconds)

	if status == 1 {
		// 在线：ZADD GT + 确保 zset key TTL + 清 Nil Marker
		member := PlatformZMember(platformID)
		_, _ = sredis.CacheZAddGT(ctx, m.redis, zKey, float64(nowMs), member, expire)
		_ = m.redis.Del(ctx, nilKey).Err()
	} else {
		// 离线：ZREM platform member，若 zset 变空则写 Nil Marker
		member := PlatformZMember(platformID)
		_, _ = sredis.CacheZRem(ctx, m.redis, zKey, member)
		_ = sredis.CacheDelOnEmptyZ(ctx, m.redis, zKey, nilKey, m.jitterTTL(userStatusZNilExpireSeconds))
	}
	return nil
}

func (m *cachedUserModel) SetUserOnlineStatus(ctx context.Context, statuses []*model.UserStatus) error {
	err := m.UserModel.SetUserOnlineStatus(ctx, statuses)
	if err != nil {
		return err
	}
	if len(statuses) == 0 || m.redis == nil {
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
		// 对于 status=0 且 status=1 在同一批同时出现（同 user+platform 不同 device），
		// 只要任一 device online 就算该 platform online（OR 语义）：
		//   先记上两种状态，循环结束后 online 优先（如果在 online map 中有 key，就不加入 offline）
		if s.Status == 1 {
			if prev, ok2 := pu.online[member]; !ok2 || ms > prev {
				pu.online[member] = ms
			}
		} else {
			pu.offline = append(pu.offline, member)
		}
	}

	expire := m.jitterTTL(userStatusZDefaultExpireSeconds)
	nilTTL := m.jitterTTL(userStatusZNilExpireSeconds)

	for userID, pu := range perUserMap {
		zKey := GetUserStatusZKey(userID)
		nilKey := GetUserStatusZNilKey(userID)

		// 1) 去重 offline：排除 offline 列表里同时在 online map 里的 member（任一 device online 即 platform online）
		if len(pu.offline) > 0 {
			remList := pu.offline[:0]
			for _, mbr := range pu.offline {
				if _, inOnline := pu.online[mbr]; inOnline {
					continue
				}
				remList = append(remList, mbr)
			}
			if len(remList) > 0 {
				_, _ = sredis.CacheZRem(ctx, m.redis, zKey, remList...)
			}
		}

		// 2) 写入 online (ZADD GT 批量)
		if len(pu.online) > 0 {
			_, _ = sredis.CacheZAddGTBatch(ctx, m.redis, zKey, pu.online, expire)
			_ = m.redis.Del(ctx, nilKey).Err()
		} else {
			// 这批里 user 全是 offline：清空后若 zset 空则写 Nil Marker
			_ = sredis.CacheDelOnEmptyZ(ctx, m.redis, zKey, nilKey, nilTTL)
		}
	}

	return nil
}
