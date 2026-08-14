package server

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/group/internal/logic"
	"github.com/PaperMan11/goim/im-rpc/group/internal/svc"
	pbgroup "github.com/PaperMan11/goim/pkg/protocol/group"
)

type GroupServer struct {
	svcCtx *svc.ServiceContext
	pbgroup.UnimplementedGroupServer
}

func NewGroupServer(svcCtx *svc.ServiceContext) *GroupServer {
	return &GroupServer{svcCtx: svcCtx}
}

func (s *GroupServer) CreateGroup(ctx context.Context, req *pbgroup.CreateGroupReq) (*pbgroup.CreateGroupResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).CreateGroup(ctx, req)
}

func (s *GroupServer) JoinGroup(ctx context.Context, req *pbgroup.JoinGroupReq) (*pbgroup.JoinGroupResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).JoinGroup(ctx, req)
}

func (s *GroupServer) QuitGroup(ctx context.Context, req *pbgroup.QuitGroupReq) (*pbgroup.QuitGroupResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).QuitGroup(ctx, req)
}

func (s *GroupServer) GetGroupsInfo(ctx context.Context, req *pbgroup.GetGroupsInfoReq) (*pbgroup.GetGroupsInfoResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupsInfo(ctx, req)
}

func (s *GroupServer) SetGroupInfo(ctx context.Context, req *pbgroup.SetGroupInfoReq) (*pbgroup.SetGroupInfoResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetGroupInfo(ctx, req)
}

func (s *GroupServer) SetGroupInfoEx(ctx context.Context, req *pbgroup.SetGroupInfoExReq) (*pbgroup.SetGroupInfoExResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetGroupInfoEx(ctx, req)
}

func (s *GroupServer) GetGroupApplicationList(ctx context.Context, req *pbgroup.GetGroupApplicationListReq) (*pbgroup.GetGroupApplicationListResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupApplicationList(ctx, req)
}

func (s *GroupServer) GetGroupApplicationUnhandledCount(ctx context.Context, req *pbgroup.GetGroupApplicationUnhandledCountReq) (*pbgroup.GetGroupApplicationUnhandledCountResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupApplicationUnhandledCount(ctx, req)
}

func (s *GroupServer) GetUserReqApplicationList(ctx context.Context, req *pbgroup.GetUserReqApplicationListReq) (*pbgroup.GetUserReqApplicationListResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetUserReqApplicationList(ctx, req)
}

func (s *GroupServer) GetGroupUsersReqApplicationList(ctx context.Context, req *pbgroup.GetGroupUsersReqApplicationListReq) (*pbgroup.GetGroupUsersReqApplicationListResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupUsersReqApplicationList(ctx, req)
}

func (s *GroupServer) GetSpecifiedUserGroupRequestInfo(ctx context.Context, req *pbgroup.GetSpecifiedUserGroupRequestInfoReq) (*pbgroup.GetSpecifiedUserGroupRequestInfoResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetSpecifiedUserGroupRequestInfo(ctx, req)
}

func (s *GroupServer) TransferGroupOwner(ctx context.Context, req *pbgroup.TransferGroupOwnerReq) (*pbgroup.TransferGroupOwnerResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).TransferGroupOwner(ctx, req)
}

func (s *GroupServer) GroupApplicationResponse(ctx context.Context, req *pbgroup.GroupApplicationResponseReq) (*pbgroup.GroupApplicationResponseResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GroupApplicationResponse(ctx, req)
}

func (s *GroupServer) GetGroupMemberList(ctx context.Context, req *pbgroup.GetGroupMemberListReq) (*pbgroup.GetGroupMemberListResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupMemberList(ctx, req)
}

func (s *GroupServer) GetGroupMembersInfo(ctx context.Context, req *pbgroup.GetGroupMembersInfoReq) (*pbgroup.GetGroupMembersInfoResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupMembersInfo(ctx, req)
}

func (s *GroupServer) KickGroupMember(ctx context.Context, req *pbgroup.KickGroupMemberReq) (*pbgroup.KickGroupMemberResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).KickGroupMember(ctx, req)
}

func (s *GroupServer) GetJoinedGroupList(ctx context.Context, req *pbgroup.GetJoinedGroupListReq) (*pbgroup.GetJoinedGroupListResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetJoinedGroupList(ctx, req)
}

func (s *GroupServer) InviteUserToGroup(ctx context.Context, req *pbgroup.InviteUserToGroupReq) (*pbgroup.InviteUserToGroupResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).InviteUserToGroup(ctx, req)
}

func (s *GroupServer) GetGroups(ctx context.Context, req *pbgroup.GetGroupsReq) (*pbgroup.GetGroupsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroups(ctx, req)
}

