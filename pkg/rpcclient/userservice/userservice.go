package userservice

import (
	"context"

	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type UserService interface {
	// 获取指定用户信息(完整字段)
	GetDesignateUsers(ctx context.Context, in *pbuser.GetDesignateUsersReq, opts ...grpc.CallOption) (*pbuser.GetDesignateUsersResp, error)
	// 更新用户信息
	UpdateUserInfo(ctx context.Context, in *pbuser.UpdateUserInfoReq, opts ...grpc.CallOption) (*pbuser.UpdateUserInfoResp, error)
	// 更新用户信息(扩展)
	UpdateUserInfoEx(ctx context.Context, in *pbuser.UpdateUserInfoExReq, opts ...grpc.CallOption) (*pbuser.UpdateUserInfoExResp, error)
	// 设置用户全局消息接收选项
	SetGlobalRecvMessageOpt(ctx context.Context, in *pbuser.SetGlobalRecvMessageOptReq, opts ...grpc.CallOption) (*pbuser.SetGlobalRecvMessageOptResp, error)
	// 获取用户全局消息接收选项(未找到时不报错)
	GetGlobalRecvMessageOpt(ctx context.Context, in *pbuser.GetGlobalRecvMessageOptReq, opts ...grpc.CallOption) (*pbuser.GetGlobalRecvMessageOptResp, error)
	// 检查用户ID是否存在
	AccountCheck(ctx context.Context, in *pbuser.AccountCheckReq, opts ...grpc.CallOption) (*pbuser.AccountCheckResp, error)
	// 分页获取用户信息(或按用户ID、昵称查询)
	GetPaginationUsers(ctx context.Context, in *pbuser.GetPaginationUsersReq, opts ...grpc.CallOption) (*pbuser.GetPaginationUsersResp, error)
	// 用户注册
	UserRegister(ctx context.Context, in *pbuser.UserRegisterReq, opts ...grpc.CallOption) (*pbuser.UserRegisterResp, error)
	// 获取所有用户ID
	GetAllUserID(ctx context.Context, in *pbuser.GetAllUserIDReq, opts ...grpc.CallOption) (*pbuser.GetAllUserIDResp, error)
	// 获取指定时间段内的用户总数和增量
	// 用户注册数量统计
	UserRegisterCount(ctx context.Context, in *pbuser.UserRegisterCountReq, opts ...grpc.CallOption) (*pbuser.UserRegisterCountResp, error)
	// 订阅或取消订阅用户在线状态
	SubscribeOrCancelUsersStatus(ctx context.Context, in *pbuser.SubscribeOrCancelUsersStatusReq, opts ...grpc.CallOption) (*pbuser.SubscribeOrCancelUsersStatusResp, error)
	// 获取订阅用户的在线状态
	GetSubscribeUsersStatus(ctx context.Context, in *pbuser.GetSubscribeUsersStatusReq, opts ...grpc.CallOption) (*pbuser.GetSubscribeUsersStatusResp, error)
	// 获取用户在线状态
	GetUserStatus(ctx context.Context, in *pbuser.GetUserStatusReq, opts ...grpc.CallOption) (*pbuser.GetUserStatusResp, error)
	// 网关同步用户在线状态到Redis
	SetUserStatus(ctx context.Context, in *pbuser.SetUserStatusReq, opts ...grpc.CallOption) (*pbuser.SetUserStatusResp, error)
	// 添加用户命令
	ProcessUserCommandAdd(ctx context.Context, in *pbuser.ProcessUserCommandAddReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandAddResp, error)
	// 更新用户命令
	ProcessUserCommandUpdate(ctx context.Context, in *pbuser.ProcessUserCommandUpdateReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandUpdateResp, error)
	// 删除用户命令
	ProcessUserCommandDelete(ctx context.Context, in *pbuser.ProcessUserCommandDeleteReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandDeleteResp, error)
	// 获取用户命令
	ProcessUserCommandGet(ctx context.Context, in *pbuser.ProcessUserCommandGetReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandGetResp, error)
	// 获取所有用户命令
	ProcessUserCommandGetAll(ctx context.Context, in *pbuser.ProcessUserCommandGetAllReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandGetAllResp, error)
	// 添加系统通知账户
	AddNotificationAccount(ctx context.Context, in *pbuser.AddNotificationAccountReq, opts ...grpc.CallOption) (*pbuser.AddNotificationAccountResp, error)
	// 更新系统通知账户信息
	UpdateNotificationAccountInfo(ctx context.Context, in *pbuser.UpdateNotificationAccountInfoReq, opts ...grpc.CallOption) (*pbuser.UpdateNotificationAccountInfoResp, error)
	// 搜索系统通知账户
	SearchNotificationAccount(ctx context.Context, in *pbuser.SearchNotificationAccountReq, opts ...grpc.CallOption) (*pbuser.SearchNotificationAccountResp, error)
	// 按用户ID获取通知账户
	GetNotificationAccount(ctx context.Context, in *pbuser.GetNotificationAccountReq, opts ...grpc.CallOption) (*pbuser.GetNotificationAccountResp, error)
	// 排序查询用户
	SortQuery(ctx context.Context, in *pbuser.SortQueryReq, opts ...grpc.CallOption) (*pbuser.SortQueryResp, error)
	// 批量设置用户在线状态
	SetUserOnlineStatus(ctx context.Context, in *pbuser.SetUserOnlineStatusReq, opts ...grpc.CallOption) (*pbuser.SetUserOnlineStatusResp, error)
	// 获取所有在线用户
	GetAllOnlineUsers(ctx context.Context, in *pbuser.GetAllOnlineUsersReq, opts ...grpc.CallOption) (*pbuser.GetAllOnlineUsersResp, error)
	// 获取用户客户端配置
	GetUserClientConfig(ctx context.Context, in *pbuser.GetUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.GetUserClientConfigResp, error)
	// 设置用户客户端配置
	SetUserClientConfig(ctx context.Context, in *pbuser.SetUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.SetUserClientConfigResp, error)
	// 删除用户客户端配置
	DelUserClientConfig(ctx context.Context, in *pbuser.DelUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.DelUserClientConfigResp, error)
	// 分页获取用户客户端配置
	PageUserClientConfig(ctx context.Context, in *pbuser.PageUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.PageUserClientConfigResp, error)
	// 是否为IM管理员
	IsIMAdmin(ctx context.Context, in *pbuser.IsIMAdminReq, opts ...grpc.CallOption) (*pbuser.IsIMAdminResp, error)
}

type defaultUserService struct {
	cli zrpc.Client
}

func NewUserService(cli zrpc.Client) UserService {
	return &defaultUserService{cli: cli}
}

// 获取指定用户信息(完整字段)
func (s *defaultUserService) GetDesignateUsers(ctx context.Context, in *pbuser.GetDesignateUsersReq, opts ...grpc.CallOption) (*pbuser.GetDesignateUsersResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.GetDesignateUsers(ctx, in, opts...)
}

// 更新用户信息
func (s *defaultUserService) UpdateUserInfo(ctx context.Context, in *pbuser.UpdateUserInfoReq, opts ...grpc.CallOption) (*pbuser.UpdateUserInfoResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.UpdateUserInfo(ctx, in, opts...)
}

// 更新用户信息(扩展)
func (s *defaultUserService) UpdateUserInfoEx(ctx context.Context, in *pbuser.UpdateUserInfoExReq, opts ...grpc.CallOption) (*pbuser.UpdateUserInfoExResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.UpdateUserInfoEx(ctx, in, opts...)
}

// 设置用户全局消息接收选项
// 设置用户全局消息接收选项
func (s *defaultUserService) SetGlobalRecvMessageOpt(ctx context.Context, in *pbuser.SetGlobalRecvMessageOptReq, opts ...grpc.CallOption) (*pbuser.SetGlobalRecvMessageOptResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.SetGlobalRecvMessageOpt(ctx, in, opts...)
}

// 获取用户全局消息接收选项(未找到时不报错)
func (s *defaultUserService) GetGlobalRecvMessageOpt(ctx context.Context, in *pbuser.GetGlobalRecvMessageOptReq, opts ...grpc.CallOption) (*pbuser.GetGlobalRecvMessageOptResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.GetGlobalRecvMessageOpt(ctx, in, opts...)
}

// 检查用户ID是否存在
func (s *defaultUserService) AccountCheck(ctx context.Context, in *pbuser.AccountCheckReq, opts ...grpc.CallOption) (*pbuser.AccountCheckResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.AccountCheck(ctx, in, opts...)
}

// 分页获取用户信息(或按用户ID、昵称查询)
func (s *defaultUserService) GetPaginationUsers(ctx context.Context, in *pbuser.GetPaginationUsersReq, opts ...grpc.CallOption) (*pbuser.GetPaginationUsersResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.GetPaginationUsers(ctx, in, opts...)
}

// 用户注册
func (s *defaultUserService) UserRegister(ctx context.Context, in *pbuser.UserRegisterReq, opts ...grpc.CallOption) (*pbuser.UserRegisterResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.UserRegister(ctx, in, opts...)
}

// 获取所有用户ID
func (s *defaultUserService) GetAllUserID(ctx context.Context, in *pbuser.GetAllUserIDReq, opts ...grpc.CallOption) (*pbuser.GetAllUserIDResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.GetAllUserID(ctx, in, opts...)
}

// 获取指定时间段内的用户总数和增量
// 用户注册数量统计
func (s *defaultUserService) UserRegisterCount(ctx context.Context, in *pbuser.UserRegisterCountReq, opts ...grpc.CallOption) (*pbuser.UserRegisterCountResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.UserRegisterCount(ctx, in, opts...)
}

// 订阅或取消订阅用户在线状态
func (s *defaultUserService) SubscribeOrCancelUsersStatus(ctx context.Context, in *pbuser.SubscribeOrCancelUsersStatusReq, opts ...grpc.CallOption) (*pbuser.SubscribeOrCancelUsersStatusResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.SubscribeOrCancelUsersStatus(ctx, in, opts...)
}

// 获取订阅用户的在线状态
func (s *defaultUserService) GetSubscribeUsersStatus(ctx context.Context, in *pbuser.GetSubscribeUsersStatusReq, opts ...grpc.CallOption) (*pbuser.GetSubscribeUsersStatusResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.GetSubscribeUsersStatus(ctx, in, opts...)
}

// 获取用户在线状态
func (s *defaultUserService) GetUserStatus(ctx context.Context, in *pbuser.GetUserStatusReq, opts ...grpc.CallOption) (*pbuser.GetUserStatusResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.GetUserStatus(ctx, in, opts...)
}

// 网关同步用户在线状态到Redis
func (s *defaultUserService) SetUserStatus(ctx context.Context, in *pbuser.SetUserStatusReq, opts ...grpc.CallOption) (*pbuser.SetUserStatusResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.SetUserStatus(ctx, in, opts...)
}

// 添加用户命令
func (s *defaultUserService) ProcessUserCommandAdd(ctx context.Context, in *pbuser.ProcessUserCommandAddReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandAddResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.ProcessUserCommandAdd(ctx, in, opts...)
}

// 更新用户命令
func (s *defaultUserService) ProcessUserCommandUpdate(ctx context.Context, in *pbuser.ProcessUserCommandUpdateReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandUpdateResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.ProcessUserCommandUpdate(ctx, in, opts...)
}

// 删除用户命令
func (s *defaultUserService) ProcessUserCommandDelete(ctx context.Context, in *pbuser.ProcessUserCommandDeleteReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandDeleteResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.ProcessUserCommandDelete(ctx, in, opts...)
}

// 获取用户命令
func (s *defaultUserService) ProcessUserCommandGet(ctx context.Context, in *pbuser.ProcessUserCommandGetReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandGetResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.ProcessUserCommandGet(ctx, in, opts...)
}

// 获取所有用户命令
func (s *defaultUserService) ProcessUserCommandGetAll(ctx context.Context, in *pbuser.ProcessUserCommandGetAllReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandGetAllResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.ProcessUserCommandGetAll(ctx, in, opts...)
}

// 添加系统通知账户
func (s *defaultUserService) AddNotificationAccount(ctx context.Context, in *pbuser.AddNotificationAccountReq, opts ...grpc.CallOption) (*pbuser.AddNotificationAccountResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.AddNotificationAccount(ctx, in, opts...)
}

// 更新系统通知账户信息
func (s *defaultUserService) UpdateNotificationAccountInfo(ctx context.Context, in *pbuser.UpdateNotificationAccountInfoReq, opts ...grpc.CallOption) (*pbuser.UpdateNotificationAccountInfoResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.UpdateNotificationAccountInfo(ctx, in, opts...)
}

// 搜索系统通知账户
func (s *defaultUserService) SearchNotificationAccount(ctx context.Context, in *pbuser.SearchNotificationAccountReq, opts ...grpc.CallOption) (*pbuser.SearchNotificationAccountResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.SearchNotificationAccount(ctx, in, opts...)
}

// 按用户ID获取通知账户
func (s *defaultUserService) GetNotificationAccount(ctx context.Context, in *pbuser.GetNotificationAccountReq, opts ...grpc.CallOption) (*pbuser.GetNotificationAccountResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.GetNotificationAccount(ctx, in, opts...)
}

// 排序查询用户
func (s *defaultUserService) SortQuery(ctx context.Context, in *pbuser.SortQueryReq, opts ...grpc.CallOption) (*pbuser.SortQueryResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.SortQuery(ctx, in, opts...)
}

// 批量设置用户在线状态
func (s *defaultUserService) SetUserOnlineStatus(ctx context.Context, in *pbuser.SetUserOnlineStatusReq, opts ...grpc.CallOption) (*pbuser.SetUserOnlineStatusResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.SetUserOnlineStatus(ctx, in, opts...)
}

// 获取所有在线用户
func (s *defaultUserService) GetAllOnlineUsers(ctx context.Context, in *pbuser.GetAllOnlineUsersReq, opts ...grpc.CallOption) (*pbuser.GetAllOnlineUsersResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.GetAllOnlineUsers(ctx, in, opts...)
}

// 获取用户客户端配置
func (s *defaultUserService) GetUserClientConfig(ctx context.Context, in *pbuser.GetUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.GetUserClientConfigResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.GetUserClientConfig(ctx, in, opts...)
}

// 设置用户客户端配置
func (s *defaultUserService) SetUserClientConfig(ctx context.Context, in *pbuser.SetUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.SetUserClientConfigResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.SetUserClientConfig(ctx, in, opts...)
}

// 删除用户客户端配置
func (s *defaultUserService) DelUserClientConfig(ctx context.Context, in *pbuser.DelUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.DelUserClientConfigResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.DelUserClientConfig(ctx, in, opts...)
}

// 分页获取用户客户端配置
func (s *defaultUserService) PageUserClientConfig(ctx context.Context, in *pbuser.PageUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.PageUserClientConfigResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.PageUserClientConfig(ctx, in, opts...)
}

// 是否为IM管理员
func (s *defaultUserService) IsIMAdmin(ctx context.Context, in *pbuser.IsIMAdminReq, opts ...grpc.CallOption) (*pbuser.IsIMAdminResp, error) {
	userClient := pbuser.NewUserClient(s.cli.Conn())
	return userClient.IsIMAdmin(ctx, in, opts...)
}

// stub
type stubUserService struct {
}

func NewStubUserService() UserService {
	return &stubUserService{}
}

// 获取指定用户信息(完整字段)
func (s *stubUserService) GetDesignateUsers(ctx context.Context, in *pbuser.GetDesignateUsersReq, opts ...grpc.CallOption) (*pbuser.GetDesignateUsersResp, error) {
	return &pbuser.GetDesignateUsersResp{}, nil
}

// 更新用户信息
func (s *stubUserService) UpdateUserInfo(ctx context.Context, in *pbuser.UpdateUserInfoReq, opts ...grpc.CallOption) (*pbuser.UpdateUserInfoResp, error) {
	return &pbuser.UpdateUserInfoResp{}, nil
}

// 更新用户信息(扩展)
func (s *stubUserService) UpdateUserInfoEx(ctx context.Context, in *pbuser.UpdateUserInfoExReq, opts ...grpc.CallOption) (*pbuser.UpdateUserInfoExResp, error) {
	return &pbuser.UpdateUserInfoExResp{}, nil
}

// 设置用户全局消息接收选项
// 设置用户全局消息接收选项
func (s *stubUserService) SetGlobalRecvMessageOpt(ctx context.Context, in *pbuser.SetGlobalRecvMessageOptReq, opts ...grpc.CallOption) (*pbuser.SetGlobalRecvMessageOptResp, error) {
	return &pbuser.SetGlobalRecvMessageOptResp{}, nil
}

// 获取用户全局消息接收选项(未找到时不报错)
func (s *stubUserService) GetGlobalRecvMessageOpt(ctx context.Context, in *pbuser.GetGlobalRecvMessageOptReq, opts ...grpc.CallOption) (*pbuser.GetGlobalRecvMessageOptResp, error) {
	return &pbuser.GetGlobalRecvMessageOptResp{}, nil
}

// 检查用户ID是否存在
func (s *stubUserService) AccountCheck(ctx context.Context, in *pbuser.AccountCheckReq, opts ...grpc.CallOption) (*pbuser.AccountCheckResp, error) {
	return &pbuser.AccountCheckResp{}, nil
}

// 分页获取用户信息(或按用户ID、昵称查询)
func (s *stubUserService) GetPaginationUsers(ctx context.Context, in *pbuser.GetPaginationUsersReq, opts ...grpc.CallOption) (*pbuser.GetPaginationUsersResp, error) {
	return &pbuser.GetPaginationUsersResp{}, nil
}

// 用户注册
func (s *stubUserService) UserRegister(ctx context.Context, in *pbuser.UserRegisterReq, opts ...grpc.CallOption) (*pbuser.UserRegisterResp, error) {
	return &pbuser.UserRegisterResp{}, nil
}

// 获取所有用户ID
func (s *stubUserService) GetAllUserID(ctx context.Context, in *pbuser.GetAllUserIDReq, opts ...grpc.CallOption) (*pbuser.GetAllUserIDResp, error) {
	return &pbuser.GetAllUserIDResp{}, nil
}

// 获取指定时间段内的用户总数和增量
// 用户注册数量统计
func (s *stubUserService) UserRegisterCount(ctx context.Context, in *pbuser.UserRegisterCountReq, opts ...grpc.CallOption) (*pbuser.UserRegisterCountResp, error) {
	return &pbuser.UserRegisterCountResp{}, nil
}

// 订阅或取消订阅用户在线状态
func (s *stubUserService) SubscribeOrCancelUsersStatus(ctx context.Context, in *pbuser.SubscribeOrCancelUsersStatusReq, opts ...grpc.CallOption) (*pbuser.SubscribeOrCancelUsersStatusResp, error) {
	return &pbuser.SubscribeOrCancelUsersStatusResp{}, nil
}

// 获取订阅用户的在线状态
func (s *stubUserService) GetSubscribeUsersStatus(ctx context.Context, in *pbuser.GetSubscribeUsersStatusReq, opts ...grpc.CallOption) (*pbuser.GetSubscribeUsersStatusResp, error) {
	return &pbuser.GetSubscribeUsersStatusResp{}, nil
}

// 获取用户在线状态
func (s *stubUserService) GetUserStatus(ctx context.Context, in *pbuser.GetUserStatusReq, opts ...grpc.CallOption) (*pbuser.GetUserStatusResp, error) {
	return &pbuser.GetUserStatusResp{}, nil
}

// 网关同步用户在线状态到Redis
func (s *stubUserService) SetUserStatus(ctx context.Context, in *pbuser.SetUserStatusReq, opts ...grpc.CallOption) (*pbuser.SetUserStatusResp, error) {
	return &pbuser.SetUserStatusResp{}, nil
}

// 添加用户命令
func (s *stubUserService) ProcessUserCommandAdd(ctx context.Context, in *pbuser.ProcessUserCommandAddReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandAddResp, error) {
	return &pbuser.ProcessUserCommandAddResp{}, nil
}

// 更新用户命令
func (s *stubUserService) ProcessUserCommandUpdate(ctx context.Context, in *pbuser.ProcessUserCommandUpdateReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandUpdateResp, error) {
	return &pbuser.ProcessUserCommandUpdateResp{}, nil
}

// 删除用户命令
func (s *stubUserService) ProcessUserCommandDelete(ctx context.Context, in *pbuser.ProcessUserCommandDeleteReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandDeleteResp, error) {
	return &pbuser.ProcessUserCommandDeleteResp{}, nil
}

// 获取用户命令
func (s *stubUserService) ProcessUserCommandGet(ctx context.Context, in *pbuser.ProcessUserCommandGetReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandGetResp, error) {
	return &pbuser.ProcessUserCommandGetResp{}, nil
}

// 获取所有用户命令
func (s *stubUserService) ProcessUserCommandGetAll(ctx context.Context, in *pbuser.ProcessUserCommandGetAllReq, opts ...grpc.CallOption) (*pbuser.ProcessUserCommandGetAllResp, error) {
	return &pbuser.ProcessUserCommandGetAllResp{}, nil
}

// 添加系统通知账户
func (s *stubUserService) AddNotificationAccount(ctx context.Context, in *pbuser.AddNotificationAccountReq, opts ...grpc.CallOption) (*pbuser.AddNotificationAccountResp, error) {
	return &pbuser.AddNotificationAccountResp{}, nil
}

// 更新系统通知账户信息
func (s *stubUserService) UpdateNotificationAccountInfo(ctx context.Context, in *pbuser.UpdateNotificationAccountInfoReq, opts ...grpc.CallOption) (*pbuser.UpdateNotificationAccountInfoResp, error) {
	return &pbuser.UpdateNotificationAccountInfoResp{}, nil
}

// 搜索系统通知账户
func (s *stubUserService) SearchNotificationAccount(ctx context.Context, in *pbuser.SearchNotificationAccountReq, opts ...grpc.CallOption) (*pbuser.SearchNotificationAccountResp, error) {
	return &pbuser.SearchNotificationAccountResp{}, nil
}

// 按用户ID获取通知账户
func (s *stubUserService) GetNotificationAccount(ctx context.Context, in *pbuser.GetNotificationAccountReq, opts ...grpc.CallOption) (*pbuser.GetNotificationAccountResp, error) {
	return &pbuser.GetNotificationAccountResp{}, nil
}

// 排序查询用户
func (s *stubUserService) SortQuery(ctx context.Context, in *pbuser.SortQueryReq, opts ...grpc.CallOption) (*pbuser.SortQueryResp, error) {
	return &pbuser.SortQueryResp{}, nil
}

// 批量设置用户在线状态
func (s *stubUserService) SetUserOnlineStatus(ctx context.Context, in *pbuser.SetUserOnlineStatusReq, opts ...grpc.CallOption) (*pbuser.SetUserOnlineStatusResp, error) {
	return &pbuser.SetUserOnlineStatusResp{}, nil
}

// 获取所有在线用户
func (s *stubUserService) GetAllOnlineUsers(ctx context.Context, in *pbuser.GetAllOnlineUsersReq, opts ...grpc.CallOption) (*pbuser.GetAllOnlineUsersResp, error) {
	return &pbuser.GetAllOnlineUsersResp{}, nil
}

// 获取用户客户端配置
func (s *stubUserService) GetUserClientConfig(ctx context.Context, in *pbuser.GetUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.GetUserClientConfigResp, error) {
	return &pbuser.GetUserClientConfigResp{}, nil
}

// 设置用户客户端配置
func (s *stubUserService) SetUserClientConfig(ctx context.Context, in *pbuser.SetUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.SetUserClientConfigResp, error) {
	return &pbuser.SetUserClientConfigResp{}, nil
}

// 删除用户客户端配置
func (s *stubUserService) DelUserClientConfig(ctx context.Context, in *pbuser.DelUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.DelUserClientConfigResp, error) {
	return &pbuser.DelUserClientConfigResp{}, nil
}

// 分页获取用户客户端配置
func (s *stubUserService) PageUserClientConfig(ctx context.Context, in *pbuser.PageUserClientConfigReq, opts ...grpc.CallOption) (*pbuser.PageUserClientConfigResp, error) {
	return &pbuser.PageUserClientConfigResp{}, nil
}

func (s *stubUserService) IsIMAdmin(ctx context.Context, in *pbuser.IsIMAdminReq, opts ...grpc.CallOption) (*pbuser.IsIMAdminResp, error) {
	return &pbuser.IsIMAdminResp{IsIMAdmin: true}, nil
}
