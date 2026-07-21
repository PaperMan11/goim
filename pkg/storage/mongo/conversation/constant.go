package conversation

import "fmt"

const (
	KeyConversationInfo   = "mongo:conversation:info:%s:%s"
	KeyConversationLatest = "mongo:conversation:latest:%s:%s"
)

func GetConversationInfoKey(owner, convID string) string {
	return fmt.Sprintf(KeyConversationInfo, owner, convID)
}

func GetConversationLatestKey(owner, convID string) string {
	return fmt.Sprintf(KeyConversationLatest, owner, convID)
}

const (
	conversationDefaultExpireSeconds = 5 * 60
	conversationNilExpireSeconds     = 60

	ttlJitterRatioPct = 10
)

var (
	sfKeyPrefixConvInfo  = "uf:cv:"
	sfKeyPrefixConvLatest = "uf:cl:"
	sfKeyPrefixBatchConv = "uf:bc:"
)
