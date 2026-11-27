package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/PaperMan11/goim/pkg/apiresp"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/metrics"
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
	stopCh          chan struct{}
	stopOnce        sync.Once
}

// statusHeartbeatBatchSize 单次心跳续期 RPC 携带的最大连接数，避免单请求过大。
const statusHeartbeatBatchSize = 200

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
		stopCh:          make(chan struct{}),
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
	go s.statusHeartbeatLoop()
	logx.Infof("WebSocket server starting on %s:%d", s.Config().Host, s.Config().Port)
	return s.server.ListenAndServe()
}

func (s *wsServer) Stop() (err error) {
	s.stopOnce.Do(func() { close(s.stopCh) })
	s.connManager.CloseAll()
	s.connManager.Stop()
	s.webhookManager.Stop()
	logx.Infof("WebSocket server stopped")
	if s.server != nil {
		err = s.server.Shutdown(context.Background())
	}
	return err
}

// statusHeartbeatLoop 周期性把本网关存活连接对应的用户在线状态写回 user-rpc。
// 作用：刷新 Mongo 的 last_seen_at 与 Redis score，使得进程异常退出（kill -9、OOM）
// 后状态能在 ZombieThresholdMs 内自动判定为离线，而不是长期"幽灵在线"。
func (s *wsServer) statusHeartbeatLoop() {
	interval := s.config.StatusHeartbeatSeconds
	if interval <= 0 || s.userService == nil {
		return
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.refreshOnlineStatus()
		}
	}
}

func (s *wsServer) refreshOnlineStatus() {
	conns := s.connManager.GetAll()
	if len(conns) == 0 {
		return
	}

	operationID := mcontext.NewOperationID("ws_status_heartbeat")
	ctx, cancel := context.WithTimeout(mcontext.SetOperationIDInContext(context.Background(), operationID), 10*time.Second)
	defer cancel()

	batch := make([]*pbuser.UserOnlineStatus, 0, statusHeartbeatBatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		_, err := s.userService.SetUserOnlineStatus(ctx, &pbuser.SetUserOnlineStatusReq{
			Status: batch,
			Silent: true,
		})
		if err != nil {
			logx.Errorf("refresh online status failed, connCount: %d, err: %v", len(batch), err)
		}
		batch = make([]*pbuser.UserOnlineStatus, 0, statusHeartbeatBatchSize)
	}

	for _, conn := range conns {
		handshakeInfo := conn.Context().HandshakeInfo()
		userID := handshakeInfo.GetUserID()
		if userID == "" {
			continue
		}
		batch = append(batch, &pbuser.UserOnlineStatus{
			UserID:   userID,
			ConnID:   conn.ID(),
			DeviceID: handshakeInfo.GetDeviceID(),
			Online:   []int32{handshakeInfo.GetPlatformID()},
		})
		if len(batch) >= statusHeartbeatBatchSize {
			flush()
		}
	}
	flush()
}

func (s *wsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	connContext := newConnContext(w, r)
	if s.connManager.Count() >= s.Config().MaxConns {
		s.handleError(nil, w, errx.ConnOverMaxNumLimit)
		metrics.RateLimitCounter.Inc()
		return
	}

	if err := connContext.ParseHandshakeRequest(); err != nil {
		s.handleError(nil, w, errx.HandshakeError.WrapWithError(err))
		return
	}

	if s.Config().EnableAuth && !s.Verify(connContext, w) {
		return
	}

	// 鉴权通过（或未开启鉴权）后，userID 才可信，注入操作人身份，
	// 之后该连接上的所有 RPC 都会通过 metadata 透传该身份。
	connContext.SetAuthenticatedUser(connContext.HandshakeInfo().GetUserID())

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logc.Errorf(connContext, "Upgrade failed: %v, remoteAddr: %s", err, connContext.RemoteAddr())
		return
	}

	wsConn := NewWsConnection(connContext, conn, s)
	connContext.SetConnID(wsConn.ID())
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
		metrics.AuthFailedCounter.Inc("token_parse_failed")
		return false
	}

	userID := connContext.HandshakeInfo().GetUserID()
	platformID := connContext.HandshakeInfo().GetPlatformID()
	logx.Debugf("Verify, userID: %s, respUserID: %s, platformID: %d, respPlatformID: %d", userID, rpcResp.UserID, platformID, rpcResp.PlatformID)
	if rpcResp.GetUserID() != userID {
		s.handleError(nil, w, errx.TokenInvalidError.Wrap(fmt.Sprintf("user id not match, expect: %s, actual: %s", rpcResp.GetUserID(), userID)))
		metrics.AuthFailedCounter.Inc("user_id_mismatch")
		return false
	}
	if rpcResp.GetPlatformID() != platformID {
		s.handleError(nil, w, errx.TokenInvalidError.Wrap(fmt.Sprintf("platform not match, expect: %d, actual: %d", rpcResp.GetPlatformID(), platformID)))
		metrics.AuthFailedCounter.Inc("platform_mismatch")
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
