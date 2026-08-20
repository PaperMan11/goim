package msgservice

import "fmt"

const (
	ConversationMaxSeqKey = "msg:conv_max_seq:%s" // 会话最大序列号缓存键
	ServerTimeKey         = "msg:server_time"      // 服务器时间缓存键
)

func GetConversationMaxSeqKey(conversationID string) string {
	return fmt.Sprintf(ConversationMaxSeqKey, conversationID)
}

func GetServerTimeKey() string {
	return ServerTimeKey
}
