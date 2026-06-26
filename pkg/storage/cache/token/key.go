package token

import (
	"github.com/PaperMan11/goim/pkg/protocol/constant"
)

const (
	TokenPrefix      = "token:"
	UserTokensPrefix = "token:user:"
	PlatformSuffix   = ":platform:"

	TokenDeleteChannel = "token:delete"
)

func GetTokenKey(token string) string {
	return TokenPrefix + token
}

func GetUserTokensKey(userID string) string {
	return UserTokensPrefix + userID + ":tokens"
}

func GetPlatformTokenKey(userID string, platformID int32) string {
	return UserTokensPrefix + userID + PlatformSuffix + constant.PlatformID2Name[int(platformID)] + ":tokens"
}
