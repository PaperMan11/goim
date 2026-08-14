package request

import (
	"context"
	"errors"

	"github.com/PaperMan11/goim/pkg/storage/model"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/syncx"
)

type cachedRequestModel struct {
	RequestModel
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

func NewCachedRequestModel(inner RequestModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) RequestModel {
	return &cachedRequestModel{
		RequestModel: inner,
		redis:        rdb,
		barrier:      barrier,
	}
}

func (m *cachedRequestModel) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (m *cachedRequestModel) InsertFriendRequest(ctx context.Context, req *model.FriendRequest) error {
	err := m.RequestModel.InsertFriendRequest(ctx, req)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetFriendRequestKey(req.FromUserID, req.ToUserID))
	return nil
}

func (m *cachedRequestModel) UpsertFriendRequest(ctx context.Context, req *model.FriendRequest) error {
	err := m.RequestModel.UpsertFriendRequest(ctx, req)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetFriendRequestKey(req.FromUserID, req.ToUserID))
	return nil
}

func (m *cachedRequestModel) HandleFriendRequest(ctx context.Context, fromUserID, toUserID, handlerUserID string, handleResult int, handleMsg string) error {
	err := m.RequestModel.HandleFriendRequest(ctx, fromUserID, toUserID, handlerUserID, handleResult, handleMsg)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetFriendRequestKey(fromUserID, toUserID))
	return nil
}

func (m *cachedRequestModel) FindFriendRequest(ctx context.Context, from, to string) (*model.FriendRequest, error) {
	if m.redis == nil {
		return m.RequestModel.FindFriendRequest(ctx, from, to)
	}

	var req model.FriendRequest
	key := GetFriendRequestKey(from, to)
	found, err := sredis.CacheGet(ctx, m.redis, key, &req)
	if err != nil {
		return nil, err
	}
	if found {
		if req.FromUserID == "" {
			return nil, ErrFriendRequestNotFound
		}
		return &req, nil
	}

	sfKey := sfKeyPrefixFriendReq + from + ":" + to
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerReq model.FriendRequest
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerReq)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerReq.FromUserID == "" {
				return nil, ErrFriendRequestNotFound
			}
			return &innerReq, nil
		}

		dbReq, err2 := m.RequestModel.FindFriendRequest(ctx, from, to)
		if err2 != nil {
			if errors.Is(err2, ErrFriendRequestNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(reqNilExpireSeconds))
			}
			return nil, err2
		}

		version := dbReq.CreateTime.UnixMilli()
		if !dbReq.HandleTime.IsZero() {
			handleVer := dbReq.HandleTime.UnixMilli()
			if handleVer > version {
				version = handleVer
			}
		}
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbReq, version, m.jitterTTL(reqDefaultExpireSeconds))
		return dbReq, nil
	})
	if err != nil {
		if errors.Is(err, ErrFriendRequestNotFound) {
			return nil, ErrFriendRequestNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrFriendRequestNotFound
	}
	return v.(*model.FriendRequest), nil
}

func (m *cachedRequestModel) DeleteFriendRequest(ctx context.Context, from, to string) error {
	err := m.RequestModel.DeleteFriendRequest(ctx, from, to)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetFriendRequestKey(from, to))
	return nil
}

func (m *cachedRequestModel) InsertGroupRequest(ctx context.Context, req *model.GroupRequest) error {
	err := m.RequestModel.InsertGroupRequest(ctx, req)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetGroupRequestKey(req.UserID, req.GroupID))
	return nil
}

func (m *cachedRequestModel) UpsertGroupRequest(ctx context.Context, req *model.GroupRequest) error {
	err := m.RequestModel.UpsertGroupRequest(ctx, req)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetGroupRequestKey(req.UserID, req.GroupID))
	return nil
}

func (m *cachedRequestModel) HandleGroupRequest(ctx context.Context, userID, groupID, handleUserID string, handleResult int, handleMsg string) error {
	err := m.RequestModel.HandleGroupRequest(ctx, userID, groupID, handleUserID, handleResult, handleMsg)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetGroupRequestKey(userID, groupID))
	return nil
}

func (m *cachedRequestModel) FindGroupRequest(ctx context.Context, userID, groupID string) (*model.GroupRequest, error) {
	if m.redis == nil {
		return m.RequestModel.FindGroupRequest(ctx, userID, groupID)
	}

	var req model.GroupRequest
	key := GetGroupRequestKey(userID, groupID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &req)
	if err != nil {
		return nil, err
	}
	if found {
		if req.UserID == "" {
			return nil, ErrGroupRequestNotFound
		}
		return &req, nil
	}

	sfKey := sfKeyPrefixGroupReq + userID + ":" + groupID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerReq model.GroupRequest
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerReq)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerReq.UserID == "" {
				return nil, ErrGroupRequestNotFound
			}
			return &innerReq, nil
		}

		dbReq, err2 := m.RequestModel.FindGroupRequest(ctx, userID, groupID)
		if err2 != nil {
			if errors.Is(err2, ErrGroupRequestNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(reqNilExpireSeconds))
			}
			return nil, err2
		}

		version := dbReq.ReqTime.UnixMilli()
		if !dbReq.HandleTime.IsZero() {
			handleVer := dbReq.HandleTime.UnixMilli()
			if handleVer > version {
				version = handleVer
			}
		}
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbReq, version, m.jitterTTL(reqDefaultExpireSeconds))
		return dbReq, nil
	})
	if err != nil {
		if errors.Is(err, ErrGroupRequestNotFound) {
			return nil, ErrGroupRequestNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrGroupRequestNotFound
	}
	return v.(*model.GroupRequest), nil
}

func (m *cachedRequestModel) CountGroupRequests(ctx context.Context, groupIDs []string, handleResults []int) (int64, error) {
	return m.RequestModel.CountGroupRequests(ctx, groupIDs, handleResults)
}

func (m *cachedRequestModel) DeleteGroupRequest(ctx context.Context, userID, groupID string) error {
	err := m.RequestModel.DeleteGroupRequest(ctx, userID, groupID)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetGroupRequestKey(userID, groupID))
	return nil
}
