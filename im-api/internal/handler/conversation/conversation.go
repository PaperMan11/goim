package conversation

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/PaperMan11/goim/pkg/a2r"
	"github.com/gin-gonic/gin"
)

func GetSortedConversationList(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.GetSortedConversationList)
	}
}

func GetAllConversations(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.GetAllConversations)
	}
}

func GetConversation(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.GetConversation)
	}
}

func GetConversations(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.GetConversations)
	}
}

func SetConversations(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.SetConversations)
	}
}

func GetConversationOfflinePushUserIDs(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.GetConversationOfflinePushUserIDs)
	}
}

func GetFullConversationIDs(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.GetFullOwnerConversationIDs)
	}
}

func GetIncrementalConversations(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.GetIncrementalConversation)
	}
}

func GetOwnerConversation(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.GetOwnerConversation)
	}
}

func GetNotNotifyConversationIDs(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.GetNotNotifyConversationIDs)
	}
}

func GetPinnedConversationIDs(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.GetPinnedConversationIDs)
	}
}

func DeleteConversations(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.DeleteConversations)
	}
}

func UpdateConversationsByUser(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ConvService.UpdateConversationsByUser)
	}
}
