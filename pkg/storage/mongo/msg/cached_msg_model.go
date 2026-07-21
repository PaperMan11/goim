package msg

import (
	"context"
	"errors"

	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/PaperMan11/goim/pkg/utils/convert"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/syncx"
)

type cachedMsgModel struct {
	MsgModel
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

func NewCachedMsgModel(inner MsgModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) MsgModel {
	return &cachedMsgModel{
		MsgModel: inner,
		redis:    rdb,
		barrier:  barrier,
	}
}

func (m *cachedMsgModel) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (m *cachedMsgModel) msgSeqCacheKeys(convID string, seqs []int64) []string {
	keys := make([]string, 0, len(seqs)+2)
	for _, seq := range seqs {
		keys = append(keys, GetMsgBySeqKey(convID, seq))
	}
	return keys
}

func (m *cachedMsgModel) Insert(ctx context.Context, msg *model.Message) error {
	err := m.MsgModel.Insert(ctx, msg)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis,
		GetMsgBySeqKey(msg.ConversationID, msg.Seq),
		GetMsgLatestKey(msg.ConversationID),
		GetMsgMaxSeqKey(msg.ConversationID),
		GetMsgMinSeqKey(msg.ConversationID),
	)
	return nil
}

func (m *cachedMsgModel) BatchInsert(ctx context.Context, msgs []*model.Message) error {
	err := m.MsgModel.BatchInsert(ctx, msgs)
	if err != nil {
		return err
	}
	seenConv := make(map[string]struct{})
	for _, msg := range msgs {
		sredis.CacheDelDouble(ctx, m.redis,
			GetMsgBySeqKey(msg.ConversationID, msg.Seq),
		)
		if _, ok := seenConv[msg.ConversationID]; !ok {
			seenConv[msg.ConversationID] = struct{}{}
			sredis.CacheDelDouble(ctx, m.redis,
				GetMsgLatestKey(msg.ConversationID),
				GetMsgMaxSeqKey(msg.ConversationID),
				GetMsgMinSeqKey(msg.ConversationID),
			)
		}
	}
	return nil
}

func (m *cachedMsgModel) UpdateRevoke(ctx context.Context, conversationID string, seq int64, revokedContent *model.MessageRevokedContent) error {
	err := m.MsgModel.UpdateRevoke(ctx, conversationID, seq, revokedContent)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis,
		GetMsgBySeqKey(conversationID, seq),
		GetMsgLatestKey(conversationID),
	)
	return nil
}

func (m *cachedMsgModel) DeleteBySeq(ctx context.Context, conversationID string, seqs []int64) error {
	err := m.MsgModel.DeleteBySeq(ctx, conversationID, seqs)
	if err != nil {
		return err
	}
	delKeys := m.msgSeqCacheKeys(conversationID, seqs)
	delKeys = append(delKeys,
		GetMsgLatestKey(conversationID),
		GetMsgMaxSeqKey(conversationID),
		GetMsgMinSeqKey(conversationID),
	)
	sredis.CacheDelDouble(ctx, m.redis, delKeys...)
	return nil
}

func (m *cachedMsgModel) FindBySeq(ctx context.Context, conversationID string, seq int64) (*model.Message, error) {
	if m.redis == nil {
		return m.MsgModel.FindBySeq(ctx, conversationID, seq)
	}

	var msg model.Message
	key := GetMsgBySeqKey(conversationID, seq)
	found, err := sredis.CacheGet(ctx, m.redis, key, &msg)
	if err != nil {
		return nil, err
	}
	if found {
		if msg.ConversationID == "" {
			return nil, ErrMsgNotFound
		}
		return &msg, nil
	}

	sfKey := sfKeyPrefixMsgSeq + conversationID + ":" + convert.ToString(seq)
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerMsg model.Message
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerMsg)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerMsg.ConversationID == "" {
				return nil, ErrMsgNotFound
			}
			return &innerMsg, nil
		}

		dbMsg, err2 := m.MsgModel.FindBySeq(ctx, conversationID, seq)
		if err2 != nil {
			if errors.Is(err2, ErrMsgNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(msgNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbMsg.SendTime.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbMsg, version, m.jitterTTL(msgDefaultExpireSeconds))
		return dbMsg, nil
	})
	if err != nil {
		if errors.Is(err, ErrMsgNotFound) {
			return nil, ErrMsgNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrMsgNotFound
	}
	return v.(*model.Message), nil
}

