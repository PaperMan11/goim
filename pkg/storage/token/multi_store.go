package token

import (
	"context"
	"sync"
	"sync/atomic"

	goredis "github.com/redis/go-redis/v9"
)

type MultiStore struct {
	localStore    *LocalStore
	redisStore    *RedisStore
	cacheEnabled  bool
	pubsub        *goredis.PubSub
	pubsubRunning atomic.Bool
	pubsubStopCtx context.Context
	pubsubCancel  context.CancelFunc
	pubsubMutex   sync.Mutex
	redisClient   goredis.UniversalClient
}

var _ TokenStore = (*MultiStore)(nil)

func NewMultiStore(redisClient goredis.UniversalClient, cacheEnabled bool) *MultiStore {
	ms := &MultiStore{
		localStore:   NewLocalStore(),
		redisStore:   NewRedisStore(redisClient),
		cacheEnabled: cacheEnabled,
		redisClient:  redisClient,
	}

	if cacheEnabled && redisClient != nil {
		ms.startPubSub()
	}

	return ms
}

func (ms *MultiStore) startPubSub() {
	if ms.pubsubRunning.CompareAndSwap(false, true) {
		ms.pubsubStopCtx, ms.pubsubCancel = context.WithCancel(context.Background())
		ms.pubsub = ms.redisClient.Subscribe(ms.pubsubStopCtx, TokenDeleteChannel)
		go ms.pubsubLoop()
	}
}

func (ms *MultiStore) pubsubLoop() {
	ch := ms.pubsub.Channel()
	for {
		select {
		case <-ms.pubsubStopCtx.Done():
			_ = ms.pubsub.Close()
			return
		case msg, ok := <-ch:
			if !ok || msg == nil {
				return
			}
			if msg.Payload != "" && ms.cacheEnabled {
				_ = ms.localStore.DeleteToken(context.Background(), msg.Payload)
			}
		}
	}
}

func (ms *MultiStore) IsPubSubRunning() bool {
	return ms.pubsubRunning.Load()
}

func (ms *MultiStore) StopPubSub() {
	ms.pubsubMutex.Lock()
	defer ms.pubsubMutex.Unlock()

	if ms.pubsubRunning.CompareAndSwap(true, false) {
		if ms.pubsubCancel != nil {
			ms.pubsubCancel()
		}
	}
}

func (ms *MultiStore) Close() error {
	ms.StopPubSub()
	return ms.localStore.Close()
}

func (ms *MultiStore) StoreToken(ctx context.Context, info *TokenInfo) error {
	err := ms.redisStore.StoreToken(ctx, info)
	if err != nil {
		return err
	}

	if ms.cacheEnabled {
		_ = ms.localStore.StoreToken(ctx, info)
	}

	return nil
}

func (ms *MultiStore) GetToken(ctx context.Context, uuid string) (*TokenInfo, error) {
	if ms.cacheEnabled {
		info, err := ms.localStore.GetToken(ctx, uuid)
		if err == nil {
			return info, nil
		}
	}

	info, err := ms.redisStore.GetToken(ctx, uuid)
	if err != nil {
		return nil, err
	}

	if ms.cacheEnabled && info != nil {
		_ = ms.localStore.StoreToken(ctx, info)
	}

	return info, nil
}

func (ms *MultiStore) DeleteToken(ctx context.Context, uuid string) error {
	err := ms.redisStore.DeleteToken(ctx, uuid)
	if err != nil {
		return err
	}

	if ms.cacheEnabled {
		_ = ms.localStore.DeleteToken(ctx, uuid)
	}

	return nil
}

func (ms *MultiStore) DeleteTokens(ctx context.Context, uuids []string) error {
	err := ms.redisStore.DeleteTokens(ctx, uuids)
	if err != nil {
		return err
	}

	if ms.cacheEnabled {
		_ = ms.localStore.DeleteTokens(ctx, uuids)
	}

	return nil
}

func (ms *MultiStore) DeleteUserTokens(ctx context.Context, userID string, platformID ...int32) error {
	err := ms.redisStore.DeleteUserTokens(ctx, userID, platformID...)
	if err != nil {
		return err
	}

	if ms.cacheEnabled {
		_ = ms.localStore.DeleteUserTokens(ctx, userID, platformID...)
	}

	return nil
}

func (ms *MultiStore) CheckTokenExists(ctx context.Context, userID string, platformID int32) (bool, error) {
	if ms.cacheEnabled {
		exists, err := ms.localStore.CheckTokenExists(ctx, userID, platformID)
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
	}

	return ms.redisStore.CheckTokenExists(ctx, userID, platformID)
}

func (ms *MultiStore) GetUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {
	if ms.cacheEnabled {
		tokens, err := ms.localStore.GetUserTokens(ctx, userID)
		if err == nil && len(tokens) > 0 {
			return tokens, nil
		}
	}

	tokens, err := ms.redisStore.GetUserTokens(ctx, userID)
	if err != nil {
		return nil, err
	}

	if ms.cacheEnabled && len(tokens) > 0 {
		for _, info := range tokens {
			_ = ms.localStore.StoreToken(ctx, info)
		}
	}

	return tokens, nil
}

func (ms *MultiStore) GetUserTokensByPlatform(ctx context.Context, userID string, platformID int32) ([]*TokenInfo, error) {
	if ms.cacheEnabled {
		tokens, err := ms.localStore.GetUserTokensByPlatform(ctx, userID, platformID)
		if err == nil && len(tokens) > 0 {
			return tokens, nil
		}
	}

	tokens, err := ms.redisStore.GetUserTokensByPlatform(ctx, userID, platformID)
	if err != nil {
		return nil, err
	}

	if ms.cacheEnabled && len(tokens) > 0 {
		for _, info := range tokens {
			_ = ms.localStore.StoreToken(ctx, info)
		}
	}

	return tokens, nil
}

func (ms *MultiStore) SetCacheEnabled(enabled bool) {
	ms.cacheEnabled = enabled
}

func (ms *MultiStore) GetLocalStore() *LocalStore {
	return ms.localStore
}

func (ms *MultiStore) GetRedisStore() *RedisStore {
	return ms.redisStore
}
