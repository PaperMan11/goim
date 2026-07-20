package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/PaperMan11/goim/im-msggateway/internal/compressor"
	"github.com/PaperMan11/goim/im-msggateway/internal/encoder"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/pushservice"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/logc"
	"google.golang.org/protobuf/proto"
)

type MessageHandler interface {
	Handle(conn Connection, req *Request) error
}

type MessageHandlerFunc func(conn Connection, req *Request) error

func (f MessageHandlerFunc) Handle(conn Connection, req *Request) error {
	return f(conn, req)
}

type MessagePipeline struct {
	handlers []Middleware
	handler  MessageHandler
}

type Middleware func(next MessageHandler) MessageHandler

func NewMessagePipeline(handler MessageHandler, middlewares ...Middleware) *MessagePipeline {
	pipeline := &MessagePipeline{
		handler: handler,
	}
	pipeline.handlers = append(pipeline.handlers, middlewares...)
	return pipeline
}

func (p *MessagePipeline) Handle(conn Connection, req *Request) error {
	handler := p.handler
	for i := len(p.handlers) - 1; i >= 0; i-- {
		handler = p.handlers[i](handler)
	}
	return handler.Handle(conn, req)
}

func (p *MessagePipeline) HandleRaw(conn Connection, message []byte) error {
	start := timex.Now()
	defer func() {
		duration := float64(time.Since(start).Milliseconds())
		msgHandleDurationHistogram.ObserveFloat(duration)
	}()

	req := MallocRequest()
	defer FreeRequest(req)

	compression := conn.Context().HandshakeInfo().GetCompression()
	compressor := compressor.GetCompressor(compression)
	decompressed, err := compressor.Decompress(message)
	if err != nil {
		msgHandleErrorCounter.Inc("decompress_failed")
		return err
	}

	sdkType := conn.Context().HandshakeInfo().GetSDKType()
	decoder := encoder.GetEncoder(sdkType)
	if err := decoder.Unmarshal(decompressed, req); err != nil {
		msgHandleErrorCounter.Inc("unmarshal_failed")
		return err
	}

	return p.Handle(conn, req)
}

// ----------------------------------------------------
// BusinessHandler 处理业务消息
// ----------------------------------------------------

type BusinessHandler struct {
	pushService pushservice.PushService
	msgService  msgservice.MsgService
}

func NewBusinessHandler(pushService pushservice.PushService, msgService msgservice.MsgService) *BusinessHandler {
	return &BusinessHandler{
		pushService: pushService,
		msgService:  msgService,
	}
}

func (h *BusinessHandler) Handle(conn Connection, req *Request) error {
	logc.Debugf(conn.Context(), "handle business message: %+v", req)
	if conn.Server().Config().EnableAuth && conn.Context().HandshakeInfo().GetUserID() != req.SenderID {
		h.reply(conn, req, errx.DataError.Wrap("senderID not match"), nil)
		return fmt.Errorf("senderID not match: %s, expect %s", req.SenderID, conn.Context().HandshakeInfo().GetUserID())
	}

	var (
		err    error
		data   []byte
		opName string
	)

	start := timex.Now()
	defer func() {
		duration := float64(time.Since(start).Milliseconds())
		businessOpDurationHistogram.ObserveFloat(duration, opName)
		if err != nil {
			businessOpErrorCounter.Inc(opName)
		}
	}()

	switch req.ReqIdentifier {
	case WSGetNewestSeq:
		opName = "get_newest_seq"
		businessOpCounter.Inc(opName)
		data, err = h.handleGetNewestSeq(conn.Context(), req)
	case WSPullMsgBySeqList:
		opName = "pull_msg_by_seq"
		businessOpCounter.Inc(opName)
		pullMsgCounter.Inc("by_seq_list")
		data, err = h.handlePullMsgBySeqList(conn.Context(), req)
	case WSSendMsg:
		opName = "send_msg"
		businessOpCounter.Inc(opName)
		sendMsgCounter.Inc("normal")
		data, err = h.handleSendMsg(conn.Context(), req)
	case WSSendSignalMsg:
		opName = "send_signal_msg"
		businessOpCounter.Inc(opName)
		sendMsgCounter.Inc("signal")
		data, err = h.handleSendSignalMsg(conn.Context(), req)
	case WSPullMsg:
		opName = "pull_msg"
		businessOpCounter.Inc(opName)
		pullMsgCounter.Inc("normal")
		data, err = h.handlePullMsg(conn.Context(), req)
	case WSGetConvMaxReadSeq:
		opName = "get_conv_max_read_seq"
		businessOpCounter.Inc(opName)
		data, err = h.handleGetConvMaxReadSeq(conn.Context(), req)
	case WsPullConvLastMessage:
		opName = "pull_conv_last_msg"
		businessOpCounter.Inc(opName)
		pullMsgCounter.Inc("last_message")
		data, err = h.handlePullConvLastMessage(conn.Context(), req)
	case WSPushMsg:
		opName = "push_msg"
		businessOpCounter.Inc(opName)
		data, err = h.handlePushMsg(conn.Context(), req)
	case WSKickOnlineMsg:
		opName = "kick_online"
		businessOpCounter.Inc(opName)
		data, err = h.handleKickOnlineMsg(conn.Context(), req)
	case WSLogoutMsg:
		opName = "logout"
		businessOpCounter.Inc(opName)
		data, err = h.handleLogoutMsg(conn.Context(), req)
	case WSSetBackgroundStatus:
		opName = "set_background_status"
		businessOpCounter.Inc(opName)
		data, err = h.handleSetBackgroundStatus(conn.Context(), req)
	case WSSubUserOnlineStatus:
		opName = "sub_online_status"
		businessOpCounter.Inc(opName)
		subscribeCounter.Inc("online_status")
		data, err = h.handleSubUserOnlineStatus(conn.Context(), req)
	case WSDataError:
		opName = "data_error"
		businessOpCounter.Inc(opName)
		data, err = h.handleDataError(conn.Context(), req)
	default:
		opName = "unknown_message"
		err = errx.UnknownMessageError.Wrap(fmt.Sprintf("reqIdentifier: %d", req.ReqIdentifier))
		logc.Errorf(conn.Context(), "unknown message: %+v", req)
	}
	return h.reply(conn, req, err, data)
}

