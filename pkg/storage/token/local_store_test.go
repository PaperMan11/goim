package token

import (
	"context"
	"testing"
	"time"
)

func TestLocalStore_StoreAndGetToken(t *testing.T) {
	ls := NewLocalStore()
	defer ls.Close()

	ctx := context.Background()
	info := &TokenInfo{
		UUID:       "test-uuid-1",
		UserID:     "user-1",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	err := ls.StoreToken(ctx, info)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	retrieved, err := ls.GetToken(ctx, info.UUID)
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

func TestLocalStore_DeleteToken(t *testing.T) {
	ls := NewLocalStore()
	defer ls.Close()

	ctx := context.Background()
	info := &TokenInfo{
		UUID:       "test-uuid-2",
		UserID:     "user-2",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	ls.StoreToken(ctx, info)

	err := ls.DeleteToken(ctx, info.UUID)
	if err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}

	_, err = ls.GetToken(ctx, info.UUID)
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got %v", err)
	}
}

func TestLocalStore_GetUserTokens(t *testing.T) {
	ls := NewLocalStore()
	defer ls.Close()

	ctx := context.Background()
	userID := "user-3"

	tokens := []*TokenInfo{
		{
			UUID:       "test-uuid-3-1",
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
		{
			UUID:       "test-uuid-3-2",
			UserID:     userID,
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		ls.StoreToken(ctx, token)
	}

	retrieved, err := ls.GetUserTokens(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserTokens failed: %v", err)
	}

	if len(retrieved) != len(tokens) {
		t.Errorf("Expected %d tokens, got %d", len(tokens), len(retrieved))
	}
}

func TestLocalStore_CheckTokenExists(t *testing.T) {
	ls := NewLocalStore()
	defer ls.Close()

	ctx := context.Background()
	info := &TokenInfo{
		UUID:       "test-uuid-4",
		UserID:     "user-4",
		PlatformID: 1,
		ExpireAt:   time.Now().Add(time.Hour).Unix(),
	}

	ls.StoreToken(ctx, info)

	exists, err := ls.CheckTokenExists(ctx, info.UserID, info.PlatformID)
	if err != nil {
		t.Fatalf("CheckTokenExists failed: %v", err)
	}

	if !exists {
		t.Errorf("Expected token to exist")
	}

	exists, err = ls.CheckTokenExists(ctx, info.UserID, 2)
	if err != nil {
		t.Fatalf("CheckTokenExists failed: %v", err)
	}

	if exists {
		t.Errorf("Expected token not to exist for platform 2")
	}
}

func TestLocalStore_DeleteUserTokens(t *testing.T) {
	ls := NewLocalStore()
	defer ls.Close()

	ctx := context.Background()
	userID := "user-5"

	tokens := []*TokenInfo{
		{
			UUID:       "test-uuid-5-1",
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
		{
			UUID:       "test-uuid-5-2",
			UserID:     userID,
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		ls.StoreToken(ctx, token)
	}

	err := ls.DeleteUserTokens(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteUserTokens failed: %v", err)
	}

	retrieved, err := ls.GetUserTokens(ctx, userID)
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound after deletion, got %v", err)
	}

	if retrieved != nil {
		t.Errorf("Expected no tokens after deletion, got %d", len(retrieved))
	}
}

func TestLocalStore_DeleteUserTokensByPlatform(t *testing.T) {
	ls := NewLocalStore()
	defer ls.Close()

	ctx := context.Background()
	userID := "user-6"

	tokens := []*TokenInfo{
		{
			UUID:       "test-uuid-6-1",
			UserID:     userID,
			PlatformID: 1,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
		{
			UUID:       "test-uuid-6-2",
			UserID:     userID,
			PlatformID: 2,
			ExpireAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, token := range tokens {
		ls.StoreToken(ctx, token)
	}

	err := ls.DeleteUserTokens(ctx, userID, 1)
	if err != nil {
		t.Fatalf("DeleteUserTokens failed: %v", err)
	}

	retrieved, err := ls.GetUserTokens(ctx, userID)
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

func TestLocalStore_TokenExpiration(t *testing.T) {
	ls := NewLocalStore()
	defer ls.Close()

	ctx := context.Background()
	expireAt := time.Now().Add(time.Second).Unix()
	info := &TokenInfo{
		UUID:       "test-uuid-7",
		UserID:     "user-7",
		PlatformID: 1,
		ExpireAt:   expireAt,
	}

	err := ls.StoreToken(ctx, info)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	retrieved, err := ls.GetToken(ctx, info.UUID)
	if err != nil {
		t.Fatalf("GetToken immediately failed: %v", err)
	}

	if retrieved.ExpireAt != expireAt {
		t.Errorf("Expected ExpireAt %d, got %d", expireAt, retrieved.ExpireAt)
	}

	t.Logf("Stored token with ExpireAt: %d, now: %d", expireAt, time.Now().Unix())

	time.Sleep(2000 * time.Millisecond)

	t.Logf("After sleep - ExpireAt: %d, now: %d", expireAt, time.Now().Unix())

	_, err = ls.GetToken(ctx, info.UUID)
	if err != ErrTokenNotFound && err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenNotFound or ErrTokenExpired, got %v", err)
	}
}
