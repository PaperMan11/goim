package authservice

import (
	"context"

	pbauth "github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type AuthService interface {
	// Generate token
	GetAdminToken(ctx context.Context, in *pbauth.GetAdminTokenReq, opts ...grpc.CallOption) (*pbauth.GetAdminTokenResp, error)
	// Admin retrieves user token
	GetUserToken(ctx context.Context, in *pbauth.GetUserTokenReq, opts ...grpc.CallOption) (*pbauth.GetUserTokenResp, error)
	// Force logout
	ForceLogout(ctx context.Context, in *pbauth.ForceLogoutReq, opts ...grpc.CallOption) (*pbauth.ForceLogoutResp, error)
	// Parse token
	ParseToken(ctx context.Context, in *pbauth.ParseTokenReq, opts ...grpc.CallOption) (*pbauth.ParseTokenResp, error)
	// Invalidate or mark the token as kicked out
	InvalidateToken(ctx context.Context, in *pbauth.InvalidateTokenReq, opts ...grpc.CallOption) (*pbauth.InvalidateTokenResp, error)
	// kick tokens
	KickTokens(ctx context.Context, in *pbauth.KickTokensReq, opts ...grpc.CallOption) (*pbauth.KickTokensResp, error)
	// Get existing token
	GetExistingToken(ctx context.Context, in *pbauth.GetExistingTokenReq, opts ...grpc.CallOption) (*pbauth.GetExistingTokenResp, error)
}

type defaultAuthService struct {
	cli zrpc.Client
}

func NewAuthService(cli zrpc.Client) AuthService {
	return &defaultAuthService{cli: cli}
}

func (s *defaultAuthService) GetAdminToken(ctx context.Context, in *pbauth.GetAdminTokenReq, opts ...grpc.CallOption) (*pbauth.GetAdminTokenResp, error) {
	authClient := pbauth.NewAuthClient(s.cli.Conn())
	return authClient.GetAdminToken(ctx, in, opts...)
}

func (s *defaultAuthService) GetUserToken(ctx context.Context, in *pbauth.GetUserTokenReq, opts ...grpc.CallOption) (*pbauth.GetUserTokenResp, error) {
	authClient := pbauth.NewAuthClient(s.cli.Conn())
	return authClient.GetUserToken(ctx, in, opts...)
}

func (s *defaultAuthService) ForceLogout(ctx context.Context, in *pbauth.ForceLogoutReq, opts ...grpc.CallOption) (*pbauth.ForceLogoutResp, error) {
	authClient := pbauth.NewAuthClient(s.cli.Conn())
	return authClient.ForceLogout(ctx, in, opts...)
}

func (s *defaultAuthService) ParseToken(ctx context.Context, in *pbauth.ParseTokenReq, opts ...grpc.CallOption) (*pbauth.ParseTokenResp, error) {
	authClient := pbauth.NewAuthClient(s.cli.Conn())
	return authClient.ParseToken(ctx, in, opts...)
}

func (s *defaultAuthService) InvalidateToken(ctx context.Context, in *pbauth.InvalidateTokenReq, opts ...grpc.CallOption) (*pbauth.InvalidateTokenResp, error) {
	authClient := pbauth.NewAuthClient(s.cli.Conn())
	return authClient.InvalidateToken(ctx, in, opts...)
}

func (s *defaultAuthService) KickTokens(ctx context.Context, in *pbauth.KickTokensReq, opts ...grpc.CallOption) (*pbauth.KickTokensResp, error) {
	authClient := pbauth.NewAuthClient(s.cli.Conn())
	return authClient.KickTokens(ctx, in, opts...)
}

func (s *defaultAuthService) GetExistingToken(ctx context.Context, in *pbauth.GetExistingTokenReq, opts ...grpc.CallOption) (*pbauth.GetExistingTokenResp, error) {
	authClient := pbauth.NewAuthClient(s.cli.Conn())
	return authClient.GetExistingToken(ctx, in, opts...)
}

// stub
type stubAuthService struct {
}

func NewStubAuthService() AuthService {
	return &stubAuthService{}
}

func (s *stubAuthService) GetAdminToken(ctx context.Context, in *pbauth.GetAdminTokenReq, opts ...grpc.CallOption) (*pbauth.GetAdminTokenResp, error) {
	return &pbauth.GetAdminTokenResp{}, nil
}

func (s *stubAuthService) GetUserToken(ctx context.Context, in *pbauth.GetUserTokenReq, opts ...grpc.CallOption) (*pbauth.GetUserTokenResp, error) {
	return &pbauth.GetUserTokenResp{}, nil
}

func (s *stubAuthService) ForceLogout(ctx context.Context, in *pbauth.ForceLogoutReq, opts ...grpc.CallOption) (*pbauth.ForceLogoutResp, error) {
	return &pbauth.ForceLogoutResp{}, nil
}

func (s *stubAuthService) ParseToken(ctx context.Context, in *pbauth.ParseTokenReq, opts ...grpc.CallOption) (*pbauth.ParseTokenResp, error) {
	return &pbauth.ParseTokenResp{}, nil
}

func (s *stubAuthService) InvalidateToken(ctx context.Context, in *pbauth.InvalidateTokenReq, opts ...grpc.CallOption) (*pbauth.InvalidateTokenResp, error) {
	return &pbauth.InvalidateTokenResp{}, nil
}

func (s *stubAuthService) KickTokens(ctx context.Context, in *pbauth.KickTokensReq, opts ...grpc.CallOption) (*pbauth.KickTokensResp, error) {
	return &pbauth.KickTokensResp{}, nil
}

func (s *stubAuthService) GetExistingToken(ctx context.Context, in *pbauth.GetExistingTokenReq, opts ...grpc.CallOption) (*pbauth.GetExistingTokenResp, error) {
	return &pbauth.GetExistingTokenResp{}, nil
}
