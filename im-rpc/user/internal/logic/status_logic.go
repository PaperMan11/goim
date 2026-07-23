package logic

import (
	"context"
	"time"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
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

	type onlineAgg struct {
		*pbuser.OnlineStatus
		platformSet map[int32]struct{}
	}
	aggMap := make(map[string]*onlineAgg)

	for _, uid := range req.GetUserIDs() {
		if uid == "" {
			continue
		}
		if _, dup := aggMap[uid]; dup {
			continue
		}
		aggMap[uid] = &onlineAgg{
			OnlineStatus: &pbuser.OnlineStatus{
				UserID:      uid,
				Status:      constant.Offline,
				PlatformIDs: make([]int32, 0, 4),
			},
			platformSet: make(map[int32]struct{}),
		}
	}

	for _, status := range statuses {
		if status == nil {
			continue
		}
		agg, ok := aggMap[status.UserID]
		if !ok {
			agg = &onlineAgg{
				OnlineStatus: &pbuser.OnlineStatus{
					UserID:      status.UserID,
					Status:      constant.Offline,
					PlatformIDs: make([]int32, 0, 4),
				},
				platformSet: make(map[int32]struct{}),
			}
			aggMap[status.UserID] = agg
		}

		if status.Status == 1 {
			agg.Status = constant.Online
		}

		p := int32(status.PlatformID)
		if _, dup := agg.platformSet[p]; !dup {
			agg.platformSet[p] = struct{}{}
			agg.PlatformIDs = append(agg.PlatformIDs, p)
		}
	}

	statusList := make([]*pbuser.OnlineStatus, 0, len(aggMap))
	for _, agg := range aggMap {
		statusList = append(statusList, agg.OnlineStatus)
	}

	return &pbuser.GetUserStatusResp{StatusList: statusList}, nil
}

func (l *Logic) SetUserStatus(ctx context.Context, req *pbuser.SetUserStatusReq) (*pbuser.SetUserStatusResp, error) {
	err := l.svcCtx.UserModel.UpdateUserStatus(ctx,
		req.GetUserID(),
		int(req.GetPlatformID()),
		req.GetDeviceID(),
		int(req.GetStatus()))
	if err != nil {
		return nil, err
	}

	return &pbuser.SetUserStatusResp{}, nil
}

func (l *Logic) SetUserOnlineStatus(ctx context.Context, req *pbuser.SetUserOnlineStatusReq) (*pbuser.SetUserOnlineStatusResp, error) {
	var statuses []*model.UserStatus
	now := timex.Now()
	expireAt := now.Add(24 * time.Hour)

	for _, status := range req.GetStatus() {
		userID := status.GetUserID()
		deviceID := status.GetDeviceID() // 可能为空串（旧网关），mongo 层会自动退化到 P0 粒度
		connID := status.GetConnID()

		for _, platform := range status.GetOnline() {
			statuses = append(statuses, &model.UserStatus{
				UserID:         userID,
				Status:         1,
				PlatformID:     int(platform),
				DeviceID:       deviceID,
				ConnID:         connID,
				LastOnlineTime: now,
				LastSeenAt:     now,
				ExpireAt:       expireAt,
				CreatedAt:      now,
				UpdatedAt:      now,
			})
		}
		for _, platform := range status.GetOffline() {
			statuses = append(statuses, &model.UserStatus{
				UserID:         userID,
				Status:         0,
				PlatformID:     int(platform),
				DeviceID:       deviceID,
				ConnID:         connID,
				LastOnlineTime: now,
				LastSeenAt:     now,
				// 离线的过期时间缩短到 1h（不再心跳，给 TTL 清理留出缓冲）
				ExpireAt:  now.Add(1 * time.Hour),
				CreatedAt: now,
				UpdatedAt: now,
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
