package group

import "fmt"

const (
	KeyGroupInfo        = "mongo:group:info:%s"
	KeyGroupMember      = "mongo:group:member:%s:%s"
	KeyGroupExists      = "mongo:group:exists:%s"
	KeyGroupMemberCount = "mongo:group:member_count:%s"
)

func GetGroupInfoKey(groupID string) string {
	return fmt.Sprintf(KeyGroupInfo, groupID)
}

func GetGroupMemberKey(groupID, userID string) string {
	return fmt.Sprintf(KeyGroupMember, groupID, userID)
}

func GetGroupExistsKey(groupID string) string {
	return fmt.Sprintf(KeyGroupExists, groupID)
}

func GetGroupMemberCountKey(groupID string) string {
	return fmt.Sprintf(KeyGroupMemberCount, groupID)
}

const (
	groupDefaultExpireSeconds = 10 * 60
	groupNilExpireSeconds     = 60

	ttlJitterRatioPct = 10
)

var (
	sfKeyPrefixGroupInfo   = "ug:gi:"
	sfKeyPrefixMember      = "ug:gm:"
	sfKeyPrefixMemberCount = "ug:mc:"
	sfKeyPrefixMemberBatch = "ug:mb:"
	sfKeyPrefixExists      = "ug:ex:"
)
