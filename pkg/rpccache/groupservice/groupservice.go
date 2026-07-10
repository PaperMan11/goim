package groupservice

import (
	"context"
	"errors"

	"github.com/PaperMan11/goim/pkg/localcache"
	pbgroup "github.com/PaperMan11/goim/pkg/protocol/group"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/rpcclient/groupservice"
	"google.golang.org/grpc"
)

type GroupServiceWrapperCache interface {
	groupservice.GroupService
	GetGroupInfo(ctx context.Context, groupID string) (*sdkws.GroupInfo, error)
	GetGroupMembersFullInfo(ctx context.Context, groupID string, userIDs []string) ([]*sdkws.GroupMemberFullInfo, error)
	GetGroupMemberFullInfo(ctx context.Context, groupID string, userID string) (*sdkws.GroupMemberFullInfo, error)
}

type GroupService struct {
	groupservice.GroupService
	localCache localcache.LocalCache
}

func NewGroupServiceWrapperCache(groupService groupservice.GroupService, cache localcache.LocalCache) GroupServiceWrapperCache {
	return &GroupService{
		GroupService: groupService,
		localCache:   cache,
	}
}

// 删除指定群组信息缓存
func (s *GroupService) deleteGroupInfoCache(_ context.Context, groupIDs []string) {
	if s.localCache != nil {
		keys := make([]string, 0, len(groupIDs))
		for _, groupID := range groupIDs {
			key := GetGroupInfoKey(groupID)
			keys = append(keys, key)
		}
		s.localCache.PublishDelete(keys)
	}
}

func (s *GroupService) deleteGroupMemberInfoCache(_ context.Context, groupID string, userIDs []string) {
	if s.localCache != nil {
		keys := make([]string, 0, len(userIDs))
		for _, userID := range userIDs {
			key := GetGroupMemberFullInfoKey(groupID, userID)
			keys = append(keys, key)
		}
		s.localCache.PublishDelete(keys)
	}
}

func (s *GroupService) deleteGroupMemberIDsCache(_ context.Context, groupID string) {
	if s.localCache != nil {
		key := GetGroupMemberIDsKey(groupID)
		s.localCache.PublishDelete([]string{key})
	}
}

// 获取指定群组信息
func (s *GroupService) GetGroupInfo(ctx context.Context, groupID string) (*sdkws.GroupInfo, error) {
	fetch := func() (*sdkws.GroupInfo, error) {
		resp, err := s.GroupService.GetGroupsInfo(ctx, &pbgroup.GetGroupsInfoReq{
			GroupIDs: []string{groupID},
		})
		if err != nil {
			return nil, err
		}
		if len(resp.GroupInfos) == 0 {
			return nil, errors.New("group not found")
		}
		return resp.GroupInfos[0], nil
	}

	if s.localCache == nil {
		return fetch()
	}
	key := GetGroupInfoKey(groupID)
	groupInfo, err := s.localCache.Take(key, func() (any, error) {
		return fetch()
	})
	if err != nil {
		return nil, err
	}
	return groupInfo.(*sdkws.GroupInfo), nil
}

// 设置群组信息
func (s *GroupService) SetGroupInfo(ctx context.Context, in *pbgroup.SetGroupInfoReq, opts ...grpc.CallOption) (*pbgroup.SetGroupInfoResp, error) {
	if s.localCache != nil {
		s.deleteGroupInfoCache(ctx, []string{in.GroupInfoForSet.GroupID})
	}
	return s.GroupService.SetGroupInfo(ctx, in, opts...)
}

// 设置群组信息(扩展)
func (s *GroupService) SetGroupInfoEx(ctx context.Context, in *pbgroup.SetGroupInfoExReq, opts ...grpc.CallOption) (*pbgroup.SetGroupInfoExResp, error) {
	if s.localCache != nil {
		s.deleteGroupInfoCache(ctx, []string{in.GroupID})
	}
	return s.GroupService.SetGroupInfoEx(ctx, in, opts...)
}

// 转让群主
func (s *GroupService) TransferGroupOwner(ctx context.Context, in *pbgroup.TransferGroupOwnerReq, opts ...grpc.CallOption) (*pbgroup.TransferGroupOwnerResp, error) {
	if s.localCache != nil {
		s.deleteGroupInfoCache(ctx, []string{in.GroupID})
	}
	return s.GroupService.TransferGroupOwner(ctx, in, opts...)
}

// 获取群组成员
func (s *GroupService) getGroupMember(ctx context.Context, groupID string, userID string) (*sdkws.GroupMemberFullInfo, error) {
	fetch := func() (*sdkws.GroupMemberFullInfo, error) {
		resp, err := s.GroupService.GetGroupMemberCache(ctx, &pbgroup.GetGroupMemberCacheReq{
			GroupID:       groupID,
			GroupMemberID: userID,
		})
		if err != nil {
			return nil, err
		}
		return resp.Member, nil
	}
	if s.localCache == nil {
		return fetch()
	}
	key := GetGroupMemberFullInfoKey(groupID, userID)
	members, err := s.localCache.Take(key, func() (any, error) {
		return fetch()
	})
	if err != nil {
		return nil, err
	}
	return members.(*sdkws.GroupMemberFullInfo), nil
}

func (s *GroupService) GetGroupMemberFullInfo(ctx context.Context, groupID string, userID string) (*sdkws.GroupMemberFullInfo, error) {
	return s.getGroupMember(ctx, groupID, userID)
}

func (s *GroupService) GetGroupMembersFullInfo(ctx context.Context, groupID string, userIDs []string) ([]*sdkws.GroupMemberFullInfo, error) {
	members := make([]*sdkws.GroupMemberFullInfo, 0, len(userIDs))
	for _, userID := range userIDs {
		member, err := s.getGroupMember(ctx, groupID, userID)
		if err != nil {
			continue
		}
		members = append(members, member)
	}
	return members, nil
}

// 退出群组
func (s *GroupService) QuitGroup(ctx context.Context, in *pbgroup.QuitGroupReq, opts ...grpc.CallOption) (*pbgroup.QuitGroupResp, error) {
	if s.localCache != nil {
		s.deleteGroupMemberInfoCache(ctx, in.GroupID, []string{in.UserID})
		s.deleteGroupMemberIDsCache(ctx, in.GroupID)
	}
	return s.GroupService.QuitGroup(ctx, in, opts...)
}

// 踢出群成员
func (s *GroupService) KickGroupMember(ctx context.Context, in *pbgroup.KickGroupMemberReq, opts ...grpc.CallOption) (*pbgroup.KickGroupMemberResp, error) {
	if s.localCache != nil {
		s.deleteGroupMemberInfoCache(ctx, in.GroupID, in.KickedUserIDs)
		s.deleteGroupMemberIDsCache(ctx, in.GroupID)
	}
	return s.GroupService.KickGroupMember(ctx, in, opts...)
}

// 获取群成员用户ID列表
func (s *GroupService) GetGroupMemberUserIDs(ctx context.Context, in *pbgroup.GetGroupMemberUserIDsReq, opts ...grpc.CallOption) (*pbgroup.GetGroupMemberUserIDsResp, error) {
	fetch := func() (*pbgroup.GetGroupMemberUserIDsResp, error) {
		resp, err := s.GroupService.GetGroupMemberUserIDs(ctx, in)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	if s.localCache == nil {
		return fetch()
	}
	key := GetGroupMemberIDsKey(in.GroupID)
	members, err := s.localCache.Take(key, func() (any, error) {
		return fetch()
	})
	if err != nil {
		return nil, err
	}
	return members.(*pbgroup.GetGroupMemberUserIDsResp), nil
}
