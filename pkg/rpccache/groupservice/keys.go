package groupservice

import "fmt"

const (
	GroupInfoKey           = "group_info:%s"       // 群组信息缓存键
	GroupMemberFullInfoKey = "group:%s:member:%s"  // 群成员完整信息缓存键
	GroupMemberIDsKey      = "group:%s:member_ids" // 群成员ID集合缓存键
)

func GetGroupInfoKey(groupIdID string) string {
	return fmt.Sprintf(GroupInfoKey, groupIdID)
}

func GetGroupMemberFullInfoKey(groupID, userID string) string {
	return fmt.Sprintf(GroupMemberFullInfoKey, groupID, userID)
}

func GetGroupMemberIDsKey(groupID string) string {
	return fmt.Sprintf(GroupMemberIDsKey, groupID)
}
