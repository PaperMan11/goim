package conversation

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/gin-gonic/gin"
)

func InitRoutes(r *gin.RouterGroup, svc *svc.ServiceContext) {
	convRoute := r.Group("/conversation")

	convRoute.POST("/get_sorted_conversation_list", GetSortedConversationList(svc))
	convRoute.POST("/get_all_conversations", GetAllConversations(svc))
	convRoute.POST("/get_conversation", GetConversation(svc))
	convRoute.POST("/get_conversations", GetConversations(svc))
	convRoute.POST("/set_conversations", SetConversations(svc))
	convRoute.POST("/get_conversation_offline_push_user_ids", GetConversationOfflinePushUserIDs(svc))
	convRoute.POST("/get_full_conversation_ids", GetFullConversationIDs(svc))
	convRoute.POST("/get_incremental_conversations", GetIncrementalConversations(svc))
	convRoute.POST("/get_owner_conversation", GetOwnerConversation(svc))
	convRoute.POST("/get_not_notify_conversation_ids", GetNotNotifyConversationIDs(svc))
	convRoute.POST("/get_pinned_conversation_ids", GetPinnedConversationIDs(svc))
	convRoute.POST("/delete_conversations", DeleteConversations(svc))
	convRoute.POST("/update_conversations_by_user", UpdateConversationsByUser(svc))
}
