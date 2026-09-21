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

func NewNotificationSender(msgService msgservice.MsgService, userService userservice.UserService) *NotificationSender {
	return &NotificationSender{
		MsgDispatcher: msgdispatcher.NewMsgDispatcher(msgService),
		MsgService:    msgService,
		UserService:   userService,
	}
}

func (s *NotificationSender) sendNotification(ctx context.Context, fromUserID, toUserID string, contentType int32, notification proto.Message) error {
	return s.MsgDispatcher.SendNotification(ctx, fromUserID, toUserID, "", contentType, msgdispatcher.SessionTypeMap[contentType], notification)
}

// 用户信息已更新
func (s *NotificationSender) UserInfoUpdatedNotification(ctx context.Context, userID string) error {
	tips := &sdkws.UserInfoUpdatedTips{
		UserID: userID,
	}
	return s.sendNotification(ctx, userID, userID, constant.UserInfoUpdatedNotification, tips)
}

// 用户状态变更
func (s *NotificationSender) UserStatusChangedNotification(ctx context.Context, fromUserID, toUserID string, status, platformID int32) error {
	tips := &sdkws.UserStatusChangeTips{
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Status:     status,
		PlatformID: platformID,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, constant.UserStatusChangeNotification, tips)
}

// 用户命令添加
func (s *NotificationSender) UserCommandAddedNotification(ctx context.Context, fromUserID, toUserID string) error {
	tips := &sdkws.UserCommandAddTips{
		FromUserID: fromUserID,
		ToUserID:   toUserID,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, constant.UserCommandAddNotification, tips)
}

// 用户命令删除
func (s *NotificationSender) UserCommandDeletedNotification(ctx context.Context, fromUserID, toUserID string) error {
	tips := &sdkws.UserCommandDeleteTips{
		FromUserID: fromUserID,
		ToUserID:   toUserID,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, constant.UserCommandDeleteNotification, tips)
}

// 用户命令更新
func (s *NotificationSender) UserCommandUpdatedNotification(ctx context.Context, fromUserID, toUserID string) error {
	tips := &sdkws.UserCommandUpdateTips{
		FromUserID: fromUserID,
		ToUserID:   toUserID,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, constant.UserCommandUpdateNotification, tips)
}
