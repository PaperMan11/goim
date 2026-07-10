package userservice

import (
	"context"
	"errors"
	"time"

	"github.com/PaperMan11/goim/pkg/localcache"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"google.golang.org/grpc"
)

type UserServiceWrapperCache interface {
	userservice.UserService
	GetUserInfo(ctx context.Context, userID string) (*sdkws.UserInfo, error)
}

type UserService struct {
	userservice.UserService
	localCache localcache.LocalCache
}

func NewUserServiceWrapperCache(userService userservice.UserService, cache localcache.LocalCache) UserServiceWrapperCache {
	return &UserService{
		UserService: userService,
		localCache:  cache,
	}
}

func (s *UserService) AccountCheck(ctx context.Context, in *pbuser.AccountCheckReq, opts ...grpc.CallOption) (*pbuser.AccountCheckResp, error) {
	if len(in.CheckUserIDs) != 1 || s.localCache == nil {
		return s.UserService.AccountCheck(ctx, in, opts...)
	}

	key := GetValidUserKey(in.CheckUserIDs[0])
	respI, ok := s.localCache.Get(key)
	if ok {
		return respI.(*pbuser.AccountCheckResp), nil
	}

	resp, err := s.UserService.AccountCheck(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	s.localCache.SetWithExpire(key, resp, 5*time.Second)
	return resp, nil
}

func (s *UserService) IsIMAdmin(ctx context.Context, in *pbuser.IsIMAdminReq, opts ...grpc.CallOption) (*pbuser.IsIMAdminResp, error) {
	if s.localCache == nil {
		return s.UserService.IsIMAdmin(ctx, in, opts...)
	}

	key := GetIMAdminKey(in.UserID)
	respI, err := s.localCache.Take(key, func() (any, error) {
		return s.UserService.IsIMAdmin(ctx, in, opts...)
	})
	if err != nil {
		return nil, err
	}
	if result, ok := respI.(*pbuser.IsIMAdminResp); ok {
		return result, nil
	}
	return nil, errors.New("invalid response type")
}

// 获取指定用户信息(完整字段)
func (s *UserService) GetUserInfo(ctx context.Context, userID string) (*sdkws.UserInfo, error) {
	fetch := func() (*sdkws.UserInfo, error) {
		userInfo, err := s.UserService.GetDesignateUsers(ctx, &pbuser.GetDesignateUsersReq{UserIDs: []string{userID}})
		if err != nil {
			return nil, err
		}
		return userInfo.UsersInfo[0], nil
	}

	if s.localCache == nil {
		return fetch()
	}
	key := GetUserInfoKey(userID)
	respI, err := s.localCache.Take(key, func() (any, error) {
		return fetch()
	})
	if err != nil {
		return nil, err
	}
	if result, ok := respI.(*sdkws.UserInfo); ok {
		return result, nil
	}
	return nil, errors.New("invalid response type")
}

// 更新用户信息
func (s *UserService) UpdateUserInfo(ctx context.Context, in *pbuser.UpdateUserInfoReq, opts ...grpc.CallOption) (*pbuser.UpdateUserInfoResp, error) {
	if s.localCache != nil && in.UserInfo.UserID != "" {
		key := GetUserInfoKey(in.UserInfo.UserID)
		s.localCache.PublishDelete([]string{key})
	}
	return s.UserService.UpdateUserInfo(ctx, in, opts...)
}

// 更新用户信息(扩展)
func (s *UserService) UpdateUserInfoEx(ctx context.Context, in *pbuser.UpdateUserInfoExReq, opts ...grpc.CallOption) (*pbuser.UpdateUserInfoExResp, error) {
	if s.localCache != nil && in.UserInfo.UserID != "" {
		key := GetUserInfoKey(in.UserInfo.UserID)
		s.localCache.PublishDelete([]string{key})
	}
	return s.UserService.UpdateUserInfoEx(ctx, in, opts...)
}

// 设置用户全局消息接收选项
func (s *UserService) SetGlobalRecvMessageOpt(ctx context.Context, in *pbuser.SetGlobalRecvMessageOptReq, opts ...grpc.CallOption) (*pbuser.SetGlobalRecvMessageOptResp, error) {
	if s.localCache != nil && in.UserID != "" {
		userRecvOptKey := GetUserRecvOptKey(in.UserID)
		s.localCache.PublishDelete([]string{userRecvOptKey})
	}
	return s.UserService.SetGlobalRecvMessageOpt(ctx, in, opts...)
}

// 获取用户全局消息接收选项(未找到时不报错)
func (s *UserService) GetGlobalRecvMessageOpt(ctx context.Context, in *pbuser.GetGlobalRecvMessageOptReq, opts ...grpc.CallOption) (*pbuser.GetGlobalRecvMessageOptResp, error) {
	if s.localCache == nil {
		return s.UserService.GetGlobalRecvMessageOpt(ctx, in, opts...)
	}
	userRecvOptKey := GetUserRecvOptKey(in.UserID)
	respI, err := s.localCache.Take(userRecvOptKey, func() (any, error) {
		return s.UserService.GetGlobalRecvMessageOpt(ctx, in, opts...)
	})
	if err != nil {
		return nil, err
	}
	if result, ok := respI.(*pbuser.GetGlobalRecvMessageOptResp); ok {
		return result, nil
	}
	return nil, errors.New("invalid response type")
}
