package groupservice

import (
	"context"

	pbgroup "github.com/PaperMan11/goim/pkg/protocol/group"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type GroupService interface {
	// 创建群组
	CreateGroup(ctx context.Context, in *pbgroup.CreateGroupReq, opts ...grpc.CallOption) (*pbgroup.CreateGroupResp, error)
	// 申请加入群组
	JoinGroup(ctx context.Context, in *pbgroup.JoinGroupReq, opts ...grpc.CallOption) (*pbgroup.JoinGroupResp, error)
	// 退出群组
	QuitGroup(ctx context.Context, in *pbgroup.QuitGroupReq, opts ...grpc.CallOption) (*pbgroup.QuitGroupResp, error)
	// 获取指定群组信息
	GetGroupsInfo(ctx context.Context, in *pbgroup.GetGroupsInfoReq, opts ...grpc.CallOption) (*pbgroup.GetGroupsInfoResp, error)
	// 设置群组信息
	SetGroupInfo(ctx context.Context, in *pbgroup.SetGroupInfoReq, opts ...grpc.CallOption) (*pbgroup.SetGroupInfoResp, error)
	// 设置群组信息(扩展)
	SetGroupInfoEx(ctx context.Context, in *pbgroup.SetGroupInfoExReq, opts ...grpc.CallOption) (*pbgroup.SetGroupInfoExResp, error)
	// 获取群申请列表(管理员/群主视角)
	GetGroupApplicationList(ctx context.Context, in *pbgroup.GetGroupApplicationListReq, opts ...grpc.CallOption) (*pbgroup.GetGroupApplicationListResp, error)
	// 获取未处理群申请数量
	GetGroupApplicationUnhandledCount(ctx context.Context, in *pbgroup.GetGroupApplicationUnhandledCountReq, opts ...grpc.CallOption) (*pbgroup.GetGroupApplicationUnhandledCountResp, error)
	// 获取用户申请加入群列表
	GetUserReqApplicationList(ctx context.Context, in *pbgroup.GetUserReqApplicationListReq, opts ...grpc.CallOption) (*pbgroup.GetUserReqApplicationListResp, error)
	// 获取群用户申请列表
	GetGroupUsersReqApplicationList(ctx context.Context, in *pbgroup.GetGroupUsersReqApplicationListReq, opts ...grpc.CallOption) (*pbgroup.GetGroupUsersReqApplicationListResp, error)
	// 获取指定用户的群申请信息
	GetSpecifiedUserGroupRequestInfo(ctx context.Context, in *pbgroup.GetSpecifiedUserGroupRequestInfoReq, opts ...grpc.CallOption) (*pbgroup.GetSpecifiedUserGroupRequestInfoResp, error)
	// 转让群主
	TransferGroupOwner(ctx context.Context, in *pbgroup.TransferGroupOwnerReq, opts ...grpc.CallOption) (*pbgroup.TransferGroupOwnerResp, error)
	// 处理群申请(管理员/群主)
	GroupApplicationResponse(ctx context.Context, in *pbgroup.GroupApplicationResponseReq, opts ...grpc.CallOption) (*pbgroup.GroupApplicationResponseResp, error)
	// 获取群成员列表
	GetGroupMemberList(ctx context.Context, in *pbgroup.GetGroupMemberListReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberListResp, error)
	// 获取指定群成员信息
	GetGroupMembersInfo(ctx context.Context, in *pbgroup.GetGroupMembersInfoReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMembersInfoResp, error)
	// 踢出群成员
	KickGroupMember(ctx context.Context, in *pbgroup.KickGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.KickGroupMemberResp, error)
	// 获取用户已加入的群组列表
	GetJoinedGroupList(ctx context.Context, in *pbgroup.GetJoinedGroupListReq, opts ...grpc.CallOption) (*pbgroup.GetJoinedGroupListResp, error)
	// 邀请用户入群
	InviteUserToGroup(ctx context.Context, in *pbgroup.InviteUserToGroupReq, opts ...grpc.CallOption) (*pbgroup.InviteUserToGroupResp, error)
	// CMS获取群组列表
	GetGroups(ctx context.Context, in *pbgroup.GetGroupsReq, opts ...grpc.CallOption) (*pbgroup.GetGroupsResp, error)
	// CMS获取群成员列表
	GetGroupMembersCMS(ctx context.Context, in *pbgroup.GetGroupMembersCMSReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMembersCMSResp, error)
	// 解散群组
	DismissGroup(ctx context.Context, in *pbgroup.DismissGroupReq, opts ...grpc.CallOption) (*pbgroup.DismissGroupResp, error)
	// 禁言群成员
	MuteGroupMember(ctx context.Context, in *pbgroup.MuteGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.MuteGroupMemberResp, error)
	// 取消禁言群成员
	CancelMuteGroupMember(ctx context.Context, in *pbgroup.CancelMuteGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.CancelMuteGroupMemberResp, error)
	// 禁言整个群
	MuteGroup(ctx context.Context, in *pbgroup.MuteGroupReq, opts ...grpc.CallOption) (*pbgroup.MuteGroupResp, error)
	// 取消群禁言
	CancelMuteGroup(ctx context.Context, in *pbgroup.CancelMuteGroupReq, opts ...grpc.CallOption) (*pbgroup.CancelMuteGroupResp, error)
	// 设置群成员信息
	SetGroupMemberInfo(ctx context.Context, in *pbgroup.SetGroupMemberInfoReq, opts ...grpc.CallOption) (*pbgroup.SetGroupMemberInfoResp, error)
	// 获取群组摘要信息
	GetGroupAbstractInfo(ctx context.Context, in *pbgroup.GetGroupAbstractInfoReq, opts ...grpc.CallOption) (*pbgroup.GetGroupAbstractInfoResp, error)
	// 获取用户在群中的成员信息
	GetUserInGroupMembers(ctx context.Context, in *pbgroup.GetUserInGroupMembersReq, opts ...grpc.CallOption) (*pbgroup.GetUserInGroupMembersResp, error)
	// 获取群成员用户ID列表
	GetGroupMemberUserIDs(ctx context.Context, in *pbgroup.GetGroupMemberUserIDsReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberUserIDsResp, error)
	// 获取指定角色等级的群成员
	GetGroupMemberRoleLevel(ctx context.Context, in *pbgroup.GetGroupMemberRoleLevelReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberRoleLevelResp, error)
	// 获取群组信息缓存
	GetGroupInfoCache(ctx context.Context, in *pbgroup.GetGroupInfoCacheReq, opts ...grpc.CallOption) (*pbgroup.GetGroupInfoCacheResp, error)
	// 获取群成员缓存
	GetGroupMemberCache(ctx context.Context, in *pbgroup.GetGroupMemberCacheReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberCacheResp, error)
	// 群组创建统计
	GroupCreateCount(ctx context.Context, in *pbgroup.GroupCreateCountReq, opts ...grpc.CallOption) (*pbgroup.GroupCreateCountResp, error)
	// 通知用户信息更新
	NotificationUserInfoUpdate(ctx context.Context, in *pbgroup.NotificationUserInfoUpdateReq, opts ...grpc.CallOption) (*pbgroup.NotificationUserInfoUpdateResp, error)
	// 获取增量群成员
	GetIncrementalGroupMember(ctx context.Context, in *pbgroup.GetIncrementalGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.GetIncrementalGroupMemberResp, error)
	// 批量获取增量群成员
	BatchGetIncrementalGroupMember(ctx context.Context, in *pbgroup.BatchGetIncrementalGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.BatchGetIncrementalGroupMemberResp, error)
	// 获取增量加入群组
	GetIncrementalJoinGroup(ctx context.Context, in *pbgroup.GetIncrementalJoinGroupReq, opts ...grpc.CallOption) (*pbgroup.GetIncrementalJoinGroupResp, error)
	// 获取完整群成员ID列表
	GetFullGroupMemberUserIDs(ctx context.Context, in *pbgroup.GetFullGroupMemberUserIDsReq, opts ...grpc.CallOption) (*pbgroup.GetFullGroupMemberUserIDsResp, error)
	// 获取完整加入群组ID列表
	GetFullJoinGroupIDs(ctx context.Context, in *pbgroup.GetFullJoinGroupIDsReq, opts ...grpc.CallOption) (*pbgroup.GetFullJoinGroupIDsResp, error)
}

type defaultGroupService struct {
	cli zrpc.Client
}

func NewGroupService(cli zrpc.Client) GroupService {
	return &defaultGroupService{cli: cli}
}

func (s *defaultGroupService) CreateGroup(ctx context.Context, in *pbgroup.CreateGroupReq, opts ...grpc.CallOption) (*pbgroup.CreateGroupResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.CreateGroup(ctx, in, opts...)
}

func (s *defaultGroupService) JoinGroup(ctx context.Context, in *pbgroup.JoinGroupReq, opts ...grpc.CallOption) (*pbgroup.JoinGroupResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.JoinGroup(ctx, in, opts...)
}

func (s *defaultGroupService) QuitGroup(ctx context.Context, in *pbgroup.QuitGroupReq, opts ...grpc.CallOption) (*pbgroup.QuitGroupResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.QuitGroup(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupsInfo(ctx context.Context, in *pbgroup.GetGroupsInfoReq, opts ...grpc.CallOption) (*pbgroup.GetGroupsInfoResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupsInfo(ctx, in, opts...)
}

func (s *defaultGroupService) SetGroupInfo(ctx context.Context, in *pbgroup.SetGroupInfoReq, opts ...grpc.CallOption) (*pbgroup.SetGroupInfoResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.SetGroupInfo(ctx, in, opts...)
}

func (s *defaultGroupService) SetGroupInfoEx(ctx context.Context, in *pbgroup.SetGroupInfoExReq, opts ...grpc.CallOption) (*pbgroup.SetGroupInfoExResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.SetGroupInfoEx(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupApplicationList(ctx context.Context, in *pbgroup.GetGroupApplicationListReq, opts ...grpc.CallOption) (*pbgroup.GetGroupApplicationListResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupApplicationList(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupApplicationUnhandledCount(ctx context.Context, in *pbgroup.GetGroupApplicationUnhandledCountReq, opts ...grpc.CallOption) (*pbgroup.GetGroupApplicationUnhandledCountResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupApplicationUnhandledCount(ctx, in, opts...)
}

func (s *defaultGroupService) GetUserReqApplicationList(ctx context.Context, in *pbgroup.GetUserReqApplicationListReq, opts ...grpc.CallOption) (*pbgroup.GetUserReqApplicationListResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetUserReqApplicationList(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupUsersReqApplicationList(ctx context.Context, in *pbgroup.GetGroupUsersReqApplicationListReq, opts ...grpc.CallOption) (*pbgroup.GetGroupUsersReqApplicationListResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupUsersReqApplicationList(ctx, in, opts...)
}

func (s *defaultGroupService) GetSpecifiedUserGroupRequestInfo(ctx context.Context, in *pbgroup.GetSpecifiedUserGroupRequestInfoReq, opts ...grpc.CallOption) (*pbgroup.GetSpecifiedUserGroupRequestInfoResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetSpecifiedUserGroupRequestInfo(ctx, in, opts...)
}

func (s *defaultGroupService) TransferGroupOwner(ctx context.Context, in *pbgroup.TransferGroupOwnerReq, opts ...grpc.CallOption) (*pbgroup.TransferGroupOwnerResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.TransferGroupOwner(ctx, in, opts...)
}

func (s *defaultGroupService) GroupApplicationResponse(ctx context.Context, in *pbgroup.GroupApplicationResponseReq, opts ...grpc.CallOption) (*pbgroup.GroupApplicationResponseResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GroupApplicationResponse(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupMemberList(ctx context.Context, in *pbgroup.GetGroupMemberListReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberListResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupMemberList(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupMembersInfo(ctx context.Context, in *pbgroup.GetGroupMembersInfoReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMembersInfoResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupMembersInfo(ctx, in, opts...)
}

func (s *defaultGroupService) KickGroupMember(ctx context.Context, in *pbgroup.KickGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.KickGroupMemberResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.KickGroupMember(ctx, in, opts...)
}

func (s *defaultGroupService) GetJoinedGroupList(ctx context.Context, in *pbgroup.GetJoinedGroupListReq, opts ...grpc.CallOption) (*pbgroup.GetJoinedGroupListResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetJoinedGroupList(ctx, in, opts...)
}

func (s *defaultGroupService) InviteUserToGroup(ctx context.Context, in *pbgroup.InviteUserToGroupReq, opts ...grpc.CallOption) (*pbgroup.InviteUserToGroupResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.InviteUserToGroup(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroups(ctx context.Context, in *pbgroup.GetGroupsReq, opts ...grpc.CallOption) (*pbgroup.GetGroupsResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroups(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupMembersCMS(ctx context.Context, in *pbgroup.GetGroupMembersCMSReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMembersCMSResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupMembersCMS(ctx, in, opts...)
}

func (s *defaultGroupService) DismissGroup(ctx context.Context, in *pbgroup.DismissGroupReq, opts ...grpc.CallOption) (*pbgroup.DismissGroupResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.DismissGroup(ctx, in, opts...)
}

func (s *defaultGroupService) MuteGroupMember(ctx context.Context, in *pbgroup.MuteGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.MuteGroupMemberResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.MuteGroupMember(ctx, in, opts...)
}

func (s *defaultGroupService) CancelMuteGroupMember(ctx context.Context, in *pbgroup.CancelMuteGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.CancelMuteGroupMemberResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.CancelMuteGroupMember(ctx, in, opts...)
}

func (s *defaultGroupService) MuteGroup(ctx context.Context, in *pbgroup.MuteGroupReq, opts ...grpc.CallOption) (*pbgroup.MuteGroupResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.MuteGroup(ctx, in, opts...)
}

func (s *defaultGroupService) CancelMuteGroup(ctx context.Context, in *pbgroup.CancelMuteGroupReq, opts ...grpc.CallOption) (*pbgroup.CancelMuteGroupResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.CancelMuteGroup(ctx, in, opts...)
}

func (s *defaultGroupService) SetGroupMemberInfo(ctx context.Context, in *pbgroup.SetGroupMemberInfoReq, opts ...grpc.CallOption) (*pbgroup.SetGroupMemberInfoResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.SetGroupMemberInfo(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupAbstractInfo(ctx context.Context, in *pbgroup.GetGroupAbstractInfoReq, opts ...grpc.CallOption) (*pbgroup.GetGroupAbstractInfoResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupAbstractInfo(ctx, in, opts...)
}

func (s *defaultGroupService) GetUserInGroupMembers(ctx context.Context, in *pbgroup.GetUserInGroupMembersReq, opts ...grpc.CallOption) (*pbgroup.GetUserInGroupMembersResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetUserInGroupMembers(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupMemberUserIDs(ctx context.Context, in *pbgroup.GetGroupMemberUserIDsReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberUserIDsResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupMemberUserIDs(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupMemberRoleLevel(ctx context.Context, in *pbgroup.GetGroupMemberRoleLevelReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberRoleLevelResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupMemberRoleLevel(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupInfoCache(ctx context.Context, in *pbgroup.GetGroupInfoCacheReq, opts ...grpc.CallOption) (*pbgroup.GetGroupInfoCacheResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupInfoCache(ctx, in, opts...)
}

func (s *defaultGroupService) GetGroupMemberCache(ctx context.Context, in *pbgroup.GetGroupMemberCacheReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberCacheResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetGroupMemberCache(ctx, in, opts...)
}

func (s *defaultGroupService) GroupCreateCount(ctx context.Context, in *pbgroup.GroupCreateCountReq, opts ...grpc.CallOption) (*pbgroup.GroupCreateCountResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GroupCreateCount(ctx, in, opts...)
}

func (s *defaultGroupService) NotificationUserInfoUpdate(ctx context.Context, in *pbgroup.NotificationUserInfoUpdateReq, opts ...grpc.CallOption) (*pbgroup.NotificationUserInfoUpdateResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.NotificationUserInfoUpdate(ctx, in, opts...)
}

func (s *defaultGroupService) GetIncrementalGroupMember(ctx context.Context, in *pbgroup.GetIncrementalGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.GetIncrementalGroupMemberResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetIncrementalGroupMember(ctx, in, opts...)
}

func (s *defaultGroupService) BatchGetIncrementalGroupMember(ctx context.Context, in *pbgroup.BatchGetIncrementalGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.BatchGetIncrementalGroupMemberResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.BatchGetIncrementalGroupMember(ctx, in, opts...)
}

func (s *defaultGroupService) GetIncrementalJoinGroup(ctx context.Context, in *pbgroup.GetIncrementalJoinGroupReq, opts ...grpc.CallOption) (*pbgroup.GetIncrementalJoinGroupResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetIncrementalJoinGroup(ctx, in, opts...)
}

func (s *defaultGroupService) GetFullGroupMemberUserIDs(ctx context.Context, in *pbgroup.GetFullGroupMemberUserIDsReq, opts ...grpc.CallOption) (*pbgroup.GetFullGroupMemberUserIDsResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetFullGroupMemberUserIDs(ctx, in, opts...)
}

func (s *defaultGroupService) GetFullJoinGroupIDs(ctx context.Context, in *pbgroup.GetFullJoinGroupIDsReq, opts ...grpc.CallOption) (*pbgroup.GetFullJoinGroupIDsResp, error) {
	groupClient := pbgroup.NewGroupClient(s.cli.Conn())
	return groupClient.GetFullJoinGroupIDs(ctx, in, opts...)
}

type stubGroupService struct {
}

func NewStubGroupService() GroupService {
	return &stubGroupService{}
}

func (s *stubGroupService) CreateGroup(ctx context.Context, in *pbgroup.CreateGroupReq, opts ...grpc.CallOption) (*pbgroup.CreateGroupResp, error) {
	return &pbgroup.CreateGroupResp{}, nil
}

func (s *stubGroupService) JoinGroup(ctx context.Context, in *pbgroup.JoinGroupReq, opts ...grpc.CallOption) (*pbgroup.JoinGroupResp, error) {
	return &pbgroup.JoinGroupResp{}, nil
}

func (s *stubGroupService) QuitGroup(ctx context.Context, in *pbgroup.QuitGroupReq, opts ...grpc.CallOption) (*pbgroup.QuitGroupResp, error) {
	return &pbgroup.QuitGroupResp{}, nil
}

func (s *stubGroupService) GetGroupsInfo(ctx context.Context, in *pbgroup.GetGroupsInfoReq, opts ...grpc.CallOption) (*pbgroup.GetGroupsInfoResp, error) {
	return &pbgroup.GetGroupsInfoResp{}, nil
}

func (s *stubGroupService) SetGroupInfo(ctx context.Context, in *pbgroup.SetGroupInfoReq, opts ...grpc.CallOption) (*pbgroup.SetGroupInfoResp, error) {
	return &pbgroup.SetGroupInfoResp{}, nil
}

func (s *stubGroupService) SetGroupInfoEx(ctx context.Context, in *pbgroup.SetGroupInfoExReq, opts ...grpc.CallOption) (*pbgroup.SetGroupInfoExResp, error) {
	return &pbgroup.SetGroupInfoExResp{}, nil
}

func (s *stubGroupService) GetGroupApplicationList(ctx context.Context, in *pbgroup.GetGroupApplicationListReq, opts ...grpc.CallOption) (*pbgroup.GetGroupApplicationListResp, error) {
	return &pbgroup.GetGroupApplicationListResp{}, nil
}

func (s *stubGroupService) GetGroupApplicationUnhandledCount(ctx context.Context, in *pbgroup.GetGroupApplicationUnhandledCountReq, opts ...grpc.CallOption) (*pbgroup.GetGroupApplicationUnhandledCountResp, error) {
	return &pbgroup.GetGroupApplicationUnhandledCountResp{}, nil
}

func (s *stubGroupService) GetUserReqApplicationList(ctx context.Context, in *pbgroup.GetUserReqApplicationListReq, opts ...grpc.CallOption) (*pbgroup.GetUserReqApplicationListResp, error) {
	return &pbgroup.GetUserReqApplicationListResp{}, nil
}

func (s *stubGroupService) GetGroupUsersReqApplicationList(ctx context.Context, in *pbgroup.GetGroupUsersReqApplicationListReq, opts ...grpc.CallOption) (*pbgroup.GetGroupUsersReqApplicationListResp, error) {
	return &pbgroup.GetGroupUsersReqApplicationListResp{}, nil
}

func (s *stubGroupService) GetSpecifiedUserGroupRequestInfo(ctx context.Context, in *pbgroup.GetSpecifiedUserGroupRequestInfoReq, opts ...grpc.CallOption) (*pbgroup.GetSpecifiedUserGroupRequestInfoResp, error) {
	return &pbgroup.GetSpecifiedUserGroupRequestInfoResp{}, nil
}

func (s *stubGroupService) TransferGroupOwner(ctx context.Context, in *pbgroup.TransferGroupOwnerReq, opts ...grpc.CallOption) (*pbgroup.TransferGroupOwnerResp, error) {
	return &pbgroup.TransferGroupOwnerResp{}, nil
}

func (s *stubGroupService) GroupApplicationResponse(ctx context.Context, in *pbgroup.GroupApplicationResponseReq, opts ...grpc.CallOption) (*pbgroup.GroupApplicationResponseResp, error) {
	return &pbgroup.GroupApplicationResponseResp{}, nil
}

func (s *stubGroupService) GetGroupMemberList(ctx context.Context, in *pbgroup.GetGroupMemberListReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberListResp, error) {
	return &pbgroup.GetGroupMemberListResp{}, nil
}

func (s *stubGroupService) GetGroupMembersInfo(ctx context.Context, in *pbgroup.GetGroupMembersInfoReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMembersInfoResp, error) {
	return &pbgroup.GetGroupMembersInfoResp{}, nil
}

func (s *stubGroupService) KickGroupMember(ctx context.Context, in *pbgroup.KickGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.KickGroupMemberResp, error) {
	return &pbgroup.KickGroupMemberResp{}, nil
}

func (s *stubGroupService) GetJoinedGroupList(ctx context.Context, in *pbgroup.GetJoinedGroupListReq, opts ...grpc.CallOption) (*pbgroup.GetJoinedGroupListResp, error) {
	return &pbgroup.GetJoinedGroupListResp{}, nil
}

func (s *stubGroupService) InviteUserToGroup(ctx context.Context, in *pbgroup.InviteUserToGroupReq, opts ...grpc.CallOption) (*pbgroup.InviteUserToGroupResp, error) {
	return &pbgroup.InviteUserToGroupResp{}, nil
}

func (s *stubGroupService) GetGroups(ctx context.Context, in *pbgroup.GetGroupsReq, opts ...grpc.CallOption) (*pbgroup.GetGroupsResp, error) {
	return &pbgroup.GetGroupsResp{}, nil
}

func (s *stubGroupService) GetGroupMembersCMS(ctx context.Context, in *pbgroup.GetGroupMembersCMSReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMembersCMSResp, error) {
	return &pbgroup.GetGroupMembersCMSResp{}, nil
}

func (s *stubGroupService) DismissGroup(ctx context.Context, in *pbgroup.DismissGroupReq, opts ...grpc.CallOption) (*pbgroup.DismissGroupResp, error) {
	return &pbgroup.DismissGroupResp{}, nil
}

func (s *stubGroupService) MuteGroupMember(ctx context.Context, in *pbgroup.MuteGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.MuteGroupMemberResp, error) {
	return &pbgroup.MuteGroupMemberResp{}, nil
}

func (s *stubGroupService) CancelMuteGroupMember(ctx context.Context, in *pbgroup.CancelMuteGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.CancelMuteGroupMemberResp, error) {
	return &pbgroup.CancelMuteGroupMemberResp{}, nil
}

func (s *stubGroupService) MuteGroup(ctx context.Context, in *pbgroup.MuteGroupReq, opts ...grpc.CallOption) (*pbgroup.MuteGroupResp, error) {
	return &pbgroup.MuteGroupResp{}, nil
}

func (s *stubGroupService) CancelMuteGroup(ctx context.Context, in *pbgroup.CancelMuteGroupReq, opts ...grpc.CallOption) (*pbgroup.CancelMuteGroupResp, error) {
	return &pbgroup.CancelMuteGroupResp{}, nil
}

func (s *stubGroupService) SetGroupMemberInfo(ctx context.Context, in *pbgroup.SetGroupMemberInfoReq, opts ...grpc.CallOption) (*pbgroup.SetGroupMemberInfoResp, error) {
	return &pbgroup.SetGroupMemberInfoResp{}, nil
}

func (s *stubGroupService) GetGroupAbstractInfo(ctx context.Context, in *pbgroup.GetGroupAbstractInfoReq, opts ...grpc.CallOption) (*pbgroup.GetGroupAbstractInfoResp, error) {
	return &pbgroup.GetGroupAbstractInfoResp{}, nil
}

func (s *stubGroupService) GetUserInGroupMembers(ctx context.Context, in *pbgroup.GetUserInGroupMembersReq, opts ...grpc.CallOption) (*pbgroup.GetUserInGroupMembersResp, error) {
	return &pbgroup.GetUserInGroupMembersResp{}, nil
}

func (s *stubGroupService) GetGroupMemberUserIDs(ctx context.Context, in *pbgroup.GetGroupMemberUserIDsReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberUserIDsResp, error) {
	return &pbgroup.GetGroupMemberUserIDsResp{}, nil
}

func (s *stubGroupService) GetGroupMemberRoleLevel(ctx context.Context, in *pbgroup.GetGroupMemberRoleLevelReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberRoleLevelResp, error) {
	return &pbgroup.GetGroupMemberRoleLevelResp{}, nil
}

func (s *stubGroupService) GetGroupInfoCache(ctx context.Context, in *pbgroup.GetGroupInfoCacheReq, opts ...grpc.CallOption) (*pbgroup.GetGroupInfoCacheResp, error) {
	return &pbgroup.GetGroupInfoCacheResp{}, nil
}

func (s *stubGroupService) GetGroupMemberCache(ctx context.Context, in *pbgroup.GetGroupMemberCacheReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberCacheResp, error) {
	return &pbgroup.GetGroupMemberCacheResp{}, nil
}

func (s *stubGroupService) GroupCreateCount(ctx context.Context, in *pbgroup.GroupCreateCountReq, opts ...grpc.CallOption) (*pbgroup.GroupCreateCountResp, error) {
	return &pbgroup.GroupCreateCountResp{}, nil
}

func (s *stubGroupService) NotificationUserInfoUpdate(ctx context.Context, in *pbgroup.NotificationUserInfoUpdateReq, opts ...grpc.CallOption) (*pbgroup.NotificationUserInfoUpdateResp, error) {
	return &pbgroup.NotificationUserInfoUpdateResp{}, nil
}

func (s *stubGroupService) GetIncrementalGroupMember(ctx context.Context, in *pbgroup.GetIncrementalGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.GetIncrementalGroupMemberResp, error) {
	return &pbgroup.GetIncrementalGroupMemberResp{}, nil
}

func (s *stubGroupService) BatchGetIncrementalGroupMember(ctx context.Context, in *pbgroup.BatchGetIncrementalGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.BatchGetIncrementalGroupMemberResp, error) {
	return &pbgroup.BatchGetIncrementalGroupMemberResp{}, nil
}

func (s *stubGroupService) GetIncrementalJoinGroup(ctx context.Context, in *pbgroup.GetIncrementalJoinGroupReq, opts ...grpc.CallOption) (*pbgroup.GetIncrementalJoinGroupResp, error) {
	return &pbgroup.GetIncrementalJoinGroupResp{}, nil
}

func (s *stubGroupService) GetFullGroupMemberUserIDs(ctx context.Context, in *pbgroup.GetFullGroupMemberUserIDsReq, opts ...grpc.CallOption) (*pbgroup.GetFullGroupMemberUserIDsResp, error) {
	return &pbgroup.GetFullGroupMemberUserIDsResp{}, nil
}

func (s *stubGroupService) GetFullJoinGroupIDs(ctx context.Context, in *pbgroup.GetFullJoinGroupIDsReq, opts ...grpc.CallOption) (*pbgroup.GetFullJoinGroupIDsResp, error) {
	return &pbgroup.GetFullJoinGroupIDsResp{}, nil
}
