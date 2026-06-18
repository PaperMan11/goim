package internal

import (
	"fmt"
	"time"

	"github.com/PaperMan11/goim/im-msggateway/internal/compressor"
	"github.com/PaperMan11/goim/im-msggateway/internal/encoder"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/zeromicro/go-zero/core/logc"
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
	start := time.Now()
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
}

func NewBusinessHandler() *BusinessHandler {
	return &BusinessHandler{}
}

func (h *BusinessHandler) Handle(conn Connection, req *Request) error {
	logc.Debugf(conn.Context(), "handle business message: %+v", req)
	if conn.Server().Config().EnableAuth && conn.Context().HandshakeInfo().GetUserID() != req.SenderID {
		// replyFn(conn, req, errx.DataError.Wrap("senderID not match"), nil)
		return fmt.Errorf("senderID not match: %s, expect %s", req.SenderID, conn.Context().HandshakeInfo().GetUserID())
	}

	var (
		err    error
		data   []byte
		opName string
	)

	start := time.Now()
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
	case WSPullMsgBySeqList:
		opName = "pull_msg_by_seq"
		businessOpCounter.Inc(opName)
		pullMsgCounter.Inc("by_seq_list")
	case WSSendMsg:
		opName = "send_msg"
		businessOpCounter.Inc(opName)
		sendMsgCounter.Inc("normal")
	case WSSendSignalMsg:
		opName = "send_signal_msg"
		businessOpCounter.Inc(opName)
		sendMsgCounter.Inc("signal")
	case WSPullMsg:
		opName = "pull_msg"
		businessOpCounter.Inc(opName)
		pullMsgCounter.Inc("normal")
	case WSGetConvMaxReadSeq:
		opName = "get_conv_max_read_seq"
		businessOpCounter.Inc(opName)
	case WsPullConvLastMessage:
		opName = "pull_conv_last_msg"
		businessOpCounter.Inc(opName)
		pullMsgCounter.Inc("last_message")
	case WSPushMsg:
		opName = "push_msg"
		businessOpCounter.Inc(opName)
	case WSKickOnlineMsg:
		opName = "kick_online"
		businessOpCounter.Inc(opName)
	case WSLogoutMsg:
		opName = "logout"
		businessOpCounter.Inc(opName)
	case WSSetBackgroundStatus:
		opName = "set_background_status"
		businessOpCounter.Inc(opName)
	case WSSubUserOnlineStatus:
		opName = "sub_online_status"
		businessOpCounter.Inc(opName)
		subscribeCounter.Inc("online_status")
	case WSDataError:
		opName = "data_error"
		businessOpCounter.Inc(opName)
	default:
		opName = "unknown_message"
		err = errx.UnknownMessageError.Wrap(fmt.Sprintf("reqIdentifier: %d", req.ReqIdentifier))
		logc.Errorf(conn.Context(), "unknown message: %+v", req)
	}
	data = []byte(opName) // 模拟返回空数据
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
