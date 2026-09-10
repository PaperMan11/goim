package user

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/gin-gonic/gin"
)

func InitRoutes(r *gin.RouterGroup, svc *svc.ServiceContext) {
	userRoute := r.Group("/user")

	userRoute.POST("/user_register", UserRegister(svc))
	userRoute.POST("/update_user_info", UpdateUserInfo(svc))
	userRoute.POST("/update_user_info_ex", UpdateUserInfoEx(svc))
	userRoute.POST("/set_global_msg_recv_opt", SetGlobalMsgRecvOpt(svc))
	userRoute.POST("/get_users_info", GetUsersInfo(svc))
	userRoute.POST("/get_all_users_uid", GetAllUsersUid(svc))
	userRoute.POST("/account_check", AccountCheck(svc))
	userRoute.POST("/get_users", GetUsers(svc))
	userRoute.POST("/get_users_online_status", GetUsersOnlineStatus(svc))
	userRoute.POST("/get_users_online_token_detail", GetUsersOnlineTokenDetail(svc))
	userRoute.POST("/subscribe_users_status", SubscribeUsersStatus(svc))
	userRoute.POST("/get_users_status", GetUsersStatus(svc))
	userRoute.POST("/get_subscribe_users_status", GetSubscribeUsersStatus(svc))

	userRoute.POST("/process_user_command_add", ProcessUserCommandAdd(svc))
	userRoute.POST("/process_user_command_delete", ProcessUserCommandDelete(svc))
	userRoute.POST("/process_user_command_update", ProcessUserCommandUpdate(svc))
	userRoute.POST("/process_user_command_get", ProcessUserCommandGet(svc))
	userRoute.POST("/process_user_command_get_all", ProcessUserCommandGetAll(svc))

	userRoute.POST("/add_notification_account", AddNotificationAccount(svc))
	userRoute.POST("/update_notification_account", UpdateNotificationAccount(svc))
	userRoute.POST("/search_notification_account", SearchNotificationAccount(svc))

	userRoute.POST("/get_user_client_config", GetUserClientConfig(svc))
	userRoute.POST("/set_user_client_config", SetUserClientConfig(svc))
	userRoute.POST("/del_user_client_config", DelUserClientConfig(svc))
	userRoute.POST("/page_user_client_config", PageUserClientConfig(svc))
}
