package logic

import (
	"context"
	"sort"
	"strings"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbrelation "github.com/PaperMan11/goim/pkg/protocol/relation"
	sdkws "github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"

	"github.com/PaperMan11/goim/pkg/utils/timex"
)

// ==================== 好友管理（版本日志 DID=ownerUserID, EID=friendUserID） ====================

// ApplyToAddFriend 创建好友申请记录到 RequestModel。不直接写版本日志（等申请被同意后才写）。
func (l *Logic) ApplyToAddFriend(ctx context.Context, req *pbrelation.ApplyToAddFriendReq) (*pbrelation.ApplyToAddFriendResp, error) {
	fromUserID := req.GetFromUserID()
	toUserID := req.GetToUserID()
	if fromUserID == "" || toUserID == "" {
		return nil, errx.ArgsError.Wrap("fromUserID and toUserID are required")
	}
	if fromUserID == toUserID {
		return nil, errx.CanNotAddYourselfError
	}

	if err := l.requireValidUser(toUserID); err != nil {
		return nil, err
	}

	// 检查是否已被对方拉黑
	inBlack, err := l.svcCtx.FriendModel.IsBlack(ctx, toUserID, fromUserID)
	if err != nil {
		l.Errorf("check black failed, owner: %s, target: %s, err: %v", toUserID, fromUserID, err)
		return nil, err
	}
	if inBlack {
		return nil, errx.BlockedByPeer
	}

	// 检查是否已经是好友
	isFriend, err := l.svcCtx.FriendModel.IsFriend(ctx, fromUserID, toUserID)
	if err != nil {
		l.Errorf("check friend failed, userA: %s, userB: %s, err: %v", fromUserID, toUserID, err)
		return nil, err
	}
	if isFriend {
		return nil, errx.RelationshipAlreadyError
	}

	now := timex.Now()
	friendReq := &model.FriendRequest{
		FromUserID:   fromUserID,
		ToUserID:     toUserID,
		HandleResult: constant.FriendResponseNotHandle,
		ReqMsg:       req.GetReqMsg(),
		CreateTime:   now,
		Extra:        req.GetEx(),
	}

	if err := l.svcCtx.RequestModel.UpsertFriendRequest(ctx, friendReq); err != nil {
		l.Errorf("upsert friend request failed, from: %s, to: %s, err: %v", fromUserID, toUserID, err)
		return nil, err
	}

	return &pbrelation.ApplyToAddFriendResp{}, nil
}

// RespondFriendApply 处理好友申请。同意时：双向插入好友记录 + 对双方都写版本日志。
func (l *Logic) RespondFriendApply(ctx context.Context, req *pbrelation.RespondFriendApplyReq) (*pbrelation.RespondFriendApplyResp, error) {
	fromUserID := req.GetFromUserID()
	toUserID := req.GetToUserID()
	handleResult := req.GetHandleResult()

	if fromUserID == "" || toUserID == "" {
		return nil, errx.ArgsError.Wrap("fromUserID and toUserID are required")
	}

	if err := l.requireSelfOrAdmin(toUserID); err != nil {
		return nil, err
	}

	friendReq, err := l.svcCtx.RequestModel.FindFriendRequest(ctx, fromUserID, toUserID)
	if err != nil {
		l.Errorf("find friend request failed, from: %s, to: %s, err: %v", fromUserID, toUserID, err)
		return nil, err
	}
	if friendReq.HandleResult != constant.FriendResponseNotHandle {
		return nil, errx.FriendRequestHandled
	}

	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	modelHandleResult := constant.FriendResponseAgree
	if handleResult == 2 {
		modelHandleResult = constant.FriendResponseRefuse
	}

	if err := l.svcCtx.RequestModel.HandleFriendRequest(ctx, fromUserID, toUserID, opUserID, modelHandleResult, req.GetHandleMsg()); err != nil {
		l.Errorf("handle friend request failed, from: %s, to: %s, err: %v", fromUserID, toUserID, err)
		return nil, err
	}

	if modelHandleResult == constant.FriendResponseAgree {
		now := timex.Now()
		// 双向插入好友记录
		friendAToB := &model.Friend{
			OwnerUserID:    fromUserID,
			FriendUserID:   toUserID,
			Remark:         "",
			CreateTime:     now,
			AddSource:      constant.BecomeFriendByApply,
			OperatorUserID: opUserID,
			UpdatedAt:      now,
		}
		friendBToA := &model.Friend{
			OwnerUserID:    toUserID,
			FriendUserID:   fromUserID,
			Remark:         "",
			CreateTime:     now,
			AddSource:      constant.BecomeFriendByApply,
			OperatorUserID: opUserID,
			UpdatedAt:      now,
		}
		if err := l.svcCtx.FriendModel.InsertFriends(ctx, []*model.Friend{friendAToB, friendBToA}); err != nil {
			l.Errorf("insert friends failed, from: %s, to: %s, err: %v", fromUserID, toUserID, err)
			return nil, err
		}

		// 分别对两个用户写版本日志
		if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, fromUserID, toUserID, model.VersionStateInsert); err != nil {
			l.Errorf("incr version log for friend insert failed, owner: %s, friend: %s, err: %v", fromUserID, toUserID, err)
		}
		if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, toUserID, fromUserID, model.VersionStateInsert); err != nil {
			l.Errorf("incr version log for friend insert failed, owner: %s, friend: %s, err: %v", toUserID, fromUserID, err)
		}
	}

	return &pbrelation.RespondFriendApplyResp{}, nil
}