func (h *BusinessHandler) reply(conn Connection, req *Request, err error, data []byte) error {
	errInfo := errx.Success
	if err != nil {
		errInfo = errx.ParseError(err)
	}
	reply := Response{
		ReqIdentifier: req.ReqIdentifier,
		MsgIncr:       req.MsgIncr,
		OperationID:   req.OperationID,
		ErrCode:       errInfo.Code,
		ErrMsg:        errInfo.Message,
		Data:          data,
	}

	if err := conn.SendResponse(&reply); err != nil {
		return err
	}

	if req.ReqIdentifier == WSLogoutMsg {
		return errx.LogoutError.Wrap(fmt.Sprintf("userID: %s", conn.Context().HandshakeInfo().GetUserID()))
	}
	return nil
}

// handleGetNewestSeq 处理获取最新消息序列请求
func (h *BusinessHandler) handleGetNewestSeq(ctx context.Context, req *Request) (respData []byte, err error) {
	var reqData sdkws.GetMaxSeqReq
	if err := proto.Unmarshal(req.Data, &reqData); err != nil {
		return nil, errx.DataError.Wrap("get_newest_seq req data unmarshal error")
	}
	resp, err := h.msgService.GetMaxSeq(ctx, &reqData)
	if err != nil {
		return nil, err
	}
	respData, err = proto.Marshal(resp)
	if err != nil {
		return nil, errx.DataError.Wrap("get_newest_seq resp data marshal error")
	}
	return respData, nil
}

// handleSendMsg 处理发送消息请求
func (h *BusinessHandler) handleSendMsg(ctx context.Context, req *Request) (respData []byte, err error) {
	var reqData sdkws.MsgData
	if err := proto.Unmarshal(req.Data, &reqData); err != nil {
		return nil, errx.DataError.Wrap("send_msg req data unmarshal error")
	}
	resp, err := h.msgService.SendMsg(ctx, &pbmsg.SendMsgReq{
		MsgData: &reqData,
	})
	if err != nil {
		return nil, err
	}
	respData, err = proto.Marshal(resp)
	if err != nil {
		return nil, errx.DataError.Wrap("send_msg resp data marshal error")
	}
	return respData, nil
}

// handleSendSignalMsg 处理发送信号消息请求
func (h *BusinessHandler) handleSendSignalMsg(ctx context.Context, req *Request) (respData []byte, err error) {
	var reqData sdkws.MsgData
	if err := proto.Unmarshal(req.Data, &reqData); err != nil {
		return nil, errx.DataError.Wrap("send_signal_msg req data unmarshal error")
	}
	resp, err := h.msgService.SendMsg(ctx, &pbmsg.SendMsgReq{
		MsgData: &reqData,
	})
	if err != nil {
		return nil, err
	}
	respData, err = proto.Marshal(resp)
	if err != nil {
		return nil, errx.DataError.Wrap("send_signal_msg resp data marshal error")
	}
	return respData, nil
}

