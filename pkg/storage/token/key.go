package token

import (
	"github.com/PaperMan11/goim/pkg/protocol/constant"
)

const (
	TokenPrefix      = "token:"
	UserTokensPrefix = "token:user:"
	PlatformSuffix   = ":platform:"

	TokenDeleteChannel = "token:delete"

	// cleanScanBatchSize 全局清理时 SCAN 的单批条数
	cleanScanBatchSize = 200
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
