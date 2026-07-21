package friend

import "fmt"

const (
	KeyFriendInfo    = "mongo:friend:info:%s:%s"
	KeyFriendVersion = "mongo:friend:version:%s"
	KeyBlackInfo     = "mongo:friend:black:%s:%s"
	KeyFriendExists  = "mongo:friend:exists:%s:%s"
)

func GetFriendInfoKey(ownerUserID, friendUserID string) string {
	return fmt.Sprintf(KeyFriendInfo, ownerUserID, friendUserID)
}

func GetFriendVersionKey(ownerUserID string) string {
	return fmt.Sprintf(KeyFriendVersion, ownerUserID)
}

func GetBlackInfoKey(ownerUserID, blackUserID string) string {
	return fmt.Sprintf(KeyBlackInfo, ownerUserID, blackUserID)
}

func GetFriendExistsKey(userA, userB string) string {
	return fmt.Sprintf(KeyFriendExists, userA, userB)
}

const (
	friendDefaultExpireSeconds = 5 * 60
	friendNilExpireSeconds     = 60

	ttlJitterRatioPct = 10
)

var (
	sfKeyPrefixFriend        = "uf:fr:"
	sfKeyPrefixBlack         = "uf:bk:"
	sfKeyPrefixFriendVersion = "uf:fv:"
	sfKeyPrefixFriendBatch   = "uf:fb:"
	sfKeyPrefixBlackBatch    = "uf:bb:"
	sfKeyPrefixFriendExists  = "uf:fe:"
)
