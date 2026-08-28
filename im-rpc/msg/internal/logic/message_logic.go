package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/msgprocessor"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbconv "github.com/PaperMan11/goim/pkg/protocol/conversation"
	pbgroup "github.com/PaperMan11/goim/pkg/protocol/group"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	pbrelation "github.com/PaperMan11/goim/pkg/protocol/relation"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/storage/model"
	msgModel "github.com/PaperMan11/goim/pkg/storage/mongo/msg"
	"github.com/PaperMan11/goim/pkg/utils/convert"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"google.golang.org/protobuf/proto"
)

// 处理消息选项
func (l *Logic) processMsgOptions(msgData *sdkws.MsgData) {
	msgData.ServerMsgID = generateServerMsgID()
	if msgData.SendTime == 0 {
		msgData.SendTime = timex.UnixMilli()
	}
	if msgData.ContentType >= constant.NotificationBegin && msgData.ContentType <= constant.NotificationEnd {
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsNotNotification, false)
	}
	switch msgData.ContentType {
	case constant.Text, constant.Picture, constant.Voice, constant.Video,
		constant.File, constant.AtText, constant.Merger, constant.Card,
		constant.Location, constant.Custom, constant.Quote, constant.AdvancedText, constant.MarkdownText:
	case constant.Revoke:
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsUnreadCount, false)
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsOfflinePush, false)
	case constant.HasReadReceipt:
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsConversationUpdate, false)
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsSenderConversationUpdate, false)
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsUnreadCount, false)
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsOfflinePush, false)
	case constant.Typing:
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsConversationUpdate, false)
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsSenderConversationUpdate, false)
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsUnreadCount, false)
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsOfflinePush, false)
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsHistory, false)
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsPersistent, false)
		msgprocessor.SetOrSwitchOption(msgData.Options, constant.IsSenderSync, true)
	}
}

func (l *Logic) sendMsg(ctx context.Context, msgData *sdkws.MsgData) error {
	l.processMsgOptions(msgData)
	switch msgData.SessionType {
	case constant.SingleChatType:
		return l.sendSingleChatMsg(ctx, msgData)
	case constant.ReadGroupChatType:
		return l.sendReadGroupChatMsg(ctx, msgData)
	case constant.NotificationChatType:
		return l.sendNotificationMsg(ctx, msgData)
	}
	return fmt.Errorf("session type %d not supported", msgData.SessionType)
}

func (l *Logic) sendSingleChatMsg(ctx context.Context, msgData *sdkws.MsgData) error {
	if err := l.validateSendMsg(ctx, msgData); err != nil {
		return err
	}

	conversationID := msgprocessor.GetChatConversationIDByMsg(msgData)
	isAllowed, err := l.isRecvMsgAllowed(ctx, msgData.RecvID, conversationID, msgData)
	if err != nil {
		return err
	}
	if !isAllowed {
		return fmt.Errorf("user %s is not allowed to send message to user %s", msgData.SendID, msgData.RecvID)
	}

	msgDataBytes, err := proto.Marshal(msgData)
	if err != nil {
		return err
	}
	err = l.svcCtx.MsgTransferProducer.PushWithKey(ctx, conversationID, string(msgDataBytes))
	if err != nil {
		return err
	}
	return nil
}

func (l *Logic) sendReadGroupChatMsg(ctx context.Context, msgData *sdkws.MsgData) error {
	if err := l.validateSendMsg(ctx, msgData); err != nil {
		return err
	}

	msgDataBytes, err := proto.Marshal(msgData)
	if err != nil {
		return err
	}
	err = l.svcCtx.MsgTransferProducer.PushWithKey(ctx, msgData.GroupID, string(msgDataBytes))
	if err != nil {
		return err
	}
	return nil
}

func (l *Logic) sendNotificationMsg(ctx context.Context, msgData *sdkws.MsgData) error {
	msgDataBytes, err := proto.Marshal(msgData)
	if err != nil {
		return err
	}
	conversationID := msgprocessor.GetNotificationConversationIDByMsg(msgData)
	err = l.svcCtx.MsgTransferProducer.PushWithKey(ctx, conversationID, string(msgDataBytes))
	if err != nil {
		return err
	}
	return nil
}

