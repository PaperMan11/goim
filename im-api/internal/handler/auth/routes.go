package auth

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/gin-gonic/gin"
)

func InitRoutes(r *gin.RouterGroup, svc *svc.ServiceContext) {
	authRoute := r.Group("/auth")

	authRoute.POST("/get_admin_token", GetAdminToken(svc))
	authRoute.POST("/get_user_token", GetUserToken(svc))
	authRoute.POST("/parse_token", ParseToken(svc))
	authRoute.POST("/force_logout", ForceLogout(svc))
}
