package mconvert

import (
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
)

func ModelToPbFriendInfo(f *model.Friend) *sdkws.FriendInfo {
	if f == nil {
		return nil
	}
	return &sdkws.FriendInfo{
		OwnerUserID: f.OwnerUserID,
		FriendUser: &sdkws.UserInfo{
			UserID: f.FriendUserID,
		},
		Remark:         f.Remark,
		CreateTime:     f.CreateTime.UnixMilli(),
		AddSource:      int32(f.AddSource),
		OperatorUserID: f.OperatorUserID,
		Ex:             f.Extra,
		IsPinned:       f.IsPinned,
	}
}

func ModelToPbBlackInfo(b *model.Black) *sdkws.BlackInfo {
	if b == nil {
		return nil
	}
	return &sdkws.BlackInfo{
		OwnerUserID: b.OwnerUserID,
		BlackUserInfo: &sdkws.PublicUserInfo{
			UserID: b.BlackUserID,
		},
		CreateTime:     b.CreateTime.UnixMilli(),
		AddSource:      int32(b.AddSource),
		OperatorUserID: b.OperatorUserID,
		Ex:             b.Extra,
	}
}

func ModelToPbFriendRequest(r *model.FriendRequest) *sdkws.FriendRequest {
	if r == nil {
		return nil
	}
	return &sdkws.FriendRequest{
		FromUserID:    r.FromUserID,
		FromNickname:  r.FromNickname,           // 申请人昵称
		FromFaceURL:   r.FromFaceURL,            // 申请人头像URL
		ToUserID:      r.ToUserID,               // 被申请人ID
		ToNickname:    r.ToNickname,             // 被申请人昵称
		ToFaceURL:     r.ToFaceURL,              // 被申请人头像URL
		HandleResult:  int32(r.HandleResult),    // 处理结果(0-未处理, 1-已同意, -1-已拒绝)
		ReqMsg:        r.ReqMsg,                 // 申请消息
		CreateTime:    r.CreateTime.UnixMilli(), // 创建时间戳
		HandlerUserID: r.HandlerUserID,          // 处理人ID
		HandleMsg:     r.HandleMsg,              // 处理消息
		HandleTime:    r.HandleTime.UnixMilli(), // 处理时间戳
		Ex:            r.Extra,                  // 扩展字段
	}
}
