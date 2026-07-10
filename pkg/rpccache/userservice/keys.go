package userservice

import "fmt"

const (
	KeyIMAdmin     = "auth:im_admin:%s"
	KeyValidUser   = "auth:valid_user:%s"
	KeyUserInfo    = "auth:user_info:%s"
	KeyUserRecvOpt = "auth:user_recv_opt:%s"
)

func GetValidUserKey(userID string) string {
	return fmt.Sprintf(KeyValidUser, userID)
}

func GetIMAdminKey(userID string) string {
	return fmt.Sprintf(KeyIMAdmin, userID)
}

func GetUserInfoKey(userID string) string {
	return fmt.Sprintf(KeyUserInfo, userID)
}

func GetUserRecvOptKey(userID string) string {
	return fmt.Sprintf(KeyUserRecvOpt, userID)
}
