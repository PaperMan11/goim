package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/PaperMan11/goim/im-rpc/msg/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
	hashx "github.com/PaperMan11/goim/pkg/utils/hash"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/logx"
)

type Logic struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logx.Logger
}

func NewLogic(ctx context.Context, svc *svc.ServiceContext) *Logic {
	return &Logic{
		svcCtx: svc,
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
	}
}

// requireSelfOrAdmin 校验：操作人必须是 targetUserID 本人，或是 IM 管理员。
// 校验失败时自动打 error log 并返回统一错误。
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

// requireAdmin 校验：操作人必须是 IM 管理员。
// 校验失败时自动打 error log 并返回统一错误。
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

// requireValidUser 校验：targetUserID 是真实存在的有效用户（未注销）。
// 校验失败时自动打 error log 并返回统一错误。
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

// ToSDKMsg 将 MsgDataModel 转为 sdkws.MsgData
func (l *Logic) ToSDKMsg(msg *model.MsgDataModel) *sdkws.MsgData {
	return &sdkws.MsgData{
		SendID:           msg.SendID,
		RecvID:           msg.RecvID,
		GroupID:          msg.GroupID,
		ClientMsgID:      msg.ClientMsgID,
		ServerMsgID:      msg.ServerMsgID,
		SenderPlatformID: msg.SenderPlatformID,
		SenderNickname:   msg.SenderNickname,
		SenderFaceURL:    msg.SenderFaceURL,
		SessionType:      msg.SessionType,
		MsgFrom:          msg.MsgFrom,
		ContentType:      msg.ContentType,
		Content:          []byte(msg.Content),
		Seq:              msg.Seq,
		SendTime:         msg.SendTime,
		CreateTime:       msg.CreateTime,
		Status:           msg.Status,
		IsRead:           msg.IsRead,
		Options:          msg.Options,
		AtUserIDList:     msg.AtUserIDList,
		AttachedInfo:     msg.AttachedInfo,
		Ex:               msg.Ex,
		OfflinePushInfo: &sdkws.OfflinePushInfo{
			Title:         msg.OfflinePush.Title,
			Desc:          msg.OfflinePush.Desc,
			Ex:            msg.OfflinePush.Ex,
			IOSPushSound:  msg.OfflinePush.IOSPushSound,
			IOSBadgeCount: msg.OfflinePush.IOSBadgeCount,
			SignalInfo:    "",
		},
	}
}

func modelToChatLog(msg *model.MsgInfoModel) *pbmsg.ChatLog {
	if msg.Msg == nil {
		return nil
	}
	return &pbmsg.ChatLog{
		ServerMsgID:      msg.Msg.ServerMsgID,
		ClientMsgID:      msg.Msg.ClientMsgID,
		SendID:           msg.Msg.SendID,
		RecvID:           msg.Msg.RecvID,
		GroupID:          msg.Msg.GroupID,
		RecvNickname:     "",
		SenderPlatformID: msg.Msg.SenderPlatformID,
		SenderNickname:   msg.Msg.SenderNickname,
		SenderFaceURL:    msg.Msg.SenderFaceURL,
		GroupName:        "",
		SessionType:      msg.Msg.SessionType,
		MsgFrom:          msg.Msg.MsgFrom,
		ContentType:      msg.Msg.ContentType,
		Content:          msg.Msg.Content,
		Status:           msg.Msg.Status,
		SendTime:         msg.Msg.SendTime,
		CreateTime:       msg.Msg.CreateTime,
		Ex:               msg.Msg.Ex,
		GroupFaceURL:     "",
		GroupMemberCount: 0,
		Seq:              msg.Msg.Seq,
		GroupOwner:       "",
		GroupType:        0,
	}
}

func generateServerMsgID() string {
	timeFormat := timex.Format(time.DateTime)
	return hashx.Md5String(fmt.Sprintf("%s-%s", timeFormat, randx.AlphaString(8)))
}
