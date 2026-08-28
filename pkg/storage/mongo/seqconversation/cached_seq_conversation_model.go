package seqconversation

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

type cachedSeqConversationModel struct {
	SeqConversationModel
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

func NewCachedSeqConversationModel(inner SeqConversationModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) SeqConversationModel {
	return &cachedSeqConversationModel{
		SeqConversationModel: inner,
		redis:                rdb,
		barrier:              barrier,
	}
}

func (m *cachedSeqConversationModel) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (m *cachedSeqConversationModel) SetConversationMaxSeq(ctx context.Context, conversationID string, maxSeq int64) error {
	err := m.SeqConversationModel.SetConversationMaxSeq(ctx, conversationID, maxSeq)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetSeqConvKey(conversationID))
	return nil
}

func (m *cachedSeqConversationModel) SetConversationMinSeq(ctx context.Context, conversationID string, minSeq int64) error {
	err := m.SeqConversationModel.SetConversationMinSeq(ctx, conversationID, minSeq)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetSeqConvKey(conversationID))
	return nil
}

func (m *cachedSeqConversationModel) GetSeqConversation(ctx context.Context, conversationID string) (*model.SeqConversation, error) {
	if m.redis == nil {
		return m.SeqConversationModel.GetSeqConversation(ctx, conversationID)
	}

	var seq model.SeqConversation
	key := GetSeqConvKey(conversationID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &seq)
	if err != nil {
		return nil, err
	}
	if found {
		if seq.ConversationID == "" {
			return nil, ErrSeqConversationNotFound
		}
		return &seq, nil
	}

	sfKey := sfKeyPrefixSeqConv + conversationID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerSeq model.SeqConversation
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerSeq)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerSeq.ConversationID == "" {
				return nil, ErrSeqConversationNotFound
			}
			return &innerSeq, nil
		}

		dbSeq, err2 := m.SeqConversationModel.GetSeqConversation(ctx, conversationID)
		if err2 != nil {
			if errors.Is(err2, ErrSeqConversationNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(seqNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbSeq.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbSeq, version, m.jitterTTL(seqDefaultExpireSeconds))
		return dbSeq, nil
	})
	if err != nil {
		if errors.Is(err, ErrSeqConversationNotFound) {
			return nil, ErrSeqConversationNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrSeqConversationNotFound
	}
	return v.(*model.SeqConversation), nil
}
