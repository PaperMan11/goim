package immsgtransfer

import (
	"context"

	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/msgprocessor"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	queuex "github.com/PaperMan11/goim/pkg/queue"
	"github.com/zeromicro/go-zero/core/logc"
	"google.golang.org/protobuf/proto"
)

// withOperationID 确保 ctx 携带 operationID（server 拦截器硬性要求），
// 并在指定 opUserID 时注入，使下游 requireSelfOrAdmin 自然通过。
// msgtransfer 消费的消息已在上游完成鉴权，此处仅继承用户身份做异步后续操作。
func withOperationID(ctx context.Context, opUserID string) context.Context {
	if mcontext.GetOperationIDFromContext(ctx) == "" {
		ctx = mcontext.SetOperationIDInContext(ctx, mcontext.NewOperationID("transfer"))
	}
	if opUserID != "" {
		ctx = mcontext.SetOpUserIDInContext(ctx, opUserID)
	}
	return ctx
}

func (mt *MsgTransfer) consumePersistentMsgs(ctx context.Context, msg queuex.Message) error {
	ctx = withOperationID(ctx, "")

	var msgData pbmsg.MsgDataToMongoByMQ
	if err := proto.Unmarshal(msg.Value(), &msgData); err != nil {
		logc.Errorf(ctx, "failed to unmarshal msg data, err: %v", err)
		return err
	}

	logc.Infof(ctx, "received msg data to mongo by mq, conversationID: %s, msgCount: %d", msgData.ConversationID, len(msgData.MsgData))

	if _, err := mt.msgService.AddMsgs(ctx, &pbmsg.AddMsgsReq{
		ConversationID: msgData.ConversationID,
		Msgs:           msgData.MsgData,
	}); err != nil {
		logc.Errorf(ctx, "failed to add msgs to db, conversationID: %s, msgCount: %d, err: %v",
			msgData.ConversationID, len(msgData.MsgData), err)
		return err
	}

	logc.Infof(ctx, "persist msgs success, conversationID: %s, msgCount: %d",
		msgData.ConversationID, len(msgData.MsgData))
	return nil
}

func (mt *MsgTransfer) consumeMsg(ctx context.Context, msg queuex.Message) error {
	ctx = withOperationID(ctx, "")

	msgData := &sdkws.MsgData{}
	if err := proto.Unmarshal(msg.Value(), msgData); err != nil {
		logc.Errorf(ctx, "failed to unmarshal msg data, err: %v", err)
		return err
	}

	logc.Infof(ctx, "received msg, sendID: %s, recvID: %s, groupID: %s, sessionType: %d, contentType: %d",
		msgData.SendID, msgData.RecvID, msgData.GroupID, msgData.SessionType, msgData.ContentType)

	// push 不进攒批：toPushTopic 本就逐条投递，攒批对 push 零收益纯延迟，
	// 且 push 不应被落库路径（RPC/produce）阻塞。msg-transfer-topic 按会话 key
	// 哈希分区，同会话消息串行到达，此处直接推送可保证同会话推送顺序。
	conversationID := msgprocessor.GetChatConversationIDByMsg(msgData)
	fixNotifyOptions(msgData)

	// seq 必须在 push 前分配，推送消息才能携带 seq，且分配顺序与推送顺序、
	// 落库顺序一致。仅持久化消息（IsHistory）参与编号；typing 等不落库消息
	// 不占 seq，避免客户端按 seq 拉取时出现永久空洞。
	if err := mt.allocateMsgSeq(ctx, conversationID, msgData); err != nil {
		return err
	}

	if err := mt.toPushTopic(ctx, conversationID, []*sdkws.MsgData{msgData}); err != nil {
		return err
	}

	if err := mt.batcher.Push(msgData); err != nil {
		logc.Errorf(ctx, "failed to push msg to batcher, err: %v", err)
		return err
	}
	return nil
}

// fixNotifyOptions 对未显式配置的默认通知（IsSendMsg）：按"实时信号不打扰"语义，
// 原地关闭离线推送与未读计数，保证在线端与离线端 pull 补齐时行为一致。
// 必须在推送前执行；注意不可用 NewOptions 重建 Options（会丢失 history/persistent
// 等标志，导致落库消息语义错乱）。
func fixNotifyOptions(msg *sdkws.MsgData) {
	msgOpts := msgprocessor.Options(msg.Options)
	if !msgOpts.IsNotNotification() && msgOpts.IsSendMsg() {
		msg.Options = msgprocessor.WithOptions(msg.Options,
			msgprocessor.WithOfflinePush(false),
			msgprocessor.WithUnreadCount(false),
		)
	}
}

func (mt *MsgTransfer) batchHandleMsg(conversationID string, msgs []*sdkws.MsgData) {
	ctx := withOperationID(context.Background(), "")
	mt.handleReadReceipt(ctx, msgs)
	storeMsg, notStoreMsg, storeNotifyMsg, notStoreNotifyMsg := mt.classifyMsgType(ctx, msgs)

	logc.Infof(ctx, "classify msg type, storeMsgCount: %d, notStoreMsgCount: %d, storeNotifyMsgCount: %d, notStoreNotifyMsgCount: %d",
		len(storeMsg), len(notStoreMsg), len(storeNotifyMsg), len(notStoreNotifyMsg))

	// push 已在 consumeMsg 阶段完成，批处理只负责落库攒批
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
		if err := proto.Unmarshal([]byte(msg.Content), &elem); err != nil {
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
			// 已读回执由该用户发出，继承其身份使下游 requireSelfOrAdmin 自然通过
			callCtx := withOperationID(ctx, userID)
			if _, err := mt.msgService.SetConversationHasReadSeq(callCtx, &pbmsg.SetConversationHasReadSeqReq{
				ConversationID: conversationID,
				HasReadSeq:     hasReadSeq,
				UserID:         userID,
				NoNotification: true,
			}); err != nil {
				// 已读进度静默丢失会导致对端已读状态回退
				logc.Errorf(ctx, "failed to set conversation has read seq, userID: %s, conversationID: %s, hasReadSeq: %d, err: %v",
					userID, conversationID, hasReadSeq, err)
			}
		}
	}
}

// allocateMsgSeq 为需要持久化（IsHistory）的消息分配会话级 seq。
// 分配必须先于 push：推送消息携带 seq 供客户端直接排序展示，无需回源拉取。
func (mt *MsgTransfer) allocateMsgSeq(ctx context.Context, conversationID string, msg *sdkws.MsgData) error {
	if !msgprocessor.Options(msg.Options).IsHistory() {
		return nil
	}
	seq, err := mt.seqAllocator.Allocate(ctx, conversationID)
	if err != nil {
		logc.Errorf(ctx, "allocate msg seq failed, err: %v, conversationID: %s, clientMsgID: %s",
			err, conversationID, msg.ClientMsgID)
		return err
	}
	msg.Seq = seq
	return nil
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
	// 按会话 key 投递：同会话批次串行落库，保证 maxSeq 更新不回退
	err = mt.msgPersistentProducer.PushWithKey(ctx, conversationID, string(pbMsg))
	if err != nil {
		logc.Errorf(ctx, "failed to push msg data to mongo by mq, err: %v", err)
		return err
	}

	for _, msg := range msgs {
		mt.webhookMessageSaved(ctx, msg)
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
		// 按会话 key 投递，保证同会话推送到客户端的顺序与 seq 一致
		mt.msgPushProducer.PushWithKey(ctx, conversationID, string(pbMsg))
	}
	return nil
}
