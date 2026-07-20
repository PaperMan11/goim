package logic

import (
	"context"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
)

func (l *Logic) sendMsg(ctx context.Context, msgData *sdkws.MsgData) error {
	return nil
}

// 验证消息是否符合要求
func (l *Logic) validateMsg(ctx context.Context, msgData *sdkws.MsgData) error {
	switch msgData.SessionType {
	case constant.SingleChatType:
	case constant.ReadGroupChatType:
	}
	return nil
}

func (l *Logic) SendMsg(ctx context.Context, req *pbmsg.SendMsgReq) (*pbmsg.SendMsgResp, error) {
	msgData := req.MsgData
	ok, err := l.svcCtx.AuthVerifier.CheckAccess(ctx, msgData.SendID)
	if err != nil {
		l.Errorf("check access error: %v", err)
		return nil, errx.InternalError.WrapWithError(err)
	}
	if !ok {
		return nil, errx.NoPermissionError
	}

	return &pbmsg.SendMsgResp{
		ServerMsgID: msgData.ServerMsgID,
		ClientMsgID: msgData.ClientMsgID,
		SendTime:    msgData.SendTime,
		Modify:      msgData,
	}, nil
}

func (l *Logic) SendSimpleMsg(ctx context.Context, req *pbmsg.SendSimpleMsgReq) (*pbmsg.SendSimpleMsgResp, error) {
	msgData := req.MsgData
	if msgData.ServerMsgID == "" {
		msgData.ServerMsgID = generateServerMsgID()
	}
	if msgData.SendTime == 0 {
		msgData.SendTime = timex.UnixMilli()
	}
	if msgData.CreateTime == 0 {
		msgData.CreateTime = msgData.SendTime
	}

	return &pbmsg.SendSimpleMsgResp{
		ServerMsgID: msgData.ServerMsgID,
		ClientMsgID: msgData.ClientMsgID,
		SendTime:    msgData.SendTime,
		Modify:      msgData,
	}, nil
}

func (l *Logic) PullMessageBySeqs(ctx context.Context, req *sdkws.PullMessageBySeqsReq) (*sdkws.PullMessageBySeqsResp, error) {
	msgs := make(map[string]*sdkws.PullMsgs)
	notificationMsgs := make(map[string]*sdkws.PullMsgs)

	for _, seqRange := range req.SeqRanges {
		dbMsgs, err := l.svcCtx.MsgModel.FindByConversationID(ctx, seqRange.ConversationID, seqRange.Begin, seqRange.End, int(seqRange.Num))
		if err != nil {
			return nil, err
		}

		var pullMsgs sdkws.PullMsgs
		var notifyPullMsgs sdkws.PullMsgs

		for _, dbMsg := range dbMsgs {
			sdkMsg := l.svcCtx.MsgModel.ToSDKMsg(dbMsg)
			if isNotificationMsg(dbMsg.ContentType) {
				notifyPullMsgs.Msgs = append(notifyPullMsgs.Msgs, sdkMsg)
			} else {
				pullMsgs.Msgs = append(pullMsgs.Msgs, sdkMsg)
			}
		}

		if len(pullMsgs.Msgs) > 0 {
			msgs[seqRange.ConversationID] = &pullMsgs
		}
		if len(notifyPullMsgs.Msgs) > 0 {
			notificationMsgs[seqRange.ConversationID] = &notifyPullMsgs
		}
	}

	return &sdkws.PullMessageBySeqsResp{
		Msgs:             msgs,
		NotificationMsgs: notificationMsgs,
	}, nil
}

func (l *Logic) GetSeqMessage(ctx context.Context, req *pbmsg.GetSeqMessageReq) (*pbmsg.GetSeqMessageResp, error) {
	msgs := make(map[string]*sdkws.PullMsgs)
	notificationMsgs := make(map[string]*sdkws.PullMsgs)

	for _, convSeqs := range req.Conversations {
		dbMsgs, err := l.svcCtx.MsgModel.FindByConversationID(ctx, convSeqs.ConversationID, 0, 0, len(convSeqs.Seqs)*2)
		if err != nil {
			return nil, err
		}

		var pullMsgs sdkws.PullMsgs
		var notifyPullMsgs sdkws.PullMsgs

		seqMap := make(map[int64]bool)
		for _, seq := range convSeqs.Seqs {
			seqMap[seq] = true
		}

		for _, dbMsg := range dbMsgs {
			if seqMap[dbMsg.Seq] {
				sdkMsg := l.svcCtx.MsgModel.ToSDKMsg(dbMsg)
				if isNotificationMsg(dbMsg.ContentType) {
					notifyPullMsgs.Msgs = append(notifyPullMsgs.Msgs, sdkMsg)
				} else {
					pullMsgs.Msgs = append(pullMsgs.Msgs, sdkMsg)
				}
			}
		}

		if len(pullMsgs.Msgs) > 0 {
			msgs[convSeqs.ConversationID] = &pullMsgs
		}
		if len(notifyPullMsgs.Msgs) > 0 {
			notificationMsgs[convSeqs.ConversationID] = &notifyPullMsgs
		}
	}

	return &pbmsg.GetSeqMessageResp{
		Msgs:             msgs,
		NotificationMsgs: notificationMsgs,
	}, nil
}

