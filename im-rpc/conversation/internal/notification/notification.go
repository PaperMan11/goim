package notification

import (
	"context"

	"github.com/PaperMan11/goim/pkg/msgdispatcher"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"google.golang.org/protobuf/proto"
)

type NotificationSender struct {
	MsgDispatcher msgdispatcher.MsgDispatcher
	MsgService    msgservice.MsgService
	UserService   userservice.UserService
}

func NewNotificationSender(
	msgService msgservice.MsgService,
	userService userservice.UserService,
) *NotificationSender {
	return &NotificationSender{
		MsgDispatcher: msgdispatcher.NewMsgDispatcher(msgService),
		MsgService:    msgService,
		UserService:   userService,
	}
}

func (s *NotificationSender) sendNotification(ctx context.Context, fromUserID, toUserID string, contentType int32, notification proto.Message) error {
	return s.MsgDispatcher.SendNotification(ctx, fromUserID, toUserID, "", contentType, msgdispatcher.SessionTypeMap[contentType], notification)
}

// 会话变更
func (s *NotificationSender) ConversationChangeNotification(ctx context.Context, userID string, conversationIDs []string) error {
	tips := &sdkws.ConversationUpdateTips{
		UserID:             userID,
		ConversationIDList: conversationIDs,
	}
	return s.sendNotification(ctx, userID, userID, constant.ConversationChangeNotification, tips)
}

// 会话私有聊天
func (s *NotificationSender) PrivatePrivateChatNotification(ctx context.Context, fromUserID, toUserID string, conversationID string, isPrivateChat bool) error {
	tips := &sdkws.ConversationSetPrivateTips{
		RecvID:         toUserID,
		SendID:         fromUserID,
		IsPrivate:      isPrivateChat,
		ConversationID: conversationID,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, constant.ConversationPrivateChatNotification, tips)
}
