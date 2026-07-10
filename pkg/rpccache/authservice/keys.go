package authservice

import "fmt"

const (
	UserTokensKey = "user_tokens:"
)

func GetUserTokensKey(userID string, platformID int32) string {
	return fmt.Sprintf("%s%s:%d", UserTokensKey, userID, platformID)
}
