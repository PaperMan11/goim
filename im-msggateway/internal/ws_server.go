package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/PaperMan11/goim/pkg/apiresp"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	pbauth "github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/PaperMan11/goim/pkg/rpcclient/authservice"
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
	Config() *WsServerConfig
	HandleMessage(conn Connection, message []byte) error
	Disconnect(conn Connection) error
}

type wsServer struct {
	upgrader        *websocket.Upgrader
	connManager     ConnManager
	config          *WsServerConfig
	server          *http.Server
	authService     authservice.AuthService
	messagePipeline *MessagePipeline
}

func NewWsServer(config *WsServerConfig, authService authservice.AuthService, pipeline *MessagePipeline) *wsServer {
	return &wsServer{
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		connManager:     NewConnManager(config.MaxConns),
		config:          config,
		authService:     authService,
		messagePipeline: pipeline,
	}
}

func (s *wsServer) Config() *WsServerConfig {
	return s.config
}

func (s *wsServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.ServeHTTP)
	addr := fmt.Sprintf("%s:%d", s.Config().Host, s.Config().Port)
	s.server = &http.Server{Addr: addr, Handler: mux}
	return s.server.ListenAndServe()
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

func (s *wsServer) Stop() error {
	s.connManager.CloseAll()
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}
	return nil
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
	logc.Debugf(conn.Context(), "disconnecting conn %s", conn.ID())
	return s.connManager.Remove(conn.ID())
}