func (l *Logic) GetMsgByConversationIDs(ctx context.Context, req *pbmsg.GetMsgByConversationIDsReq) (*pbmsg.GetMsgByConversationIDsResp, error) {
	msgDatas := make(map[string]*sdkws.MsgData)

	for _, convID := range req.ConversationIDs {
		maxSeq, ok := req.MaxSeqs[convID]
		if !ok {
			maxSeq = 0
		}

		dbMsg, err := l.svcCtx.MsgModel.FindLatestMsg(ctx, convID)
		if err != nil {
			continue
		}

		if dbMsg != nil && (maxSeq == 0 || dbMsg.Seq > maxSeq) {
			sdkMsg := l.svcCtx.MsgModel.ToSDKMsg(dbMsg)
			msgDatas[convID] = sdkMsg
		}
	}

	return &pbmsg.GetMsgByConversationIDsResp{
		MsgDatas: msgDatas,
	}, nil
}

func (l *Logic) GetLastMessage(ctx context.Context, req *pbmsg.GetLastMessageReq) (*pbmsg.GetLastMessageResp, error) {
	msgDatas := make(map[string]*sdkws.MsgData)

	for _, convID := range req.ConversationIDs {
		dbMsg, err := l.svcCtx.MsgModel.FindLatestMsg(ctx, convID)
		if err != nil {
			continue
		}
		if dbMsg != nil {
			sdkMsg := l.svcCtx.MsgModel.ToSDKMsg(dbMsg)
			msgDatas[convID] = sdkMsg
		}
	}

	return &pbmsg.GetLastMessageResp{
		Msgs: msgDatas,
	}, nil
}

func (l *Logic) RevokeMsg(ctx context.Context, req *pbmsg.RevokeMsgReq) (*pbmsg.RevokeMsgResp, error) {
	revokedContent := &model.MessageRevokedContent{
		RevokerID:  req.UserID,
		RevokeTime: timex.Now(),
		Seq:        req.Seq,
	}

	err := l.svcCtx.MsgModel.UpdateRevoke(ctx, req.ConversationID, req.Seq, revokedContent)
	if err != nil {
		return nil, err
	}

	return &pbmsg.RevokeMsgResp{}, nil
}

func (l *Logic) MarkMsgsAsRead(ctx context.Context, req *pbmsg.MarkMsgsAsReadReq) (*pbmsg.MarkMsgsAsReadResp, error) {
	var maxReadSeq int64
	for _, seq := range req.Seqs {
		if seq > maxReadSeq {
			maxReadSeq = seq
		}
	}

	if maxReadSeq > 0 {
		err := l.svcCtx.SeqUserModel.SetUserReadSeq(ctx, req.UserID, req.ConversationID, maxReadSeq)
		if err != nil {
			return nil, err
		}
	}

	return &pbmsg.MarkMsgsAsReadResp{}, nil
}

func (l *Logic) MarkConversationAsRead(ctx context.Context, req *pbmsg.MarkConversationAsReadReq) (*pbmsg.MarkConversationAsReadResp, error) {
	maxReadSeq := req.HasReadSeq
	for _, seq := range req.Seqs {
		if seq > maxReadSeq {
			maxReadSeq = seq
		}
	}

	if maxReadSeq > 0 {
		err := l.svcCtx.SeqUserModel.SetUserReadSeq(ctx, req.UserID, req.ConversationID, maxReadSeq)
		if err != nil {
			return nil, err
		}
	}

	return &pbmsg.MarkConversationAsReadResp{}, nil
}

func (l *Logic) DeleteMsgs(ctx context.Context, req *pbmsg.DeleteMsgsReq) (*pbmsg.DeleteMsgsResp, error) {
	err := l.svcCtx.MsgModel.DeleteBySeq(ctx, req.ConversationID, req.Seqs)
	if err != nil {
		return nil, err
	}

	var minSeq int64 = 0
	if len(req.Seqs) > 0 {
		minSeq = req.Seqs[0]
		for _, seq := range req.Seqs {
			if seq < minSeq {
				minSeq = seq
			}
		}
	}

	err = l.svcCtx.SeqUserModel.SetUserMinSeq(ctx, req.UserID, req.ConversationID, minSeq)
	if err != nil {
		return nil, err
	}

	return &pbmsg.DeleteMsgsResp{}, nil
}

