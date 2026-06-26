package server

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/logic"
	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/auth"
)

type AuthServer struct {
	svcCtx *svc.ServiceContext
	auth.UnimplementedAuthServer
}

func NewAuthServer(svcCtx *svc.ServiceContext) *AuthServer {
	return &AuthServer{
		svcCtx: svcCtx,
	}
}

func (s *AuthServer) GetAdminToken(ctx context.Context, req *auth.GetAdminTokenReq) (*auth.GetAdminTokenResp, error) {
	logic := logic.NewGetAdminTokenLogic(ctx, s.svcCtx)
	return logic.GetAdminToken(req)
}

func (s *AuthServer) GetUserToken(ctx context.Context, req *auth.GetUserTokenReq) (*auth.GetUserTokenResp, error) {
	logic := logic.NewGetUserTokenLogic(ctx, s.svcCtx)
	return logic.GetUserToken(req)
}

func (s *AuthServer) ForceLogout(ctx context.Context, req *auth.ForceLogoutReq) (*auth.ForceLogoutResp, error) {
	logic := logic.NewForceLogoutLogic(ctx, s.svcCtx)
	return logic.ForceLogout(req)
}

func (s *AuthServer) ParseToken(ctx context.Context, req *auth.ParseTokenReq) (*auth.ParseTokenResp, error) {
	logic := logic.NewParseTokenLogic(ctx, s.svcCtx)
	return logic.ParseToken(req)
}

func (s *AuthServer) InvalidateToken(ctx context.Context, req *auth.InvalidateTokenReq) (*auth.InvalidateTokenResp, error) {
	logic := logic.NewInvalidateTokenLogic(ctx, s.svcCtx)
	return logic.InvalidateToken(req)
}

func (s *AuthServer) KickTokens(ctx context.Context, req *auth.KickTokensReq) (*auth.KickTokensResp, error) {
	logic := logic.NewKickTokensLogic(ctx, s.svcCtx)
	return logic.KickTokens(req)
}

func (s *AuthServer) GetExistingToken(ctx context.Context, req *auth.GetExistingTokenReq) (*auth.GetExistingTokenResp, error) {
	logic := logic.NewGetExistingTokenLogic(ctx, s.svcCtx)
	return logic.GetExistingToken(req)
}
