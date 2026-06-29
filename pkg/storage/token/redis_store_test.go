package token

import (
	"context"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

func setupTestRedis(t *testing.T) *redis.Redis {
	redisHost := "192.168.241.128:6379"
	redisPass := "123456"
	r, err := redis.NewRedis(redis.RedisConf{
		Host: redisHost,
		Type: "node",
		Pass: redisPass,
	})
	if err != nil {
		t.Skipf("Redis not available at %s: %v", redisHost, err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		r.DelCtx(ctx, "token:*")
		r.DelCtx(ctx, "user:*")
	})

	return r
}

func TestRedisStore_StoreAndGetToken(t *testing.T) {
	r := setupTestRedis(t)

	rs := NewRedisStore(r)
	ctx := context.Background()

	info := &TokenInfo{
		UUID:       "redis-uuid-1",
		UserID:     "redis-user-1",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	err := rs.StoreToken(ctx, info)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	retrieved, err := rs.GetToken(ctx, info.UUID)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	if retrieved.UUID != info.UUID {
		t.Errorf("Expected UUID %s, got %s", info.UUID, retrieved.UUID)
	}

	if retrieved.UserID != info.UserID {
		t.Errorf("Expected UserID %s, got %s", info.UserID, retrieved.UserID)
	}
}

func TestRedisStore_DeleteToken(t *testing.T) {
	r := setupTestRedis(t)

	rs := NewRedisStore(r)
	ctx := context.Background()

	info := &TokenInfo{
		UUID:       "redis-uuid-2",
		UserID:     "redis-user-2",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	rs.StoreToken(ctx, info)

	err := rs.DeleteToken(ctx, info.UUID)
	if err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}

	_, err = rs.GetToken(ctx, info.UUID)
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got %v", err)
	}
}

func TestRedisStore_GetUserTokens(t *testing.T) {
	r := setupTestRedis(t)

	rs := NewRedisStore(r)
	ctx := context.Background()
	userID := "redis-user-3"

	tokens := []*TokenInfo{
		{
			UUID:       "redis-uuid-3-1",
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
		{
			UUID:       "redis-uuid-3-2",
			UserID:     userID,
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		rs.StoreToken(ctx, token)
	}

	retrieved, err := rs.GetUserTokens(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserTokens failed: %v", err)
	}

	if len(retrieved) != len(tokens) {
		t.Errorf("Expected %d tokens, got %d", len(tokens), len(retrieved))
	}
}

func TestRedisStore_CheckTokenExists(t *testing.T) {
	r := setupTestRedis(t)

	rs := NewRedisStore(r)
	ctx := context.Background()

	info := &TokenInfo{
		UUID:       "redis-uuid-4",
		UserID:     "redis-user-4",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	rs.StoreToken(ctx, info)

	exists, err := rs.CheckTokenExists(ctx, info.UserID, info.PlatformID)
	if err != nil {
		t.Fatalf("CheckTokenExists failed: %v", err)
	}

	if !exists {
		t.Errorf("Expected token to exist")
	}

	exists, err = rs.CheckTokenExists(ctx, info.UserID, 2)
	if err != nil {
		t.Fatalf("CheckTokenExists failed: %v", err)
	}

	if exists {
		t.Errorf("Expected token not to exist for platform 2")
	}
}

func TestRedisStore_DeleteUserTokens(t *testing.T) {
	r := setupTestRedis(t)

	rs := NewRedisStore(r)
	ctx := context.Background()
	userID := "redis-user-5"

	tokens := []*TokenInfo{
		{
			UUID:       "redis-uuid-5-1",
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
		{
			UUID:       "redis-uuid-5-2",
			UserID:     userID,
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		rs.StoreToken(ctx, token)
	}

	err := rs.DeleteUserTokens(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteUserTokens failed: %v", err)
	}

	retrieved, err := rs.GetUserTokens(ctx, userID)
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound after deletion, got %v", err)
	}

	if retrieved != nil {
		t.Errorf("Expected no tokens after deletion, got %d", len(retrieved))
	}
}

func TestRedisStore_DeleteUserTokensByPlatform(t *testing.T) {
	r := setupTestRedis(t)

	rs := NewRedisStore(r)
	ctx := context.Background()
	userID := "redis-user-6"

	tokens := []*TokenInfo{
		{
			UUID:       "redis-uuid-6-1",
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
		{
			UUID:       "redis-uuid-6-2",
			UserID:     userID,
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		rs.StoreToken(ctx, token)
	}

	err := rs.DeleteUserTokens(ctx, userID, 1)
	if err != nil {
		t.Fatalf("DeleteUserTokens failed: %v", err)
	}

	retrieved, err := rs.GetUserTokens(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserTokens failed: %v", err)
	}

	if len(retrieved) != 1 {
		t.Errorf("Expected 1 token after platform deletion, got %d", len(retrieved))
	}

	if retrieved[0].PlatformID != 2 {
		t.Errorf("Expected platform 2 token, got platform %d", retrieved[0].PlatformID)
	}
}

func TestRedisStore_TokenExpiration(t *testing.T) {
	r := setupTestRedis(t)

	rs := NewRedisStore(r)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info := &TokenInfo{
		UUID:       "redis-uuid-7",
		UserID:     "redis-user-7",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Second).Unix(),
	}

	err := rs.StoreToken(ctx, info)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	_, err = rs.GetToken(ctx, info.UUID)
	if err != ErrTokenNotFound && err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenNotFound or ErrTokenExpired, got %v", err)
	}
}

func TestRedisStore_CleanExpiredTokens(t *testing.T) {
	r := setupTestRedis(t)

	rs := NewRedisStore(r)
	ctx := context.Background()
	userID := "redis-user-8"

	tokens := []*TokenInfo{
		{
			UUID:       "redis-uuid-8-1",
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(-time.Hour).Unix(),
		},
		{
			UUID:       "redis-uuid-8-2",
			UserID:     userID,
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		rs.StoreToken(ctx, token)
	}

	err := rs.CleanExpiredTokens(ctx, userID)
	if err != nil {
		t.Fatalf("CleanExpiredTokens failed: %v", err)
	}

	retrieved, err := rs.GetUserTokens(ctx, userID)
	if err != nil && err != ErrTokenNotFound {
		t.Fatalf("GetUserTokens failed: %v", err)
	}

	if retrieved != nil && len(retrieved) != 1 {
		t.Errorf("Expected 1 valid token after cleanup, got %d", len(retrieved))
	}

	if retrieved != nil && retrieved[0].PlatformID != 2 {
		t.Errorf("Expected platform 2 token, got platform %d", retrieved[0].PlatformID)
	}
}

func TestRedisStore_ClearAll(t *testing.T) {
	r := setupTestRedis(t)

	rs := NewRedisStore(r)
	ctx := context.Background()

	tokens := []*TokenInfo{
		{
			UUID:       "redis-uuid-9-1",
			UserID:     "redis-user-9-1",
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
		{
			UUID:       "redis-uuid-9-2",
			UserID:     "redis-user-9-2",
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		rs.StoreToken(ctx, token)
	}

	err := rs.ClearAll(ctx)
	if err != nil {
		t.Fatalf("ClearAll failed: %v", err)
	}

	for _, token := range tokens {
		_, err := rs.GetToken(ctx, token.UUID)
		if err != ErrTokenNotFound {
			t.Errorf("Expected ErrTokenNotFound for %s, got %v", token.UUID, err)
		}
	}
}

func TestRedisStore_BatchDeletePerformance(t *testing.T) {
	r := setupTestRedis(t)

	rs := NewRedisStore(r)
	ctx := context.Background()
	userID := "redis-user-batch"

	const batchSize = 10
	tokens := make([]*TokenInfo, batchSize)
	for i := 0; i < batchSize; i++ {
		tokens[i] = &TokenInfo{
			UUID:       "redis-uuid-batch-" + string(rune('a'+i)),
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		}
		rs.StoreToken(ctx, tokens[i])
	}

	start := time.Now()
	err := rs.DeleteUserTokens(ctx, userID)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("DeleteUserTokens failed: %v", err)
	}

	t.Logf("Batch delete %d tokens took %v", batchSize, duration)

	retrieved, err := rs.GetUserTokens(ctx, userID)
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound after batch deletion, got %v", err)
	}

	if retrieved != nil {
		t.Errorf("Expected no tokens after batch deletion, got %d", len(retrieved))
	}
}
