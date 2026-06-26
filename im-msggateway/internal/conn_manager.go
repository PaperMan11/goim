package internal

import (
	"sync"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/loginstrategy"
)

type ConnManager interface {
	Add(conn Connection) error
	Remove(connID string) error
	KickOut(connID string) error
	Get(connID string) (Connection, error)
	GetByUserID(userID string) ([]Connection, error)
	GetAll() []Connection
	Count() int64
	CountByUserID(userID string) int64
	Broadcast(message []byte)
	SendTo(connID string, message []byte) error
	SendToUser(userID string, message []byte) error
	CloseAll()
}

type ConnChangeCallback func(conn Connection)

type connManager struct {
	connections         map[string][]Connection // userID -> conn list
	connIndex           map[string]Connection   // connID -> conn
	mu                  sync.RWMutex
	maxConns            int64
	connCount           int64
	loginStrategyConfig LoginStrategyConf
	onRemove            ConnChangeCallback
	onAdd               ConnChangeCallback
}

type Option func(*connManager)

func WithOnRemove(onRemove ConnChangeCallback) Option {
	return func(cm *connManager) {
		cm.onRemove = onRemove
	}
}

func WithOnAdd(onAdd ConnChangeCallback) Option {
	return func(cm *connManager) {
		cm.onAdd = onAdd
	}
}

func NewConnManager(maxConns int64, loginStrategy LoginStrategyConf, opts ...Option) *connManager {
	manager := &connManager{
		connections:         make(map[string][]Connection),
		connIndex:           make(map[string]Connection),
		maxConns:            maxConns,
		loginStrategyConfig: loginStrategy,
	}
	for _, opt := range opts {
		opt(manager)
	}
	return manager
}

func (cm *connManager) Add(conn Connection) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.connCount >= cm.maxConns {
		return errx.ConnOverMaxNumLimit
	}

	if _, exists := cm.connIndex[conn.ID()]; exists {
		return errx.ConnResetError
	}

	return cm.addWithStrategy(conn)
}

func (cm *connManager) addDirectly(conn Connection) error {
	userID := conn.Context().HandshakeInfo().GetUserID()
	cm.connections[userID] = append(cm.connections[userID], conn)
	cm.connIndex[conn.ID()] = conn
	cm.connCount++

	totalConnCounter.Inc()
	activeConnGauge.Set(float64(cm.connCount))
	return nil
}

func (cm *connManager) addWithStrategy(conn Connection) (err error) {
	handshakeInfo := conn.Context().HandshakeInfo()
	userID := handshakeInfo.GetUserID()
	platformID := handshakeInfo.GetPlatformID()

	switch cm.loginStrategyConfig.LoginStrategy {
	case loginstrategy.LoginStrategySingle:
		err = cm.handleSingleLogin(userID, conn)
	case loginstrategy.LoginStrategyReplace:
		err = cm.handleReplaceLogin(userID, conn)
	case loginstrategy.LoginStrategyReplaceSamePlatform:
		err = cm.handleReplaceSamePlatformLogin(userID, platformID, conn)
	case loginstrategy.LoginStrategyAllowMulti:
		fallthrough
	default:
		err = cm.handleAllowMultiLogin(userID, platformID, conn)
	}
	if err != nil && cm.onAdd != nil {
		cm.onAdd(conn)
	}
	return err
}

func (cm *connManager) handleSingleLogin(userID string, conn Connection) error {
	if conns, exists := cm.connections[userID]; exists && len(conns) > 0 {
		return errx.ConnResetError.Wrap("user already logged in on another device")
	}
	return cm.addDirectly(conn)
}

func (cm *connManager) handleReplaceLogin(userID string, conn Connection) error {
	if conns, exists := cm.connections[userID]; exists {
		for _, existingConn := range conns {
			_ = existingConn.SendResponse(&Response{
				ReqIdentifier: WSKickOnlineMsg,
			})
			cm.removeLocked(existingConn.ID())
		}
	}
	return cm.addDirectly(conn)
}

