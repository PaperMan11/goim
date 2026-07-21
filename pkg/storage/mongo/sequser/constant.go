package sequser

import "fmt"

const (
	KeySeqUser = "mongo:sequser:%s:%s"
)

func GetSeqUserKey(userID string, conversationID string) string {
	return fmt.Sprintf(KeySeqUser, userID, conversationID)
}

const (
	userSeqDefaultExpireSeconds = 60 * 60
	userSeqNilExpireSeconds     = 30

	ttlJitterRatioPct = 10
)

var (
	sfKeyPrefixSeqUser = "uq:su:"
)