// 验证消息是否符合可发送条件
func (l *Logic) validateSendMsg(ctx context.Context, msgData *sdkws.MsgData) error {
	if msgprocessor.IsNotificationByMsg(msgData) {
		// 通知消息直接放行
		return nil
	}

	switch msgData.SessionType {
	case constant.SingleChatType:
		senderUid, receiverUid := msgData.SendID, msgData.RecvID
		if !l.svcCtx.Config.SendMsgNeedRelationVerify {
			return nil
		}

		// 管理员直接放行
		isAdminResp, _ := l.svcCtx.UserService.IsIMAdmin(ctx, &pbuser.IsIMAdminReq{UserID: senderUid})
		if isAdminResp.GetIsIMAdmin() {
			return nil
		}

		isBlackResp, _ := l.svcCtx.RelationService.IsBlack(ctx, &pbrelation.IsBlackReq{
			UserID1: senderUid,
			UserID2: receiverUid,
		})
		if isBlackResp.GetInUser1Blacks() || isBlackResp.GetInUser2Blacks() {
			return errx.BlackByPeer.Wrap(fmt.Sprintf("user %s is black to user %s", senderUid, receiverUid))
		}
		isFriendResp, _ := l.svcCtx.RelationService.IsFriend(ctx, &pbrelation.IsFriendReq{
			UserID1: senderUid,
			UserID2: receiverUid,
		})
		if !isFriendResp.GetInUser1Friends() {
			return errx.NotPeersFriend.Wrap(fmt.Sprintf("user %s is not friend to user %s", senderUid, receiverUid))
		}

	case constant.ReadGroupChatType:
		groupInfoResp, _ := l.svcCtx.GroupService.GetGroupInfoCache(ctx, &pbgroup.GetGroupInfoCacheReq{GroupID: msgData.GroupID})
		groupInfo := groupInfoResp.GetGroupInfo()
		if groupInfo == nil {
			return errx.GroupNotFoundError
		}
		if groupInfo.Status == constant.GroupStatusDismissed && msgData.ContentType != constant.GroupDismissedNotification {
			return errx.GroupDismissedError
		}
		if groupInfo.Status == constant.SuperGroup {
			return nil
		}
		groupMemberCacheResp, _ := l.svcCtx.GroupService.GetGroupMemberCache(ctx, &pbgroup.GetGroupMemberCacheReq{
			GroupID:       msgData.GroupID,
			GroupMemberID: msgData.SendID,
		})
		memberInfo := groupMemberCacheResp.GetMember()
		if memberInfo == nil {
			return errx.NotInGroupYetError.Wrap(fmt.Sprintf("user %s is not member of group %s", msgData.SendID, msgData.GroupID))
		}
		if memberInfo.RoleLevel == constant.GroupOwner {
			return nil
		} else {
			if memberInfo.MuteEndTime >= timex.UnixMilli() {
				return errx.GroupMemberMutedError.Wrap(fmt.Sprintf("member %s is muted", memberInfo.UserID))
			}
			if groupInfo.Status == constant.GroupStatusMuted && memberInfo.RoleLevel < constant.GroupAdmin {
				return errx.GroupMutedError
			}
		}
	}
	return nil
}

// 判断接受者是否接收消息
func (l *Logic) isRecvMsgAllowed(ctx context.Context, userID, conversationID string, msg *sdkws.MsgData) (bool, error) {
	if msgprocessor.IsNotificationByMsg(msg) {
		return true, nil
	}
	recvOptResp, err := l.svcCtx.UserService.GetGlobalRecvMessageOpt(ctx, &pbuser.GetGlobalRecvMessageOptReq{UserID: userID})
	if err != nil {
		return false, err
	}
	switch recvOptResp.GetGlobalRecvMsgOpt() {
	case constant.ReceiveMessage:
	case constant.NotReceiveMessage:
		return false, nil
	case constant.ReceiveNotNotifyMessage:
		if msg.Options == nil {
			msg.Options = make(map[string]bool, 10)
		}
		msgprocessor.SetOrSwitchOption(msg.Options, constant.IsOfflinePush, false)
		return true, nil
	}

	conversationResp, err := l.svcCtx.ConvService.GetConversation(ctx, &pbconv.GetConversationReq{
		ConversationID: conversationID,
		OwnerUserID:    userID,
	})
	if err != nil {
		return false, err
	}
	switch conversationResp.GetConversation().GetRecvMsgOpt() {
	case constant.ReceiveMessage:
	case constant.NotReceiveMessage:
		return false, nil
	case constant.ReceiveNotNotifyMessage:
		if msg.Options == nil {
			msg.Options = make(map[string]bool, 10)
		}
		msgprocessor.SetOrSwitchOption(msg.Options, constant.IsOfflinePush, false)
		return true, nil
	}

	return true, nil
}

func (l *Logic) SendMsg(ctx context.Context, req *pbmsg.SendMsgReq) (*pbmsg.SendMsgResp, error) {
	msgData := req.MsgData
	if err := l.requireSelfOrAdmin(msgData.SendID); err != nil {
		return nil, err
	}
	if err := l.sendMsg(ctx, msgData); err != nil {
		l.Errorf("send msg failed, err: %v, msgData: %+v", err, msgData)
		return nil, err
	}

	return &pbmsg.SendMsgResp{
		ServerMsgID: msgData.ServerMsgID,
		ClientMsgID: msgData.ClientMsgID,
		SendTime:    msgData.SendTime,
		Modify:      msgData,
	}, nil
}

func (l *Logic) SendSimpleMsg(ctx context.Context, req *pbmsg.SendSimpleMsgReq) (*pbmsg.SendSimpleMsgResp, error) {
	msgData := req.GetMsgData()
	if msgData == nil {
		return nil, errx.ArgsError.Wrap("msgData is nil")
	}

	userResp, err := l.svcCtx.UserService.GetDesignateUsers(ctx, &pbuser.GetDesignateUsersReq{UserIDs: []string{msgData.SendID}})
	if err != nil {
		return nil, err
	}
	users := userResp.GetUsersInfo()
	if len(users) == 0 {
		l.Errorf("user %s %s not found", msgData.SendID, msgData.SendID)
		return nil, err
	}
	user := users[0]
	msgData.SenderFaceURL = user.GetFaceURL()
	msgData.SenderNickname = user.GetNickname()
	resp, err := l.SendMsg(ctx, &pbmsg.SendMsgReq{MsgData: msgData})
	if err != nil {
		return nil, err
	}
	return &pbmsg.SendSimpleMsgResp{
		ServerMsgID: resp.ServerMsgID,
		ClientMsgID: resp.ClientMsgID,
		SendTime:    resp.SendTime,
		Modify:      resp.Modify,
	}, nil
}

