package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/user/internal/svc"
	sdkws "github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/zeromicro/go-zero/core/logx"
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

func modelToUserInfo(user *model.User) *sdkws.UserInfo {
	return &sdkws.UserInfo{
		UserID:           user.UserID,
		Nickname:         user.Nickname,
		FaceURL:          user.FaceURL,
		Ex:               user.Extra,
		CreateTime:       user.CreatedAt.Unix(),
		AppMangerLevel:   int32(user.AppManagerLevel),
		GlobalRecvMsgOpt: int32(user.GlobalRecvMsgOpt),
	}
}

func userInfoToModel(userInfo *sdkws.UserInfo) *model.User {
	return &model.User{
		UserID:           userInfo.GetUserID(),
		Nickname:         userInfo.GetNickname(),
		FaceURL:          userInfo.GetFaceURL(),
		Extra:            userInfo.GetEx(),
		AppManagerLevel:  int(userInfo.GetAppMangerLevel()),
		GlobalRecvMsgOpt: int(userInfo.GetGlobalRecvMsgOpt()),
	}
}

func boolToStatus(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
