package msg

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/gin-gonic/gin"
)

func InitRoutes(r *gin.RouterGroup, svc *svc.ServiceContext) {
	msgRoute := r.Group("/msg")

	msgRoute.POST("/newest_seq", NewestSeq(svc))
	msgRoute.POST("/search_msg", SearchMsg(svc))
	msgRoute.POST("/send_msg", SendMsg(svc))
	msgRoute.POST("/send_business_notification", SendBusinessNotification(svc))
	msgRoute.POST("/pull_msg_by_seq", PullMsgBySeq(svc))
	msgRoute.POST("/revoke_msg", RevokeMsg(svc))
	msgRoute.POST("/mark_msgs_as_read", MarkMsgsAsRead(svc))
	msgRoute.POST("/mark_conversation_as_read", MarkConversationAsRead(svc))
	msgRoute.POST("/get_conversations_has_read_and_max_seq", GetConversationsHasReadAndMaxSeq(svc))
	msgRoute.POST("/set_conversation_has_read_seq", SetConversationHasReadSeq(svc))

	msgRoute.POST("/clear_conversation_msg", ClearConversationMsg(svc))
	msgRoute.POST("/user_clear_all_msg", UserClearAllMsg(svc))
	msgRoute.POST("/delete_msgs", DeleteMsgs(svc))
	msgRoute.POST("/delete_msg_phsical_by_seq", DeleteMsgPhsicalBySeq(svc))
	msgRoute.POST("/delete_msg_physical", DeleteMsgPhysical(svc))

	msgRoute.POST("/batch_send_msg", BatchSendMsg(svc))
	msgRoute.POST("/send_simple_msg", SendSimpleMsg(svc))
	msgRoute.POST("/check_msg_is_send_success", CheckMsgIsSendSuccess(svc))
	msgRoute.POST("/get_server_time", GetServerTime(svc))
}
