package relationservice

import (
	"context"

	"github.com/PaperMan11/goim/pkg/localcache"
	pbrelation "github.com/PaperMan11/goim/pkg/protocol/relation"
	"github.com/PaperMan11/goim/pkg/rpcclient/relationservice"
	"google.golang.org/grpc"
)

type RelationServiceWrapperCache interface {
	relationservice.RelationService
}

type RelationService struct {
	relationservice.RelationService
	localCache localcache.LocalCache
}

func NewRelationServiceWrapperCache(relationService relationservice.RelationService, cache localcache.LocalCache) RelationServiceWrapperCache {
	return &RelationService{
		RelationService: relationService,
		localCache:      cache,
	}
}

// 获取好友ID列表
func (s *RelationService) GetFriendIDs(ctx context.Context, in *pbrelation.GetFriendIDsReq, opts ...grpc.CallOption) (*pbrelation.GetFriendIDsResp, error) {
	if s.localCache == nil {
		return s.RelationService.GetFriendIDs(ctx, in, opts...)
	}
	key := GetFriendIDListKey(in.UserID)
	resp, err := s.localCache.Take(key, func() (any, error) {
		return s.RelationService.GetFriendIDs(ctx, in, opts...)
	})
	if err != nil {
		return nil, err
	}
	return resp.(*pbrelation.GetFriendIDsResp), nil
}

// 删除好友
func (s *RelationService) DeleteFriend(ctx context.Context, in *pbrelation.DeleteFriendReq, opts ...grpc.CallOption) (*pbrelation.DeleteFriendResp, error) {
	if s.localCache == nil {
		return s.RelationService.DeleteFriend(ctx, in, opts...)
	}
	key := GetFriendInfoKey(in.OwnerUserID)
	s.localCache.PublishDelete([]string{key})
	return s.RelationService.DeleteFriend(ctx, in, opts...)
}
