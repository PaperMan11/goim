package logic

import (
	"context"
	"time"

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
	// P1-3: 把 request 里的 DeviceID 透传给 UserModel.UpdateUserStatus，
	// 实现 user+platform+device 粒度的在线状态 upsert。
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

	// P1-2: 把网关传过来的 DeviceID / ConnID 填到 model.UserStatus 里，
	//        让 mongo 层的 BulkUpsert 就能按 user+platform+device 三列唯一键精确去重。
	//        P2 预留的 ExpireAt 默认给一个 24h 的过期窗口，配合 TTL 索引自动清僵尸。
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
