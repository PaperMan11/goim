package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/PaperMan11/goim/pkg/apiresp"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	pbauth "github.com/PaperMan11/goim/pkg/protocol/auth"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/rpcclient/authservice"
	"github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/PaperMan11/goim/pkg/webhooks"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
)

type Server interface {
	Start() error
	Stop() error
}

type WsServer interface {
	Server
	Config() *WsServerConf
	// HandleMessage 处理接收到的消息，由 Connection 内部调用
	HandleMessage(conn Connection, message []byte) error
	GetAllConnections() []Connection
	GetUserConnections(userID string) []Connection
	// Disconnect 静默断开连接，不发送踢下线消息
	Disconnect(conn Connection) error
	// Kick 踢下线连接，发送踢下线消息后断开
	Kick(conn Connection) error
	CheckMultiTerminalLogin(connID string) error
}

type wsServer struct {
	upgrader        *websocket.Upgrader
	connManager     ConnManager
	config          *WsServerConf
	server          *http.Server
	authService     authservice.AuthService
	userService     userservice.UserService
	messagePipeline *MessagePipeline
	webhookManager  *webhooks.Manager
}

func NewWsServer(config *WsServerConf, pipeline *MessagePipeline, webhookManager *webhooks.Manager,
	authService authservice.AuthService,
	userService userservice.UserService,
) *wsServer {
	s := &wsServer{
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		config:          config,
		authService:     authService,
		userService:     userService,
		messagePipeline: pipeline,
		webhookManager:  webhookManager,
	}
	s.connManager = NewConnManager(config.MaxConns, config.LoginStrategy,
		WithOnRemove(s.onConnRemove),
		WithOnAdd(s.onConnAdd))
	return s
}

func (s *wsServer) Config() *WsServerConf {
	return s.config
}

func (s *wsServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.ServeHTTP)
	addr := fmt.Sprintf("%s:%d", s.Config().Host, s.Config().Port)
	s.server = &http.Server{Addr: addr, Handler: mux}
	s.webhookManager.Start()
	logx.Infof("WebSocket server starting on %s:%d", s.Config().Host, s.Config().Port)
	return s.server.ListenAndServe()
}

func (s *wsServer) Stop() (err error) {
	s.connManager.CloseAll()
	s.connManager.Stop()
	s.webhookManager.Stop()
	logx.Infof("WebSocket server stopped")
	if s.server != nil {
		err = s.server.Shutdown(context.Background())
	}
	return err
}

func (s *wsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	connContext := newConnContext(w, r)
	if s.connManager.Count() >= s.Config().MaxConns {
		s.handleError(nil, w, errx.ConnOverMaxNumLimit)
		rateLimitCounter.Inc()
		return
	}

	if err := connContext.ParseHandshakeRequest(); err != nil {
		s.handleError(nil, w, errx.HandshakeError.WrapWithError(err))
		return
	}

	if s.Config().EnableAuth && !s.Verify(connContext, w) {
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logc.Errorf(connContext, "Upgrade failed: %v, remoteAddr: %s", err, connContext.RemoteAddr())
		return
	}

	wsConn := NewWsConnection(connContext, conn, s)
	if err := s.connManager.Add(wsConn); err != nil {
		s.handleError(wsConn, w, err)
		_ = wsConn.Close()
		return
	}

	wsConn.Start()
}

func (s *wsServer) Verify(connContext ConnContext, w http.ResponseWriter) bool {
	rpcResp, err := s.authService.ParseToken(connContext, &pbauth.ParseTokenReq{
		Token: connContext.HandshakeInfo().GetToken(),
	})
	if err != nil {
		s.handleError(nil, w, err)
		authFailedCounter.Inc("token_parse_failed")
		return false
	}

	userID := connContext.HandshakeInfo().GetUserID()
	platformID := connContext.HandshakeInfo().GetPlatformID()
	logx.Debugf("Verify, userID: %s, respUserID: %s, platformID: %d, respPlatformID: %d", userID, rpcResp.UserID, platformID, rpcResp.PlatformID)
	if rpcResp.GetUserID() != userID {
		s.handleError(nil, w, errx.TokenInvalidError.Wrap(fmt.Sprintf("user id not match, expect: %s, actual: %s", rpcResp.GetUserID(), userID)))
		authFailedCounter.Inc("user_id_mismatch")
		return false
	}
	if rpcResp.GetPlatformID() != platformID {
		s.handleError(nil, w, errx.TokenInvalidError.Wrap(fmt.Sprintf("platform not match, expect: %d, actual: %d", rpcResp.GetPlatformID(), platformID)))
		authFailedCounter.Inc("platform_mismatch")
		return false
	}
	return true
}

