package logic

import (
	"context"

	"github.com/PaperMan11/goim/pkg/utils/timex"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbconv "github.com/PaperMan11/goim/pkg/protocol/conversation"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
)

// sendRevokeNotification 发送撤回消息通知
func (l *Logic) sendRevokeNotification(ctx context.Context, userID string, dbMsg *model.MsgDataModel, role int32, conversationID string, seq int64) {
	tips := sdkws.RevokeMsgTips{
		RevokerUserID:  userID,
		ClientMsgID:    dbMsg.ClientMsgID,
		RevokeTime:     timex.UnixMilli(),
		Seq:            seq,
		SesstionType:   dbMsg.SessionType,
		ConversationID: conversationID,
		IsAdminRevoke:  role >= constant.GroupAdmin,
	}
	var recvID string
	if dbMsg.SessionType == constant.ReadGroupChatType {
		recvID = dbMsg.GroupID
	} else {
		recvID = dbMsg.RecvID
	}
	l.svcCtx.NotificationSender.SendNotification(ctx, userID, recvID, dbMsg.GroupID, constant.MsgRevokeNotification, dbMsg.SessionType, &tips)
}

// sendMarkAsReadNotification 发送消息已读通知
func (l *Logic) sendMarkAsReadNotification(ctx context.Context, userID string, conversation *pbconv.Conversation, seqs []int64, hasReadSeq int64) {
	tips := &sdkws.MarkAsReadTips{
		MarkAsReadUserID: userID,
		ConversationID:   conversation.ConversationID,
		Seqs:             seqs,
		HasReadSeq:       hasReadSeq,
	}
	var recvID string
	if conversation.ConversationType == constant.ReadGroupChatType {
		recvID = conversation.GroupID
	} else {
		recvID = conversation.UserID
	}
	l.svcCtx.NotificationSender.SendNotification(ctx, userID, recvID, conversation.GroupID, constant.HasReadReceipt, conversation.ConversationType, tips)
}

// sendDeleteMsgsNotification 发送删除消息通知
func (l *Logic) sendDeleteMsgsNotification(ctx context.Context, userID string, conversationID string, seqs []int64, conv *pbconv.Conversation) {
	tips := &sdkws.DeleteMsgsTips{UserID: userID, ConversationID: conversationID, Seqs: seqs}
	recvID := userID
	if conv.ConversationType == constant.SingleChatType || conv.ConversationType == constant.NotificationChatType {
		if userID == conv.OwnerUserID {
			recvID = conv.UserID
		} else {
			recvID = conv.OwnerUserID
		}
	} else {
		recvID = conv.GroupID
	}
	err := l.svcCtx.NotificationSender.SendNotification(ctx, userID, recvID, conv.GroupID, constant.DeleteMsgsNotification, conv.ConversationType, tips)
	if err != nil {
		l.Errorf("send delete msg notification failed, err: %v, userID: %s, conversationID: %s, seqs: %v", err, userID, conversationID, seqs)
	}
}

// sendDeleteMsgsSelfNotification 仅同步给自己的删除消息通知
func (l *Logic) sendDeleteMsgsSelfNotification(ctx context.Context, userID string, conversationID string, seqs []int64) {
	tips := &sdkws.DeleteMsgsTips{UserID: userID, ConversationID: conversationID, Seqs: seqs}
	err := l.svcCtx.NotificationSender.SendNotification(ctx, userID, userID, "", constant.DeleteMsgsNotification, constant.SingleChatType, tips)
	if err != nil {
		l.Errorf("send delete msg notification failed, err: %v, userID: %s, conversationID: %s, seqs: %v", err, userID, conversationID, seqs)
	}
}

func (l *Logic) sendClearConversationsMsgNotification(ctx context.Context, sendID, recvID string, conversationIDs []string, sessionType int32) {
	tips := &sdkws.ClearConversationTips{UserID: sendID, ConversationIDs: conversationIDs}
	l.svcCtx.NotificationSender.SendNotification(ctx, sendID, recvID, "", constant.ClearConversationNotification, sessionType, tips)
}
