package token

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenInvalid  = errors.New("token invalid")
	ErrUserExists    = errors.New("user already logged in")
)

type TokenInfo struct {
	Token      string            `json:"token"`
	ExpireAt   int64             `json:"expireAt"`
	UserID     string            `json:"userId"`
	Roles      string            `json:"roles"`
	UUID       string            `json:"uuid"`
	PlatformID int32             `json:"platformId"`
	DeviceID   string            `json:"deviceId"`
	Extra      map[string]string `json:"extra,omitempty"`
}

func (t *TokenInfo) IsExpired() bool {
	return t.ExpireAt > 0 && t.ExpireAt < time.Now().Unix()
}

type TokenStore interface {
	StoreToken(ctx context.Context, info *TokenInfo) error
	GetToken(ctx context.Context, uuid string) (*TokenInfo, error)
	DeleteToken(ctx context.Context, uuid string) error
	DeleteTokens(ctx context.Context, uuids []string) error
	DeleteUserTokens(ctx context.Context, userID string, platformID ...int32) error
	CheckTokenExists(ctx context.Context, userID string, platformID int32) (bool, error)
	GetUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error)
	GetUserTokensByPlatform(ctx context.Context, userID string, platformID int32) ([]*TokenInfo, error)
	Close() error
}
