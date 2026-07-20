package jwtx

import (
	"errors"
	"strings"
	"time"

	"github.com/PaperMan11/goim/pkg/utils/convert"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("token invalid")
	ErrTokenType    = errors.New("token type mismatch")
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
	TokenTypeAdmin   TokenType = "admin"
)

type TokenClaims struct {
	Type       TokenType         `json:"type"`
	UserID     string            `json:"userId"`
	Roles      string            `json:"roles"`
	UUID       string            `json:"uuid"`
	PlatformID int32             `json:"platformId"`
	DeviceID   string            `json:"deviceId"`
	Extra      map[string]string `json:"extra,omitempty"`
	jwt.RegisteredClaims
}

func NewTokenClaims(tokenType TokenType) *TokenClaims {
	return &TokenClaims{
		Type:  tokenType,
		Extra: make(map[string]string),
	}
}

func (c *TokenClaims) SetUserID(userID int64) *TokenClaims {
	c.UserID = convert.ToString(userID)
	return c
}

func (c *TokenClaims) SetPlatformID(platformID int32) *TokenClaims {
	c.PlatformID = platformID
	return c
}

func (c *TokenClaims) SetDeviceID(deviceID string) *TokenClaims {
	c.DeviceID = deviceID
	return c
}

func (c *TokenClaims) SetUserIDStr(userID string) *TokenClaims {
	c.UserID = userID
	return c
}

func (c *TokenClaims) SetRoles(roles []string) *TokenClaims {
	c.Roles = strings.Join(roles, ",")
	return c
}

func (c *TokenClaims) SetUUID(uuid string) *TokenClaims {
	c.UUID = uuid
	return c
}

func (c *TokenClaims) SetExtra(key, value string) *TokenClaims {
	if c.Extra == nil {
		c.Extra = make(map[string]string)
	}
	c.Extra[key] = value
	return c
}

func (c *TokenClaims) SetIssuer(issuer string) *TokenClaims {
	c.RegisteredClaims.Issuer = issuer
	return c
}

func (c *TokenClaims) SetExpire(expireSeconds int64) *TokenClaims {
	now := timex.Now()
	c.RegisteredClaims.IssuedAt = jwt.NewNumericDate(now)
	c.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(expireSeconds) * time.Second))
	return c
}

func (c *TokenClaims) SetExpireAt(expireTime time.Time) *TokenClaims {
	c.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(expireTime)
	return c
}

func (c *TokenClaims) SetNotBefore(notBefore time.Time) *TokenClaims {
	c.RegisteredClaims.NotBefore = jwt.NewNumericDate(notBefore)
	return c
}

func (c *TokenClaims) SetAudience(audience []string) *TokenClaims {
	c.RegisteredClaims.Audience = audience
	return c
}

func (c *TokenClaims) SetSubject(subject string) *TokenClaims {
	c.RegisteredClaims.Subject = subject
	return c
}

func GenerateToken(claims *TokenClaims, secretKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

func ParseToken(tokenString, secretKey string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, ErrTokenExpired
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

func ParseTokenWithType(tokenString, secretKey string, expectedType TokenType) (*TokenClaims, error) {
	claims, err := ParseToken(tokenString, secretKey)
	if err != nil {
		return nil, err
	}

	if claims.Type != expectedType {
		return nil, ErrTokenType
	}

	return claims, nil
}

func GenerateAccessToken(uuid string, issuer string, userID string, platformID int32, deviceID string, roles []string, secretKey string, accessExpire int64) (string, error) {
	claims := NewTokenClaims(TokenTypeAccess).
		SetUUID(uuid).
		SetIssuer(issuer).
		SetUserIDStr(userID).
		SetPlatformID(platformID).
		SetDeviceID(deviceID).
		SetRoles(roles).
		SetExpire(accessExpire)

	return GenerateToken(claims, secretKey)
}

func GenerateRefreshToken(uuid string, issuer string, userID string, platformID int32, deviceID string, roles []string, refreshSecretKey string, refreshExpire int64) (string, error) {
	claims := NewTokenClaims(TokenTypeRefresh).
		SetUUID(uuid).
		SetIssuer(issuer).
		SetUserIDStr(userID).
		SetPlatformID(platformID).
		SetDeviceID(deviceID).
		SetRoles(roles).
		SetExpire(refreshExpire)

	return GenerateToken(claims, refreshSecretKey)
}

func GenerateAdminToken(uuid string, issuer string, userID string, platformID int32, deviceID string, roles []string, secretKey string, expire int64) (string, error) {
	claims := NewTokenClaims(TokenTypeAdmin).
		SetUUID(uuid).
		SetIssuer(issuer).
		SetUserIDStr(userID).
		SetPlatformID(platformID).
		SetDeviceID(deviceID).
		SetRoles(roles).
		SetExpire(expire)

	return GenerateToken(claims, secretKey)
}

func ParseAccessToken(tokenString, secretKey string) (*TokenClaims, error) {
	return ParseTokenWithType(tokenString, secretKey, TokenTypeAccess)
}

func ParseRefreshToken(tokenString, refreshSecretKey string) (*TokenClaims, error) {
	return ParseTokenWithType(tokenString, refreshSecretKey, TokenTypeRefresh)
}

func (c *TokenClaims) GetUserID() int64 {
	return convert.ToInt64(c.UserID)
}

func (c *TokenClaims) GetUserIDStr() string {
	return c.UserID
}

func (c *TokenClaims) GetRoles() []string {
	if c.Roles == "" {
		return nil
	}
	return strings.Split(c.Roles, ",")
}

func (c *TokenClaims) GetPlatformID() int32 {
	return convert.ToInt32(c.PlatformID)
}

func (c *TokenClaims) GetDeviceID() string {
	return c.DeviceID
}

func (c *TokenClaims) HasRole(role string) bool {
	roles := c.GetRoles()
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func (c *TokenClaims) IsAccess() bool {
	return c.Type == TokenTypeAccess
}

func (c *TokenClaims) IsRefresh() bool {
	return c.Type == TokenTypeRefresh
}

func (c *TokenClaims) IsAdmin() bool {
	return c.Type == TokenTypeAdmin
}

func (c *TokenClaims) GetExtra(key string) string {
	if c.Extra == nil {
		return ""
	}
	return c.Extra[key]
}

func (c *TokenClaims) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return c.ExpiresAt.Time.Before(timex.Now())
}

func (c *TokenClaims) GetRemainingTime() time.Duration {
	if c.ExpiresAt == nil {
		return 0
	}
	return time.Until(c.ExpiresAt.Time)
}