func (l *Logic) PullMessageBySeqs(ctx context.Context, req *sdkws.PullMessageBySeqsReq) (*sdkws.PullMessageBySeqsResp, error) {
	msgs := make(map[string]*sdkws.PullMsgs)
	notificationMsgs := make(map[string]*sdkws.PullMsgs)
	userID := req.UserID

	for _, seqRange := range req.SeqRanges {
		pullMsgs, isNotification, err := l.pullMessage(ctx, userID, seqRange.ConversationID, seqRange.Begin, seqRange.End, int(seqRange.Num), req.Order)
		if err != nil {
			return nil, err
		}
		if !isNotification {
			msgs[seqRange.ConversationID] = pullMsgs
		} else {
			notificationMsgs[seqRange.ConversationID] = pullMsgs
		}
	}

	return &sdkws.PullMessageBySeqsResp{
		Msgs:             msgs,
		NotificationMsgs: notificationMsgs,
	}, nil
}

func (l *Logic) pullMessage(ctx context.Context, userID string, conversationID string, seqStart, seqEnd int64, limit int, order sdkws.PullOrder) (pullMsg *sdkws.PullMsgs, isNotification bool, err error) {
	seqUser, err := l.svcCtx.SeqUserModel.GetUserSeq(ctx, userID, conversationID)
	if err != nil {
		l.Errorf("get user seq failed, err: %v, userID: %s, conversationID: %s", err, userID, conversationID)
		return nil, false, err
	}
	seqConversation, err := l.svcCtx.SeqConversationModel.GetSeqConversation(ctx, conversationID)
	if err != nil {
		l.Errorf("get seq conversation failed, err: %v, conversationID: %s", err, conversationID)
		return nil, false, err
	}
	minSeq, maxSeq := seqConversation.MinSeq, seqConversation.MaxSeq
	minSeq = max(max(minSeq, seqUser.MinSeq), seqStart)
	maxSeq = min(min(maxSeq, seqUser.MaxSeq), seqEnd)
	if minSeq > maxSeq {
		return &sdkws.PullMsgs{
			IsEnd:  true,
			EndSeq: minSeq,
			Msgs:   make([]*sdkws.MsgData, 0),
		}, msgprocessor.IsNotification(conversationID), nil
	}

	dbMsgs, err := l.svcCtx.MsgModel.FindByConversationID(ctx, conversationID, minSeq, maxSeq, limit)
	if err != nil {
		l.Errorf("find msg by conversation id failed, err: %v, conversationID: %s", err, conversationID)
		return nil, false, err
	}
	if len(dbMsgs) == 0 {
		return nil, false, nil
	}
	l.handleDeletedAndRevoked(userID, dbMsgs)
	l.handleQuote(ctx, conversationID, userID, dbMsgs)

	var (
		isEnd  bool
		endSeq int64
	)
	switch order {
	case sdkws.PullOrder_PullOrderAsc:
		isEnd = seqEnd >= maxSeq
		endSeq = seqConversation.MaxSeq
	case sdkws.PullOrder_PullOrderDesc:
		isEnd = seqStart <= minSeq
		endSeq = seqConversation.MinSeq
	}

	pullMsgs := &sdkws.PullMsgs{
		IsEnd:  isEnd,
		EndSeq: endSeq,
	}
	for _, dbMsg := range dbMsgs {
		sdkMsg := l.ToSDKMsg(dbMsg.Msg)
		pullMsgs.Msgs = append(pullMsgs.Msgs, sdkMsg)
	}

	return pullMsgs, msgprocessor.IsNotification(conversationID), nil
}

func (l *Logic) handleDeletedAndRevoked(userID string, msgs []*model.MsgInfoModel) {
	for _, msg := range msgs {
		if msg == nil || msg.Msg == nil {
			continue
		}
		msg.Msg.IsRead = msg.IsRead
		if slices.Contains(msg.DelList, userID) {
			msg.Msg.Content = ""
			msg.Msg.Status = constant.MsgDeleted
		}
		if msg.Revoke != nil {
			msg.Msg.ContentType = constant.MsgRevokeNotification
			revokeContent := &sdkws.MessageRevokedContent{
				RevokerID:                   msg.Revoke.UserID,
				RevokerRole:                 msg.Revoke.Role,
				ClientMsgID:                 msg.Msg.ClientMsgID,
				RevokerNickname:             msg.Revoke.Nickname,
				RevokeTime:                  msg.Revoke.Time,
				SourceMessageSendTime:       msg.Msg.SendTime,
				SourceMessageSendID:         msg.Msg.SendID,
				SourceMessageSenderNickname: msg.Msg.SenderNickname,
				SessionType:                 msg.Msg.SessionType,
				Seq:                         msg.Msg.Seq,
				Ex:                          msg.Msg.Ex,
			}
			revokeContentJson, err := json.Marshal(revokeContent)
			if err != nil {
				l.Errorf("marshal revoke content failed, err: %v, revokeContent: %+v", err, revokeContent)
				continue
			}
			elem := &sdkws.NotificationElem{Detail: string(revokeContentJson)}
			jsonStr, err := json.Marshal(elem)
			if err != nil {
				l.Errorf("marshal notification elem failed, err: %v, elem: %+v", err, elem)
				continue
			}
			msg.Msg.Content = string(jsonStr)
		}
	}
}

