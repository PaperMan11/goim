package immsgtransfer

import (
	"context"

	"github.com/PaperMan11/goim/pkg/msgprocessor"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	queuex "github.com/PaperMan11/goim/pkg/queue"
	"github.com/PaperMan11/goim/pkg/webhooks"
	"github.com/zeromicro/go-zero/core/logc"
	"google.golang.org/protobuf/proto"
)

func (mt *MsgTransfer) handleMsg(ctx context.Context, msg queuex.Message) error {
	var msgData sdkws.MsgData
	if err := proto.Unmarshal(msg.Value(), &msgData); err != nil {
		logc.Errorf(ctx, "failed to unmarshal msg data, err: %v", err)
		return err
	}

	logc.Infof(ctx, "received msg, sendID: %s, recvID: %s, groupID: %s, sessionType: %d, contentType: %d",
		msgData.SendID, msgData.RecvID, msgData.GroupID, msgData.SessionType, msgData.ContentType)

	storeMsg, notStoreMsg, storeNotifyMsg, notStoreNotifyMsg := mt.classifyMsgType(ctx, []*sdkws.MsgData{&msgData})

	// 推送消息到 push topic
	conversationID := msgprocessor.GetConversationIDByMsg(&msgData)
	mt.toPushTopic(ctx, conversationID, &msgData)

	if len(storeMsg) > 0 {
		if err := mt.toMongoTopic(ctx, conversationID, 0, storeMsg); err != nil {
			logc.Errorf(ctx, "failed to persist msg, err: %v", err)
		}
	}

	if len(notStoreMsg) > 0 {

	}

	if len(storeNotifyMsg) > 0 {

	}

	if len(notStoreNotifyMsg) > 0 {

	}

	mt.triggerWebhook(ctx, &msgData)

	return nil
}

func (mt *MsgTransfer) classifyMsgType(_ context.Context, msgs []*sdkws.MsgData) (storeMsg, notStoreMsg, storeNotifyMsg, notStoreNotifyMsg []*sdkws.MsgData) {
	for _, msg := range msgs {
		msgOpts := msgprocessor.Options(msg.Options)
		if msgOpts.IsNotNotification() {
			if msgOpts.IsHistory() {
				storeMsg = append(storeMsg, msg)
			} else {
				notStoreMsg = append(notStoreMsg, msg)
			}
		} else {
			if msgOpts.IsSendMsg() {
				cloneMsg := proto.Clone(msg).(*sdkws.MsgData)
				cloneMsg.Options = msgprocessor.NewOptions(
					msgprocessor.WithOfflinePush(msgOpts.IsOfflinePush()),
					msgprocessor.WithUnreadCount(msgOpts.IsUnreadCount()),
				)
				storeNotifyMsg = append(storeNotifyMsg, cloneMsg)

				msg.Options = msgprocessor.WithOptions(msg.Options,
					msgprocessor.WithOfflinePush(false),
					msgprocessor.WithUnreadCount(false),
				)
			}
			if msgOpts.IsHistory() {
				storeNotifyMsg = append(storeNotifyMsg, msg)
			} else {
				notStoreNotifyMsg = append(notStoreNotifyMsg, msg)
			}
		}
	}
	return
}

func (mt *MsgTransfer) handleReadReceipt(ctx context.Context, msg *sdkws.MsgData) {
	if msg.ContentType != constant.HasReadReceipt {
		return
	}

	var elem sdkws.NotificationElem
	if err := proto.Unmarshal(msg.Content, &elem); err != nil {
		logc.Errorf(ctx, "failed to unmarshal read receipt elem, err: %v", err)
		return
	}

	var tips sdkws.MarkAsReadTips
	if err := proto.Unmarshal([]byte(elem.Detail), &tips); err != nil {
		logc.Errorf(ctx, "failed to unmarshal read receipt tips, err: %v", err)
		return
	}

	if tips.ConversationID == "" || tips.HasReadSeq < 0 {
		return
	}

	for _, seq := range tips.Seqs {
		if seq > tips.HasReadSeq {
			tips.HasReadSeq = seq
		}
	}

	logc.Infof(ctx, "read receipt, conversationID: %s, hasReadSeq: %d", tips.ConversationID, tips.HasReadSeq)
}

func (mt *MsgTransfer) toMongoTopic(ctx context.Context, conversationID string, lastSeq int64, msg []*sdkws.MsgData) error {
	if len(msg) == 0 {
		return nil
	}
	pbMsg, err := proto.Marshal(&pbmsg.MsgDataToMongoByMQ{
		LastSeq:        lastSeq,
		ConversationID: conversationID,
		MsgData:        msg,
	})
	if err != nil {
		logc.Errorf(ctx, "failed to marshal msg data to mongo by mq, err: %v", err)
		return err
	}
	mt.msgPersistentProducer.Push(ctx, string(pbMsg))
	return nil
}

func (mt *MsgTransfer) toPushTopic(ctx context.Context, conversationID string, msg *sdkws.MsgData) error {
	pbMsg, err := proto.Marshal(&pbmsg.PushMsgDataToMQ{
		ConversationID: conversationID,
		MsgData:        msg,
	})
	if err != nil {
		logc.Errorf(ctx, "failed to marshal msg data to push by mq, err: %v", err)
		return err
	}
	mt.msgPushProducer.Push(ctx, string(pbMsg))
	return nil
}

func (mt *MsgTransfer) triggerWebhook(ctx context.Context, msg *sdkws.MsgData) {
	eventData := &webhooks.MessageEventData{
		MessageID:      msg.ClientMsgID,
		ServerMsgID:    msg.ServerMsgID,
		ClientMsgID:    msg.ClientMsgID,
		SenderID:       msg.SendID,
		SenderNickname: msg.SenderNickname,
		ReceiverID:     msg.RecvID,
		GroupID:        msg.GroupID,
		ContentType:    int(msg.ContentType),
		Content:        string(msg.Content),
		SessionType:    int(msg.SessionType),
		SendTime:       msg.SendTime,
		Seq:            msg.Seq,
		PlatformID:     int(msg.SenderPlatformID),
		Ex:             msg.Ex,
		AtUserList:     msg.AtUserIDList,
	}

	webhookEvent := webhooks.NewMessageEvent(eventData)
	if err := mt.webhookManager.Dispatch(webhookEvent); err != nil {
		logc.Errorf(ctx, "failed to dispatch webhook event, err: %v", err)
	}
}