func (l *Logic) DeleteMsgPhysicalBySeq(ctx context.Context, req *pbmsg.DeleteMsgPhysicalBySeqReq) (*pbmsg.DeleteMsgPhysicalBySeqResp, error) {
	err := l.svcCtx.MsgModel.DeleteBySeq(ctx, req.ConversationID, req.Seqs)
	if err != nil {
		return nil, err
	}

	return &pbmsg.DeleteMsgPhysicalBySeqResp{}, nil
}

func (l *Logic) DeleteMsgPhysical(ctx context.Context, req *pbmsg.DeleteMsgPhysicalReq) (*pbmsg.DeleteMsgPhysicalResp, error) {
	err := l.svcCtx.MsgModel.DeleteByTimestamp(ctx, req.ConversationIDs, req.Timestamp)
	if err != nil {
		return nil, err
	}

	return &pbmsg.DeleteMsgPhysicalResp{}, nil
}

func (l *Logic) ClearConversationsMsg(ctx context.Context, req *pbmsg.ClearConversationsMsgReq) (*pbmsg.ClearConversationsMsgResp, error) {
	for _, convID := range req.ConversationIDs {
		err := l.svcCtx.SeqUserModel.SetUserMinSeq(ctx, req.UserID, convID, 0)
		if err != nil {
			return nil, err
		}
	}

	return &pbmsg.ClearConversationsMsgResp{}, nil
}

func (l *Logic) UserClearAllMsg(ctx context.Context, req *pbmsg.UserClearAllMsgReq) (*pbmsg.UserClearAllMsgResp, error) {
	return &pbmsg.UserClearAllMsgResp{}, nil
}

func (l *Logic) GetLastMessageSeqByTime(ctx context.Context, req *pbmsg.GetLastMessageSeqByTimeReq) (*pbmsg.GetLastMessageSeqByTimeResp, error) {
	return &pbmsg.GetLastMessageSeqByTimeResp{
		Seq: 0,
	}, nil
}

func (l *Logic) SearchMessage(ctx context.Context, req *pbmsg.SearchMessageReq) (*pbmsg.SearchMessageResp, error) {
	return &pbmsg.SearchMessageResp{
		ChatLogs:    []*pbmsg.SearchChatLog{},
		ChatLogsNum: 0,
	}, nil
}

func (l *Logic) GetActiveUser(ctx context.Context, req *pbmsg.GetActiveUserReq) (*pbmsg.GetActiveUserResp, error) {
	return &pbmsg.GetActiveUserResp{
		MsgCount:  0,
		UserCount: 0,
		DateCount: make(map[string]int64),
		Users:     []*pbmsg.ActiveUser{},
	}, nil
}

func (l *Logic) GetActiveGroup(ctx context.Context, req *pbmsg.GetActiveGroupReq) (*pbmsg.GetActiveGroupResp, error) {
	return &pbmsg.GetActiveGroupResp{
		MsgCount:   0,
		GroupCount: 0,
		DateCount:  make(map[string]int64),
		Groups:     []*pbmsg.ActiveGroup{},
	}, nil
}

func (l *Logic) GetServerTime(ctx context.Context, req *pbmsg.GetServerTimeReq) (*pbmsg.GetServerTimeResp, error) {
	return &pbmsg.GetServerTimeResp{
		ServerTime: timex.UnixMilli(),
	}, nil
}

func (l *Logic) ClearMsg(ctx context.Context, req *pbmsg.ClearMsgReq) (*pbmsg.ClearMsgResp, error) {
	return &pbmsg.ClearMsgResp{}, nil
}

func (l *Logic) DestructMsgs(ctx context.Context, req *pbmsg.DestructMsgsReq) (*pbmsg.DestructMsgsResp, error) {
	return &pbmsg.DestructMsgsResp{
		Count: 0,
	}, nil
}

func (l *Logic) GetActiveConversation(ctx context.Context, req *pbmsg.GetActiveConversationReq) (*pbmsg.GetActiveConversationResp, error) {
	return &pbmsg.GetActiveConversationResp{
		Conversations: []*pbmsg.ActiveConversation{},
	}, nil
}

func (l *Logic) SetSendMsgStatus(ctx context.Context, req *pbmsg.SetSendMsgStatusReq) (*pbmsg.SetSendMsgStatusResp, error) {
	return &pbmsg.SetSendMsgStatusResp{}, nil
}

func (l *Logic) GetSendMsgStatus(ctx context.Context, req *pbmsg.GetSendMsgStatusReq) (*pbmsg.GetSendMsgStatusResp, error) {
	return &pbmsg.GetSendMsgStatusResp{
		Status: 0,
	}, nil
}

func generateServerMsgID() string {
	return timex.Now().Format("20060102150405") + "_" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[int(timex.UnixNano())%len(letters)]
	}
	return string(b)
}

func isNotificationMsg(contentType int) bool {
	return contentType >= constant.FriendApplicationApprovedNotification &&
		contentType <= constant.DeleteMsgsNotification
}
