package conversation

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

type cachedConversationModel struct {
	ConversationModel
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

func NewCachedConversationModel(inner ConversationModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) ConversationModel {
	return &cachedConversationModel{
		ConversationModel: inner,
		redis:             rdb,
		barrier:           barrier,
	}
}

func (m *cachedConversationModel) convCacheKeys(convID string) []string {
	return []string{
		GetConversationInfoKey(convID),
		GetConversationLatestKey(convID),
	}
}

func (m *cachedConversationModel) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (m *cachedConversationModel) InsertConversation(ctx context.Context, convs []*model.Conversation) error {
	err := m.ConversationModel.InsertConversation(ctx, convs)
	if err != nil {
		return err
	}
	for _, conv := range convs {
		sredis.CacheDelDouble(ctx, m.redis, m.convCacheKeys(conv.ConversationID)...)
	}
	return nil
}

func (m *cachedConversationModel) UpsertConversation(ctx context.Context, conv *model.Conversation) error {
	err := m.ConversationModel.UpsertConversation(ctx, conv)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.convCacheKeys(conv.ConversationID)...)
	return nil
}

func (m *cachedConversationModel) FindConversation(ctx context.Context, ownerUserID, conversationID string) (*model.Conversation, error) {
	if m.redis == nil {
		return m.ConversationModel.FindConversation(ctx, ownerUserID, conversationID)
	}

	var conv model.Conversation
	key := GetConversationInfoKey(conversationID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &conv)
	if err != nil {
		return nil, err
	}
	if found {
		if conv.ConversationID == "" || conv.OwnerUserID != ownerUserID {
			return nil, ErrConversationNotFound
		}
		return &conv, nil
	}

	sfKey := sfKeyPrefixConvInfo + conversationID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerConv model.Conversation
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerConv)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerConv.ConversationID == "" {
				return nil, ErrConversationNotFound
			}
			return &innerConv, nil
		}

		dbConv, err2 := m.ConversationModel.FindConversation(ctx, ownerUserID, conversationID)
		if err2 != nil {
			if errors.Is(err2, ErrConversationNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(conversationNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbConv.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbConv, version, m.jitterTTL(conversationDefaultExpireSeconds))
		return dbConv, nil
	})
	if err != nil {
		if errors.Is(err, ErrConversationNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	if v == nil || v.(*model.Conversation).OwnerUserID != ownerUserID {
		return nil, ErrConversationNotFound
	}
	return v.(*model.Conversation), nil
}

func (m *cachedConversationModel) FindConversationsByIDs(ctx context.Context, ownerUserID string, convIDs []string) ([]*model.Conversation, error) {
	if m.redis == nil {
		return m.ConversationModel.FindConversationsByIDs(ctx, ownerUserID, convIDs)
	}

	result := make([]*model.Conversation, 0, len(convIDs))
	missIDs := make([]string, 0, len(convIDs))

	for _, convID := range convIDs {
		var conv model.Conversation
		found, err := sredis.CacheGet(ctx, m.redis, GetConversationInfoKey(convID), &conv)
		if err != nil {
			return nil, err
		}
		if found && conv.ConversationID != "" && conv.OwnerUserID == ownerUserID {
			result = append(result, &conv)
			continue
		}
		missIDs = append(missIDs, convID)
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	sort.Strings(missIDs)
	sum := sha1.Sum([]byte(strings.Join(missIDs, ",")))
	sfKey := sfKeyPrefixBatchConv + hex.EncodeToString(sum[:])

	v, err := m.barrier.Do(sfKey, func() (any, error) {
		for i, cid := range missIDs {
			c, errLoad := m.FindConversation(ctx, ownerUserID, cid)
			if errLoad != nil && !errors.Is(errLoad, ErrConversationNotFound) {
				return nil, errLoad
			}
			if c != nil {
				missIDs[i] = ""
				result = append(result, c)
			}
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}
	_ = v

	for _, cid := range missIDs {
		if cid == "" {
			continue
		}
		var conv model.Conversation
		found, err := sredis.CacheGet(ctx, m.redis, GetConversationInfoKey(cid), &conv)
		if err != nil {
			return nil, err
		}
		if found && conv.ConversationID != "" {
			result = append(result, &conv)
		}
	}

	return result, nil
}

func (m *cachedConversationModel) FindConversationsByOwner(ctx context.Context, ownerUserID string) ([]*model.Conversation, error) {
	return m.ConversationModel.FindConversationsByOwner(ctx, ownerUserID)
}

func (m *cachedConversationModel) UpdateConversation(ctx context.Context, owner, convID string, updates map[string]any) error {
	err := m.ConversationModel.UpdateConversation(ctx, owner, convID, updates)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.convCacheKeys(convID)...)
	return nil
}

func (m *cachedConversationModel) UpdateConversations(ctx context.Context, ownerUserID string, convIDs []string, updates map[string]any) error {
	err := m.ConversationModel.UpdateConversations(ctx, ownerUserID, convIDs, updates)
	if err != nil {
		return err
	}
	for _, convID := range convIDs {
		sredis.CacheDelDouble(ctx, m.redis, m.convCacheKeys(convID)...)
	}
	return nil
}

func (m *cachedConversationModel) DeleteConversation(ctx context.Context, owner, convID string) error {
	err := m.ConversationModel.DeleteConversation(ctx, owner, convID)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.convCacheKeys(convID)...)
	return nil
}

func (m *cachedConversationModel) DeleteConversationsByOwner(ctx context.Context, ownerUserID string) error {
	convs, err := m.ConversationModel.FindConversationsByOwner(ctx, ownerUserID)
	if err != nil {
		return err
	}
	err = m.ConversationModel.DeleteConversationsByOwner(ctx, ownerUserID)
	if err != nil {
		return err
	}
	for _, conv := range convs {
		sredis.CacheDelDouble(ctx, m.redis, m.convCacheKeys(conv.ConversationID)...)
	}
	return nil
}

func (m *cachedConversationModel) UpsertConversationLatestMsg(ctx context.Context, latest *model.ConversationLatestMsg) error {
	err := m.ConversationModel.UpsertConversationLatestMsg(ctx, latest)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.convCacheKeys(latest.ConversationID)...)
	return nil
}

func (m *cachedConversationModel) FindLatestMsg(ctx context.Context, owner, convID string) (*model.ConversationLatestMsg, error) {
	if m.redis == nil {
		return m.ConversationModel.FindLatestMsg(ctx, owner, convID)
	}

	var latest model.ConversationLatestMsg
	key := GetConversationLatestKey(convID)
	found, err := sredis.CacheGet(ctx, m.redis, key, &latest)
	if err != nil {
		return nil, err
	}
	if found {
		if latest.ConversationID == "" {
			return nil, ErrLatestMsgNotFound
		}
		return &latest, nil
	}

	sfKey := sfKeyPrefixConvLatest + convID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerLatest model.ConversationLatestMsg
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerLatest)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerLatest.ConversationID == "" {
				return nil, ErrLatestMsgNotFound
			}
			return &innerLatest, nil
		}

		dbLatest, err2 := m.ConversationModel.FindLatestMsg(ctx, owner, convID)
		if err2 != nil {
			if errors.Is(err2, ErrLatestMsgNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(conversationNilExpireSeconds))
			}
			return nil, err2
		}
		version := dbLatest.UpdatedAt.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbLatest, version, m.jitterTTL(conversationDefaultExpireSeconds))
		return dbLatest, nil
	})
	if err != nil {
		if errors.Is(err, ErrLatestMsgNotFound) {
			return nil, ErrLatestMsgNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrLatestMsgNotFound
	}
	return v.(*model.ConversationLatestMsg), nil
}

func (m *cachedConversationModel) FindLatestMsgsByOwner(ctx context.Context, owner string, limit int) ([]*model.ConversationLatestMsg, error) {
	return m.ConversationModel.FindLatestMsgsByOwner(ctx, owner, limit)
}

func (m *cachedConversationModel) DeleteLatestMsg(ctx context.Context, owner, convID string) error {
	err := m.ConversationModel.DeleteLatestMsg(ctx, owner, convID)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, m.convCacheKeys(convID)...)
	return nil
}
