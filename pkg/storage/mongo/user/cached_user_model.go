package user

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/syncx"
)

type cachedUserModel struct {
	UserModel
	userInfo   *userInfoCache
	userStatus *userStatusCache
	barrier    syncx.SingleFlight
}

func NewCachedUserModel(inner UserModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) UserModel {
	return &cachedUserModel{
		UserModel:  inner,
		userInfo:   newUserInfoCache(rdb),
		userStatus: newUserStatusCache(rdb),
		barrier:    barrier,
	}
}

// =====================================================
// UserInfo
// =====================================================

func (m *cachedUserModel) Insert(ctx context.Context, users []*model.User) error {
	err := m.UserModel.Insert(ctx, users)
	if err != nil {
		return err
	}
	for _, user := range users {
		m.userInfo.Del(ctx, user.UserID)
	}
	return nil
}

func (m *cachedUserModel) FindByIDs(ctx context.Context, userIDs []string) ([]*model.User, error) {
	result := make([]*model.User, 0, len(userIDs))
	missIDs := make([]string, 0, len(userIDs))

	for _, userID := range userIDs {
		user, found, err := m.userInfo.GetUserInfo(ctx, userID)
		if err != nil {
			return nil, err
		}
		if found {
			result = append(result, user)
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

	_, err := m.barrier.Do(sfKey, func() (any, error) {
		for _, uid := range missIDs {
			_, found2, err2 := m.userInfo.GetUserInfo(ctx, uid)
			if err2 != nil {
				return nil, err2
			}
			if found2 {
				continue
			}

			dbUser, errDB := m.UserModel.FindByID(ctx, uid)
			if errDB != nil {
				if errors.Is(errDB, ErrUserNotFound) {
					m.userInfo.SetUserInfo(ctx, uid, nil, 0)
					continue
				}
				return nil, errDB
			}
			m.userInfo.SetUserInfo(ctx, uid, dbUser, dbUser.UpdatedAt.UnixMilli())
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}

	for _, uid := range missIDs {
		user, found, err2 := m.userInfo.GetUserInfo(ctx, uid)
		if err2 != nil {
			return nil, err2
		}
		if found {
			result = append(result, user)
		}
	}

	return result, nil
}

func (m *cachedUserModel) FindByID(ctx context.Context, userID string) (*model.User, error) {
	if m.userInfo == nil {
		return m.UserModel.FindByID(ctx, userID)
	}

	user, found, err := m.userInfo.GetUserInfo(ctx, userID)
	if err != nil {
		return nil, err
	}
	if found {
		return user, nil
	}

	sfKey := sfKeyPrefixUserInfo + userID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		user2, found2, err2 := m.userInfo.GetUserInfo(ctx, userID)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			return user2, nil
		}

		dbUser, errDB := m.UserModel.FindByID(ctx, userID)
		if errDB != nil {
			if errors.Is(errDB, ErrUserNotFound) {
				m.userInfo.SetUserInfo(ctx, userID, nil, 0)
			}
			return nil, errDB
		}
		m.userInfo.SetUserInfo(ctx, userID, dbUser, dbUser.UpdatedAt.UnixMilli())
		return dbUser, nil
	})
	if err != nil {
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
	m.userInfo.Del(ctx, user.UserID)
	return nil
}

func (m *cachedUserModel) UpdateEx(ctx context.Context, userID string, updates map[string]any) error {
	err := m.UserModel.UpdateEx(ctx, userID, updates)
	if err != nil {
		return err
	}
	m.userInfo.Del(ctx, userID)
	return nil
}

func (m *cachedUserModel) Delete(ctx context.Context, userID string) error {
	err := m.UserModel.Delete(ctx, userID)
	if err != nil {
		return err
	}
	m.userInfo.Del(ctx, userID)
	return nil
}

func (m *cachedUserModel) CheckExists(ctx context.Context, userIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(userIDs))
	missIDs := make([]string, 0, len(userIDs))

	for _, userID := range userIDs {
		exists, found, err := m.userInfo.GetUserExists(ctx, userID)
		if err != nil {
			return nil, err
		}
		if found {
			result[userID] = exists
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
		for userID, exists := range dbResult {
			m.userInfo.SetUserExists(ctx, userID, exists)
			if _, ok := result[userID]; !ok {
				result[userID] = exists
			}
		}
		for _, userID := range missIDs {
			if _, ok := result[userID]; !ok {
				m.userInfo.SetUserExists(ctx, userID, false)
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
		exists, found, err2 := m.userInfo.GetUserExists(ctx, userID)
		if err2 != nil {
			return nil, err2
		}
		if found {
			result[userID] = exists
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
	m.userInfo.DelGlobalRecvOpt(ctx, userID)
	return nil
}

func (m *cachedUserModel) GetGlobalRecvMsgOpt(ctx context.Context, userID string) (int, error) {
	if m.userInfo == nil {
		return m.UserModel.GetGlobalRecvMsgOpt(ctx, userID)
	}

	opt, found, err := m.userInfo.GetGlobalRecvMsgOpt(ctx, userID)
	if err != nil {
		return 0, err
	}
	if found {
		return opt, nil
	}

	sfKey := sfKeyPrefixRecvOpt + userID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		opt2, found2, err2 := m.userInfo.GetGlobalRecvMsgOpt(ctx, userID)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			return opt2, nil
		}

		dbOpt, errDB := m.UserModel.GetGlobalRecvMsgOpt(ctx, userID)
		if errDB != nil {
			if errors.Is(errDB, ErrUserNotFound) {
				m.userInfo.SetGlobalRecvMsgOpt(ctx, userID, 0)
				return 0, nil
			}
			return nil, errDB
		}
		m.userInfo.SetGlobalRecvMsgOpt(ctx, userID, dbOpt)
		return dbOpt, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int), nil
}

func (m *cachedUserModel) IsIMAdmin(ctx context.Context, userID string) (bool, error) {
	if m.userInfo == nil {
		return m.UserModel.IsIMAdmin(ctx, userID)
	}

	isAdmin, found, err := m.userInfo.GetIMAdmin(ctx, userID)
	if err != nil {
		return false, err
	}
	if found {
		return isAdmin, nil
	}

	sfKey := sfKeyPrefixIMAdmin + userID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		isAdmin2, found2, err2 := m.userInfo.GetIMAdmin(ctx, userID)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			return isAdmin2, nil
		}

		dbIsAdmin, errDB := m.UserModel.IsIMAdmin(ctx, userID)
		if errDB != nil {
			if errors.Is(errDB, ErrUserNotFound) {
				m.userInfo.SetIMAdmin(ctx, userID, false)
				return false, nil
			}
			return nil, errDB
		}
		m.userInfo.SetIMAdmin(ctx, userID, dbIsAdmin)
		return dbIsAdmin, nil
	})
	if err != nil {
		return false, err
	}
	return v.(bool), nil
}

// =====================================================
// UserStatus 相关方法
// =====================================================

func (m *cachedUserModel) InsertUserStatus(ctx context.Context, status *model.UserStatus) error {
	err := m.UserModel.InsertUserStatus(ctx, status)
	if err != nil {
		return err
	}
	if status != nil && status.UserID != "" {
		m.userStatus.Del(ctx, status.UserID)
	}
	return nil
}

func (m *cachedUserModel) UpdateUserStatus(ctx context.Context, userID string, platformID int, deviceID string, status int) error {
	err := m.UserModel.UpdateUserStatus(ctx, userID, platformID, deviceID, status)
	if err != nil {
		return err
	}
	if userID == "" || m.userStatus == nil {
		return nil
	}
	if status == 1 {
		m.userStatus.SetOnline(ctx, userID, platformID, timex.UnixMilli())
	} else {
		m.userStatus.SetOffline(ctx, userID, platformID)
	}
	return nil
}

func (m *cachedUserModel) GetUserStatus(ctx context.Context, userIDs []string) ([]*model.UserStatus, error) {
	if m.userStatus == nil {
		return m.UserModel.GetUserStatus(ctx, userIDs)
	}
	if len(userIDs) == 0 {
		return []*model.UserStatus{}, nil
	}

	cacheRows, missIDs, err := m.userStatus.GetUserStatus(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	if len(missIDs) == 0 {
		return cacheRows, nil
	}

	sort.Strings(missIDs)
	sum := sha1.Sum([]byte(strings.Join(missIDs, ",")))
	sfKey := sfKeyPrefixBatchStatus + hex.EncodeToString(sum[:])

	sfRowsAny, err := m.barrier.Do(sfKey, func() (any, error) {
		sfRows := make(map[string][]*model.UserStatus, len(missIDs))
		realMiss := make([]string, 0, len(missIDs))

		for _, uid := range missIDs {
			rows, needDB, err2 := m.userStatus.zRowsForUser(ctx, uid)
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

		for _, uid := range realMiss {
			usrRows := groupMap[uid]

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
				for member, ms := range onlineMax {
					m.userStatus.SetOnline(ctx, uid, ParsePlatformZMember(member), ms)
				}
			} else {
				m.userStatus.SetNilMarker(ctx, uid)
			}
		}
		return sfRows, nil
	})
	if err != nil {
		return nil, err
	}
	sfRows, _ := sfRowsAny.(map[string][]*model.UserStatus)
	for _, uid := range missIDs {
		cacheRows = append(cacheRows, sfRows[uid]...)
	}

	return cacheRows, nil
}

func (m *cachedUserModel) SetUserOnlineStatus(ctx context.Context, statuses []*model.UserStatus) error {
	err := m.UserModel.SetUserOnlineStatus(ctx, statuses)
	if err != nil {
		return err
	}
	if len(statuses) == 0 || m.userStatus == nil {
		return nil
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

	for userID, pu := range perUserMap {
		if len(pu.online) > 0 {
			for member, ms := range pu.online {
				m.userStatus.SetOnline(ctx, userID, ParsePlatformZMember(member), ms)
			}
		}
		for _, member := range pu.offline {
			if _, ok := pu.online[member]; !ok {
				m.userStatus.SetOffline(ctx, userID, ParsePlatformZMember(member))
			}
		}
		if len(pu.online) == 0 && len(pu.offline) > 0 {
			m.userStatus.SetNilMarker(ctx, userID)
		}
	}

	return nil
}

func (m *cachedUserModel) GetAllOnlineUsers(ctx context.Context) ([]string, error) {
	if m.userStatus == nil {
		return m.UserModel.GetAllOnlineUsers(ctx)
	}

	userIDs, err := m.userStatus.GetAllOnlineUsers(ctx)
	if err != nil {
		return nil, err
	}
	if len(userIDs) > 0 {
		return userIDs, nil
	}

	sfKey := "user:all_online"
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		userIDs2, err2 := m.userStatus.GetAllOnlineUsers(ctx)
		if err2 != nil {
			return nil, err2
		}
		if len(userIDs2) > 0 {
			return userIDs2, nil
		}

		dbUserIDs, errDB := m.UserModel.GetAllOnlineUsers(ctx)
		if errDB != nil {
			return nil, errDB
		}

		if len(dbUserIDs) > 0 {
			for _, uid := range dbUserIDs {
				m.userStatus.AddToOnlineSet(ctx, uid)
			}
		}

		return dbUserIDs, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}
