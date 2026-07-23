package user

import (
	"context"

	"github.com/PaperMan11/goim/pkg/storage/model"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/syncx"
)

type cachedUserModel struct {
	UserModel
	userInfo   *userInfoCache
	userStatus *userStatusCache
}

func NewCachedUserModel(inner UserModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) UserModel {
	return &cachedUserModel{
		UserModel:  inner,
		userInfo:   newUserInfoCache(rdb, barrier),
		userStatus: newUserStatusCache(rdb, barrier),
	}
}

// =====================================================
// UserInfo 相关方法（委托给 userInfoCache）
// =====================================================

func (m *cachedUserModel) Insert(ctx context.Context, users []*model.User) error {
	err := m.UserModel.Insert(ctx, users)
	if err != nil {
		return err
	}
	for _, user := range users {
		m.userInfo.invalidate(ctx, user.UserID)
	}
	return nil
}

func (m *cachedUserModel) FindByIDs(ctx context.Context, userIDs []string) ([]*model.User, error) {
	return m.userInfo.FindByIDs(ctx, m.UserModel, userIDs)
}

func (m *cachedUserModel) FindByID(ctx context.Context, userID string) (*model.User, error) {
	return m.userInfo.FindByID(ctx, m.UserModel, userID)
}

func (m *cachedUserModel) Update(ctx context.Context, user *model.User) error {
	err := m.UserModel.Update(ctx, user)
	if err != nil {
		return err
	}
	m.userInfo.invalidate(ctx, user.UserID)
	return nil
}

func (m *cachedUserModel) UpdateEx(ctx context.Context, userID string, updates map[string]any) error {
	err := m.UserModel.UpdateEx(ctx, userID, updates)
	if err != nil {
		return err
	}
	m.userInfo.invalidate(ctx, userID)
	return nil
}

func (m *cachedUserModel) Delete(ctx context.Context, userID string) error {
	err := m.UserModel.Delete(ctx, userID)
	if err != nil {
		return err
	}
	m.userInfo.invalidate(ctx, userID)
	return nil
}

func (m *cachedUserModel) CheckExists(ctx context.Context, userIDs []string) (map[string]bool, error) {
	return m.userInfo.CheckExists(ctx, m.UserModel, userIDs)
}

func (m *cachedUserModel) SetGlobalRecvMsgOpt(ctx context.Context, userID string, opt int) error {
	err := m.UserModel.SetGlobalRecvMsgOpt(ctx, userID, opt)
	if err != nil {
		return err
	}
	m.userInfo.invalidateGlobalRecvOpt(ctx, userID)
	return nil
}

func (m *cachedUserModel) GetGlobalRecvMsgOpt(ctx context.Context, userID string) (int, error) {
	return m.userInfo.GetGlobalRecvMsgOpt(ctx, m.UserModel, userID)
}

func (m *cachedUserModel) IsIMAdmin(ctx context.Context, userID string) (bool, error) {
	return m.userInfo.IsIMAdmin(ctx, m.UserModel, userID)
}

// =====================================================
// UserStatus 相关方法（委托给 userStatusCache）
// =====================================================

func (m *cachedUserModel) InsertUserStatus(ctx context.Context, status *model.UserStatus) error {
	err := m.UserModel.InsertUserStatus(ctx, status)
	if err != nil {
		return err
	}
	if status != nil && status.UserID != "" {
		m.userStatus.invalidate(ctx, status.UserID)
	}
	return nil
}

func (m *cachedUserModel) UpdateUserStatus(ctx context.Context, userID string, platformID int, deviceID string, status int) error {
	return m.userStatus.UpdateUserStatus(ctx, m.UserModel, userID, platformID, deviceID, status)
}

func (m *cachedUserModel) GetUserStatus(ctx context.Context, userIDs []string) ([]*model.UserStatus, error) {
	return m.userStatus.GetUserStatus(ctx, m.UserModel, userIDs)
}

func (m *cachedUserModel) SetUserOnlineStatus(ctx context.Context, statuses []*model.UserStatus) error {
	return m.userStatus.SetUserOnlineStatus(ctx, m.UserModel, statuses)
}

// =====================================================
// 透传方法（直接调用 inner，无需缓存）
// =====================================================

func (m *cachedUserModel) Count(ctx context.Context) (int64, error) {
	return m.UserModel.Count(ctx)
}

func (m *cachedUserModel) Page(ctx context.Context, page, size int64, userID, nickname string) ([]*model.User, int64, error) {
	return m.UserModel.Page(ctx, page, size, userID, nickname)
}

func (m *cachedUserModel) SortQuery(ctx context.Context, userIDName map[string]string, asc bool) ([]*model.User, error) {
	return m.UserModel.SortQuery(ctx, userIDName, asc)
}

func (m *cachedUserModel) GetAllUserIDs(ctx context.Context, page, size int64) ([]string, int64, error) {
	return m.UserModel.GetAllUserIDs(ctx, page, size)
}

func (m *cachedUserModel) RegisterCount(ctx context.Context, start, end int64) (int64, int64, map[string]int64, error) {
	return m.UserModel.RegisterCount(ctx, start, end)
}

func (m *cachedUserModel) GetAllOnlineUsers(ctx context.Context) ([]string, error) {
	return m.UserModel.GetAllOnlineUsers(ctx)
}

func (m *cachedUserModel) InsertUserCommand(ctx context.Context, cmd *model.UserCommand) error {
	return m.UserModel.InsertUserCommand(ctx, cmd)
}

func (m *cachedUserModel) UpdateUserCommand(ctx context.Context, userID, uuid string, value string) error {
	return m.UserModel.UpdateUserCommand(ctx, userID, uuid, value)
}

func (m *cachedUserModel) DeleteUserCommand(ctx context.Context, userID, uuid string) error {
	return m.UserModel.DeleteUserCommand(ctx, userID, uuid)
}

func (m *cachedUserModel) GetUserCommand(ctx context.Context, userID, uuid string) (*model.UserCommand, error) {
	return m.UserModel.GetUserCommand(ctx, userID, uuid)
}

func (m *cachedUserModel) GetAllUserCommands(ctx context.Context, userID string) ([]*model.UserCommand, error) {
	return m.UserModel.GetAllUserCommands(ctx, userID)
}
