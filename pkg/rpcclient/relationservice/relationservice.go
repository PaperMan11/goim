package relationservice

import (
	"context"

	pbrelation "github.com/PaperMan11/goim/pkg/protocol/relation"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type RelationService interface {
	// 申请添加好友
	ApplyToAddFriend(ctx context.Context, in *pbrelation.ApplyToAddFriendReq, opts ...grpc.CallOption) (*pbrelation.ApplyToAddFriendResp, error)
	// 获取收到的好友申请列表
	GetPaginationFriendsApplyTo(ctx context.Context, in *pbrelation.GetPaginationFriendsApplyToReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationFriendsApplyToResp, error)
	// 获取发出的好友申请列表
	GetPaginationFriendsApplyFrom(ctx context.Context, in *pbrelation.GetPaginationFriendsApplyFromReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationFriendsApplyFromResp, error)
	// 获取未处理好友申请数量
	GetSelfUnhandledApplyCount(ctx context.Context, in *pbrelation.GetSelfUnhandledApplyCountReq, opts ...grpc.CallOption) (*pbrelation.GetSelfUnhandledApplyCountResp, error)
	// 获取指定好友申请
	GetDesignatedFriendsApply(ctx context.Context, in *pbrelation.GetDesignatedFriendsApplyReq, opts ...grpc.CallOption) (*pbrelation.GetDesignatedFriendsApplyResp, error)
	// 获取增量好友申请(收到的)
	GetIncrementalFriendsApplyTo(ctx context.Context, in *pbrelation.GetIncrementalFriendsApplyToReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalFriendsApplyToResp, error)
	// 获取增量好友申请(发出的)
	GetIncrementalFriendsApplyFrom(ctx context.Context, in *pbrelation.GetIncrementalFriendsApplyFromReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalFriendsApplyFromResp, error)
	// 添加黑名单
	AddBlack(ctx context.Context, in *pbrelation.AddBlackReq, opts ...grpc.CallOption) (*pbrelation.AddBlackResp, error)
	// 移除黑名单
	RemoveBlack(ctx context.Context, in *pbrelation.RemoveBlackReq, opts ...grpc.CallOption) (*pbrelation.RemoveBlackResp, error)
	// 验证好友关系
	IsFriend(ctx context.Context, in *pbrelation.IsFriendReq, opts ...grpc.CallOption) (*pbrelation.IsFriendResp, error)
	// 验证黑名单关系
	IsBlack(ctx context.Context, in *pbrelation.IsBlackReq, opts ...grpc.CallOption) (*pbrelation.IsBlackResp, error)
	// 获取黑名单列表
	GetPaginationBlacks(ctx context.Context, in *pbrelation.GetPaginationBlacksReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationBlacksResp, error)
	// 获取指定黑名单信息
	GetSpecifiedBlacks(ctx context.Context, in *pbrelation.GetSpecifiedBlacksReq, opts ...grpc.CallOption) (*pbrelation.GetSpecifiedBlacksResp, error)
	// 删除好友
	DeleteFriend(ctx context.Context, in *pbrelation.DeleteFriendReq, opts ...grpc.CallOption) (*pbrelation.DeleteFriendResp, error)
	// 处理好友申请(同意或拒绝)
	RespondFriendApply(ctx context.Context, in *pbrelation.RespondFriendApplyReq, opts ...grpc.CallOption) (*pbrelation.RespondFriendApplyResp, error)
	// 更新好友信息(置顶、备注等)
	UpdateFriends(ctx context.Context, in *pbrelation.UpdateFriendsReq, opts ...grpc.CallOption) (*pbrelation.UpdateFriendsResp, error)
	// 设置好友备注
	SetFriendRemark(ctx context.Context, in *pbrelation.SetFriendRemarkReq, opts ...grpc.CallOption) (*pbrelation.SetFriendRemarkResp, error)
	// 导入好友关系
	ImportFriends(ctx context.Context, in *pbrelation.ImportFriendReq, opts ...grpc.CallOption) (*pbrelation.ImportFriendResp, error)
	// 获取指定好友信息(未找到不报错)
	GetDesignatedFriends(ctx context.Context, in *pbrelation.GetDesignatedFriendsReq, opts ...grpc.CallOption) (*pbrelation.GetDesignatedFriendsResp, error)
	// 分页获取好友列表(ID不存在报错)
	GetPaginationFriends(ctx context.Context, in *pbrelation.GetPaginationFriendsReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationFriendsResp, error)
	// 获取好友ID列表
	GetFriendIDs(ctx context.Context, in *pbrelation.GetFriendIDsReq, opts ...grpc.CallOption) (*pbrelation.GetFriendIDsResp, error)
	// 获取指定用户的详细信息
	GetSpecifiedFriendsInfo(ctx context.Context, in *pbrelation.GetSpecifiedFriendsInfoReq, opts ...grpc.CallOption) (*pbrelation.GetSpecifiedFriendsInfoResp, error)
	// 获取增量好友列表
	GetIncrementalFriends(ctx context.Context, in *pbrelation.GetIncrementalFriendsReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalFriendsResp, error)
	// 获取增量黑名单列表
	GetIncrementalBlacks(ctx context.Context, in *pbrelation.GetIncrementalBlacksReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalBlacksResp, error)
	// 获取完整好友ID列表
	GetFullFriendUserIDs(ctx context.Context, in *pbrelation.GetFullFriendUserIDsReq, opts ...grpc.CallOption) (*pbrelation.GetFullFriendUserIDsResp, error)
	// 通知用户信息更新
	NotificationUserInfoUpdate(ctx context.Context, in *pbrelation.NotificationUserInfoUpdateReq, opts ...grpc.CallOption) (*pbrelation.NotificationUserInfoUpdateResp, error)
	// 获取好友信息(简化版)
	GetFriendInfo(ctx context.Context, in *pbrelation.GetFriendInfoReq, opts ...grpc.CallOption) (*pbrelation.GetFriendInfoResp, error)
}

type defaultRelationService struct {
	cli zrpc.Client
}

func NewRelationService(cli zrpc.Client) RelationService {
	return &defaultRelationService{cli: cli}
}

func (s *defaultRelationService) ApplyToAddFriend(ctx context.Context, in *pbrelation.ApplyToAddFriendReq, opts ...grpc.CallOption) (*pbrelation.ApplyToAddFriendResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.ApplyToAddFriend(ctx, in, opts...)
}

func (s *defaultRelationService) GetPaginationFriendsApplyTo(ctx context.Context, in *pbrelation.GetPaginationFriendsApplyToReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationFriendsApplyToResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetPaginationFriendsApplyTo(ctx, in, opts...)
}

func (s *defaultRelationService) GetPaginationFriendsApplyFrom(ctx context.Context, in *pbrelation.GetPaginationFriendsApplyFromReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationFriendsApplyFromResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetPaginationFriendsApplyFrom(ctx, in, opts...)
}

func (s *defaultRelationService) GetSelfUnhandledApplyCount(ctx context.Context, in *pbrelation.GetSelfUnhandledApplyCountReq, opts ...grpc.CallOption) (*pbrelation.GetSelfUnhandledApplyCountResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetSelfUnhandledApplyCount(ctx, in, opts...)
}

func (s *defaultRelationService) GetDesignatedFriendsApply(ctx context.Context, in *pbrelation.GetDesignatedFriendsApplyReq, opts ...grpc.CallOption) (*pbrelation.GetDesignatedFriendsApplyResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetDesignatedFriendsApply(ctx, in, opts...)
}

func (s *defaultRelationService) GetIncrementalFriendsApplyTo(ctx context.Context, in *pbrelation.GetIncrementalFriendsApplyToReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalFriendsApplyToResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetIncrementalFriendsApplyTo(ctx, in, opts...)
}

func (s *defaultRelationService) GetIncrementalFriendsApplyFrom(ctx context.Context, in *pbrelation.GetIncrementalFriendsApplyFromReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalFriendsApplyFromResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetIncrementalFriendsApplyFrom(ctx, in, opts...)
}

func (s *defaultRelationService) AddBlack(ctx context.Context, in *pbrelation.AddBlackReq, opts ...grpc.CallOption) (*pbrelation.AddBlackResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.AddBlack(ctx, in, opts...)
}

func (s *defaultRelationService) RemoveBlack(ctx context.Context, in *pbrelation.RemoveBlackReq, opts ...grpc.CallOption) (*pbrelation.RemoveBlackResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.RemoveBlack(ctx, in, opts...)
}

func (s *defaultRelationService) IsFriend(ctx context.Context, in *pbrelation.IsFriendReq, opts ...grpc.CallOption) (*pbrelation.IsFriendResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.IsFriend(ctx, in, opts...)
}

func (s *defaultRelationService) IsBlack(ctx context.Context, in *pbrelation.IsBlackReq, opts ...grpc.CallOption) (*pbrelation.IsBlackResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.IsBlack(ctx, in, opts...)
}

func (s *defaultRelationService) GetPaginationBlacks(ctx context.Context, in *pbrelation.GetPaginationBlacksReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationBlacksResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetPaginationBlacks(ctx, in, opts...)
}

func (s *defaultRelationService) GetSpecifiedBlacks(ctx context.Context, in *pbrelation.GetSpecifiedBlacksReq, opts ...grpc.CallOption) (*pbrelation.GetSpecifiedBlacksResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetSpecifiedBlacks(ctx, in, opts...)
}

func (s *defaultRelationService) DeleteFriend(ctx context.Context, in *pbrelation.DeleteFriendReq, opts ...grpc.CallOption) (*pbrelation.DeleteFriendResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.DeleteFriend(ctx, in, opts...)
}

func (s *defaultRelationService) RespondFriendApply(ctx context.Context, in *pbrelation.RespondFriendApplyReq, opts ...grpc.CallOption) (*pbrelation.RespondFriendApplyResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.RespondFriendApply(ctx, in, opts...)
}

func (s *defaultRelationService) UpdateFriends(ctx context.Context, in *pbrelation.UpdateFriendsReq, opts ...grpc.CallOption) (*pbrelation.UpdateFriendsResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.UpdateFriends(ctx, in, opts...)
}

func (s *defaultRelationService) SetFriendRemark(ctx context.Context, in *pbrelation.SetFriendRemarkReq, opts ...grpc.CallOption) (*pbrelation.SetFriendRemarkResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.SetFriendRemark(ctx, in, opts...)
}

func (s *defaultRelationService) ImportFriends(ctx context.Context, in *pbrelation.ImportFriendReq, opts ...grpc.CallOption) (*pbrelation.ImportFriendResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.ImportFriends(ctx, in, opts...)
}

func (s *defaultRelationService) GetDesignatedFriends(ctx context.Context, in *pbrelation.GetDesignatedFriendsReq, opts ...grpc.CallOption) (*pbrelation.GetDesignatedFriendsResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetDesignatedFriends(ctx, in, opts...)
}

func (s *defaultRelationService) GetPaginationFriends(ctx context.Context, in *pbrelation.GetPaginationFriendsReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationFriendsResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetPaginationFriends(ctx, in, opts...)
}

func (s *defaultRelationService) GetFriendIDs(ctx context.Context, in *pbrelation.GetFriendIDsReq, opts ...grpc.CallOption) (*pbrelation.GetFriendIDsResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetFriendIDs(ctx, in, opts...)
}

func (s *defaultRelationService) GetSpecifiedFriendsInfo(ctx context.Context, in *pbrelation.GetSpecifiedFriendsInfoReq, opts ...grpc.CallOption) (*pbrelation.GetSpecifiedFriendsInfoResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetSpecifiedFriendsInfo(ctx, in, opts...)
}

func (s *defaultRelationService) GetIncrementalFriends(ctx context.Context, in *pbrelation.GetIncrementalFriendsReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalFriendsResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetIncrementalFriends(ctx, in, opts...)
}

func (s *defaultRelationService) GetIncrementalBlacks(ctx context.Context, in *pbrelation.GetIncrementalBlacksReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalBlacksResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetIncrementalBlacks(ctx, in, opts...)
}

func (s *defaultRelationService) GetFullFriendUserIDs(ctx context.Context, in *pbrelation.GetFullFriendUserIDsReq, opts ...grpc.CallOption) (*pbrelation.GetFullFriendUserIDsResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetFullFriendUserIDs(ctx, in, opts...)
}

func (s *defaultRelationService) NotificationUserInfoUpdate(ctx context.Context, in *pbrelation.NotificationUserInfoUpdateReq, opts ...grpc.CallOption) (*pbrelation.NotificationUserInfoUpdateResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.NotificationUserInfoUpdate(ctx, in, opts...)
}

func (s *defaultRelationService) GetFriendInfo(ctx context.Context, in *pbrelation.GetFriendInfoReq, opts ...grpc.CallOption) (*pbrelation.GetFriendInfoResp, error) {
	friendClient := pbrelation.NewFriendClient(s.cli.Conn())
	return friendClient.GetFriendInfo(ctx, in, opts...)
}

type stubRelationService struct {
}

func NewStubRelationService() RelationService {
	return &stubRelationService{}
}

func (s *stubRelationService) ApplyToAddFriend(ctx context.Context, in *pbrelation.ApplyToAddFriendReq, opts ...grpc.CallOption) (*pbrelation.ApplyToAddFriendResp, error) {
	return &pbrelation.ApplyToAddFriendResp{}, nil
}

func (s *stubRelationService) GetPaginationFriendsApplyTo(ctx context.Context, in *pbrelation.GetPaginationFriendsApplyToReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationFriendsApplyToResp, error) {
	return &pbrelation.GetPaginationFriendsApplyToResp{}, nil
}

func (s *stubRelationService) GetPaginationFriendsApplyFrom(ctx context.Context, in *pbrelation.GetPaginationFriendsApplyFromReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationFriendsApplyFromResp, error) {
	return &pbrelation.GetPaginationFriendsApplyFromResp{}, nil
}

func (s *stubRelationService) GetSelfUnhandledApplyCount(ctx context.Context, in *pbrelation.GetSelfUnhandledApplyCountReq, opts ...grpc.CallOption) (*pbrelation.GetSelfUnhandledApplyCountResp, error) {
	return &pbrelation.GetSelfUnhandledApplyCountResp{}, nil
}

func (s *stubRelationService) GetDesignatedFriendsApply(ctx context.Context, in *pbrelation.GetDesignatedFriendsApplyReq, opts ...grpc.CallOption) (*pbrelation.GetDesignatedFriendsApplyResp, error) {
	return &pbrelation.GetDesignatedFriendsApplyResp{}, nil
}

func (s *stubRelationService) GetIncrementalFriendsApplyTo(ctx context.Context, in *pbrelation.GetIncrementalFriendsApplyToReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalFriendsApplyToResp, error) {
	return &pbrelation.GetIncrementalFriendsApplyToResp{}, nil
}

func (s *stubRelationService) GetIncrementalFriendsApplyFrom(ctx context.Context, in *pbrelation.GetIncrementalFriendsApplyFromReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalFriendsApplyFromResp, error) {
	return &pbrelation.GetIncrementalFriendsApplyFromResp{}, nil
}

func (s *stubRelationService) AddBlack(ctx context.Context, in *pbrelation.AddBlackReq, opts ...grpc.CallOption) (*pbrelation.AddBlackResp, error) {
	return &pbrelation.AddBlackResp{}, nil
}

func (s *stubRelationService) RemoveBlack(ctx context.Context, in *pbrelation.RemoveBlackReq, opts ...grpc.CallOption) (*pbrelation.RemoveBlackResp, error) {
	return &pbrelation.RemoveBlackResp{}, nil
}

func (s *stubRelationService) IsFriend(ctx context.Context, in *pbrelation.IsFriendReq, opts ...grpc.CallOption) (*pbrelation.IsFriendResp, error) {
	return &pbrelation.IsFriendResp{}, nil
}

func (s *stubRelationService) IsBlack(ctx context.Context, in *pbrelation.IsBlackReq, opts ...grpc.CallOption) (*pbrelation.IsBlackResp, error) {
	return &pbrelation.IsBlackResp{}, nil
}

func (s *stubRelationService) GetPaginationBlacks(ctx context.Context, in *pbrelation.GetPaginationBlacksReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationBlacksResp, error) {
	return &pbrelation.GetPaginationBlacksResp{}, nil
}

func (s *stubRelationService) GetSpecifiedBlacks(ctx context.Context, in *pbrelation.GetSpecifiedBlacksReq, opts ...grpc.CallOption) (*pbrelation.GetSpecifiedBlacksResp, error) {
	return &pbrelation.GetSpecifiedBlacksResp{}, nil
}

func (s *stubRelationService) DeleteFriend(ctx context.Context, in *pbrelation.DeleteFriendReq, opts ...grpc.CallOption) (*pbrelation.DeleteFriendResp, error) {
	return &pbrelation.DeleteFriendResp{}, nil
}

func (s *stubRelationService) RespondFriendApply(ctx context.Context, in *pbrelation.RespondFriendApplyReq, opts ...grpc.CallOption) (*pbrelation.RespondFriendApplyResp, error) {
	return &pbrelation.RespondFriendApplyResp{}, nil
}

func (s *stubRelationService) UpdateFriends(ctx context.Context, in *pbrelation.UpdateFriendsReq, opts ...grpc.CallOption) (*pbrelation.UpdateFriendsResp, error) {
	return &pbrelation.UpdateFriendsResp{}, nil
}

func (s *stubRelationService) SetFriendRemark(ctx context.Context, in *pbrelation.SetFriendRemarkReq, opts ...grpc.CallOption) (*pbrelation.SetFriendRemarkResp, error) {
	return &pbrelation.SetFriendRemarkResp{}, nil
}

func (s *stubRelationService) ImportFriends(ctx context.Context, in *pbrelation.ImportFriendReq, opts ...grpc.CallOption) (*pbrelation.ImportFriendResp, error) {
	return &pbrelation.ImportFriendResp{}, nil
}

func (s *stubRelationService) GetDesignatedFriends(ctx context.Context, in *pbrelation.GetDesignatedFriendsReq, opts ...grpc.CallOption) (*pbrelation.GetDesignatedFriendsResp, error) {
	return &pbrelation.GetDesignatedFriendsResp{}, nil
}

func (s *stubRelationService) GetPaginationFriends(ctx context.Context, in *pbrelation.GetPaginationFriendsReq, opts ...grpc.CallOption) (*pbrelation.GetPaginationFriendsResp, error) {
	return &pbrelation.GetPaginationFriendsResp{}, nil
}

func (s *stubRelationService) GetFriendIDs(ctx context.Context, in *pbrelation.GetFriendIDsReq, opts ...grpc.CallOption) (*pbrelation.GetFriendIDsResp, error) {
	return &pbrelation.GetFriendIDsResp{}, nil
}

func (s *stubRelationService) GetSpecifiedFriendsInfo(ctx context.Context, in *pbrelation.GetSpecifiedFriendsInfoReq, opts ...grpc.CallOption) (*pbrelation.GetSpecifiedFriendsInfoResp, error) {
	return &pbrelation.GetSpecifiedFriendsInfoResp{}, nil
}

func (s *stubRelationService) GetIncrementalFriends(ctx context.Context, in *pbrelation.GetIncrementalFriendsReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalFriendsResp, error) {
	return &pbrelation.GetIncrementalFriendsResp{}, nil
}

func (s *stubRelationService) GetIncrementalBlacks(ctx context.Context, in *pbrelation.GetIncrementalBlacksReq, opts ...grpc.CallOption) (*pbrelation.GetIncrementalBlacksResp, error) {
	return &pbrelation.GetIncrementalBlacksResp{}, nil
}

func (s *stubRelationService) GetFullFriendUserIDs(ctx context.Context, in *pbrelation.GetFullFriendUserIDsReq, opts ...grpc.CallOption) (*pbrelation.GetFullFriendUserIDsResp, error) {
	return &pbrelation.GetFullFriendUserIDsResp{}, nil
}

func (s *stubRelationService) NotificationUserInfoUpdate(ctx context.Context, in *pbrelation.NotificationUserInfoUpdateReq, opts ...grpc.CallOption) (*pbrelation.NotificationUserInfoUpdateResp, error) {
	return &pbrelation.NotificationUserInfoUpdateResp{}, nil
}

func (s *stubRelationService) GetFriendInfo(ctx context.Context, in *pbrelation.GetFriendInfoReq, opts ...grpc.CallOption) (*pbrelation.GetFriendInfoResp, error) {
	return &pbrelation.GetFriendInfoResp{}, nil
}