// quoteMsgData 引用消息中 quoteMessage 字段的结构，只关心撤回判断与改写所需字段
type quoteMsgData struct {
	ClientMsgID string `json:"clientMsgID,omitempty"`
	Seq         int64  `json:"seq,omitempty"`
	ContentType int32  `json:"contentType,omitempty"`
	Content     string `json:"content,omitempty"`
	SendID      string `json:"sendID,omitempty"`
	SendTime    int64  `json:"sendTime,omitempty"`
}

// handleQuote 处理引用消息：若被引用消息已被撤回，把引用体改写为撤回通知，
// 使客户端展示"原消息已撤回"而非原始内容。cache 在批量消息间复用回源结果，避免同一 seq 重复回源。
func (l *Logic) handleQuote(ctx context.Context, conversationID, userID string, msgs []*model.MsgInfoModel) {
	quoteCache := make(map[int64]*model.MsgInfoModel)
	for _, msg := range msgs {
		if msg == nil || msg.Msg == nil {
			continue
		}
		msg.Msg.IsRead = msg.IsRead
		if slices.Contains(msg.DelList, userID) {
			msg.Msg.Content = ""
			msg.Msg.Status = constant.MsgDeleted
		}
		if msg.Msg.ContentType != constant.Quote || msg.Msg.Content == "" {
			continue
		}

		// 引用消息的 content 结构：{text, quoteMessage, messageEntityList}
		var quoteMsg struct {
			Text              string          `json:"text,omitempty"`
			QuoteMessage      *quoteMsgData   `json:"quoteMessage,omitempty"`
			MessageEntityList json.RawMessage `json:"messageEntityList,omitempty"`
		}
		if err := json.Unmarshal([]byte(msg.Msg.Content), &quoteMsg); err != nil {
			l.Errorf("handleQuote unmarshal failed, err=%v, conversationID=%s", err, conversationID)
			continue
		}
		if quoteMsg.QuoteMessage == nil {
			continue
		}

		// 兼容部分客户端把空对象 {} 以 base64("e30=") 编码
		if quoteMsg.QuoteMessage.Content == "e30=" {
			quoteMsg.QuoteMessage.Content = "{}"
			if data, err := json.Marshal(&quoteMsg); err == nil {
				msg.Msg.Content = string(data)
			}
		}

		// 无 seq 且已是撤回通知：无法回源，跳过
		if quoteMsg.QuoteMessage.Seq <= 0 && quoteMsg.QuoteMessage.ContentType == constant.MsgRevokeNotification {
			continue
		}

		// 回源取被引用消息（带 Revoke）
		var quoted *model.MsgInfoModel
		if v, ok := quoteCache[quoteMsg.QuoteMessage.Seq]; ok {
			quoted = v
		} else if quoteMsg.QuoteMessage.Seq > 0 {
			info, err := l.svcCtx.MsgModel.FindInfoBySeq(ctx, conversationID, quoteMsg.QuoteMessage.Seq)
			if err != nil && !errors.Is(err, msgModel.ErrMsgNotFound) {
				l.Errorf("handleQuote FindInfoBySeq failed, err=%v, conversationID=%s, seq=%d",
					err, conversationID, quoteMsg.QuoteMessage.Seq)
				continue
			}
			quoted = info
			quoteCache[quoteMsg.QuoteMessage.Seq] = quoted // nil 也缓存，避免重复回源
		}

		// 被引用消息未撤回：不改写（查不到时保持原样，不误标为撤回）
		if quoted == nil || quoted.Msg == nil || quoted.Revoke == nil {
			continue
		}

		// 已撤回：改写引用体为撤回通知（复用 handleDeletedAndRevoked 的撤回内容构造）
		quoteMsg.QuoteMessage.ContentType = constant.MsgRevokeNotification
		quoteMsg.QuoteMessage.Content = quoted.Msg.Content

		data, err := json.Marshal(&quoteMsg)
		if err != nil {
			l.Errorf("handleQuote marshal quoteMsg failed, err=%v", err)
			continue
		}
		msg.Msg.Content = string(data)
	}
}

