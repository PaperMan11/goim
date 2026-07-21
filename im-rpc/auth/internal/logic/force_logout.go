package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/auth/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/auth"
	pbmsggateway "github.com/PaperMan11/goim/pkg/protocol/msggateway"
	"github.com/zeromicro/go-zero/core/logx"
)

type ForceLogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewForceLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ForceLogoutLogic {
	return &ForceLogoutLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ForceLogoutLogic) ForceLogout(req *auth.ForceLogoutReq) (*auth.ForceLogoutResp, error) {
	if err := requireUserIsAdmin(l.ctx, l.svcCtx, l, req.UserID); err != nil {
		return nil, err
	}

	// kick user offline
	_, err := l.svcCtx.MsgGatewayService.KickUserOffline(l.ctx, &pbmsggateway.KickUserOfflineReq{
		PlatformID:     req.PlatformID,
		KickUserIDList: []string{req.UserID},
	})
	if err != nil {
		l.Errorf("failed to kick user offline, platformID: %s, userID: %s, err: %v", req.PlatformID, req.UserID, err)
		return nil, err
	}

	// delete user tokens
	err = l.svcCtx.TokenStore.DeleteUserTokens(l.ctx, req.UserID, req.PlatformID)
	if err != nil {
		l.Errorf("failed to delete user tokens, platformID: %s, userID: %s, err: %v", req.PlatformID, req.UserID, err)
		return nil, err
	}
	return &auth.ForceLogoutResp{}, nil
}
