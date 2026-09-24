package mconvert

import (
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
)

// ModelToPbMsgData 将 MsgDataModel 转为 sdkws.MsgData
func ModelToPbMsgData(msg *model.MsgDataModel) *sdkws.MsgData {
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

func ModelToPbChatLog(msg *model.MsgInfoModel) *pbmsg.ChatLog {
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

func PbToModelMsgData(msg *sdkws.MsgData) *model.MsgDataModel {
	return &model.MsgDataModel{
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
		Content:          string(msg.Content),
		Seq:              msg.Seq,
		SendTime:         msg.SendTime,
		Status:           msg.Status,
		IsRead:           msg.IsRead,
		CreateTime:       msg.CreateTime,
		Options:          msg.Options, // 消息选项
		OfflinePush: &model.OfflinePushModel{
			Title:         msg.OfflinePushInfo.Title,
			Desc:          msg.OfflinePushInfo.Desc,
			Ex:            msg.OfflinePushInfo.Ex,
			IOSPushSound:  msg.OfflinePushInfo.IOSPushSound,
			IOSBadgeCount: msg.OfflinePushInfo.IOSBadgeCount,
		}, // 离线推送信息
		AtUserIDList: msg.AtUserIDList, // @用户ID列表
		AttachedInfo: msg.AttachedInfo, // 附加信息
		Ex:           msg.Ex,
	}
}
