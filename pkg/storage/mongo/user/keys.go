package user

import "fmt"

const (
	KeyUserInfo          = "mongo:user:info:%s"
	KeyUserGlobalRecvOpt = "mongo:user:global_recv_opt:%s"
	KeyUserExists        = "mongo:user:exists:%s"
	KeyIMAdmin           = "mongo:user:im_admin:%s"
)

func GetUserInfoKey(userID string) string {
	return fmt.Sprintf(KeyUserInfo, userID)
}

func GetUserGlobalRecvOptKey(userID string) string {
	return fmt.Sprintf(KeyUserGlobalRecvOpt, userID)
}

func GetUserExistsKey(userID string) string {
	return fmt.Sprintf(KeyUserExists, userID)
}

func GetIMAdminKey(userID string) string {
	return fmt.Sprintf(KeyIMAdmin, userID)
}