// ImportFriends 批量导入好友。IncrVersionLogBatch(owner, friendUserIDs, Insert)。
func (l *Logic) ImportFriends(ctx context.Context, req *pbrelation.ImportFriendReq) (*pbrelation.ImportFriendResp, error) {
	ownerUserID := req.GetOwnerUserID()
	friendUserIDs := req.GetFriendUserIDs()
	if ownerUserID == "" || len(friendUserIDs) == 0 {
		return nil, errx.ArgsError.Wrap("ownerUserID and friendUserIDs are required")
	}

	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	now := timex.Now()
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	var friends []*model.Friend
	for _, friendUserID := range friendUserIDs {
		friends = append(friends, &model.Friend{
			OwnerUserID:    ownerUserID,
			FriendUserID:   friendUserID,
			CreateTime:     now,
			AddSource:      constant.BecomeFriendByImport,
			OperatorUserID: opUserID,
			UpdatedAt:      now,
		})
	}

	if err := l.svcCtx.FriendModel.InsertFriends(ctx, friends); err != nil {
		l.Errorf("insert friends failed, owner: %s, err: %v", ownerUserID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, ownerUserID, friendUserIDs, model.VersionStateInsert); err != nil {
		l.Errorf("incr version log batch for friend insert failed, owner: %s, err: %v", ownerUserID, err)
	}

	return &pbrelation.ImportFriendResp{}, nil
}

// DeleteFriend 删除好友。IncrVersionLog(owner, friendUserID, Delete)。
func (l *Logic) DeleteFriend(ctx context.Context, req *pbrelation.DeleteFriendReq) (*pbrelation.DeleteFriendResp, error) {
	ownerUserID := req.GetOwnerUserID()
	friendUserID := req.GetFriendUserID()
	if ownerUserID == "" || friendUserID == "" {
		return nil, errx.ArgsError.Wrap("ownerUserID and friendUserID are required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	if err := l.svcCtx.FriendModel.DeleteFriend(ctx, ownerUserID, friendUserID); err != nil {
		l.Errorf("delete friend failed, owner: %s, friend: %s, err: %v", ownerUserID, friendUserID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, ownerUserID, friendUserID, model.VersionStateDelete); err != nil {
		l.Errorf("incr version log for friend delete failed, owner: %s, friend: %s, err: %v", ownerUserID, friendUserID, err)
	}

	return &pbrelation.DeleteFriendResp{}, nil
}

// UpdateFriends 更新好友信息（remark, isPinned, ex）。isPinned 变更时额外写 VersionSortChangeID。
func (l *Logic) UpdateFriends(ctx context.Context, req *pbrelation.UpdateFriendsReq) (*pbrelation.UpdateFriendsResp, error) {
	ownerUserID := req.GetOwnerUserID()
	friendUserIDs := req.GetFriendUserIDs()
	if ownerUserID == "" || len(friendUserIDs) == 0 {
		return nil, errx.ArgsError.Wrap("ownerUserID and friendUserIDs are required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	updates := make(map[string]any)
	isPinnedChanged := false
	if req.IsPinned != nil {
		updates["is_pinned"] = req.IsPinned.GetValue()
		isPinnedChanged = true
	}
	if req.Remark != nil {
		updates["remark"] = req.Remark.GetValue()
	}
	if req.Ex != nil {
		updates["extra"] = req.Ex.GetValue()
	}

	if len(updates) == 0 {
		return &pbrelation.UpdateFriendsResp{}, nil
	}

	for _, friendUserID := range friendUserIDs {
		if err := l.svcCtx.FriendModel.UpdateFriend(ctx, ownerUserID, friendUserID, updates); err != nil {
			l.Errorf("update friend failed, owner: %s, friend: %s, err: %v", ownerUserID, friendUserID, err)
			return nil, err
		}
		// isPinned 变更会影响好友列表排序顺序，合并好友更新 + 排序变更为一次 batch
		if isPinnedChanged {
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, ownerUserID, []string{friendUserID, model.VersionSortChangeID}, model.VersionStateUpdate); err != nil {
				l.Errorf("incr version log batch for friend+sort update failed, owner: %s, friend: %s, err: %v", ownerUserID, friendUserID, err)
			}
		} else {
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, ownerUserID, friendUserID, model.VersionStateUpdate); err != nil {
				l.Errorf("incr version log for friend update failed, owner: %s, friend: %s, err: %v", ownerUserID, friendUserID, err)
			}
		}
	}

	return &pbrelation.UpdateFriendsResp{}, nil
}

// SetFriendRemark 设置备注。IncrVersionLog(owner, friendUserID, Update)。
func (l *Logic) SetFriendRemark(ctx context.Context, req *pbrelation.SetFriendRemarkReq) (*pbrelation.SetFriendRemarkResp, error) {
	ownerUserID := req.GetOwnerUserID()
	friendUserID := req.GetFriendUserID()
	if ownerUserID == "" || friendUserID == "" {
		return nil, errx.ArgsError.Wrap("ownerUserID and friendUserID are required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	if err := l.svcCtx.FriendModel.UpdateFriend(ctx, ownerUserID, friendUserID, map[string]any{
		"remark": req.GetRemark(),
	}); err != nil {
		l.Errorf("update friend remark failed, owner: %s, friend: %s, err: %v", ownerUserID, friendUserID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, ownerUserID, friendUserID, model.VersionStateUpdate); err != nil {
		l.Errorf("incr version log for friend update failed, owner: %s, friend: %s, err: %v", ownerUserID, friendUserID, err)
	}

	return &pbrelation.SetFriendRemarkResp{}, nil
}

// ==================== 黑名单管理（版本日志 DID=ownerUserID, EID=blackUserID） ====================

// AddBlack 插入黑名单 + IncrVersionLog(owner, blackUserID, Insert)。
func (l *Logic) AddBlack(ctx context.Context, req *pbrelation.AddBlackReq) (*pbrelation.AddBlackResp, error) {
	ownerUserID := req.GetOwnerUserID()
	blackUserID := req.GetBlackUserID()
	if ownerUserID == "" || blackUserID == "" {
		return nil, errx.ArgsError.Wrap("ownerUserID and blackUserID are required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	now := timex.Now()
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	black := &model.Black{
		OwnerUserID:    ownerUserID,
		BlackUserID:    blackUserID,
		CreateTime:     now,
		AddSource:      constant.BecomeFriendByApply,
		OperatorUserID: opUserID,
		Extra:          req.GetEx(),
		UpdatedAt:      now,
	}

	if err := l.svcCtx.FriendModel.InsertBlack(ctx, black); err != nil {
		l.Errorf("insert black failed, owner: %s, black: %s, err: %v", ownerUserID, blackUserID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, ownerUserID, blackUserID, model.VersionStateInsert); err != nil {
		l.Errorf("incr version log for black insert failed, owner: %s, black: %s, err: %v", ownerUserID, blackUserID, err)
	}

	return &pbrelation.AddBlackResp{}, nil
}

// RemoveBlack 删除黑名单 + IncrVersionLog(owner, blackUserID, Delete)。
func (l *Logic) RemoveBlack(ctx context.Context, req *pbrelation.RemoveBlackReq) (*pbrelation.RemoveBlackResp, error) {
	ownerUserID := req.GetOwnerUserID()
	blackUserID := req.GetBlackUserID()
	if ownerUserID == "" || blackUserID == "" {
		return nil, errx.ArgsError.Wrap("ownerUserID and blackUserID are required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	if err := l.svcCtx.FriendModel.DeleteBlack(ctx, ownerUserID, blackUserID); err != nil {
		l.Errorf("delete black failed, owner: %s, black: %s, err: %v", ownerUserID, blackUserID, err)
		return nil, err
	}

	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, ownerUserID, blackUserID, model.VersionStateDelete); err != nil {
		l.Errorf("incr version log for black delete failed, owner: %s, black: %s, err: %v", ownerUserID, blackUserID, err)
	}

	return &pbrelation.RemoveBlackResp{}, nil
}

// ==================== 好友申请管理 ====================

// GetPaginationFriendsApplyTo 分页查询收到的好友申请
func (l *Logic) GetPaginationFriendsApplyTo(ctx context.Context, req *pbrelation.GetPaginationFriendsApplyToReq) (*pbrelation.GetPaginationFriendsApplyToResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	pagination := req.GetPagination()
	page := int64(pagination.GetPageNumber())
	size := int64(pagination.GetShowNumber())
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 50
	}

	requests, total, err := l.svcCtx.RequestModel.FindFriendRequestsByTo(ctx, userID, page, size)
	if err != nil {
		l.Errorf("find friend requests by to failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	var friendRequests []*sdkws.FriendRequest
	for _, r := range requests {
		friendRequests = append(friendRequests, modelToSDKFriendRequest(r))
	}

	return &pbrelation.GetPaginationFriendsApplyToResp{
		FriendRequests: friendRequests,
		Total:          int32(total),
	}, nil
}

// GetPaginationFriendsApplyFrom 分页查询发出的好友申请
func (l *Logic) GetPaginationFriendsApplyFrom(ctx context.Context, req *pbrelation.GetPaginationFriendsApplyFromReq) (*pbrelation.GetPaginationFriendsApplyFromResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	pagination := req.GetPagination()
	page := int64(pagination.GetPageNumber())
	size := int64(pagination.GetShowNumber())
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 50
	}

	requests, total, err := l.svcCtx.RequestModel.FindFriendRequestsByFrom(ctx, userID, page, size)
	if err != nil {
		l.Errorf("find friend requests by from failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	var friendRequests []*sdkws.FriendRequest
	for _, r := range requests {
		friendRequests = append(friendRequests, modelToSDKFriendRequest(r))
	}

	return &pbrelation.GetPaginationFriendsApplyFromResp{
		FriendRequests: friendRequests,
		Total:          int32(total),
	}, nil
}

// GetSelfUnhandledApplyCount 查询未处理好友申请数量
func (l *Logic) GetSelfUnhandledApplyCount(ctx context.Context, req *pbrelation.GetSelfUnhandledApplyCountReq) (*pbrelation.GetSelfUnhandledApplyCountResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	requests, _, err := l.svcCtx.RequestModel.FindFriendRequestsByTo(ctx, userID, 1, 0)
	if err != nil {
		l.Errorf("find friend requests by to failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	var count int64
	for _, r := range requests {
		if r.HandleResult == constant.FriendResponseNotHandle {
			count++
		}
	}

	return &pbrelation.GetSelfUnhandledApplyCountResp{
		Count: count,
	}, nil
}

// GetDesignatedFriendsApply 查询指定好友申请
func (l *Logic) GetDesignatedFriendsApply(ctx context.Context, req *pbrelation.GetDesignatedFriendsApplyReq) (*pbrelation.GetDesignatedFriendsApplyResp, error) {
	fromUserID := req.GetFromUserID()
	toUserID := req.GetToUserID()
	if fromUserID == "" || toUserID == "" {
		return nil, errx.ArgsError.Wrap("fromUserID and toUserID are required")
	}

	friendReq, err := l.svcCtx.RequestModel.FindFriendRequest(ctx, fromUserID, toUserID)
	if err != nil {
		l.Errorf("find friend request failed, from: %s, to: %s, err: %v", fromUserID, toUserID, err)
		return nil, err
	}

	return &pbrelation.GetDesignatedFriendsApplyResp{
		FriendRequests: []*sdkws.FriendRequest{modelToSDKFriendRequest(friendReq)},
	}, nil
}

// GetIncrementalFriendsApplyTo 增量同步收到的好友申请（DID=userID）
func (l *Logic) GetIncrementalFriendsApplyTo(ctx context.Context, req *pbrelation.GetIncrementalFriendsApplyToReq) (*pbrelation.GetIncrementalFriendsApplyToResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	clientVersion := uint(req.GetVersion())
	clientVersionID := req.GetVersionID()

	verLog, err := l.svcCtx.VersionLogModel.FindChangeLog(ctx, userID, clientVersion, SyncLimit)
	if err != nil {
		l.Errorf("find change log failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	// 空 Logs → 全量同步
	if len(verLog.Logs) == 0 || (clientVersionID != "" && clientVersionID != verLog.ID.Hex()) {
		return l.fullFriendsApplyToResp(ctx, userID)
	}

	// 增量同步：分类处理变更日志
	var (
		deleteIDs []string
		changeIDs []string
		seenChg   = make(map[string]struct{})
		seenDel   = make(map[string]struct{})
	)
	for _, log := range verLog.Logs {
		switch log.State {
		case model.VersionStateDelete:
			if _, ok := seenDel[log.EID]; !ok {
				seenDel[log.EID] = struct{}{}
				deleteIDs = append(deleteIDs, log.EID)
			}
		case model.VersionStateInsert, model.VersionStateUpdate:
			if _, ok := seenChg[log.EID]; !ok {
				seenChg[log.EID] = struct{}{}
				changeIDs = append(changeIDs, log.EID)
			}
		}
	}

	resp := &pbrelation.GetIncrementalFriendsApplyToResp{
		Version:       uint64(verLog.Version),
		VersionID:     userID,
		Full:          false,
		DeleteUserIds: deleteIDs,
	}

	// 拉取变更的好友申请详情
	for _, fromUserID := range changeIDs {
		friendReq, err2 := l.svcCtx.RequestModel.FindFriendRequest(ctx, fromUserID, userID)
		if err2 != nil {
			l.Errorf("find friend request failed, from: %s, to: %s, err: %v", fromUserID, userID, err2)
			continue
		}
		resp.Changes = append(resp.Changes, modelToSDKFriendRequest(friendReq))
	}

	return resp, nil
}

// fullFriendsApplyToResp 构造收到的好友申请全量同步响应
func (l *Logic) fullFriendsApplyToResp(ctx context.Context, userID string) (*pbrelation.GetIncrementalFriendsApplyToResp, error) {
	requests, _, err := l.svcCtx.RequestModel.FindFriendRequestsByTo(ctx, userID, 1, 0)
	if err != nil {
		l.Errorf("find friend requests by to failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	var changes []*sdkws.FriendRequest
	for _, r := range requests {
		changes = append(changes, modelToSDKFriendRequest(r))
	}
	var curVersion uint64
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, userID); err2 == nil && verLog != nil {
		curVersion = uint64(verLog.Version)
	}
	return &pbrelation.GetIncrementalFriendsApplyToResp{
		Version:   curVersion,
		VersionID: userID,
		Full:      true,
		Changes:   changes,
	}, nil
}

// GetIncrementalFriendsApplyFrom 增量同步发出的好友申请（DID=userID）
func (l *Logic) GetIncrementalFriendsApplyFrom(ctx context.Context, req *pbrelation.GetIncrementalFriendsApplyFromReq) (*pbrelation.GetIncrementalFriendsApplyFromResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	clientVersion := uint(req.GetVersion())
	clientVersionID := req.GetVersionID()

	verLog, err := l.svcCtx.VersionLogModel.FindChangeLog(ctx, userID, clientVersion, SyncLimit)
	if err != nil {
		l.Errorf("find change log failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	// 空 Logs → 全量同步
	if len(verLog.Logs) == 0 || (clientVersionID != "" && clientVersionID != verLog.ID.Hex()) {
		return l.fullFriendsApplyFromResp(ctx, userID)
	}

	// 增量同步：分类处理变更日志
	var (
		deleteIDs []string
		changeIDs []string
		seenChg   = make(map[string]struct{})
		seenDel   = make(map[string]struct{})
	)
	for _, log := range verLog.Logs {
		switch log.State {
		case model.VersionStateDelete:
			if _, ok := seenDel[log.EID]; !ok {
				seenDel[log.EID] = struct{}{}
				deleteIDs = append(deleteIDs, log.EID)
			}
		case model.VersionStateInsert, model.VersionStateUpdate:
			if _, ok := seenChg[log.EID]; !ok {
				seenChg[log.EID] = struct{}{}
				changeIDs = append(changeIDs, log.EID)
			}
		}
	}

	resp := &pbrelation.GetIncrementalFriendsApplyFromResp{
		Version:       uint64(verLog.Version),
		VersionID:     userID,
		Full:          false,
		DeleteUserIds: deleteIDs,
	}

	// 拉取变更的好友申请详情
	for _, toUserID := range changeIDs {
		friendReq, err2 := l.svcCtx.RequestModel.FindFriendRequest(ctx, userID, toUserID)
		if err2 != nil {
			l.Errorf("find friend request failed, from: %s, to: %s, err: %v", userID, toUserID, err2)
			continue
		}
		resp.Changes = append(resp.Changes, modelToSDKFriendRequest(friendReq))
	}

	return resp, nil
}

// fullFriendsApplyFromResp 构造发出的好友申请全量同步响应
func (l *Logic) fullFriendsApplyFromResp(ctx context.Context, userID string) (*pbrelation.GetIncrementalFriendsApplyFromResp, error) {
	requests, _, err := l.svcCtx.RequestModel.FindFriendRequestsByFrom(ctx, userID, 1, 0)
	if err != nil {
		l.Errorf("find friend requests by from failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	var changes []*sdkws.FriendRequest
	for _, r := range requests {
		changes = append(changes, modelToSDKFriendRequest(r))
	}
	var curVersion uint64
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, userID); err2 == nil && verLog != nil {
		curVersion = uint64(verLog.Version)
	}
	return &pbrelation.GetIncrementalFriendsApplyFromResp{
		Version:   curVersion,
		VersionID: userID,
		Full:      true,
		Changes:   changes,
	}, nil
}

// modelToSDKFriendRequest 将 model.FriendRequest 转换为 sdkws.FriendRequest
func modelToSDKFriendRequest(r *model.FriendRequest) *sdkws.FriendRequest {
	if r == nil {
		return nil
	}
	return &sdkws.FriendRequest{
		FromUserID:    r.FromUserID,
		FromNickname:  r.FromNickname,
		FromFaceURL:   r.FromFaceURL,
		ToUserID:      r.ToUserID,
		ToNickname:    r.ToNickname,
		ToFaceURL:     r.ToFaceURL,
		HandleResult:  int32(r.HandleResult),
		ReqMsg:        r.ReqMsg,
		CreateTime:    r.CreateTime.Unix(),
		HandlerUserID: r.HandlerUserID,
		HandleMsg:     r.HandleMsg,
		HandleTime:    r.HandleTime.Unix(),
		Ex:            r.Extra,
	}
}

// ==================== 查询方法 ====================

// GetPaginationFriends 分页获取好友列表
func (l *Logic) GetPaginationFriends(ctx context.Context, req *pbrelation.GetPaginationFriendsReq) (*pbrelation.GetPaginationFriendsResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	friends, err := l.svcCtx.FriendModel.FindFriendsByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find friends by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	pagination := req.GetPagination()
	page := int(pagination.GetPageNumber())
	size := int(pagination.GetShowNumber())
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = len(friends)
		if size == 0 {
			size = 50
		}
	}

	start := (page - 1) * size
	end := start + size
	if start > len(friends) {
		start = len(friends)
	}
	if end > len(friends) {
		end = len(friends)
	}

	var friendsInfo []*sdkws.FriendInfo
	for _, f := range friends[start:end] {
		friendsInfo = append(friendsInfo, modelToFriendInfo(f))
	}

	return &pbrelation.GetPaginationFriendsResp{
		FriendsInfo: friendsInfo,
		Total:       int32(len(friends)),
	}, nil
}

// GetDesignatedFriends 获取指定好友信息
func (l *Logic) GetDesignatedFriends(ctx context.Context, req *pbrelation.GetDesignatedFriendsReq) (*pbrelation.GetDesignatedFriendsResp, error) {
	ownerUserID := req.GetOwnerUserID()
	friendUserIDs := req.GetFriendUserIDs()
	if ownerUserID == "" || len(friendUserIDs) == 0 {
		return &pbrelation.GetDesignatedFriendsResp{}, errx.ArgsError.Wrap("ownerUserID or friendUserIDs is required")
	}

	friends, err := l.svcCtx.FriendModel.FindFriendsByIDs(ctx, ownerUserID, friendUserIDs)
	if err != nil {
		l.Errorf("find friends by ids failed, owner: %s, err: %v", ownerUserID, err)
		return nil, err
	}

	var friendsInfo []*sdkws.FriendInfo
	for _, f := range friends {
		friendsInfo = append(friendsInfo, modelToFriendInfo(f))
	}

	return &pbrelation.GetDesignatedFriendsResp{
		FriendsInfo: friendsInfo,
	}, nil
}

// GetFriendIDs 获取好友ID列表
func (l *Logic) GetFriendIDs(ctx context.Context, req *pbrelation.GetFriendIDsReq) (*pbrelation.GetFriendIDsResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	friends, err := l.svcCtx.FriendModel.FindFriendsByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find friends by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	friendIDs := make([]string, 0, len(friends))
	for _, f := range friends {
		friendIDs = append(friendIDs, f.FriendUserID)
	}

	return &pbrelation.GetFriendIDsResp{
		FriendIDs: friendIDs,
	}, nil
}

// GetSpecifiedFriendsInfo 获取指定用户的详细信息（好友信息+用户信息+黑名单信息）
func (l *Logic) GetSpecifiedFriendsInfo(ctx context.Context, req *pbrelation.GetSpecifiedFriendsInfoReq) (*pbrelation.GetSpecifiedFriendsInfoResp, error) {
	ownerUserID := req.GetOwnerUserID()
	userIDList := req.GetUserIDList()
	if ownerUserID == "" || len(userIDList) == 0 {
		return &pbrelation.GetSpecifiedFriendsInfoResp{}, nil
	}

	var infos []*pbrelation.GetSpecifiedFriendsInfoInfo
	for _, targetUserID := range userIDList {
		info := &pbrelation.GetSpecifiedFriendsInfoInfo{}

		// 获取用户信息
		userInfo, err := l.svcCtx.UserService.GetUserInfo(ctx, targetUserID)
		if err != nil {
			l.Errorf("get user info failed, userID: %s, err: %v", targetUserID, err)
		} else {
			info.UserInfo = userInfo
		}

		// 获取好友信息
		friend, err := l.svcCtx.FriendModel.FindFriend(ctx, ownerUserID, targetUserID)
		if err != nil && !isFriendNotFound(err) {
			l.Errorf("find friend failed, owner: %s, friend: %s, err: %v", ownerUserID, targetUserID, err)
		} else if friend != nil {
			info.FriendInfo = modelToFriendInfo(friend)
		}

		// 获取黑名单信息
		black, err := l.svcCtx.FriendModel.FindBlack(ctx, ownerUserID, targetUserID)
		if err != nil && !isBlackNotFound(err) {
			l.Errorf("find black failed, owner: %s, black: %s, err: %v", ownerUserID, targetUserID, err)
		} else if black != nil {
			info.BlackInfo = modelToBlackInfo(black)
		}

		infos = append(infos, info)
	}

	return &pbrelation.GetSpecifiedFriendsInfoResp{
		Infos: infos,
	}, nil
}

// GetSpecifiedBlacks 获取指定黑名单信息
func (l *Logic) GetSpecifiedBlacks(ctx context.Context, req *pbrelation.GetSpecifiedBlacksReq) (*pbrelation.GetSpecifiedBlacksResp, error) {
	ownerUserID := req.GetOwnerUserID()
	userIDList := req.GetUserIDList()
	if ownerUserID == "" || len(userIDList) == 0 {
		return &pbrelation.GetSpecifiedBlacksResp{}, nil
	}

	var blacks []*sdkws.BlackInfo
	for _, targetUserID := range userIDList {
		black, err := l.svcCtx.FriendModel.FindBlack(ctx, ownerUserID, targetUserID)
		if err != nil {
			if isBlackNotFound(err) {
				continue
			}
			l.Errorf("find black failed, owner: %s, black: %s, err: %v", ownerUserID, targetUserID, err)
			continue
		}
		blacks = append(blacks, modelToBlackInfo(black))
	}

	return &pbrelation.GetSpecifiedBlacksResp{
		Blacks: blacks,
		Total:  int32(len(blacks)),
	}, nil
}

// GetPaginationBlacks 分页获取黑名单列表
func (l *Logic) GetPaginationBlacks(ctx context.Context, req *pbrelation.GetPaginationBlacksReq) (*pbrelation.GetPaginationBlacksResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	blacks, err := l.svcCtx.FriendModel.FindBlacksByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find blacks by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	pagination := req.GetPagination()
	page := int(pagination.GetPageNumber())
	size := int(pagination.GetShowNumber())
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = len(blacks)
		if size == 0 {
			size = 50
		}
	}

	start := (page - 1) * size
	end := start + size
	if start > len(blacks) {
		start = len(blacks)
	}
	if end > len(blacks) {
		end = len(blacks)
	}

	var blackInfos []*sdkws.BlackInfo
	for _, b := range blacks[start:end] {
		blackInfos = append(blackInfos, modelToBlackInfo(b))
	}

	return &pbrelation.GetPaginationBlacksResp{
		Blacks: blackInfos,
		Total:  int32(len(blacks)),
	}, nil
}

// IsFriend 验证好友关系
func (l *Logic) IsFriend(ctx context.Context, req *pbrelation.IsFriendReq) (*pbrelation.IsFriendResp, error) {
	userID1 := req.GetUserID1()
	userID2 := req.GetUserID2()
	if userID1 == "" || userID2 == "" {
		return nil, errx.ArgsError.Wrap("userID1 and userID2 are required")
	}

	inUser1Friends, err := l.svcCtx.FriendModel.IsFriend(ctx, userID1, userID2)
	if err != nil {
		l.Errorf("check friend failed, user1: %s, user2: %s, err: %v", userID1, userID2, err)
		return nil, err
	}

	inUser2Friends, err := l.svcCtx.FriendModel.IsFriend(ctx, userID2, userID1)
	if err != nil {
		l.Errorf("check friend failed, user1: %s, user2: %s, err: %v", userID2, userID1, err)
		return nil, err
	}

	return &pbrelation.IsFriendResp{
		InUser1Friends: inUser1Friends,
		InUser2Friends: inUser2Friends,
	}, nil
}

// IsBlack 验证黑名单关系
func (l *Logic) IsBlack(ctx context.Context, req *pbrelation.IsBlackReq) (*pbrelation.IsBlackResp, error) {
	userID1 := req.GetUserID1()
	userID2 := req.GetUserID2()
	if userID1 == "" || userID2 == "" {
		return nil, errx.ArgsError.Wrap("userID1 and userID2 are required")
	}

	inUser1Blacks, err := l.svcCtx.FriendModel.IsBlack(ctx, userID1, userID2)
	if err != nil {
		l.Errorf("check black failed, owner: %s, target: %s, err: %v", userID1, userID2, err)
		return nil, err
	}

	inUser2Blacks, err := l.svcCtx.FriendModel.IsBlack(ctx, userID2, userID1)
	if err != nil {
		l.Errorf("check black failed, owner: %s, target: %s, err: %v", userID2, userID1, err)
		return nil, err
	}

	return &pbrelation.IsBlackResp{
		InUser1Blacks: inUser1Blacks,
		InUser2Blacks: inUser2Blacks,
	}, nil
}

// GetFriendInfo 获取好友信息（简化版 FriendInfoOnly）
func (l *Logic) GetFriendInfo(ctx context.Context, req *pbrelation.GetFriendInfoReq) (*pbrelation.GetFriendInfoResp, error) {
	ownerUserID := req.GetOwnerUserID()
	friendUserIDs := req.GetFriendUserIDs()
	if ownerUserID == "" || len(friendUserIDs) == 0 {
		return &pbrelation.GetFriendInfoResp{}, nil
	}

	friends, err := l.svcCtx.FriendModel.FindFriendsByIDs(ctx, ownerUserID, friendUserIDs)
	if err != nil {
		l.Errorf("find friends by ids failed, owner: %s, err: %v", ownerUserID, err)
		return nil, err
	}

	var friendInfos []*pbrelation.FriendInfoOnly
	for _, f := range friends {
		friendInfos = append(friendInfos, &pbrelation.FriendInfoOnly{
			OwnerUserID:    f.OwnerUserID,
			FriendUserID:   f.FriendUserID,
			Remark:         f.Remark,
			CreateTime:     f.CreateTime.Unix(),
			AddSource:      int32(f.AddSource),
			OperatorUserID: f.OperatorUserID,
			Ex:             f.Extra,
			IsPinned:       f.IsPinned,
		})
	}

	return &pbrelation.GetFriendInfoResp{
		FriendInfos: friendInfos,
	}, nil
}

// ==================== 增量同步方法 ====================

// GetIncrementalFriends 获取增量好友列表。DID=userID。
// 使用 FindChangeLog 拉取增量变更。空 Logs → 全量同步。
func (l *Logic) GetIncrementalFriends(ctx context.Context, req *pbrelation.GetIncrementalFriendsReq) (*pbrelation.GetIncrementalFriendsResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	clientVersion := uint(req.GetVersion())
	clientVersionID := req.GetVersionID()

	verLog, err := l.svcCtx.VersionLogModel.FindChangeLog(ctx, userID, clientVersion, SyncLimit)
	if err != nil {
		l.Errorf("find change log failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	// 空 Logs → 全量同步
	if len(verLog.Logs) == 0 || (clientVersionID != "" && clientVersionID != verLog.ID.Hex()) {
		return l.fullFriendsResp(ctx, userID)
	}

	// 增量同步：分类处理变更日志
	c := model.ClassifyIncrementalLogs(verLog.Logs)

	resp := &pbrelation.GetIncrementalFriendsResp{
		Version:   uint64(verLog.Version),
		VersionID: userID,
		Full:      false,
		Delete:    c.DeleteIDs,
	}
	if c.SortChanged {
		resp.SortVersion = c.SortVersion
	}

	// 拉取新增/更新好友的详情
	fetchIDs := append(append([]string{}, c.InsertIDs...), c.UpdateIDs...)
	if len(fetchIDs) > 0 {
		friends, err2 := l.svcCtx.FriendModel.FindFriendsByIDs(ctx, userID, fetchIDs)
		if err2 != nil {
			l.Errorf("find friends by ids failed, userID: %s, ids: %v, err: %v", userID, fetchIDs, err2)
			return nil, err2
		}
		friendMap := make(map[string]*model.Friend, len(friends))
		for _, f := range friends {
			friendMap[f.FriendUserID] = f
		}
		for _, id := range c.InsertIDs {
			if f, ok := friendMap[id]; ok {
				resp.Insert = append(resp.Insert, modelToFriendInfo(f))
			}
		}
		for _, id := range c.UpdateIDs {
			if f, ok := friendMap[id]; ok {
				resp.Update = append(resp.Update, modelToFriendInfo(f))
			}
		}
	}

	return resp, nil
}

// fullFriendsResp 构造好友全量同步响应
func (l *Logic) fullFriendsResp(ctx context.Context, userID string) (*pbrelation.GetIncrementalFriendsResp, error) {
	friends, err := l.svcCtx.FriendModel.FindFriendsByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find friends by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	inserts := make([]*sdkws.FriendInfo, 0, len(friends))
	for _, f := range friends {
		inserts = append(inserts, modelToFriendInfo(f))
	}
	var curVersion uint64
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, userID); err2 == nil && verLog != nil {
		curVersion = uint64(verLog.Version)
	}
	return &pbrelation.GetIncrementalFriendsResp{
		Version:     curVersion,
		VersionID:   userID,
		Full:        true,
		Insert:      inserts,
		SortVersion: curVersion,
	}, nil
}

// GetIncrementalBlacks 获取增量黑名单列表。DID=userID。
func (l *Logic) GetIncrementalBlacks(ctx context.Context, req *pbrelation.GetIncrementalBlacksReq) (*pbrelation.GetIncrementalBlacksResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	clientVersion := uint(req.GetVersion())
	clientVersionID := req.GetVersionID()

	verLog, err := l.svcCtx.VersionLogModel.FindChangeLog(ctx, userID, clientVersion, SyncLimit)
	if err != nil {
		l.Errorf("find change log failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	// 空 Logs → 全量同步
	if len(verLog.Logs) == 0 || (clientVersionID != "" && clientVersionID != verLog.ID.Hex()) {
		return l.fullBlacksResp(ctx, userID)
	}

	// 增量同步：分类处理变更日志
	c := model.ClassifyIncrementalLogs(verLog.Logs)

	resp := &pbrelation.GetIncrementalBlacksResp{
		Version:   uint64(verLog.Version),
		VersionID: userID,
		Full:      false,
		Delete:    c.DeleteIDs,
	}

	// 拉取新增/更新黑名单的详情
	fetchIDs := append(append([]string{}, c.InsertIDs...), c.UpdateIDs...)
	insertSet := make(map[string]struct{}, len(c.InsertIDs))
	for _, id := range c.InsertIDs {
		insertSet[id] = struct{}{}
	}
	for _, blackUserID := range fetchIDs {
		black, err2 := l.svcCtx.FriendModel.FindBlack(ctx, userID, blackUserID)
		if err2 != nil {
			l.Errorf("find black failed, owner: %s, black: %s, err: %v", userID, blackUserID, err2)
			continue
		}
		if black != nil {
			if _, isInsert := insertSet[blackUserID]; isInsert {
				resp.Insert = append(resp.Insert, modelToBlackInfo(black))
			} else {
				resp.Update = append(resp.Update, modelToBlackInfo(black))
			}
		}
	}

	return resp, nil
}

// fullBlacksResp 构造黑名单全量同步响应
func (l *Logic) fullBlacksResp(ctx context.Context, userID string) (*pbrelation.GetIncrementalBlacksResp, error) {
	blacks, err := l.svcCtx.FriendModel.FindBlacksByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find blacks by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	inserts := make([]*sdkws.BlackInfo, 0, len(blacks))
	for _, b := range blacks {
		inserts = append(inserts, modelToBlackInfo(b))
	}
	var curVersion uint64
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, userID); err2 == nil && verLog != nil {
		curVersion = uint64(verLog.Version)
	}
	return &pbrelation.GetIncrementalBlacksResp{
		Version:   curVersion,
		VersionID: userID,
		Full:      true,
		Insert:    inserts,
	}, nil
}

// GetFullFriendUserIDs 返回全量好友ID列表 + FNV-1a 哈希比对。
func (l *Logic) GetFullFriendUserIDs(ctx context.Context, req *pbrelation.GetFullFriendUserIDsReq) (*pbrelation.GetFullFriendUserIDsResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	friends, err := l.svcCtx.FriendModel.FindFriendsByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find friends by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	userIDs := make([]string, 0, len(friends))
	for _, f := range friends {
		userIDs = append(userIDs, f.FriendUserID)
	}
	sort.Strings(userIDs)
	curHash := hashIDs(userIDs)

	resp := &pbrelation.GetFullFriendUserIDsResp{
		Equal:   req.GetIdHash() != 0 && req.GetIdHash() == curHash,
		UserIDs: userIDs,
	}
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, userID); err2 == nil && verLog != nil {
		resp.VersionID = verLog.ID.Hex()
		resp.Version = uint64(verLog.Version)
	} else if err2 != nil {
		l.Errorf("get version log failed, userID: %s, err: %v", userID, err2)
	}
	return resp, nil
}

// ==================== 通知方法 ====================

// NotificationUserInfoUpdate 用户信息变更通知，更新好友表中的 friendNickname 和 friendFaceURL
func (l *Logic) NotificationUserInfoUpdate(ctx context.Context, req *pbrelation.NotificationUserInfoUpdateReq) (*pbrelation.NotificationUserInfoUpdateResp, error) {
	// model.Friend 不直接存储 friendNickname/friendFaceURL，好友信息通过 FriendUser.UserInfo 关联获取。
	// 此处仅返回成功，实际用户信息更新由 UserService 处理并触发缓存失效。
	return &pbrelation.NotificationUserInfoUpdateResp{}, nil
}

// ==================== 辅助函数 ====================

func isFriendNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "friend not found")
}

func isBlackNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "black not found")
}
