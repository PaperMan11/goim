package mconvert

import (
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
)

func ModelToPbUserInfo(user *model.User) *sdkws.UserInfo {
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

func PbToModelUserInfo(userInfo *sdkws.UserInfo) *model.User {
	return &model.User{
		UserID:           userInfo.GetUserID(),
		Nickname:         userInfo.GetNickname(),
		FaceURL:          userInfo.GetFaceURL(),
		Extra:            userInfo.GetEx(),
		AppManagerLevel:  int(userInfo.GetAppMangerLevel()),
		GlobalRecvMsgOpt: int(userInfo.GetGlobalRecvMsgOpt()),
	}
}