func (cm *connManager) handleReplaceSamePlatformLogin(userID string, platformID int32, conn Connection) error {
	if conns, exists := cm.connections[userID]; exists {
		for _, existingConn := range conns {
			if existingConn.Context().HandshakeInfo().GetPlatformID() == platformID {
				_ = existingConn.SendResponse(&Response{
					ReqIdentifier: WSKickOnlineMsg,
				})
				cm.removeLocked(existingConn.ID())
			}
		}
	}
	return cm.addDirectly(conn)
}

func (cm *connManager) handleAllowMultiLogin(userID string, platformID int32, conn Connection) error {
	existingConns, exists := cm.connections[userID]
	if !exists {
		return cm.addDirectly(conn)
	}

	if int64(len(existingConns)) >= cm.loginStrategyConfig.MaxConnPerUser {
		return errx.ConnOverMaxNumLimit.Wrap("user connection count exceeded")
	}

	samePlatformCount := int64(0)
	for _, existingConn := range existingConns {
		if existingConn.Context().HandshakeInfo().GetPlatformID() == platformID {
			samePlatformCount++
		}
	}

	if samePlatformCount >= cm.loginStrategyConfig.MaxConnPerUserPerPlatform {
		return errx.ConnOverMaxNumLimit.Wrap("user connection count exceeded for this platform")
	}

	return cm.addDirectly(conn)
}

func (cm *connManager) KickOut(connID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	conn, exists := cm.connIndex[connID]
	if !exists {
		return nil
	}
	_ = conn.SendResponse(&Response{
		ReqIdentifier: WSKickOnlineMsg,
	})
	return cm.removeLocked(connID)
}

func (cm *connManager) Remove(connID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.removeLocked(connID)
}

func (cm *connManager) removeLocked(connID string) error {
	conn, exists := cm.connIndex[connID]
	if !exists {
		return nil
	}

	userID := conn.Context().HandshakeInfo().GetUserID()

	delete(cm.connIndex, connID)

	if conns, ok := cm.connections[userID]; ok {
		for i, c := range conns {
			if c.ID() == connID {
				cm.connections[userID] = append(conns[:i], conns[i+1:]...)
				if len(cm.connections[userID]) == 0 {
					delete(cm.connections, userID)
				}
				break
			}
		}
	}

	cm.connCount--

	if cm.onRemove != nil {
		cm.onRemove(conn)
	}

	_ = conn.Close()

	connCloseCounter.Inc("normal")
	activeConnGauge.Set(float64(cm.connCount))
	return nil
}

func (cm *connManager) Get(connID string) (Connection, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.connIndex[connID]
	if !exists {
		return nil, errx.ConnResetError
	}
	return conn, nil
}

func (cm *connManager) GetByUserID(userID string) ([]Connection, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conns, exists := cm.connections[userID]
	if !exists {
		return nil, errx.ConnResetError
	}
	return conns, nil
}

func (cm *connManager) GetAll() []Connection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conns := make([]Connection, 0, cm.connCount)
	for _, userConns := range cm.connections {
		conns = append(conns, userConns...)
	}
	return conns
}

func (cm *connManager) Count() int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.connCount
}

func (cm *connManager) CountByUserID(userID string) int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if conns, exists := cm.connections[userID]; exists {
		return int64(len(conns))
	}
	return 0
}

func (cm *connManager) Broadcast(message []byte) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, conn := range cm.connIndex {
		conn.Send(message)
	}
}

func (cm *connManager) SendTo(connID string, message []byte) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.connIndex[connID]
	if !exists {
		return errx.ConnResetError
	}
	conn.Send(message)
	return nil
}

func (cm *connManager) SendToUser(userID string, message []byte) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conns, exists := cm.connections[userID]
	if !exists {
		return errx.ConnResetError
	}

	for _, conn := range conns {
		conn.Send(message)
	}
	return nil
}

func (cm *connManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	closedCount := int64(0)
	for _, conn := range cm.connIndex {
		if cm.onRemove != nil {
			cm.onRemove(conn)
		}
		_ = conn.Close()
		closedCount++
	}
	cm.connections = make(map[string][]Connection)
	cm.connIndex = make(map[string]Connection)
	cm.connCount = 0

	connCloseCounter.Add(float64(closedCount), "server_stop")
	activeConnGauge.Set(0)
}
