package conversation

import "fmt"

const (
	UserConversationKey          = "user:%s:conversation:%s"
	UserConversationIDsKey       = "user:%s:conversationIDs"
	RecvMsgNotNotifyUserIDsKey   = "conversation:%s:notNotifyUserIDs"
	UserPinnedConversationIDsKey = "user:%s:conversation:pinnedIDs"
)

func GetUserConversationKey(userID string, conversationID string) string {
	return fmt.Sprintf(UserConversationKey, userID, conversationID)
}

func GetUserConversationIDsKey(userID string) string {
	return fmt.Sprintf(UserConversationIDsKey, userID)
}

func GetRecvMsgNotNotifyUserIDsKey(conversationID string) string {
	return fmt.Sprintf(RecvMsgNotNotifyUserIDsKey, conversationID)
}

func GetUserPinnedConversationIDsKey(userID string) string {
	return fmt.Sprintf(UserPinnedConversationIDsKey, userID)
}