func (m *cachedMsgModel) FindLatestMsg(ctx context.Context, conversationID string) (*model.Message, error) {
	if m.redis == nil {
		return m.MsgModel.FindLatestMsg(ctx, conversationID)
	}

	var msg model.Message
	key := GetMsgLatestKey(conversationID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &msg)
	if err != nil {
		return nil, err
	}
	if found {
		if msg.ConversationID == "" {
			return nil, ErrMsgNotFound
		}
		return &msg, nil
	}

	sfKey := sfKeyPrefixMsgLatest + conversationID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerMsg model.Message
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerMsg)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerMsg.ConversationID == "" {
				return nil, ErrMsgNotFound
			}
			return &innerMsg, nil
		}

		dbMsg, err2 := m.MsgModel.FindLatestMsg(ctx, conversationID)
		if err2 != nil {
			if errors.Is(err2, ErrMsgNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(msgNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbMsg.SendTime.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbMsg, version, m.jitterTTL(msgDefaultExpireSeconds))
		return dbMsg, nil
	})
	if err != nil {
		if errors.Is(err, ErrMsgNotFound) {
			return nil, ErrMsgNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrMsgNotFound
	}
	return v.(*model.Message), nil
}

func (m *cachedMsgModel) GetMaxSeq(ctx context.Context, conversationID string) (int64, error) {
	if m.redis == nil {
		return m.MsgModel.GetMaxSeq(ctx, conversationID)
	}

	key := GetMsgMaxSeqKey(conversationID)
	data, found, err := sredis.CacheGetString(ctx, m.redis, key)
	if err != nil {
		return 0, err
	}
	if found {
		if data == "" {
			return 0, nil
		}
		val, errConv := convert.ToInt64E(data)
		if errConv == nil {
			return val, nil
		}
		_ = sredis.CacheDel(ctx, m.redis, key)
	}

	sfKey := sfKeyPrefixMsgMax + conversationID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		data2, found2, err2 := sredis.CacheGetString(ctx, m.redis, key)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if data2 == "" {
				return int64(0), nil
			}
			val, errConv := convert.ToInt64E(data2)
			if errConv == nil {
				return val, nil
			}
			_ = sredis.CacheDel(ctx, m.redis, key)
		}

		maxSeq, err2 := m.MsgModel.GetMaxSeq(ctx, conversationID)
		if err2 != nil {
			return nil, err2
		}
		_, _ = sredis.CacheSetCASString(ctx, m.redis, key, convert.ToString(maxSeq), timex.UnixMilli(), m.jitterTTL(msgDefaultExpireSeconds))
		return maxSeq, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}

func (m *cachedMsgModel) GetMinSeq(ctx context.Context, conversationID string) (int64, error) {
	if m.redis == nil {
		return m.MsgModel.GetMinSeq(ctx, conversationID)
	}

	key := GetMsgMinSeqKey(conversationID)
	data, found, err := sredis.CacheGetString(ctx, m.redis, key)
	if err != nil {
		return 0, err
	}
	if found {
		if data == "" {
			return 0, nil
		}
		val, errConv := convert.ToInt64E(data)
		if errConv == nil {
			return val, nil
		}
		_ = sredis.CacheDel(ctx, m.redis, key)
	}

	sfKey := sfKeyPrefixMsgMin + conversationID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		data2, found2, err2 := sredis.CacheGetString(ctx, m.redis, key)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if data2 == "" {
				return int64(0), nil
			}
			val, errConv := convert.ToInt64E(data2)
			if errConv == nil {
				return val, nil
			}
			_ = sredis.CacheDel(ctx, m.redis, key)
		}

		minSeq, err2 := m.MsgModel.GetMinSeq(ctx, conversationID)
		if err2 != nil {
			return nil, err2
		}
		_, _ = sredis.CacheSetCASString(ctx, m.redis, key, convert.ToString(minSeq), timex.UnixMilli(), m.jitterTTL(msgDefaultExpireSeconds))
		return minSeq, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}

func (m *cachedMsgModel) FindByConversationID(ctx context.Context, conversationID string, seqStart int64, seqEnd int64, limit int) ([]*model.Message, error) {
	return m.MsgModel.FindByConversationID(ctx, conversationID, seqStart, seqEnd, limit)
}

func (m *cachedMsgModel) ToSDKMsg(msg *model.Message) *sdkws.MsgData {
	return m.MsgModel.ToSDKMsg(msg)
}
