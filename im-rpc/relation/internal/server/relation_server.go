package server

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/relation/internal/logic"
	"github.com/PaperMan11/goim/im-rpc/relation/internal/svc"
	pbrelation "github.com/PaperMan11/goim/pkg/protocol/relation"
)

type RelationServer struct {
	svcCtx *svc.ServiceContext
	pbrelation.UnimplementedFriendServer
}

func NewRelationServer(svcCtx *svc.ServiceContext) *RelationServer {
	return &RelationServer{svcCtx: svcCtx}
}

func (s *RelationServer) ApplyToAddFriend(ctx context.Context, req *pbrelation.ApplyToAddFriendReq) (*pbrelation.ApplyToAddFriendResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).ApplyToAddFriend(ctx, req)
}

func (s *RelationServer) GetPaginationFriendsApplyTo(ctx context.Context, req *pbrelation.GetPaginationFriendsApplyToReq) (*pbrelation.GetPaginationFriendsApplyToResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetPaginationFriendsApplyTo(ctx, req)
}

func (s *RelationServer) GetPaginationFriendsApplyFrom(ctx context.Context, req *pbrelation.GetPaginationFriendsApplyFromReq) (*pbrelation.GetPaginationFriendsApplyFromResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetPaginationFriendsApplyFrom(ctx, req)
}

func (s *RelationServer) GetSelfUnhandledApplyCount(ctx context.Context, req *pbrelation.GetSelfUnhandledApplyCountReq) (*pbrelation.GetSelfUnhandledApplyCountResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetSelfUnhandledApplyCount(ctx, req)
}

func (s *RelationServer) GetDesignatedFriendsApply(ctx context.Context, req *pbrelation.GetDesignatedFriendsApplyReq) (*pbrelation.GetDesignatedFriendsApplyResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetDesignatedFriendsApply(ctx, req)
}

func (s *RelationServer) GetIncrementalFriendsApplyTo(ctx context.Context, req *pbrelation.GetIncrementalFriendsApplyToReq) (*pbrelation.GetIncrementalFriendsApplyToResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetIncrementalFriendsApplyTo(ctx, req)
}

func (s *RelationServer) GetIncrementalFriendsApplyFrom(ctx context.Context, req *pbrelation.GetIncrementalFriendsApplyFromReq) (*pbrelation.GetIncrementalFriendsApplyFromResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetIncrementalFriendsApplyFrom(ctx, req)
}

func (s *RelationServer) AddBlack(ctx context.Context, req *pbrelation.AddBlackReq) (*pbrelation.AddBlackResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).AddBlack(ctx, req)
}

func (s *RelationServer) RemoveBlack(ctx context.Context, req *pbrelation.RemoveBlackReq) (*pbrelation.RemoveBlackResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).RemoveBlack(ctx, req)
}

func (s *RelationServer) IsFriend(ctx context.Context, req *pbrelation.IsFriendReq) (*pbrelation.IsFriendResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).IsFriend(ctx, req)
}

func (s *RelationServer) IsBlack(ctx context.Context, req *pbrelation.IsBlackReq) (*pbrelation.IsBlackResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).IsBlack(ctx, req)
}

func (s *RelationServer) GetPaginationBlacks(ctx context.Context, req *pbrelation.GetPaginationBlacksReq) (*pbrelation.GetPaginationBlacksResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetPaginationBlacks(ctx, req)
}

func (s *RelationServer) GetSpecifiedBlacks(ctx context.Context, req *pbrelation.GetSpecifiedBlacksReq) (*pbrelation.GetSpecifiedBlacksResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetSpecifiedBlacks(ctx, req)
}

func (s *RelationServer) DeleteFriend(ctx context.Context, req *pbrelation.DeleteFriendReq) (*pbrelation.DeleteFriendResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).DeleteFriend(ctx, req)
}

func (s *RelationServer) RespondFriendApply(ctx context.Context, req *pbrelation.RespondFriendApplyReq) (*pbrelation.RespondFriendApplyResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).RespondFriendApply(ctx, req)
}

func (s *RelationServer) UpdateFriends(ctx context.Context, req *pbrelation.UpdateFriendsReq) (*pbrelation.UpdateFriendsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).UpdateFriends(ctx, req)
}

func (s *RelationServer) SetFriendRemark(ctx context.Context, req *pbrelation.SetFriendRemarkReq) (*pbrelation.SetFriendRemarkResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetFriendRemark(ctx, req)
}

func (s *RelationServer) ImportFriends(ctx context.Context, req *pbrelation.ImportFriendReq) (*pbrelation.ImportFriendResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).ImportFriends(ctx, req)
}

func (s *RelationServer) GetDesignatedFriends(ctx context.Context, req *pbrelation.GetDesignatedFriendsReq) (*pbrelation.GetDesignatedFriendsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetDesignatedFriends(ctx, req)
}

func (s *RelationServer) GetPaginationFriends(ctx context.Context, req *pbrelation.GetPaginationFriendsReq) (*pbrelation.GetPaginationFriendsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetPaginationFriends(ctx, req)
}

func (s *RelationServer) GetFriendIDs(ctx context.Context, req *pbrelation.GetFriendIDsReq) (*pbrelation.GetFriendIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetFriendIDs(ctx, req)
}

func (s *RelationServer) GetSpecifiedFriendsInfo(ctx context.Context, req *pbrelation.GetSpecifiedFriendsInfoReq) (*pbrelation.GetSpecifiedFriendsInfoResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetSpecifiedFriendsInfo(ctx, req)
}

func (s *RelationServer) GetIncrementalFriends(ctx context.Context, req *pbrelation.GetIncrementalFriendsReq) (*pbrelation.GetIncrementalFriendsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetIncrementalFriends(ctx, req)
}

func (s *RelationServer) GetIncrementalBlacks(ctx context.Context, req *pbrelation.GetIncrementalBlacksReq) (*pbrelation.GetIncrementalBlacksResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetIncrementalBlacks(ctx, req)
}

func (s *RelationServer) GetFullFriendUserIDs(ctx context.Context, req *pbrelation.GetFullFriendUserIDsReq) (*pbrelation.GetFullFriendUserIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetFullFriendUserIDs(ctx, req)
}

func (s *RelationServer) NotificationUserInfoUpdate(ctx context.Context, req *pbrelation.NotificationUserInfoUpdateReq) (*pbrelation.NotificationUserInfoUpdateResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).NotificationUserInfoUpdate(ctx, req)
}

func (s *RelationServer) GetFriendInfo(ctx context.Context, req *pbrelation.GetFriendInfoReq) (*pbrelation.GetFriendInfoResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetFriendInfo(ctx, req)
}
