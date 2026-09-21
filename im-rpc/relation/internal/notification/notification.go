package notification

import (
	"context"

	"github.com/PaperMan11/goim/pkg/mconvert"
	"github.com/PaperMan11/goim/pkg/msgdispatcher"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	friendModel "github.com/PaperMan11/goim/pkg/storage/mongo/friend"
	requestModel "github.com/PaperMan11/goim/pkg/storage/mongo/request"
	"google.golang.org/protobuf/proto"
)

type NotificationSender struct {
	MsgDispatcher msgdispatcher.MsgDispatcher
	MsgService    msgservice.MsgService
	UserService   userservice.UserService
	RequestModel  requestModel.RequestModel
	FriendModel   friendModel.FriendModel
}

func NewNotificationSender(
	msgService msgservice.MsgService,
	userService userservice.UserService,
	requestModel requestModel.RequestModel,
	friendModel friendModel.FriendModel,
) *NotificationSender {
	return &NotificationSender{
		MsgDispatcher: msgdispatcher.NewMsgDispatcher(msgService),
		MsgService:    msgService,
		UserService:   userService,
		RequestModel:  requestModel,
		FriendModel:   friendModel,
	}
}

func (s *NotificationSender) sendNotification(ctx context.Context, fromUserID, toUserID string, contentType int32, notification proto.Message) error {
	return s.MsgDispatcher.SendNotification(ctx, fromUserID, toUserID, "", contentType, msgdispatcher.SessionTypeMap[contentType], notification)
}

func (s *NotificationSender) getFriendRequest(ctx context.Context, fromUserID, toUserID string) (*sdkws.FriendRequest, error) {
	req, err := s.RequestModel.FindFriendRequest(ctx, fromUserID, toUserID)
	if err != nil {
		return nil, err
	}
	return mconvert.ModelToPbFriendRequest(req), nil
}

// 好友申请已同意通知
func (s *NotificationSender) FriendApplyAgreedNotification(ctx context.Context, fromUserID, toUserID string, handleMsg string, version uint64, versionID string) error {
	req, err := s.getFriendRequest(ctx, fromUserID, toUserID)
	if err != nil {
		return err
	}
	notification := &sdkws.FriendApplicationApprovedTips{
		FromToUserID: &sdkws.FromToUserID{
			FromUserID: fromUserID,
			ToUserID:   toUserID,
		},
		HandleMsg:       handleMsg,
		FriendVersion:   version,
		FriendVersionID: versionID,
		Request:         req,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, int32(constant.FriendApplicationApprovedNotification), notification)
}

// 好友申请已拒绝通知
func (s *NotificationSender) FriendApplyRejectedNotification(ctx context.Context, fromUserID, toUserID string, handleMsg string) error {
	req, err := s.getFriendRequest(ctx, fromUserID, toUserID)
	if err != nil {
		return err
	}
	notification := &sdkws.FriendApplicationRejectedTips{
		FromToUserID: &sdkws.FromToUserID{
			FromUserID: fromUserID,
			ToUserID:   toUserID,
		},
		HandleMsg: handleMsg,
		Request:   req,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, int32(constant.FriendApplicationRejectedNotification), notification)
}

// 收到好友申请通知
func (s *NotificationSender) FriendApplyReceived(ctx context.Context, fromUserID, toUserID string) error {
	req, err := s.getFriendRequest(ctx, fromUserID, toUserID)
	if err != nil {
		return err
	}
	notification := &sdkws.FriendApplicationTips{
		FromToUserID: &sdkws.FromToUserID{
			FromUserID: fromUserID,
			ToUserID:   toUserID,
		},
		Request: req,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, int32(constant.FriendApplicationNotification), notification)
}

// 已添加好友通知
// func (s *NotificationSender) FriendAddedNotification(ctx context.Context, fromUserID, toUserID string) error {
// 	notification := &sdkws.FriendAddedTips{}
// 	return s.sendNotification(ctx, fromUserID, toUserID, int32(constant.FriendAddedNotification), notification)
// }

// 已删除好友通知
func (s *NotificationSender) FriendDeletedNotification(ctx context.Context, fromUserID, toUserID string, version uint64, versionID string) error {
	notification := &sdkws.FriendDeletedTips{
		FromToUserID: &sdkws.FromToUserID{
			FromUserID: fromUserID,
			ToUserID:   toUserID,
		},
		FriendVersion:   version,
		FriendVersionID: versionID,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, int32(constant.FriendDeletedNotification), notification)
}

// 好友备注已设置通知
func (s *NotificationSender) FriendRemarkSetNotification(ctx context.Context, fromUserID, toUserID string, sortVersion, version uint64, versionID string) error {
	notification := &sdkws.FriendInfoChangedTips{
		FromToUserID: &sdkws.FromToUserID{
			FromUserID: fromUserID,
			ToUserID:   toUserID,
		},
		FriendSortVersion: sortVersion,
		FriendVersion:     version,
		FriendVersionID:   versionID,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, int32(constant.FriendRemarkSetNotification), notification)
}

// 已加入黑名单通知
func (s *NotificationSender) FriendBlacklistAddedNotification(ctx context.Context, fromUserID, toUserID string) error {
	notification := &sdkws.BlackAddedTips{
		FromToUserID: &sdkws.FromToUserID{
			FromUserID: fromUserID,
			ToUserID:   toUserID,
		},
	}
	return s.sendNotification(ctx, fromUserID, toUserID, int32(constant.BlackAddedNotification), notification)
}

// 已移出黑名单通知
func (s *NotificationSender) FriendBlacklistRemovedNotification(ctx context.Context, fromUserID, toUserID string) error {
	notification := &sdkws.BlackDeletedTips{
		FromToUserID: &sdkws.FromToUserID{
			FromUserID: fromUserID,
			ToUserID:   toUserID,
		},
	}
	return s.sendNotification(ctx, fromUserID, toUserID, int32(constant.BlackDeletedNotification), notification)
}

// 好友信息已更新通知
func (s *NotificationSender) FriendInfoChangedNotification(ctx context.Context, fromUserID, toUserID, changedUserID string) error {
	notification := &sdkws.UserInfoUpdatedTips{
		UserID: changedUserID,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, int32(constant.FriendInfoUpdatedNotification), notification)
}

// 好友信息更新通知
func (s *NotificationSender) FriendInfoUpdatedNotification(ctx context.Context, fromUserID, toUserID string, version uint64, versionID string) error {
	notification := &sdkws.FriendsInfoUpdateTips{
		FromToUserID: &sdkws.FromToUserID{
			FromUserID: fromUserID,
			ToUserID:   toUserID,
		},
		FriendVersion:   version,
		FriendVersionID: versionID,
	}
	return s.sendNotification(ctx, fromUserID, toUserID, int32(constant.FriendsInfoUpdateNotification), notification)
}
