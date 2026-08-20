package conversation

import "fmt"

const (
	KeyConversationInfo   = "mongo:conversation:info:%s"
	KeyConversationLatest = "mongo:conversation:latest:%s"
)

func GetConversationInfoKey(convID string) string {
	return fmt.Sprintf(KeyConversationInfo, convID)
}

func GetConversationLatestKey(convID string) string {
	return fmt.Sprintf(KeyConversationLatest, convID)
}

const (
	conversationDefaultExpireSeconds = 5 * 60
	conversationNilExpireSeconds     = 60

	ttlJitterRatioPct = 10
)

var (
	sfKeyPrefixConvInfo   = "uf:cv:"
	sfKeyPrefixConvLatest = "uf:cl:"
	sfKeyPrefixBatchConv  = "uf:bc:"
)
