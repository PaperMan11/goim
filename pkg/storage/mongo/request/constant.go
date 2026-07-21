package request

import "fmt"

const (
	KeyFriendRequest = "mongo:request:friend:%s:%s"
	KeyGroupRequest  = "mongo:request:group:%s:%s"
)

func GetFriendRequestKey(fromUserID, toUserID string) string {
	return fmt.Sprintf(KeyFriendRequest, fromUserID, toUserID)
}

func GetGroupRequestKey(userID, groupID string) string {
	return fmt.Sprintf(KeyGroupRequest, userID, groupID)
}

const (
	reqDefaultExpireSeconds = 5 * 60
	reqNilExpireSeconds     = 60

	ttlJitterRatioPct = 10
)

var (
	sfKeyPrefixFriendReq = "ur:fq:"
	sfKeyPrefixGroupReq  = "ur:gq:"
)
