package auth

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/PaperMan11/goim/pkg/a2r"
	"github.com/gin-gonic/gin"
)

func GetAdminToken(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.AuthService.GetAdminToken)
	}
}

func GetUserToken(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.AuthService.GetUserToken)
	}
}

func ParseToken(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.AuthService.ParseToken)
	}
}

func ForceLogout(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.AuthService.ForceLogout)
	}
}
