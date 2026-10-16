package msgdispatcher

import (
	"context"
	"encoding/json"

	"github.com/PaperMan11/goim/pkg/msgprocessor"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/proto"
)

type GetUserInfoFunc func(ctx context.Context, userID string) (*sdkws.UserInfo, error)

type notificationOpt struct {
	RpcGetUsername bool
	SendMessage    *bool
}

type NotificationOptions func(*notificationOpt)

func WithRpcGetUsername() NotificationOptions {
	return func(opt *notificationOpt) {
		opt.RpcGetUsername = true
	}
}

func WithSendMessage(enable bool) NotificationOptions {
	return func(opt *notificationOpt) {
		opt.SendMessage = &enable
	}
}

type MsgDispatcher interface {
	SendNotification(ctx context.Context, sendID, recvID, groupID string, contentType, sessionType int32, m proto.Message, opts ...NotificationOptions) error
	SendBatchNotification(ctx context.Context, sendID string, recvIDs []string, contentType, sessionType int32, m proto.Message, opts ...NotificationOptions) error
	SendMsg(ctx context.Context, msgData *sdkws.MsgData) (*pbmsg.SendMsgResp, error)
}

type defaultMsgDispatcher struct {
	msgService     msgservice.MsgService
	getUserInfo    GetUserInfoFunc
	contentTypeMap map[int32]*NotificationConfig
	sessionTypeMap map[int32]int32
}

func NewMsgDispatcher(msgService msgservice.MsgService) MsgDispatcher {
	return &defaultMsgDispatcher{
		msgService:     msgService,
		contentTypeMap: ContentTypeMap,
		sessionTypeMap: SessionTypeMap,
	}
}

func NewMsgDispatcherWithConfig(msgService msgservice.MsgService, notification *Notification, getUserInfo GetUserInfoFunc) MsgDispatcher {
	return &defaultMsgDispatcher{
		msgService:     msgService,
		getUserInfo:    getUserInfo,
		contentTypeMap: BuildContentTypeMap(notification),
		sessionTypeMap: SessionTypeMap,
	}
}

func (d *defaultMsgDispatcher) SendNotification(ctx context.Context, sendID, recvID, groupID string, contentType, sessionType int32, m proto.Message, opts ...NotificationOptions) error {
	// 构建通知内容
	content, err := d.buildNotificationContent(m)
	if err != nil {
		return err
	}

	// 构建消息
	msg := d.buildBaseMsg(sendID, recvID, groupID, contentType, sessionType, content)

	// 构建消息选项
	notificationOpt := d.buildNotificationOpt(opts)
	_ = d.fillSenderInfo(ctx, notificationOpt, msg)
	optionsConfig := d.getNotificationConfig(contentType)
	msg.Options = d.buildMessageOptions(optionsConfig, notificationOpt.SendMessage)
	d.adjustOptionsByContentType(msg.Options, contentType)

	msg.OfflinePushInfo = &sdkws.OfflinePushInfo{
		Title: optionsConfig.OfflinePush.Title,
		Desc:  optionsConfig.OfflinePush.Desc,
		Ex:    optionsConfig.OfflinePush.Ext,
	}

	return d.sendMsg(ctx, msg)
}

func (d *defaultMsgDispatcher) SendBatchNotification(ctx context.Context, sendID string, recvIDs []string, contentType, sessionType int32, m proto.Message, opts ...NotificationOptions) error {
	for _, recvID := range recvIDs {
		if err := d.SendNotification(ctx, sendID, recvID, "", contentType, sessionType, m, opts...); err != nil {
			logx.WithContext(ctx).Errorf("SendBatchNotification failed, sendID: %s, recvID: %s, contentType: %d, err: %v", sendID, recvID, contentType, err)
		}
	}
	return nil
}

func (d *defaultMsgDispatcher) SendMsg(ctx context.Context, msgData *sdkws.MsgData) (*pbmsg.SendMsgResp, error) {
	req := &pbmsg.SendMsgReq{MsgData: msgData}
	return d.msgService.SendMsg(ctx, req)
}

func (d *defaultMsgDispatcher) buildNotificationContent(m proto.Message) (string, error) {
	n := sdkws.NotificationElem{Detail: jsonutilStructToJsonString(m)}
	content, err := json.Marshal(&n)
	if err != nil {
		logx.Error("json.Marshal failed", "msg", jsonutilStructToJsonString(m), "err", err)
		return "", err
	}
	return string(content), nil
}

