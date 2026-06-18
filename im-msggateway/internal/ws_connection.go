package internal

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/PaperMan11/goim/im-msggateway/internal/compressor"
	"github.com/PaperMan11/goim/im-msggateway/internal/encoder"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logc"
)

var (
	ErrConnClosed = errors.New("conn closed")
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Connection interface {
	ID() string
	Start()
	Close() error
	Send(message []byte) error
	SendResponse(resp *Response) error
	Context() ConnContext
	Error() error
	Server() WsServer
}

type WsConnection struct {
	ctx    ConnContext
	id     string
	conn   *websocket.Conn
	server WsServer
	sendCh chan []byte
	doneCh chan struct{}
	closed atomic.Bool
	err    atomic.Pointer[error]
}

func NewWsConnection(ctx ConnContext, conn *websocket.Conn, server WsServer) *WsConnection {
	return &WsConnection{
		ctx:    ctx,
		id:     fmt.Sprintf("%s-%d", ctx.RemoteAddr(), time.Now().UnixMilli()),
		conn:   conn,
		server: server,
		sendCh: make(chan []byte, 256),
		doneCh: make(chan struct{}),
	}
}

func (c *WsConnection) Start() {
	go c.readPump()
	go c.writePump()
}

func (c *WsConnection) readPump() {
	defer func() {
		c.server.Disconnect(c)
		logc.Debugf(c.ctx, "conn read pump %s closed", c.ID())
	}()

	c.conn.SetReadLimit(c.server.Config().MaxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		logc.Debugf(c.ctx, "conn %s pong message", c.ID())
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	c.conn.SetPingHandler(func(appData string) error {
		logc.Debugf(c.ctx, "conn %s received ping message, sending pong", c.ID())
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		return c.conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	logc.Debugf(c.ctx, "conn read pump %s started", c.ID())

	for {
		select {
		case <-c.doneCh:
			return
		default:
			messageType, message, err := c.conn.ReadMessage()
			if err != nil {
				// 端点主动下线/异常关闭/无状态码关闭
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
					logc.Errorf(c.ctx, "conn %s read message error: %v", c.ID(), err)
				}
				c.closeWithError(err)
				return
			}

			switch messageType {
			case websocket.TextMessage, websocket.BinaryMessage:
				msgType := "text"
				if messageType == websocket.BinaryMessage {
					msgType = "binary"
				}
				msgReceivedCounter.Inc(msgType)
				msgSizeHistogram.ObserveFloat(float64(len(message)))

				err = c.server.HandleMessage(c, message)
				if err != nil {
					c.closeWithError(err)
					if !errors.Is(err, errx.LogoutError) {
						logc.Errorf(c.ctx, "conn %s handle message error: %v", c.ID(), err)
						msgHandleErrorCounter.Inc("handle_failed")
					}
					return
				}
			case websocket.CloseMessage:
				logc.Debugf(c.ctx, "conn %s received close message", c.ID())
				return
			default:
				c.closeWithError(fmt.Errorf("unknown message type: %d", messageType))
				logc.Errorf(c.ctx, "conn %s handle message error: %v", c.ID(), fmt.Errorf("unknown message type: %d", messageType))
				return
			}
		}
	}
}

func (c *WsConnection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.server.Disconnect(c)
		ticker.Stop()
		logc.Debugf(c.ctx, "conn write pump %s closed", c.ID())
	}()

	logc.Debugf(c.ctx, "conn write pump %s started", c.ID())

	for {
		select {
		case <-c.doneCh:
			return
		case message, ok := <-c.sendCh:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				c.closeWithError(err)
				logc.Errorf(c.ctx, "conn %s write message error: %v", c.ID(), err)
				msgHandleErrorCounter.Inc("send_failed")
				return
			}
			msgSentCounter.Inc("binary")
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.closeWithError(err)
				logc.Errorf(c.ctx, "conn %s write ping message error: %v", c.ID(), err)
				return
			}
			logc.Debugf(c.ctx, "conn %s write ping message", c.ID())
		}
	}
}

func (c *WsConnection) flushSendQueue() {
	logc.Debugf(c.ctx, "flushing send queue for conn %s", c.ID())
	for message := range c.sendCh {
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
			c.closeWithError(err)
			logc.Errorf(c.ctx, "conn %s flush send queue error: %v", c.ID(), err)
			return
		}
	}
}

func (c *WsConnection) Server() WsServer {
	return c.server
}

func (c *WsConnection) Close() error {
	if c.closed.Load() {
		return ErrConnClosed
	}
	logc.Debugf(c.ctx, "conn %s closed", c.ID())

	c.closed.Store(true)
	close(c.sendCh)
	close(c.doneCh)
	c.flushSendQueue()
	c.ctx.Close()
	return c.conn.Close()
}

func (c *WsConnection) Send(message []byte) error {
	if c.closed.Load() {
		return ErrConnClosed
	}
	c.sendCh <- message
	return nil
}

func (c *WsConnection) SendResponse(resp *Response) error {
	if c.closed.Load() {
		return ErrConnClosed
	}

	encoder := encoder.GetEncoder(c.ctx.HandshakeInfo().GetSDKType())
	encodeData, err := encoder.Marshal(resp)
	if err != nil {
		logc.Errorf(c.ctx, "SendResponse marshal error: %v", err)
		return err
	}
	compressor := compressor.GetCompressor(c.ctx.HandshakeInfo().GetCompression())
	compressed, err := compressor.Compress(encodeData)
	if err != nil {
		logc.Errorf(c.ctx, "SendResponse compress error: %v", err)
		return err
	}

	select {
	case c.sendCh <- compressed:
		return nil
	case <-time.After(time.Second):
		logc.Errorf(c.ctx, "SendResponse timeout, conn: %s", c.ID())
		return fmt.Errorf("send response timeout")
	}
}

func (c *WsConnection) ReadMessage() (int, []byte, error) {
	return c.conn.ReadMessage()
}

func (c *WsConnection) ID() string {
	return c.id
}

func (c *WsConnection) Context() ConnContext {
	return c.ctx
}

func (c *WsConnection) closeWithError(err error) {
	c.err.CompareAndSwap(nil, &err)
}

func (c *WsConnection) Error() error {
	return *c.err.Load()
}
