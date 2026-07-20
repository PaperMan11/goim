package server

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/user/internal/logic"
	"github.com/PaperMan11/goim/im-rpc/user/internal/svc"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
)

type UserServer struct {
	svcCtx *svc.ServiceContext
	pbuser.UnimplementedUserServer
}

func NewUserServer(svcCtx *svc.ServiceContext) *UserServer {
	return &UserServer{svcCtx: svcCtx}
}

func (s *UserServer) GetDesignateUsers(ctx context.Context, req *pbuser.GetDesignateUsersReq) (*pbuser.GetDesignateUsersResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetDesignateUsers(ctx, req)
}

func (s *UserServer) UpdateUserInfo(ctx context.Context, req *pbuser.UpdateUserInfoReq) (*pbuser.UpdateUserInfoResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).UpdateUserInfo(ctx, req)
}

func (s *UserServer) UpdateUserInfoEx(ctx context.Context, req *pbuser.UpdateUserInfoExReq) (*pbuser.UpdateUserInfoExResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).UpdateUserInfoEx(ctx, req)
}

func (s *UserServer) SetGlobalRecvMessageOpt(ctx context.Context, req *pbuser.SetGlobalRecvMessageOptReq) (*pbuser.SetGlobalRecvMessageOptResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetGlobalRecvMessageOpt(ctx, req)
}

func (s *UserServer) GetGlobalRecvMessageOpt(ctx context.Context, req *pbuser.GetGlobalRecvMessageOptReq) (*pbuser.GetGlobalRecvMessageOptResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetGlobalRecvMessageOpt(ctx, req)
}

func (s *UserServer) AccountCheck(ctx context.Context, req *pbuser.AccountCheckReq) (*pbuser.AccountCheckResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).AccountCheck(ctx, req)
}

func (s *UserServer) GetPaginationUsers(ctx context.Context, req *pbuser.GetPaginationUsersReq) (*pbuser.GetPaginationUsersResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetPaginationUsers(ctx, req)
}

func (s *UserServer) UserRegister(ctx context.Context, req *pbuser.UserRegisterReq) (*pbuser.UserRegisterResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).UserRegister(ctx, req)
}

func (s *UserServer) GetAllUserID(ctx context.Context, req *pbuser.GetAllUserIDReq) (*pbuser.GetAllUserIDResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetAllUserID(ctx, req)
}

func (s *UserServer) UserRegisterCount(ctx context.Context, req *pbuser.UserRegisterCountReq) (*pbuser.UserRegisterCountResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).UserRegisterCount(ctx, req)
}

func (s *UserServer) SubscribeOrCancelUsersStatus(ctx context.Context, req *pbuser.SubscribeOrCancelUsersStatusReq) (*pbuser.SubscribeOrCancelUsersStatusResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SubscribeOrCancelUsersStatus(ctx, req)
}

func (s *UserServer) GetSubscribeUsersStatus(ctx context.Context, req *pbuser.GetSubscribeUsersStatusReq) (*pbuser.GetSubscribeUsersStatusResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetSubscribeUsersStatus(ctx, req)
}

func (s *UserServer) GetUserStatus(ctx context.Context, req *pbuser.GetUserStatusReq) (*pbuser.GetUserStatusResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetUserStatus(ctx, req)
}

func (s *UserServer) SetUserStatus(ctx context.Context, req *pbuser.SetUserStatusReq) (*pbuser.SetUserStatusResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetUserStatus(ctx, req)
}

func (s *UserServer) ProcessUserCommandAdd(ctx context.Context, req *pbuser.ProcessUserCommandAddReq) (*pbuser.ProcessUserCommandAddResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).ProcessUserCommandAdd(ctx, req)
}

func (s *UserServer) ProcessUserCommandUpdate(ctx context.Context, req *pbuser.ProcessUserCommandUpdateReq) (*pbuser.ProcessUserCommandUpdateResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).ProcessUserCommandUpdate(ctx, req)
}

func (s *UserServer) ProcessUserCommandDelete(ctx context.Context, req *pbuser.ProcessUserCommandDeleteReq) (*pbuser.ProcessUserCommandDeleteResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).ProcessUserCommandDelete(ctx, req)
}

func (s *UserServer) ProcessUserCommandGet(ctx context.Context, req *pbuser.ProcessUserCommandGetReq) (*pbuser.ProcessUserCommandGetResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).ProcessUserCommandGet(ctx, req)
}

func (s *UserServer) ProcessUserCommandGetAll(ctx context.Context, req *pbuser.ProcessUserCommandGetAllReq) (*pbuser.ProcessUserCommandGetAllResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).ProcessUserCommandGetAll(ctx, req)
}

func (s *UserServer) AddNotificationAccount(ctx context.Context, req *pbuser.AddNotificationAccountReq) (*pbuser.AddNotificationAccountResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).AddNotificationAccount(ctx, req)
}

func (s *UserServer) UpdateNotificationAccountInfo(ctx context.Context, req *pbuser.UpdateNotificationAccountInfoReq) (*pbuser.UpdateNotificationAccountInfoResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).UpdateNotificationAccountInfo(ctx, req)
}

func (s *UserServer) SearchNotificationAccount(ctx context.Context, req *pbuser.SearchNotificationAccountReq) (*pbuser.SearchNotificationAccountResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SearchNotificationAccount(ctx, req)
}

func (s *UserServer) GetNotificationAccount(ctx context.Context, req *pbuser.GetNotificationAccountReq) (*pbuser.GetNotificationAccountResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetNotificationAccount(ctx, req)
}

func (s *UserServer) SortQuery(ctx context.Context, req *pbuser.SortQueryReq) (*pbuser.SortQueryResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SortQuery(ctx, req)
}

func (s *UserServer) SetUserOnlineStatus(ctx context.Context, req *pbuser.SetUserOnlineStatusReq) (*pbuser.SetUserOnlineStatusResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetUserOnlineStatus(ctx, req)
}

func (s *UserServer) GetAllOnlineUsers(ctx context.Context, req *pbuser.GetAllOnlineUsersReq) (*pbuser.GetAllOnlineUsersResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetAllOnlineUsers(ctx, req)
}

func (s *UserServer) GetUserClientConfig(ctx context.Context, req *pbuser.GetUserClientConfigReq) (*pbuser.GetUserClientConfigResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetUserClientConfig(ctx, req)
}

func (s *UserServer) SetUserClientConfig(ctx context.Context, req *pbuser.SetUserClientConfigReq) (*pbuser.SetUserClientConfigResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetUserClientConfig(ctx, req)
}

func (s *UserServer) DelUserClientConfig(ctx context.Context, req *pbuser.DelUserClientConfigReq) (*pbuser.DelUserClientConfigResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).DelUserClientConfig(ctx, req)
}

func (s *UserServer) PageUserClientConfig(ctx context.Context, req *pbuser.PageUserClientConfigReq) (*pbuser.PageUserClientConfigResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).PageUserClientConfig(ctx, req)
}

func (s *UserServer) IsIMAdmin(ctx context.Context, req *pbuser.IsIMAdminReq) (*pbuser.IsIMAdminResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).IsIMAdmin(ctx, req)
}
