package sequser

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

type cachedSeqUserModel struct {
	SeqUserModel
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

func NewCachedSeqUserModel(inner SeqUserModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) SeqUserModel {
	return &cachedSeqUserModel{
		SeqUserModel: inner,
		redis:        rdb,
		barrier:      barrier,
	}
}

func (m *cachedSeqUserModel) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (m *cachedSeqUserModel) UpsertUserSeq(ctx context.Context, userID, conversationID string, minSeq, maxSeq, readSeq int64) error {
	err := m.SeqUserModel.UpsertUserSeq(ctx, userID, conversationID, minSeq, maxSeq, readSeq)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetSeqUserKey(userID, conversationID))
	return nil
}

func (m *cachedSeqUserModel) SetUserReadSeq(ctx context.Context, userID, conversationID string, readSeq int64) error {
	err := m.SeqUserModel.SetUserReadSeq(ctx, userID, conversationID, readSeq)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetSeqUserKey(userID, conversationID))
	return nil
}

func (m *cachedSeqUserModel) SetUserMaxSeq(ctx context.Context, userID, conversationID string, maxSeq int64) error {
	err := m.SeqUserModel.SetUserMaxSeq(ctx, userID, conversationID, maxSeq)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetSeqUserKey(userID, conversationID))
	return nil
}

func (m *cachedSeqUserModel) SetUserMinSeq(ctx context.Context, userID, conversationID string, minSeq int64) error {
	err := m.SeqUserModel.SetUserMinSeq(ctx, userID, conversationID, minSeq)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetSeqUserKey(userID, conversationID))
	return nil
}

func (m *cachedSeqUserModel) GetUserSeq(ctx context.Context, userID, conversationID string) (*model.SeqUser, error) {
	if m.redis == nil {
		return m.SeqUserModel.GetUserSeq(ctx, userID, conversationID)
	}

	var seq model.SeqUser
	key := GetSeqUserKey(userID, conversationID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &seq)
	if err != nil {
		return nil, err
	}
	if found {
		if seq.UserID == "" {
			return nil, ErrSeqUserNotFound
		}
		return &seq, nil
	}

	sfKey := sfKeyPrefixSeqUser + userID + ":" + conversationID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerSeq model.SeqUser
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerSeq)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerSeq.UserID == "" {
				return nil, ErrSeqUserNotFound
			}
			return &innerSeq, nil
		}

		dbSeq, err2 := m.SeqUserModel.GetUserSeq(ctx, userID, conversationID)
		if err2 != nil {
			if errors.Is(err2, ErrSeqUserNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(userSeqNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbSeq.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbSeq, version, m.jitterTTL(userSeqDefaultExpireSeconds))
		return dbSeq, nil
	})
	if err != nil {
		if errors.Is(err, ErrSeqUserNotFound) {
			return nil, ErrSeqUserNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrSeqUserNotFound
	}
	return v.(*model.SeqUser), nil
}