func (l *Logic) GetSeqMessage(ctx context.Context, req *pbmsg.GetSeqMessageReq) (*pbmsg.GetSeqMessageResp, error) {
	msgs := make(map[string]*sdkws.PullMsgs)
	notificationMsgs := make(map[string]*sdkws.PullMsgs)

	for _, convSeqs := range req.Conversations {
		if len(convSeqs.Seqs) == 0 {
			continue
		}
		slices.Sort(convSeqs.Seqs)
		minSeq := convSeqs.Seqs[0]
		maxSeq := convSeqs.Seqs[len(convSeqs.Seqs)-1]
		limit := int(maxSeq - minSeq + 1)
		pullMsgs, isNotification, err := l.pullMessage(ctx, req.UserID, convSeqs.ConversationID, minSeq, maxSeq, limit, req.Order)
		if err != nil {
			continue
		}
		if !isNotification {
			msgs[convSeqs.ConversationID] = pullMsgs
		} else {
			notificationMsgs[convSeqs.ConversationID] = pullMsgs
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
		maxSeq := req.MaxSeqs[convID]
		if maxSeq == 0 {
			dbMsg, err := l.svcCtx.MsgModel.FindLatestMsg(ctx, convID)
			if err != nil {
				l.Errorf("find latest msg failed, err: %v, conversationID: %s", err, convID)
				continue
			}

			if dbMsg != nil {
				sdkMsg := l.ToSDKMsg(dbMsg)
				msgDatas[convID] = sdkMsg
			}
		} else {
			dbMsg, err := l.svcCtx.MsgModel.FindInfoBySeq(ctx, convID, maxSeq)
			if err != nil {
				l.Errorf("find msg by seq failed, err: %v, conversationID: %s, seq: %d", err, convID, maxSeq)
				continue
			}
			if dbMsg != nil {
				sdkMsg := l.ToSDKMsg(dbMsg.Msg)
				msgDatas[convID] = sdkMsg
			}
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
			l.Errorf("find latest msg failed, err: %v, conversationID: %s", err, convID)
			continue
		}
		if dbMsg != nil {
			sdkMsg := l.ToSDKMsg(dbMsg)
			msgDatas[convID] = sdkMsg
		}
	}

	return &pbmsg.GetLastMessageResp{
		Msgs: msgDatas,
	}, nil
}

func (l *Logic) RevokeMsg(ctx context.Context, req *pbmsg.RevokeMsgReq) (*pbmsg.RevokeMsgResp, error) {
	if err := l.requireSelfOrAdmin(req.UserID); err != nil {
		return nil, err
	}

	designateUsers, err := l.svcCtx.UserService.GetDesignateUsers(ctx, &pbuser.GetDesignateUsersReq{
		UserIDs: []string{req.UserID},
	})
	if err != nil || len(designateUsers.GetUsersInfo()) == 0 {
		l.Errorf("get designate users failed, err: %v, userID: %s", err, req.UserID)
		return nil, err
	}
	userInfo := designateUsers.GetUsersInfo()[0]

	dbMsg, err := l.svcCtx.MsgModel.FindBySeq(ctx, req.ConversationID, req.Seq)
	if err != nil {
		l.Errorf("find msg by seq failed, err: %v, conversationID: %s, seq: %d", err, req.ConversationID, req.Seq)
		return nil, err
	}

	var role int32
	switch dbMsg.SessionType {
	case constant.SingleChatType:
		role = userInfo.AppMangerLevel
	case constant.ReadGroupChatType:
		opMember, err := l.svcCtx.GroupService.GetGroupMemberCache(ctx, &pbgroup.GetGroupMemberCacheReq{
			GroupID:       dbMsg.GroupID,
			GroupMemberID: userInfo.UserID,
		})
		if err != nil {
			l.Errorf("get group member cache failed, err: %v, groupID: %s, userID: %s", err, dbMsg.GroupID, userInfo.UserID)
			return nil, err
		}
		if req.UserID != dbMsg.SendID {
			role = opMember.GetMember().RoleLevel
		} else {
			senderMember, err := l.svcCtx.GroupService.GetGroupMemberCache(ctx, &pbgroup.GetGroupMemberCacheReq{
				GroupID:       dbMsg.GroupID,
				GroupMemberID: dbMsg.SendID,
			})
			if err != nil {
				l.Errorf("get group member cache failed, err: %v, groupID: %s, userID: %s", err, dbMsg.GroupID, dbMsg.SendID)
				return nil, err
			}
			switch opMember.GetMember().RoleLevel {
			case constant.GroupOwner:
				role = constant.GroupOwner
			case constant.GroupAdmin:
				if senderMember.GetMember().RoleLevel >= constant.GroupAdmin {
					return nil, errx.NoPermissionError.Wrap("only owner or admin can revoke msg")
				}
				role = constant.GroupAdmin
			default:
				return nil, errx.NoPermissionError.Wrap("only owner or admin can revoke msg")
			}
		}
	default:
		return nil, errx.InternalError.Wrap(fmt.Sprintf("session type %d not support", dbMsg.SessionType))
	}

	err = l.svcCtx.MsgModel.UpdateRevoke(ctx, req.ConversationID, req.Seq, &model.RevokeModel{
		Role:     role,
		UserID:   userInfo.UserID,
		Nickname: userInfo.Nickname,
		Time:     timex.UnixMilli(),
	})
	if err != nil {
		return nil, err
	}

	// notify user that msg is revoked
	l.sendRevokeNotification(ctx, userInfo.UserID, dbMsg, role, req.ConversationID, req.Seq)

	return &pbmsg.RevokeMsgResp{}, nil
}

func (l *Logic) MarkMsgsAsRead(ctx context.Context, req *pbmsg.MarkMsgsAsReadReq) (*pbmsg.MarkMsgsAsReadResp, error) {
	var maxReadSeq int64
	for _, seq := range req.Seqs {
		if seq > maxReadSeq {
			maxReadSeq = seq
		}
	}

	conv, err := l.svcCtx.ConvService.GetConversation(ctx, &pbconv.GetConversationReq{
		ConversationID: req.ConversationID,
	})
	if err != nil {
		l.Errorf("get conversation failed, err: %v, conversationID: %s", err, req.ConversationID)
		return nil, err
	}
	conversation := conv.GetConversation()

	maxSeq, err := l.svcCtx.SeqConversationModel.GetConversationMaxSeq(ctx, req.ConversationID)
	if err != nil {
		l.Errorf("get conversation max seq failed, err: %v, conversationID: %s", err, req.ConversationID)
		return nil, err
	}
	if maxReadSeq > maxSeq {
		maxReadSeq = maxSeq
	}

	if maxReadSeq > 0 {
		err := l.svcCtx.SeqUserModel.SetUserReadSeq(ctx, req.UserID, req.ConversationID, maxReadSeq)
		if err != nil {
			l.Errorf("set user read seq failed, err: %v, userID: %s, conversationID: %s, readSeq: %d", err, req.UserID, req.ConversationID, maxReadSeq)
			return nil, err
		}

		if conversation.ConversationType == constant.SingleChatType {
			err = l.svcCtx.MsgModel.MarkReadBySeqs(ctx, req.UserID, req.ConversationID, req.Seqs)
			if err != nil {
				return nil, err
			}
		}
	}

	// notify user that msgs are read
	l.sendMarkAsReadNotification(ctx, req.UserID, conversation, req.Seqs, maxReadSeq)

	return &pbmsg.MarkMsgsAsReadResp{}, nil
}

func (l *Logic) MarkConversationAsRead(ctx context.Context, req *pbmsg.MarkConversationAsReadReq) (*pbmsg.MarkConversationAsReadResp, error) {
	maxReadSeq := req.HasReadSeq
	for _, seq := range req.Seqs {
		if seq > maxReadSeq {
			maxReadSeq = seq
		}
	}

	conv, err := l.svcCtx.ConvService.GetConversation(ctx, &pbconv.GetConversationReq{
		ConversationID: req.ConversationID,
	})
	if err != nil {
		l.Errorf("get conversation failed, err: %v, conversationID: %s", err, req.ConversationID)
		return nil, err
	}
	conversation := conv.GetConversation()

	maxSeq, err := l.svcCtx.SeqConversationModel.GetConversationMaxSeq(ctx, req.ConversationID)
	if err != nil {
		l.Errorf("get conversation max seq failed, err: %v, conversationID: %s", err, req.ConversationID)
		return nil, err
	}
	if maxReadSeq > maxSeq {
		maxReadSeq = maxSeq
	}

	if maxReadSeq > 0 {
		err := l.svcCtx.SeqUserModel.SetUserReadSeq(ctx, req.UserID, req.ConversationID, maxReadSeq)
		if err != nil {
			return nil, err
		}

		if conversation.ConversationType == constant.SingleChatType {
			err = l.svcCtx.MsgModel.MarkReadBySeqs(ctx, req.UserID, req.ConversationID, req.Seqs)
			if err != nil {
				return nil, err
			}
		}
	}

	// notify user that msgs are read
	l.sendMarkAsReadNotification(ctx, req.UserID, conversation, req.Seqs, maxReadSeq)

	return &pbmsg.MarkConversationAsReadResp{}, nil
}

func (l *Logic) DeleteMsgs(ctx context.Context, req *pbmsg.DeleteMsgsReq) (*pbmsg.DeleteMsgsResp, error) {
	err := l.requireSelfOrAdmin(req.UserID)
	if err != nil {
		return nil, err
	}

	isSyncSelf := req.GetDeleteSyncOpt().GetIsSyncSelf()
	isSyncOther := req.GetDeleteSyncOpt().GetIsSyncOther()

	err = l.deleteMsgs(ctx, req.UserID, req.ConversationID, req.Seqs, isSyncOther, isSyncSelf)
	if err != nil {
		return nil, err
	}

	return &pbmsg.DeleteMsgsResp{}, nil
}

func (l *Logic) deleteMsgs(ctx context.Context, userID, conversationID string, seqs []int64, isSyncOther, isSyncSelf bool) (err error) {
	if isSyncOther {
		err = l.svcCtx.MsgModel.DeleteBySeq(ctx, conversationID, seqs)
		if err != nil {
			l.Errorf("delete msg by seq failed, err: %v, conversationID: %s, seqs: %v", err, conversationID, seqs)
			return err
		}

		convResp, _ := l.svcCtx.ConvService.GetConversation(ctx, &pbconv.GetConversationReq{
			ConversationID: conversationID,
		})
		conv := convResp.GetConversation()
		if conv != nil {
			// notify group members that msg is deleted
			l.sendDeleteMsgsNotification(ctx, userID, conversationID, seqs, conv)
		}
	} else {
		err = l.svcCtx.MsgModel.MarkDeleteBySeqs(ctx, userID, conversationID, seqs)
		if err != nil {
			l.Errorf("mark delete msg by seq failed, err: %v, userID: %s, conversationID: %s, seqs: %v", err, userID, conversationID, seqs)
			return err
		}
		if isSyncSelf {
			// notify user that msg is deleted
			l.sendDeleteMsgsSelfNotification(ctx, userID, conversationID, seqs)
		}
	}
	return nil
}

func (l *Logic) DeleteMsgPhysicalBySeq(ctx context.Context, req *pbmsg.DeleteMsgPhysicalBySeqReq) (*pbmsg.DeleteMsgPhysicalBySeqResp, error) {
	err := l.requireAdmin()
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.MsgModel.DeleteBySeq(ctx, req.ConversationID, req.Seqs)
	if err != nil {
		l.Errorf("delete msg by seq failed, err: %v, conversationID: %s, seqs: %v", err, req.ConversationID, req.Seqs)
		return nil, err
	}

	return &pbmsg.DeleteMsgPhysicalBySeqResp{}, nil
}

func (l *Logic) DeleteMsgPhysical(ctx context.Context, req *pbmsg.DeleteMsgPhysicalReq) (*pbmsg.DeleteMsgPhysicalResp, error) {
	err := l.requireAdmin()
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.MsgModel.DeleteByTimestamp(ctx, req.ConversationIDs, req.Timestamp)
	if err != nil {
		l.Errorf("delete msg by timestamp failed, err: %v, conversationIDs: %s, timestamp: %v", err, req.ConversationIDs, req.Timestamp)
		return nil, err
	}

	return &pbmsg.DeleteMsgPhysicalResp{}, nil
}

func (l *Logic) ClearConversationsMsg(ctx context.Context, req *pbmsg.ClearConversationsMsgReq) (*pbmsg.ClearConversationsMsgResp, error) {
	err := l.requireSelfOrAdmin(req.UserID)
	if err != nil {
		return nil, err
	}

	isSyncSelf := req.GetDeleteSyncOpt().GetIsSyncSelf()
	isSyncOther := req.GetDeleteSyncOpt().GetIsSyncOther()
	err = l.clearConversationsMsg(ctx, req.UserID, req.ConversationIDs, isSyncSelf, isSyncOther)
	if err != nil {
		return nil, err
	}

	return &pbmsg.ClearConversationsMsgResp{}, nil
}

func (l *Logic) UserClearAllMsg(ctx context.Context, req *pbmsg.UserClearAllMsgReq) (*pbmsg.UserClearAllMsgResp, error) {
	err := l.requireSelfOrAdmin(req.UserID)
	if err != nil {
		return nil, err
	}

	convsResp, err := l.svcCtx.ConvService.GetConversationIDs(ctx, &pbconv.GetConversationIDsReq{
		UserID: req.UserID,
	})
	if err != nil {
		l.Errorf("get conversation ids failed, err: %v, userID: %s", err, req.UserID)
		return nil, err
	}

	isSyncSelf := req.GetDeleteSyncOpt().GetIsSyncSelf()
	isSyncOther := req.GetDeleteSyncOpt().GetIsSyncOther()
	err = l.clearConversationsMsg(ctx, req.UserID, convsResp.GetConversationIDs(), isSyncSelf, isSyncOther)
	if err != nil {
		return nil, err
	}
	return &pbmsg.UserClearAllMsgResp{}, nil
}

func (l *Logic) clearConversationsMsg(ctx context.Context, userID string, conversationIDs []string, isSyncSelf, isSyncOther bool) error {
	convResp, err := l.svcCtx.ConvService.GetConversations(ctx, &pbconv.GetConversationsReq{
		ConversationIDs: conversationIDs,
		OwnerUserID:     userID,
	})
	if err != nil {
		l.Errorf("get conversations failed, err: %v, conversationIDs: %s, userID: %s", err, conversationIDs, userID)
		return err
	}
	convs := convResp.GetConversations()
	existingConvs := make([]string, 0, len(convs))
	for _, conv := range convs {
		existingConvs = append(existingConvs, conv.GetConversationID())
	}
	convSeqsMap, err := l.svcCtx.SeqConversationModel.BatchGetConversationSeqs(ctx, existingConvs)
	if err != nil {
		l.Errorf("batch get conversation seqs failed, err: %v, conversationIDs: %s", err, existingConvs)
		return err
	}

	if !isSyncOther {
		for _, convID := range existingConvs {
			convSeq := convSeqsMap[convID]
			if convSeq == nil {
				continue
			}
			err = l.svcCtx.SeqUserModel.SetUserMinSeq(ctx, userID, convID, convSeq.MaxSeq+1)
			if err != nil {
				l.Errorf("set user min seq failed, err: %v, userID: %s, conversationID: %s", err, userID, convID)
			}
		}
		if isSyncSelf {
			l.sendClearConversationsMsgNotification(ctx, userID, userID, existingConvs, constant.SingleChatType)
		}
	} else {
		for _, conv := range convs {
			convSeq := convSeqsMap[conv.GetConversationID()]
			if convSeq == nil {
				continue
			}
			err = l.svcCtx.SeqConversationModel.SetConversationMinSeq(ctx, conv.GetConversationID(), convSeq.MaxSeq+1)
			if err != nil {
				l.Errorf("set conversation min seq failed, err: %v, conversationID: %s", err, conv.GetConversationID())
				continue
			}

			// notify group members that conversation is cleared
			recvID := userID
			if conv.ConversationType == constant.SingleChatType || conv.ConversationType == constant.NotificationChatType {
				if userID == conv.OwnerUserID {
					recvID = conv.UserID
				} else {
					recvID = conv.OwnerUserID
				}
			} else {
				recvID = conv.GroupID
			}
			l.sendClearConversationsMsgNotification(ctx, userID, recvID, []string{conv.GetConversationID()}, conv.ConversationType)
		}
	}
	return nil
}

func (l *Logic) GetLastMessageSeqByTime(ctx context.Context, req *pbmsg.GetLastMessageSeqByTimeReq) (*pbmsg.GetLastMessageSeqByTimeResp, error) {
	lastSeq, err := l.svcCtx.MsgModel.GetLastMessageSeqByTimestamp(ctx, req.ConversationID, req.Time)
	if err != nil {
		l.Errorf("get last message seq by timestamp failed, err: %v, conversationID: %s, timestamp: %d", err, req.ConversationID, req.Time)
		return nil, err
	}
	return &pbmsg.GetLastMessageSeqByTimeResp{
		Seq: lastSeq,
	}, nil
}

func (l *Logic) SearchMessage(ctx context.Context, req *pbmsg.SearchMessageReq) (*pbmsg.SearchMessageResp, error) {
	msgs, err := l.svcCtx.MsgModel.SearchMessage(ctx, &model.SearchMessageReq{
		SendID:      req.SendID,
		RecvID:      req.RecvID,
		SessionType: req.SessionType,
		ContentType: req.ContentType,
		SendTime:    req.SendTime,
		Pagination: model.Pagination{
			Page:     req.Pagination.PageNumber,
			PageSize: req.Pagination.ShowNumber,
		},
	})
	if err != nil {
		l.Errorf("search message failed, err: %v, req: %v", err, req)
		return nil, err
	}

	var (
		chatLogs        []*pbmsg.SearchChatLog
		sendIDs         []string
		recvIDs         []string
		groupIDs        []string
		sendNicknameMap = make(map[string]string)
		recvNicknameMap = make(map[string]string)
		groupMap        = make(map[string]*sdkws.GroupInfo)
	)
	for _, msg := range msgs {
		if msg.Msg.SenderNickname == "" {
			sendIDs = append(sendIDs, msg.Msg.SendID)
		}
		switch msg.Msg.SessionType {
		case constant.SingleChatType, constant.NotificationChatType:
			recvIDs = append(recvIDs, msg.Msg.RecvID)
		case constant.WriteGroupChatType, constant.ReadGroupChatType:
			groupIDs = append(groupIDs, msg.Msg.GroupID)
		}
	}
	if len(sendIDs) > 0 {
		sendResp, err := l.svcCtx.UserService.GetDesignateUsers(ctx, &pbuser.GetDesignateUsersReq{UserIDs: sendIDs})
		if err != nil {
			l.Errorf("get designate users failed, err: %v, sendIDs: %s", err, sendIDs)
			return nil, err
		}
		for _, user := range sendResp.GetUsersInfo() {
			sendNicknameMap[user.GetUserID()] = user.GetNickname()
		}
	}
	if len(recvIDs) > 0 {
		recvResp, err := l.svcCtx.UserService.GetDesignateUsers(ctx, &pbuser.GetDesignateUsersReq{UserIDs: recvIDs})
		if err != nil {
			l.Errorf("get designate users failed, err: %v, recvIDs: %s", err, recvIDs)
			return nil, err
		}
		for _, user := range recvResp.GetUsersInfo() {
			recvNicknameMap[user.GetUserID()] = user.GetNickname()
		}
	}
	if len(groupIDs) > 0 {
		for _, groupID := range groupIDs {
			groupResp, err := l.svcCtx.GroupService.GetGroupInfoCache(ctx, &pbgroup.GetGroupInfoCacheReq{GroupID: groupID})
			if err != nil {
				l.Errorf("get group info cache failed, err: %v, groupIDs: %s", err, groupIDs)
				return nil, err
			}
			if groupResp.GetGroupInfo() != nil {
				groupMap[groupResp.GetGroupInfo().GetGroupID()] = groupResp.GetGroupInfo()
			}
		}
	}

	for _, msg := range msgs {
		pbchatLog := modelToChatLog(msg)
		if msg.Msg.SenderNickname == "" {
			pbchatLog.SenderNickname = sendNicknameMap[msg.Msg.SendID]
		}
		switch msg.Msg.SessionType {
		case constant.SingleChatType, constant.NotificationChatType:
			pbchatLog.RecvNickname = recvNicknameMap[msg.Msg.RecvID]
		case constant.ReadGroupChatType:
			groupInfo := groupMap[msg.Msg.GroupID]
			pbchatLog.SenderFaceURL = groupInfo.FaceURL
			pbchatLog.GroupMemberCount = groupInfo.MemberCount // Reflects actual member count
			pbchatLog.RecvID = groupInfo.GroupID
			pbchatLog.GroupName = groupInfo.GroupName
			pbchatLog.GroupOwner = groupInfo.OwnerUserID
			pbchatLog.GroupType = groupInfo.GroupType
		}
		searchChatLog := &pbmsg.SearchChatLog{ChatLog: pbchatLog, IsRevoked: msg.Revoke != nil}

		chatLogs = append(chatLogs, searchChatLog)
	}

	return &pbmsg.SearchMessageResp{
		ChatLogs:    chatLogs,
		ChatLogsNum: int32(len(chatLogs)),
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

// SetSendMsgStatus 设置发送消息状态
func (l *Logic) SetSendMsgStatus(ctx context.Context, req *pbmsg.SetSendMsgStatusReq) (*pbmsg.SetSendMsgStatusResp, error) {
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	err := l.svcCtx.RedisClient.Set(ctx, fmt.Sprintf("send_msg_status:%s", opUserID), convert.ToString(req.Status), time.Hour*24).Err()
	if err != nil {
		l.Errorf("set send msg status failed, err: %v, userID: %s, status: %d", err, opUserID, req.Status)
		return nil, err
	}
	return &pbmsg.SetSendMsgStatusResp{}, nil
}

// GetSendMsgStatus 获取发送消息状态
func (l *Logic) GetSendMsgStatus(ctx context.Context, req *pbmsg.GetSendMsgStatusReq) (*pbmsg.GetSendMsgStatusResp, error) {
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	status, err := l.svcCtx.RedisClient.Get(ctx, fmt.Sprintf("send_msg_status:%s", opUserID)).Result()
	if err != nil {
		l.Errorf("get send msg status failed, err: %v, userID: %s", err, opUserID)
		return nil, err
	}
	return &pbmsg.GetSendMsgStatusResp{
		Status: convert.ToInt32(status),
	}, nil
}
