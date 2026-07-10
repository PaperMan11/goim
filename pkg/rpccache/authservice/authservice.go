package authservice

import (
	"context"

	"github.com/PaperMan11/goim/pkg/localcache"
	pbauth "github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/PaperMan11/goim/pkg/rpcclient/authservice"
	"google.golang.org/grpc"
)

type AuthServiceWrapperCache interface {
	authservice.AuthService
	GetExistingUserTokens(ctx context.Context, userID string, platformID int32) (map[string]int32, error)
}

type AuthService struct {
	authservice.AuthService
	localCache localcache.LocalCache
}

func NewAuthServiceWrapperCache(authService authservice.AuthService, cache localcache.LocalCache) AuthServiceWrapperCache {
	return &AuthService{
		AuthService: authService,
		localCache:  cache,
	}
}

func (s *AuthService) ForceLogout(ctx context.Context, in *pbauth.ForceLogoutReq, opts ...grpc.CallOption) (*pbauth.ForceLogoutResp, error) {
	if s.localCache != nil && in.UserID != "" {
		key := GetUserTokensKey(in.UserID, in.PlatformID)
		s.localCache.PublishDelete([]string{key})

	}
	return s.AuthService.ForceLogout(ctx, in, opts...)
}

func (s *AuthService) GetExistingUserTokens(ctx context.Context, userID string, platformID int32) (map[string]int32, error) {
	fetch := func() (map[string]int32, error) {
		resp, err := s.AuthService.GetExistingToken(ctx, &pbauth.GetExistingTokenReq{
			UserID:     userID,
			PlatformID: platformID,
		})
		if err != nil {
			return nil, err
		}

		tokens := make(map[string]int32)
		for token, state := range resp.TokenStates {
			tokens[token] = state
		}
		return tokens, nil
	}

	if s.localCache == nil {
		return fetch()
	}

	key := GetUserTokensKey(userID, platformID)
	tokens, err := s.localCache.Take(key, func() (any, error) {
		return fetch()
	})
	if err != nil {
		return nil, err
	}

	return tokens.(map[string]int32), nil
}