func (d *defaultMsgDispatcher) buildNotificationOpt(opts []NotificationOptions) *notificationOpt {
	opt := &notificationOpt{}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

func (d *defaultMsgDispatcher) buildBaseMsg(sendID, recvID, groupID string, contentType, sessionType int32, content string) *sdkws.MsgData {
	msg := sdkws.MsgData{
		SendID:      sendID,
		RecvID:      recvID,
		Content:     content,
		MsgFrom:     MsgFromSystem,
		ContentType: contentType,
		SessionType: sessionType,
		CreateTime:  timex.UnixMilli(),
		ClientMsgID: generateClientMsgID(sendID),
	}

	if msg.SessionType == SessionTypeGroup {
		msg.GroupID = groupID
	}

	return &msg
}

func (d *defaultMsgDispatcher) fillSenderInfo(ctx context.Context, opt *notificationOpt, msg *sdkws.MsgData) error {
	if !opt.RpcGetUsername || d.getUserInfo == nil {
		return nil
	}

	userInfo, err := d.getUserInfo(ctx, msg.SendID)
	if err != nil {
		logx.WithContext(ctx).Errorf("getUserInfo failed, sendID: %s, err: %v", msg.SendID, err)
		return err
	}

	msg.SenderNickname = userInfo.Nickname
	msg.SenderFaceURL = userInfo.FaceURL
	return nil
}

func (d *defaultMsgDispatcher) buildMessageOptions(config *NotificationConfig, sendMessage *bool) msgprocessor.Options {
	opts := msgprocessor.NewOptions()

	if sendMessage != nil {
		config.IsSendMsg = *sendMessage
	}

	if config.IsSendMsg {
		opts = msgprocessor.WithOptions(opts, msgprocessor.WithUnreadCount(true))
	}

	if config.OfflinePush.Enable {
		opts = msgprocessor.WithOptions(opts, msgprocessor.WithOfflinePush(true))
	}

	// 两个 Reliable 级别都表示"保证送达"：通知必须落库，离线客户端才能通过 pull 补齐。
	// 二者的差别只体现在通知本身是否携带消息内容（由 config 的 IsSendMsg/UnreadCount/
	// OfflinePush 表达），不应体现为"是否落库"，否则 ReliableNotificationMsg 会退化成
	// UnreliableNotification 的行为（实时推送一次即丢）。
	switch config.ReliabilityLevel {
	case constant.ReliableNotificationNoMsg, constant.ReliableNotificationMsg:
		opts = msgprocessor.WithOptions(opts, msgprocessor.WithHistory(true), msgprocessor.WithPersistent())
	}

	opts = msgprocessor.WithOptions(opts, msgprocessor.WithSendMsg(config.IsSendMsg))
	return opts
}

func (d *defaultMsgDispatcher) adjustOptionsByContentType(options map[string]bool, contentType int32) {
	switch contentType {
	case constant.UserStatusChangeNotification:
		options[constant.IsSenderSync] = false
	}
}

func (d *defaultMsgDispatcher) sendMsg(ctx context.Context, msg *sdkws.MsgData) error {
	req := &pbmsg.SendMsgReq{MsgData: msg}
	_, err := d.msgService.SendMsg(ctx, req)
	if err != nil {
		logx.WithContext(ctx).Errorf("SendMsg failed, err: %v, sendID: %s, recvID: %s", err, msg.SendID, msg.RecvID)
	}
	return err
}

func (d *defaultMsgDispatcher) getNotificationConfig(contentType int32) *NotificationConfig {
	if config, ok := d.contentTypeMap[contentType]; ok {
		return config
	}
	// 未显式配置的通知类型默认按"实时信号"处理（不落库、不进历史、不计未读）：
	// 需要进入会话历史的通知必须在 contentTypeMap 中显式声明可靠级别，
	// 避免新增通知类型默认污染会话历史与未读数。
	return &NotificationConfig{
		IsSendMsg:        true,
		ReliabilityLevel: constant.UnreliableNotification,
		UnreadCount:      true,
		OfflinePush:      OfflinePushConfig{},
	}
}

func jsonutilStructToJsonString(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func generateClientMsgID(senderID string) string {
	s, _ := randx.SecureString(16, randx.CharsAlphaNum)
	return "cli_" + senderID + "_" + s
}
