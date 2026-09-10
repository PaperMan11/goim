package user

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/PaperMan11/goim/pkg/a2r"
	"github.com/gin-gonic/gin"
)

func UserRegister(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.UserRegister)
	}
}

func UpdateUserInfo(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.UpdateUserInfo)
	}
}

func UpdateUserInfoEx(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.UpdateUserInfoEx)
	}
}

func SetGlobalMsgRecvOpt(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.SetGlobalRecvMessageOpt)
	}
}

func GetUsersInfo(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.GetDesignateUsers)
	}
}

func GetAllUsersUid(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.GetAllUserID)
	}
}

func AccountCheck(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.AccountCheck)
	}
}

func GetUsers(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.GetPaginationUsers)
	}
}

func GetUsersOnlineStatus(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.GetUserStatus)
	}
}

func GetUsersOnlineTokenDetail(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.GetAllOnlineUsers)
	}
}

func SubscribeUsersStatus(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.SubscribeOrCancelUsersStatus)
	}
}

func GetUsersStatus(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.GetUserStatus)
	}
}

func GetSubscribeUsersStatus(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.GetSubscribeUsersStatus)
	}
}

func ProcessUserCommandAdd(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.ProcessUserCommandAdd)
	}
}

func ProcessUserCommandDelete(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.ProcessUserCommandDelete)
	}
}

func ProcessUserCommandUpdate(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.ProcessUserCommandUpdate)
	}
}

func ProcessUserCommandGet(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.ProcessUserCommandGet)
	}
}

func ProcessUserCommandGetAll(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.ProcessUserCommandGetAll)
	}
}

func AddNotificationAccount(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.AddNotificationAccount)
	}
}

func UpdateNotificationAccount(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.UpdateNotificationAccountInfo)
	}
}

func SearchNotificationAccount(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.SearchNotificationAccount)
	}
}

func GetUserClientConfig(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.GetUserClientConfig)
	}
}

func SetUserClientConfig(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.SetUserClientConfig)
	}
}

func DelUserClientConfig(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.DelUserClientConfig)
	}
}

func PageUserClientConfig(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.UserService.PageUserClientConfig)
	}
}
