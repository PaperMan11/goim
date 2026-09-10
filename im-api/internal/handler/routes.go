package handler

import (
	"github.com/PaperMan11/goim/im-api/internal/handler/auth"
	"github.com/PaperMan11/goim/im-api/internal/handler/conversation"
	"github.com/PaperMan11/goim/im-api/internal/handler/group"
	"github.com/PaperMan11/goim/im-api/internal/handler/msg"
	"github.com/PaperMan11/goim/im-api/internal/handler/relation"
	"github.com/PaperMan11/goim/im-api/internal/handler/user"
	"github.com/PaperMan11/goim/im-api/internal/middlewares"
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/gin-gonic/gin"
)

func InitRoutes(svc *svc.ServiceContext) *gin.Engine {
	// 初始化路由
	r := gin.New()
	r.Use(middlewares.Logger(), middlewares.Recovery())
	r.Use(middlewares.Cors())
	baseRoute := r.Group("/api/v1")

	auth.InitRoutes(baseRoute, svc)
	msg.InitRoutes(baseRoute, svc)
	user.InitRoutes(baseRoute, svc)
	group.InitRoutes(baseRoute, svc)
	relation.InitRoutes(baseRoute, svc)
	conversation.InitRoutes(baseRoute, svc)
	return r
}
