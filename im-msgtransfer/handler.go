package immsgtransfer

import (
	"context"

	"github.com/PaperMan11/goim/pkg/msgprocessor"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	queuex "github.com/PaperMan11/goim/pkg/queue"
	"github.com/zeromicro/go-zero/core/logc"
	"google.golang.org/protobuf/proto"
)

func (mt *MsgTransfer) consumePersistentMsgs(ctx context.Context, msg queuex.Message) error {
	var msgData pbmsg.MsgDataToMongoByMQ
	if err := proto.Unmarshal(msg.Value(), &msgData); err != nil {
		logc.Errorf(ctx, "failed to unmarshal msg data, err: %v", err)
		return err
	}

	logc.Infof(ctx, "received msg data to mongo by mq, lastSeq: %d, conversationID: %s, msgData: %v",
		msgData.LastSeq, msgData.ConversationID, msgData.MsgData)

	if _, err := mt.msgService.AddMsgs(ctx, &pbmsg.AddMsgsReq{
		ConversationID: msgData.ConversationID,
		Msgs:           msgData.MsgData,
	}); err != nil {
		logc.Errorf(ctx, "failed to add msgs to db, err: %v", err)
		return err
	}

	return nil
}

func (mt *MsgTransfer) consumeMsg(ctx context.Context, msg queuex.Message) error {
	msgData := mt.batcher.Get()
	if err := proto.Unmarshal(msg.Value(), msgData); err != nil {
		logc.Errorf(ctx, "failed to unmarshal msg data, err: %v", err)
		return err
	}

	logc.Infof(ctx, "received msg, sendID: %s, recvID: %s, groupID: %s, sessionType: %d, contentType: %d",
		msgData.SendID, msgData.RecvID, msgData.GroupID, msgData.SessionType, msgData.ContentType)

	if err := mt.batcher.Push(msgData); err != nil {
		logc.Errorf(ctx, "failed to push msg to batcher, err: %v", err)
		return err
	}
	return nil
}

func (mt *MsgTransfer) batchHandleMsg(conversationID string, msgs []*sdkws.MsgData) {
	ctx := context.Background()
	mt.handleReadReceipt(ctx, msgs)
	storeMsg, notStoreMsg, storeNotifyMsg, notStoreNotifyMsg := mt.classifyMsgType(ctx, msgs)

	// 推送消息到 push topic
	mt.toPushTopic(ctx, conversationID, msgs)

	if len(storeMsg) > 0 {
		mt.toMongoTopic(ctx, conversationID, 0, storeMsg)
	}

	if len(notStoreMsg) > 0 {

	}

	if len(storeNotifyMsg) > 0 {
		mt.toMongoTopic(ctx, conversationID, 0, storeNotifyMsg)
	}

	if len(notStoreNotifyMsg) > 0 {

	}
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

func (mt *MsgTransfer) handleReadReceipt(ctx context.Context, msgs []*sdkws.MsgData) {
	convUserHasReadSeq := make(map[string]map[string]int64)
	for _, msg := range msgs {
		if msg.ContentType != constant.HasReadReceipt {
			continue
		}

		var elem sdkws.NotificationElem
		if err := proto.Unmarshal(msg.Content, &elem); err != nil {
			logc.Errorf(ctx, "failed to unmarshal read receipt elem, err: %v", err)
			continue
		}

		var tips sdkws.MarkAsReadTips
		if err := proto.Unmarshal([]byte(elem.Detail), &tips); err != nil {
			logc.Errorf(ctx, "failed to unmarshal read receipt tips, err: %v", err)
			continue
		}

		if tips.ConversationID == "" || tips.HasReadSeq < 0 {
			continue
		}

		for _, seq := range tips.Seqs {
			if seq > tips.HasReadSeq {
				tips.HasReadSeq = seq
			}
		}

		if _, ok := convUserHasReadSeq[tips.ConversationID]; !ok {
			convUserHasReadSeq[tips.ConversationID] = make(map[string]int64)
		}
		if convUserHasReadSeq[tips.ConversationID][tips.MarkAsReadUserID] < tips.HasReadSeq {
			convUserHasReadSeq[tips.ConversationID][tips.MarkAsReadUserID] = tips.HasReadSeq
		}

	}
	logc.Infof(ctx, "read receipt, convUserHasReadSeq: %v", convUserHasReadSeq)

	for conversationID, hasReadSeqMap := range convUserHasReadSeq {
		for userID, hasReadSeq := range hasReadSeqMap {
			mt.msgService.SetConversationHasReadSeq(ctx, &pbmsg.SetConversationHasReadSeqReq{
				ConversationID: conversationID,
				HasReadSeq:     hasReadSeq,
				UserID:         userID,
				NoNotification: true,
			})
		}
	}
}

func (mt *MsgTransfer) toMongoTopic(ctx context.Context, conversationID string, lastSeq int64, msgs []*sdkws.MsgData) error {
	if len(msgs) == 0 {
		return nil
	}
	pbMsg, err := proto.Marshal(&pbmsg.MsgDataToMongoByMQ{
		LastSeq:        lastSeq,
		ConversationID: conversationID,
		MsgData:        msgs,
	})
	if err != nil {
		logc.Errorf(ctx, "failed to marshal msg data to mongo by mq, err: %v", err)
		return err
	}
	err = mt.msgPersistentProducer.Push(ctx, string(pbMsg))
	if err != nil {
		logc.Errorf(ctx, "failed to push msg data to mongo by mq, err: %v", err)
		return err
	}

	for _, msg := range msgs {
		mt.triggerMessageSavedEvent(ctx, msg)
	}
	return nil
}

func (mt *MsgTransfer) toPushTopic(ctx context.Context, conversationID string, msgs []*sdkws.MsgData) error {
	if len(msgs) == 0 {
		return nil
	}

	for _, msg := range msgs {
		pbMsg, err := proto.Marshal(&pbmsg.PushMsgDataToMQ{
			ConversationID: conversationID,
			MsgData:        msg,
		})
		if err != nil {
			logc.Errorf(ctx, "failed to marshal msg data to push by mq, err: %v", err)
			return err
		}
		mt.msgPushProducer.Push(ctx, string(pbMsg))
	}
	return nil
}