func (s *wsServer) handleError(conn Connection, w http.ResponseWriter, err error) {
	logx.Errorf("HandleError, %v", err)

	if conn == nil {
		apiresp.Error(w, err)
		return
	}
	ctx := conn.Context()
	handshakeRequest := ctx.HandshakeInfo()
	if handshakeRequest == nil || !handshakeRequest.GetSendResponse() {
		apiresp.Error(w, err)
		return
	}
	bytes, marshalErr := jsonx.Marshal(errx.ParseError(err))
	if marshalErr != nil {
		logc.Errorf(ctx, "Marshal error: %v, remoteAddr: %s", marshalErr, ctx.RemoteAddr())
		apiresp.Error(w, err)
		return
	}
	conn.Send(bytes)
}

func (s *wsServer) HandleMessage(conn Connection, message []byte) error {
	if s.messagePipeline == nil {
		logc.Errorf(conn.Context(), "messagePipeline is nil")
		return errors.New("messagePipeline is nil")
	}
	return s.messagePipeline.HandleRaw(conn, message)
}

func (s *wsServer) Disconnect(conn Connection) error {
	return s.connManager.Remove(conn.ID())
}

func (s *wsServer) Kick(conn Connection) error {
	return s.connManager.Kick(conn.ID())
}

// onConnAdded 触发用户上线webhook事件
func (s *wsServer) onConnAdd(conn Connection) {
	ctx := conn.Context()
	handshakeInfo := ctx.HandshakeInfo()
	if s.userService != nil {
		s.userService.SetUserOnlineStatus(ctx, &pbuser.SetUserOnlineStatusReq{
			Status: []*pbuser.UserOnlineStatus{
				{
					UserID:   handshakeInfo.GetUserID(),
					ConnID:   conn.ID(),
					DeviceID: handshakeInfo.GetDeviceID(), // P1新增：把设备ID透传到user-rpc
					Online:   []int32{handshakeInfo.GetPlatformID()},
					Offline:  nil,
				},
			},
		})
	}

	if s.webhookManager != nil {
		userData := &webhooks.UserEventData{
			UserID:       handshakeInfo.GetUserID(),
			PlatformID:   int(handshakeInfo.GetPlatformID()),
			DeviceID:     handshakeInfo.GetDeviceID(),
			OnlineStatus: 1, // 1表示在线
			Extra: map[string]string{
				"remoteAddr":   ctx.RemoteAddr(),
				"isBackground": fmt.Sprintf("%v", handshakeInfo.GetIsBackground()),
				"sdkType":      handshakeInfo.GetSDKType(),
				"sdkVersion":   handshakeInfo.GetSDKVersion(),
			},
		}

		event := webhooks.NewUserOnlineEvent(userData)
		event.OperationID = ctx.HandshakeInfo().GetOperationID()
		if err := s.webhookManager.Dispatch(event); err != nil {
			logc.Errorf(ctx, "Failed to dispatch user online webhook event: %v", err)
		}
	}
}

// onConnRemoved 触发用户下线webhook事件
func (s *wsServer) onConnRemove(conn Connection) {
	ctx := conn.Context()
	handshakeInfo := ctx.HandshakeInfo()

	if s.authService != nil {
		s.authService.KickTokens(ctx, &pbauth.KickTokensReq{
			Tokens: []string{handshakeInfo.GetToken()},
		})
	}

	if s.userService != nil {
		s.userService.SetUserOnlineStatus(ctx, &pbuser.SetUserOnlineStatusReq{
			Status: []*pbuser.UserOnlineStatus{
				{
					UserID:   handshakeInfo.GetUserID(),
					ConnID:   conn.ID(),
					DeviceID: handshakeInfo.GetDeviceID(), // P1新增：把设备ID透传到user-rpc
					Online:   nil,
					Offline:  []int32{handshakeInfo.GetPlatformID()},
				},
			},
		})
	}

	if s.webhookManager != nil {
		userData := &webhooks.UserEventData{
			UserID:       handshakeInfo.GetUserID(),
			PlatformID:   int(handshakeInfo.GetPlatformID()),
			DeviceID:     handshakeInfo.GetDeviceID(),
			OnlineStatus: 0, // 0表示离线
			Extra: map[string]string{
				"remoteAddr":   ctx.RemoteAddr(),
				"isBackground": fmt.Sprintf("%v", handshakeInfo.GetIsBackground()),
				"sdkType":      handshakeInfo.GetSDKType(),
				"sdkVersion":   handshakeInfo.GetSDKVersion(),
			},
		}

		event := webhooks.NewUserOfflineEvent(userData)
		event.OperationID = ctx.HandshakeInfo().GetOperationID()
		if err := s.webhookManager.Dispatch(event); err != nil {
			logc.Errorf(ctx, "Failed to dispatch user offline webhook event: %v", err)
		}
	}
}

func (s *wsServer) GetAllConnections() []Connection {
	return s.connManager.GetAll()
}

func (s *wsServer) GetUserConnections(userID string) []Connection {
	conns, err := s.connManager.GetByUserID(userID)
	if err != nil {
		return nil
	}
	return conns
}

func (s *wsServer) CheckMultiTerminalLogin(connID string) error {
	return s.connManager.CheckMultiTerminalLogin(connID)
}
