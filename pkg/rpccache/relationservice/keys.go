package relationservice

import "fmt"

const (
	FriendInfoKey   = "friend_info:%s"    // 好友信息缓存键
	FriendIDListKey = "friend_id_list:%s" // 好友ID列表缓存键
)

func GetFriendInfoKey(userID string) string {
	return fmt.Sprintf(FriendInfoKey, userID)
}

func GetFriendIDListKey(userID string) string {
	return fmt.Sprintf(FriendIDListKey, userID)
}