func (s *GroupServer) GetGroupMembersCMS(ctx context.Context, req *pbgroup.GetGroupMembersCMSReq) (*pbgroup.GetGroupMembersCMSResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupMembersCMS(ctx, req)
}

func (s *GroupServer) DismissGroup(ctx context.Context, req *pbgroup.DismissGroupReq) (*pbgroup.DismissGroupResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).DismissGroup(ctx, req)
}

func (s *GroupServer) MuteGroupMember(ctx context.Context, req *pbgroup.MuteGroupMemberReq) (*pbgroup.MuteGroupMemberResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).MuteGroupMember(ctx, req)
}

func (s *GroupServer) CancelMuteGroupMember(ctx context.Context, req *pbgroup.CancelMuteGroupMemberReq) (*pbgroup.CancelMuteGroupMemberResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).CancelMuteGroupMember(ctx, req)
}

func (s *GroupServer) MuteGroup(ctx context.Context, req *pbgroup.MuteGroupReq) (*pbgroup.MuteGroupResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).MuteGroup(ctx, req)
}

func (s *GroupServer) CancelMuteGroup(ctx context.Context, req *pbgroup.CancelMuteGroupReq) (*pbgroup.CancelMuteGroupResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).CancelMuteGroup(ctx, req)
}

func (s *GroupServer) SetGroupMemberInfo(ctx context.Context, req *pbgroup.SetGroupMemberInfoReq) (*pbgroup.SetGroupMemberInfoResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetGroupMemberInfo(ctx, req)
}

func (s *GroupServer) GetGroupAbstractInfo(ctx context.Context, req *pbgroup.GetGroupAbstractInfoReq) (*pbgroup.GetGroupAbstractInfoResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupAbstractInfo(ctx, req)
}

func (s *GroupServer) GetUserInGroupMembers(ctx context.Context, req *pbgroup.GetUserInGroupMembersReq) (*pbgroup.GetUserInGroupMembersResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetUserInGroupMembers(ctx, req)
}

func (s *GroupServer) GetGroupMemberUserIDs(ctx context.Context, req *pbgroup.GetGroupMemberUserIDsReq) (*pbgroup.GetGroupMemberUserIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupMemberUserIDs(ctx, req)
}

func (s *GroupServer) GetGroupMemberRoleLevel(ctx context.Context, req *pbgroup.GetGroupMemberRoleLevelReq) (*pbgroup.GetGroupMemberRoleLevelResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupMemberRoleLevel(ctx, req)
}

func (s *GroupServer) GetGroupInfoCache(ctx context.Context, req *pbgroup.GetGroupInfoCacheReq) (*pbgroup.GetGroupInfoCacheResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupInfoCache(ctx, req)
}

func (s *GroupServer) GetGroupMemberCache(ctx context.Context, req *pbgroup.GetGroupMemberCacheReq) (*pbgroup.GetGroupMemberCacheResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGroupMemberCache(ctx, req)
}

func (s *GroupServer) GroupCreateCount(ctx context.Context, req *pbgroup.GroupCreateCountReq) (*pbgroup.GroupCreateCountResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GroupCreateCount(ctx, req)
}

func (s *GroupServer) NotificationUserInfoUpdate(ctx context.Context, req *pbgroup.NotificationUserInfoUpdateReq) (*pbgroup.NotificationUserInfoUpdateResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).NotificationUserInfoUpdate(ctx, req)
}

func (s *GroupServer) GetIncrementalGroupMember(ctx context.Context, req *pbgroup.GetIncrementalGroupMemberReq) (*pbgroup.GetIncrementalGroupMemberResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetIncrementalGroupMember(ctx, req)
}

func (s *GroupServer) BatchGetIncrementalGroupMember(ctx context.Context, req *pbgroup.BatchGetIncrementalGroupMemberReq) (*pbgroup.BatchGetIncrementalGroupMemberResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).BatchGetIncrementalGroupMember(ctx, req)
}

func (s *GroupServer) GetIncrementalJoinGroup(ctx context.Context, req *pbgroup.GetIncrementalJoinGroupReq) (*pbgroup.GetIncrementalJoinGroupResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetIncrementalJoinGroup(ctx, req)
}

func (s *GroupServer) GetFullGroupMemberUserIDs(ctx context.Context, req *pbgroup.GetFullGroupMemberUserIDsReq) (*pbgroup.GetFullGroupMemberUserIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetFullGroupMemberUserIDs(ctx, req)
}

func (s *GroupServer) GetFullJoinGroupIDs(ctx context.Context, req *pbgroup.GetFullJoinGroupIDsReq) (*pbgroup.GetFullJoinGroupIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetFullJoinGroupIDs(ctx, req)
}