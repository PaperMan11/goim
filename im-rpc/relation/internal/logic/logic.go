package logic

import (
	"context"
	"hash/fnv"
	"strings"

	"github.com/PaperMan11/goim/im-rpc/relation/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	sdkws "github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	SyncLimit = 200
)

type Logic struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logx.Logger
}

func NewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *Logic {
	return &Logic{
		svcCtx: svcCtx,
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *Logic) requireSelfOrAdmin(targetUserID string) error {
	opUserID := mcontext.GetOpUserIDFromContext(l.ctx)
	ok, err := l.svcCtx.AuthVerifier.CheckAccess(l.ctx, targetUserID)
	if err != nil {
		l.Errorf("check access failed, opUserID=%s targetUserID=%s err=%v", opUserID, targetUserID, err)
		return errx.InternalError.WrapWithError(err)
	}
	if !ok {
		l.Errorf("access denied, opUserID=%s targetUserID=%s", opUserID, targetUserID)
		return errx.NoPermissionError
	}
	return nil
}

func (l *Logic) requireAdmin() error {
	opUserID := mcontext.GetOpUserIDFromContext(l.ctx)
	ok, err := l.svcCtx.AuthVerifier.IsIMAdmin(l.ctx, opUserID)
	if err != nil {
		l.Errorf("check admin failed, opUserID=%s err=%v", opUserID, err)
		return errx.InternalError.WrapWithError(err)
	}
	if !ok {
		l.Errorf("not admin, opUserID=%s", opUserID)
		return errx.NoPermissionError
	}
	return nil
}

func (l *Logic) requireValidUser(targetUserID string) error {
	ok, err := l.svcCtx.AuthVerifier.IsValidUser(l.ctx, targetUserID)
	if err != nil {
		l.Errorf("check valid user failed, targetUserID=%s err=%v", targetUserID, err)
		return errx.InternalError.WrapWithError(err)
	}
	if !ok {
		l.Errorf("invalid user, targetUserID=%s", targetUserID)
		return errx.UserIDNotFoundError
	}
	return nil
}

func modelToFriendInfo(f *model.Friend) *sdkws.FriendInfo {
	if f == nil {
		return nil
	}
	return &sdkws.FriendInfo{
		OwnerUserID: f.OwnerUserID,
		FriendUser: &sdkws.UserInfo{
			UserID: f.FriendUserID,
		},
		Remark:         f.Remark,
		CreateTime:     f.CreateTime.Unix(),
		AddSource:      int32(f.AddSource),
		OperatorUserID: f.OperatorUserID,
		Ex:             f.Extra,
		IsPinned:       f.IsPinned,
	}
}

func modelToBlackInfo(b *model.Black) *sdkws.BlackInfo {
	if b == nil {
		return nil
	}
	return &sdkws.BlackInfo{
		OwnerUserID: b.OwnerUserID,
		BlackUserInfo: &sdkws.PublicUserInfo{
			UserID: b.BlackUserID,
		},
		CreateTime:     b.CreateTime.Unix(),
		AddSource:      int32(b.AddSource),
		OperatorUserID: b.OperatorUserID,
		Ex:             b.Extra,
	}
}

// hashIDs 计算有序ID列表的 FNV-1a 哈希，用于全量同步的等值比较。
func hashIDs(ids []string) uint64 {
	if len(ids) == 0 {
		return 0
	}
	h := fnv.New64a()
	h.Write([]byte(strings.Join(ids, ",")))
	return h.Sum64()
}