// handlePullMsgBySeqList 处理拉取消消息列表请求
func (h *BusinessHandler) handlePullMsgBySeqList(ctx context.Context, req *Request) (respData []byte, err error) {
	var reqData sdkws.PullMessageBySeqsReq
	if err := proto.Unmarshal(req.Data, &reqData); err != nil {
		return nil, errx.DataError.Wrap("pull_msg_by_seq_list req data unmarshal error")
	}
	resp, err := h.msgService.PullMessageBySeqs(ctx, &reqData)
	if err != nil {
		return nil, err
	}
	respData, err = proto.Marshal(resp)
	if err != nil {
		return nil, errx.DataError.Wrap("pull_msg_by_seq_list resp data marshal error")
	}
	return respData, nil
}

// handlePullMsg 处理拉取消消息请求
func (h *BusinessHandler) handlePullMsg(ctx context.Context, req *Request) (respData []byte, err error) {
	var reqData pbmsg.GetSeqMessageReq
	if err := proto.Unmarshal(req.Data, &reqData); err != nil {
		return nil, errx.DataError.Wrap("pull_msg req data unmarshal error")
	}
	resp, err := h.msgService.GetSeqMessage(ctx, &reqData)
	if err != nil {
		return nil, err
	}
	respData, err = proto.Marshal(resp)
	if err != nil {
		return nil, errx.DataError.Wrap("pull_msg resp data marshal error")
	}
	return respData, nil
}

// handleGetConvMaxReadSeq 处理获取会话最大已读序列请求
func (h *BusinessHandler) handleGetConvMaxReadSeq(ctx context.Context, req *Request) (respData []byte, err error) {
	var reqData pbmsg.GetConversationsHasReadAndMaxSeqReq
	if err := proto.Unmarshal(req.Data, &reqData); err != nil {
		return nil, errx.DataError.Wrap("get_conv_max_read_seq req data unmarshal error")
	}
	resp, err := h.msgService.GetConversationsHasReadAndMaxSeq(ctx, &reqData)
	if err != nil {
		return nil, err
	}
	respData, err = proto.Marshal(resp)
	if err != nil {
		return nil, errx.DataError.Wrap("get_conv_max_read_seq resp data marshal error")
	}
	return respData, nil
}

// handlePullConvLastMessage 处理拉取会话最后一条消息请求
func (h *BusinessHandler) handlePullConvLastMessage(ctx context.Context, req *Request) (respData []byte, err error) {
	var reqData pbmsg.GetLastMessageReq
	if err := proto.Unmarshal(req.Data, &reqData); err != nil {
		return nil, errx.DataError.Wrap("pull_conv_last_msg req data unmarshal error")
	}
	resp, err := h.msgService.GetLastMessage(ctx, &reqData)
	if err != nil {
		return nil, err
	}
	respData, err = proto.Marshal(resp)
	if err != nil {
		return nil, errx.DataError.Wrap("pull_conv_last_msg resp data marshal error")
	}
	return respData, nil
}

// handlePushMsg 处理推送消息请求
func (h *BusinessHandler) handlePushMsg(ctx context.Context, req *Request) (respData []byte, err error) {
	return nil, errx.DataError.Wrap("push_msg req not support")
}

// handleKickOnlineMsg 处理踢掉旧连接请求
func (h *BusinessHandler) handleKickOnlineMsg(ctx context.Context, req *Request) (respData []byte, err error) {
	return nil, errx.DataError.Wrap("kick_online_msg req not support")
}

// handleLogoutMsg 处理登出请求
func (h *BusinessHandler) handleLogoutMsg(ctx context.Context, req *Request) (respData []byte, err error) {
	return nil, errx.DataError.Wrap("logout_msg req not support")
}

// handleSetBackgroundStatus 处理设置背景状态请求
func (h *BusinessHandler) handleSetBackgroundStatus(ctx context.Context, req *Request) (respData []byte, err error) {
	return nil, errx.DataError.Wrap("set_background_status req not support")
}

// handleSubUserOnlineStatus 处理订阅用户在线状态请求
func (h *BusinessHandler) handleSubUserOnlineStatus(ctx context.Context, req *Request) (respData []byte, err error) {
	return nil, errx.DataError.Wrap("sub_user_online_status req not support")
}

// handleDataError 处理数据错误请求
func (h *BusinessHandler) handleDataError(ctx context.Context, req *Request) (respData []byte, err error) {
	return nil, errx.DataError.Wrap("data_error req not support")
}
