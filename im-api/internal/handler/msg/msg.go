package msg

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/PaperMan11/goim/pkg/a2r"
	"github.com/gin-gonic/gin"
)

func NewestSeq(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.GetMaxSeq)
	}
}

func SearchMsg(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.SearchMessage)
	}
}

func SendMsg(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.SendMsg)
	}
}

func SendBusinessNotification(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.SendMsg)
	}
}

func PullMsgBySeq(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.PullMessageBySeqs)
	}
}

func RevokeMsg(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.RevokeMsg)
	}
}

func MarkMsgsAsRead(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.MarkMsgsAsRead)
	}
}

func MarkConversationAsRead(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.MarkConversationAsRead)
	}
}

func GetConversationsHasReadAndMaxSeq(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.GetConversationsHasReadAndMaxSeq)
	}
}

func SetConversationHasReadSeq(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.SetConversationHasReadSeq)
	}
}

func ClearConversationMsg(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.ClearConversationsMsg)
	}
}

func UserClearAllMsg(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.UserClearAllMsg)
	}
}

func DeleteMsgs(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.DeleteMsgs)
	}
}

func DeleteMsgPhsicalBySeq(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.DeleteMsgPhysicalBySeq)
	}
}

func DeleteMsgPhysical(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.DeleteMsgPhysical)
	}
}

func BatchSendMsg(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.SendMsg)
	}
}

func SendSimpleMsg(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.SendSimpleMsg)
	}
}

func CheckMsgIsSendSuccess(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.GetSendMsgStatus)
	}
}

func GetServerTime(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.MsgService.GetServerTime)
	}
}
