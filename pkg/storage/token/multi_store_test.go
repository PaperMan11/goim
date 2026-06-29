package token

import (
	"context"
	"testing"
	"time"
)

func TestMultiStore_StoreAndGetToken(t *testing.T) {
	r := setupTestRedis(t)

	ms := NewMultiStore(r, true)
	defer ms.Close()

	ctx := context.Background()

	info := &TokenInfo{
		UUID:       "multi-uuid-1",
		UserID:     "multi-user-1",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	err := ms.StoreToken(ctx, info)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	retrieved, err := ms.GetToken(ctx, info.UUID)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	if retrieved.UUID != info.UUID {
		t.Errorf("Expected UUID %s, got %s", info.UUID, retrieved.UUID)
	}
}

func TestMultiStore_DeleteToken(t *testing.T) {
	r := setupTestRedis(t)

	ms := NewMultiStore(r, true)
	defer ms.Close()

	ctx := context.Background()

	info := &TokenInfo{
		UUID:       "multi-uuid-2",
		UserID:     "multi-user-2",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	ms.StoreToken(ctx, info)

	err := ms.DeleteToken(ctx, info.UUID)
	if err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}

	_, err = ms.GetToken(ctx, info.UUID)
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got %v", err)
	}
}

func TestMultiStore_GetUserTokens(t *testing.T) {
	r := setupTestRedis(t)

	ms := NewMultiStore(r, true)
	defer ms.Close()

	ctx := context.Background()
	userID := "multi-user-3"

	tokens := []*TokenInfo{
		{
			UUID:       "multi-uuid-3-1",
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
		{
			UUID:       "multi-uuid-3-2",
			UserID:     userID,
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		err := ms.StoreToken(ctx, token)
		if err != nil {
			t.Fatalf("StoreToken failed: %v", err)
		}
	}

	retrieved, err := ms.GetUserTokens(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserTokens failed: %v", err)
	}

	if len(retrieved) != len(tokens) {
		t.Errorf("Expected %d tokens, got %d", len(tokens), len(retrieved))
	}
}

func TestMultiStore_CheckTokenExists(t *testing.T) {
	r := setupTestRedis(t)

	ms := NewMultiStore(r, true)
	defer ms.Close()

	ctx := context.Background()

	info := &TokenInfo{
		UUID:       "multi-uuid-4",
		UserID:     "multi-user-4",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	ms.StoreToken(ctx, info)

	exists, err := ms.CheckTokenExists(ctx, info.UserID, info.PlatformID)
	if err != nil {
		t.Fatalf("CheckTokenExists failed: %v", err)
	}

	if !exists {
		t.Errorf("Expected token to exist")
	}

	exists, err = ms.CheckTokenExists(ctx, info.UserID, 2)
	if err != nil {
		t.Fatalf("CheckTokenExists failed: %v", err)
	}

	if exists {
		t.Errorf("Expected token not to exist for platform 2")
	}
}

func TestMultiStore_DeleteUserTokens(t *testing.T) {
	r := setupTestRedis(t)

	ms := NewMultiStore(r, true)
	defer ms.Close()

	ctx := context.Background()
	userID := "multi-user-5"

	tokens := []*TokenInfo{
		{
			UUID:       "multi-uuid-5-1",
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
		{
			UUID:       "multi-uuid-5-2",
			UserID:     userID,
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		ms.StoreToken(ctx, token)
	}

	err := ms.DeleteUserTokens(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteUserTokens failed: %v", err)
	}

	retrieved, err := ms.GetUserTokens(ctx, userID)
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound after deletion, got %v", err)
	}

	if retrieved != nil {
		t.Errorf("Expected no tokens after deletion, got %d", len(retrieved))
	}
}

func TestMultiStore_DeleteUserTokensByPlatform(t *testing.T) {
	r := setupTestRedis(t)

	ms := NewMultiStore(r, true)
	defer ms.Close()

	ctx := context.Background()
	userID := "multi-user-6"

	tokens := []*TokenInfo{
		{
			UUID:       "multi-uuid-6-1",
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
		{
			UUID:       "multi-uuid-6-2",
			UserID:     userID,
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		ms.StoreToken(ctx, token)
	}

	err := ms.DeleteUserTokens(ctx, userID, 1)
	if err != nil {
		t.Fatalf("DeleteUserTokens failed: %v", err)
	}

	retrieved, err := ms.GetUserTokens(ctx, userID)
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

func TestMultiStore_CacheFallback(t *testing.T) {
	r := setupTestRedis(t)

	ms := NewMultiStore(r, true)
	defer ms.Close()

	ctx := context.Background()

	info := &TokenInfo{
		UUID:       "multi-uuid-fallback",
		UserID:     "multi-user-fallback",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	rs := NewRedisStore(r)
	rs.StoreToken(ctx, info)

	retrieved, err := ms.GetToken(ctx, info.UUID)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	if retrieved.UUID != info.UUID {
		t.Errorf("Expected UUID %s, got %s", info.UUID, retrieved.UUID)
	}
}

func TestMultiStore_CacheDisabled(t *testing.T) {
	r := setupTestRedis(t)

	ms := NewMultiStore(r, false)
	defer ms.Close()

	ctx := context.Background()

	info := &TokenInfo{
		UUID:       "multi-uuid-disabled",
		UserID:     "multi-user-disabled",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	ms.StoreToken(ctx, info)

	retrieved, err := ms.GetToken(ctx, info.UUID)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	if retrieved.UUID != info.UUID {
		t.Errorf("Expected UUID %s, got %s", info.UUID, retrieved.UUID)
	}
}

func TestMultiStore_TokenExpiration(t *testing.T) {
	r := setupTestRedis(t)

	ms := NewMultiStore(r, true)
	defer ms.Close()

	ctx := context.Background()

	expireAt := time.Now().Add(time.Second).Unix()
	info := &TokenInfo{
		UUID:       "multi-uuid-expire",
		UserID:     "multi-user-expire",
		PlatformID: 1,
		ExpireAt:   expireAt,
	}

	err := ms.StoreToken(ctx, info)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	time.Sleep(2000 * time.Millisecond)

	_, err = ms.GetToken(ctx, info.UUID)
	if err != ErrTokenNotFound && err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenNotFound or ErrTokenExpired, got %v", err)
	}
}

func TestMultiStore_PubSubDeleteSync(t *testing.T) {
	r := setupTestRedis(t)

	ms1 := NewMultiStore(r, true)
	defer ms1.Close()

	ms2 := NewMultiStore(r, true)
	defer ms2.Close()

	time.Sleep(100 * time.Millisecond)

	if !ms1.IsPubSubRunning() {
		t.Fatal("ms1 PubSub is not running")
	}
	if !ms2.IsPubSubRunning() {
		t.Fatal("ms2 PubSub is not running")
	}

	ctx := context.Background()

	info := &TokenInfo{
		UUID:       "multi-uuid-pubsub",
		UserID:     "multi-user-pubsub",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	err := ms1.StoreToken(ctx, info)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	err = ms2.DeleteToken(ctx, info.UUID)
	if err != nil {
		t.Fatalf("DeleteToken from ms2 failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	_, err = ms1.GetToken(ctx, info.UUID)
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound after pubsub delete, got %v", err)
	}
}
