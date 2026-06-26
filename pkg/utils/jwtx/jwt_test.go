package jwtx

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

const (
	testSecretKey        = "test-secret-key-12345"
	testRefreshSecretKey = "test-refresh-secret-key-67890"
	testIssuer           = "goim-test"
	testUserID           = "test-user-id-12345"
	testUUID             = "test-uuid-abc-123"
)

var testRoles = []string{"user", "admin"}

func TestGenerateAndParseToken(t *testing.T) {
	token, err := GenerateAccessToken(testUUID, testIssuer, testUserID, 1, "test-device-id", testRoles, testSecretKey, 3600)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ParseToken(token, testSecretKey)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeAccess, claims.Type)
	assert.Equal(t, testIssuer, claims.Issuer)
	assert.Equal(t, testUUID, claims.UUID)
	assert.Equal(t, testUserID, claims.GetUserID())
	assert.Equal(t, testRoles, claims.GetRoles())
	assert.True(t, claims.IsAccess())
	assert.False(t, claims.IsRefresh())
	assert.False(t, claims.IsAdmin())
	assert.False(t, claims.IsExpired())
}

func TestGenerateAndParseRefreshToken(t *testing.T) {
	token, err := GenerateRefreshToken(testUUID, testIssuer, testUserID, 1, "test-device-id", testRoles, testRefreshSecretKey, 86400)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ParseRefreshToken(token, testRefreshSecretKey)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeRefresh, claims.Type)
	assert.True(t, claims.IsRefresh())
}

func TestGenerateAndParseAdminToken(t *testing.T) {
	token, err := GenerateAdminToken(testUUID, testIssuer, testUserID, 1, "test-device-id", testRoles, testSecretKey, 3600)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ParseToken(token, testSecretKey)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeAdmin, claims.Type)
	assert.True(t, claims.IsAdmin())
}

func TestTokenExpired(t *testing.T) {
	token, err := GenerateAccessToken(testUUID, testIssuer, testUserID, 1, "test-device-id", testRoles, testSecretKey, -1)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	claims, err := ParseToken(token, testSecretKey)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestTokenTypeMismatch(t *testing.T) {
	accessToken, err := GenerateAccessToken(testUUID, testIssuer, testUserID, 1, "test-device-id", testRoles, testSecretKey, 3600)
	assert.NoError(t, err)

	_, err = ParseTokenWithType(accessToken, testSecretKey, TokenTypeRefresh)
	assert.Error(t, err)
	assert.Equal(t, ErrTokenType, err)

	_, err = ParseTokenWithType(accessToken, testSecretKey, TokenTypeAdmin)
	assert.Error(t, err)
	assert.Equal(t, ErrTokenType, err)
}

func TestTokenClaimsChain(t *testing.T) {
	claims := NewTokenClaims(TokenTypeAccess).
		SetUUID(testUUID).
		SetIssuer(testIssuer).
		SetUserIDStr(testUserID).
		SetRoles(testRoles).
		SetExtra("platform", "web").
		SetExtra("device", "pc").
		SetExpire(3600)

	token, err := GenerateToken(claims, testSecretKey)
	assert.NoError(t, err)

	parsedClaims, err := ParseToken(token, testSecretKey)
	assert.NoError(t, err)
	assert.Equal(t, testUUID, parsedClaims.UUID)
	assert.Equal(t, testIssuer, parsedClaims.Issuer)
	assert.Equal(t, "web", parsedClaims.GetExtra("platform"))
	assert.Equal(t, "pc", parsedClaims.GetExtra("device"))
	assert.Empty(t, parsedClaims.GetExtra("nonexistent"))
}

func TestHasRole(t *testing.T) {
	claims := NewTokenClaims(TokenTypeAccess).
		SetRoles([]string{"user", "admin", "moderator"})

	assert.True(t, claims.HasRole("user"))
	assert.True(t, claims.HasRole("admin"))
	assert.True(t, claims.HasRole("moderator"))
	assert.False(t, claims.HasRole("guest"))
	assert.False(t, claims.HasRole(""))

	emptyClaims := NewTokenClaims(TokenTypeAccess)
	assert.False(t, emptyClaims.HasRole("user"))
}

func TestGetRemainingTime(t *testing.T) {
	claims := NewTokenClaims(TokenTypeAccess).
		SetExpire(10)

	remaining := claims.GetRemainingTime()
	assert.Greater(t, remaining, 5*time.Second)
	assert.Less(t, remaining, 10*time.Second)
}

func TestParseTokenInvalidSecret(t *testing.T) {
	token, err := GenerateAccessToken(testUUID, testIssuer, testUserID, 1, "test-device-id", testRoles, testSecretKey, 3600)
	assert.NoError(t, err)

	_, err = ParseToken(token, "wrong-secret")
	assert.Error(t, err)
}

func TestGetUserIDStr(t *testing.T) {
	claims := NewTokenClaims(TokenTypeAccess).
		SetUserIDStr(testUserID)

	assert.Equal(t, "12345", claims.GetUserIDStr())

	strClaims := NewTokenClaims(TokenTypeAccess).
		SetUserIDStr("user-abc")

	assert.Equal(t, "user-abc", strClaims.GetUserIDStr())
}

func TestTokenClaimsMethods(t *testing.T) {
	now := time.Now()
	claims := NewTokenClaims(TokenTypeAccess).
		SetAudience([]string{"mobile", "web"}).
		SetSubject("subject-test").
		SetNotBefore(now).
		SetExpireAt(now.Add(1 * time.Hour))

	token, err := GenerateToken(claims, testSecretKey)
	assert.NoError(t, err)

	parsedClaims, err := ParseToken(token, testSecretKey)
	assert.NoError(t, err)
	assert.Equal(t, jwt.ClaimStrings{"mobile", "web"}, parsedClaims.Audience)
	assert.Equal(t, "subject-test", parsedClaims.Subject)
	assert.False(t, parsedClaims.IsExpired())
}

func TestEmptyRoles(t *testing.T) {
	claims := NewTokenClaims(TokenTypeAccess).
		SetRoles(nil)

	assert.Nil(t, claims.GetRoles())

	claims2 := NewTokenClaims(TokenTypeAccess).
		SetRoles([]string{})

	assert.Empty(t, claims2.GetRoles())
}

func TestConcurrentTokenGeneration(t *testing.T) {
	tokenChan := make(chan string, 10)

	for i := 0; i < 10; i++ {
		go func() {
			token, _ := GenerateAccessToken(testUUID, testIssuer, testUserID, 1, "test-device-id", testRoles, testSecretKey, 3600)
			tokenChan <- token
		}()
	}

	for i := 0; i < 10; i++ {
		token := <-tokenChan
		assert.NotEmpty(t, token)
		claims, err := ParseToken(token, testSecretKey)
		assert.NoError(t, err)
		assert.Equal(t, TokenTypeAccess, claims.Type)
	}
}
