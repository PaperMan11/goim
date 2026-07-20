package logic

import (
	"context"
	"time"

	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/storage/model"
)

func (l *Logic) SubscribeOrCancelUsersStatus(ctx context.Context, req *pbuser.SubscribeOrCancelUsersStatusReq) (*pbuser.SubscribeOrCancelUsersStatusResp, error) {
	return &pbuser.SubscribeOrCancelUsersStatusResp{}, nil
}

func (l *Logic) GetSubscribeUsersStatus(ctx context.Context, req *pbuser.GetSubscribeUsersStatusReq) (*pbuser.GetSubscribeUsersStatusResp, error) {
	return &pbuser.GetSubscribeUsersStatusResp{}, nil
}

func (l *Logic) GetUserStatus(ctx context.Context, req *pbuser.GetUserStatusReq) (*pbuser.GetUserStatusResp, error) {
	statuses, err := l.svcCtx.UserModel.GetUserStatus(ctx, req.GetUserIDs())
	if err != nil {
		return nil, err
	}

	statusMap := make(map[string]*pbuser.OnlineStatus)
	for _, status := range statuses {
		if _, ok := statusMap[status.UserID]; !ok {
			statusMap[status.UserID] = &pbuser.OnlineStatus{
				UserID:      status.UserID,
				Status:      int32(status.Status),
				PlatformIDs: []int32{},
			}
		}
		statusMap[status.UserID].PlatformIDs = append(statusMap[status.UserID].PlatformIDs, int32(status.PlatformID))
	}

	var statusList []*pbuser.OnlineStatus
	for _, status := range statusMap {
		statusList = append(statusList, status)
	}

	return &pbuser.GetUserStatusResp{StatusList: statusList}, nil
}

func (l *Logic) SetUserStatus(ctx context.Context, req *pbuser.SetUserStatusReq) (*pbuser.SetUserStatusResp, error) {
	err := l.svcCtx.UserModel.UpdateUserStatus(ctx, req.GetUserID(), int(req.GetPlatformID()), int(req.GetStatus()))
	if err != nil {
		return nil, err
	}

	return &pbuser.SetUserStatusResp{}, nil
}

func (l *Logic) SetUserOnlineStatus(ctx context.Context, req *pbuser.SetUserOnlineStatusReq) (*pbuser.SetUserOnlineStatusResp, error) {
	var statuses []*model.UserStatus
	for _, status := range req.GetStatus() {
		for _, platform := range status.GetOnline() {
			statuses = append(statuses, &model.UserStatus{
				UserID:         status.GetUserID(),
				Status:         1,
				PlatformID:     int(platform),
				LastOnlineTime: time.Now(),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			})
		}
		for _, platform := range status.GetOffline() {
			statuses = append(statuses, &model.UserStatus{
				UserID:         status.GetUserID(),
				Status:         0,
				PlatformID:     int(platform),
				LastOnlineTime: time.Now(),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			})
		}
	}

	err := l.svcCtx.UserModel.SetUserOnlineStatus(ctx, statuses)
	if err != nil {
		return nil, err
	}

	return &pbuser.SetUserOnlineStatusResp{}, nil
}

func (l *Logic) GetAllOnlineUsers(ctx context.Context, req *pbuser.GetAllOnlineUsersReq) (*pbuser.GetAllOnlineUsersResp, error) {
	userIDs, err := l.svcCtx.UserModel.GetAllOnlineUsers(ctx)
	if err != nil {
		return nil, err
	}

	var statusList []*pbuser.OnlineStatus
	for _, userID := range userIDs {
		statusList = append(statusList, &pbuser.OnlineStatus{
			UserID:      userID,
			Status:      1,
			PlatformIDs: []int32{},
		})
	}

	return &pbuser.GetAllOnlineUsersResp{StatusList: statusList, NextCursor: 0}, nil
}
