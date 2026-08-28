package internal

import (
	"runtime"
	"sync"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/loginstrategy"
	"github.com/PaperMan11/goim/pkg/metrics"
	"github.com/PaperMan11/goim/pkg/utils/workerpool"
)

type ConnManager interface {
	// 连接管理
	Add(conn Connection) error
	// Remove 静默移除连接，不发送踢下线消息
	Remove(connID string) error
	// Kick 踢下线连接，发送踢下线消息后移除
	Kick(connID string) error

	// 连接查询
	Get(connID string) (Connection, error)
	GetByUserID(userID string) ([]Connection, error)
	GetAll() []Connection
	Count() int64
	CountByUserID(userID string) int64

	// 消息发送
	Broadcast(message []byte)
	SendTo(connID string, message []byte) error
	SendToUser(userID string, message []byte) error

	// 批量操作
	CloseAll()
	// CheckMultiTerminalLogin 检查多终端登录策略
	CheckMultiTerminalLogin(connID string) error
	Stop()
}

type ConnChangeCallback func(conn Connection)

type strategyCheckRequest struct {
	conn   Connection
	addSeq uint64
}

type connManager struct {
	connections         map[string][]Connection // userID -> conn list
	connIndex           map[string]Connection   // connID -> conn
	mu                  sync.RWMutex
	maxConns            int64
	connCount           int64
	loginStrategyConfig LoginStrategyConf
	onRemove            ConnChangeCallback
	onAdd               ConnChangeCallback

	strategyCheckPool *workerpool.WorkerPool[*strategyCheckRequest]
	workerCount       int
	queueSize         int

	connSeq    uint64
	connAddSeq map[string]uint64 // connID -> sequence number
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

func WithStrategyCheckQueue(queueSize int, workerCount int) Option {
	return func(cm *connManager) {
		cm.queueSize = queueSize
		cm.workerCount = workerCount
	}
}

func NewConnManager(maxConns int64, loginStrategy LoginStrategyConf, opts ...Option) *connManager {
	manager := &connManager{
		connections:         make(map[string][]Connection),
		connIndex:           make(map[string]Connection),
		maxConns:            maxConns,
		loginStrategyConfig: loginStrategy,
		workerCount:         runtime.NumCPU(),
		queueSize:           1024,
		connAddSeq:          make(map[string]uint64),
	}
	for _, opt := range opts {
		opt(manager)
	}

	manager.strategyCheckPool = workerpool.New(manager.processStrategyCheck, manager.workerCount, manager.queueSize)
	manager.strategyCheckPool.Start()

	return manager
}

func (cm *connManager) Stop() {
	cm.strategyCheckPool.Stop()
}

func (cm *connManager) Add(conn Connection) error {
	cm.mu.Lock()

	if cm.connCount >= cm.maxConns {
		cm.mu.Unlock()
		return errx.ConnOverMaxNumLimit
	}

	if _, exists := cm.connIndex[conn.ID()]; exists {
		cm.mu.Unlock()
		return errx.ConnResetError
	}

	// handshakeInfo := conn.Context().HandshakeInfo()
	// userID := handshakeInfo.GetUserID()
	// platformID := handshakeInfo.GetPlatformID()

	// if cm.loginStrategyConfig.LoginStrategy == loginstrategy.LoginStrategyAllowMulti {
	// 	existingConns, exists := cm.connections[userID]
	// 	if exists && int64(len(existingConns)) >= cm.loginStrategyConfig.MaxConnPerUser {
	// 		cm.mu.Unlock()
	// 		return errx.ConnOverMaxNumLimit.Wrap("user connection count exceeded")
	// 	}

	// 	samePlatformCount := int64(0)
	// 	if exists {
	// 		for _, existingConn := range existingConns {
	// 			if existingConn.Context().HandshakeInfo().GetPlatformID() == platformID {
	// 				samePlatformCount++
	// 			}
	// 		}
	// 	}
	// 	if samePlatformCount >= cm.loginStrategyConfig.MaxConnPerUserPerPlatform {
	// 		cm.mu.Unlock()
	// 		return errx.ConnOverMaxNumLimit.Wrap("user connection count exceeded for this platform")
	// 	}
	// }

	// 分配序列号，确保在锁内进行
	cm.connSeq++
	addSeq := cm.connSeq

	if err := cm.addDirectly(conn, addSeq); err != nil {
		cm.mu.Unlock()
		return err
	}
	cm.mu.Unlock()

	// if cm.loginStrategyConfig.LoginStrategy != loginstrategy.LoginStrategyAllowMulti {
	cm.strategyCheckPool.Submit(&strategyCheckRequest{conn: conn, addSeq: addSeq})
	// }

	return nil
}

func (cm *connManager) addDirectly(conn Connection, addSeq uint64) error {
	userID := conn.Context().HandshakeInfo().GetUserID()
	cm.connections[userID] = append(cm.connections[userID], conn)
	cm.connIndex[conn.ID()] = conn
	cm.connAddSeq[conn.ID()] = addSeq
	cm.connCount++

	metrics.TotalConnCounter.Inc()
	metrics.ActiveConnGauge.Set(float64(cm.connCount))
	return nil
}

func (cm *connManager) CheckMultiTerminalLogin(connID string) error {
	cm.mu.RLock()
	conn, exists := cm.connIndex[connID]
	if !exists {
		cm.mu.RUnlock()
		return errx.ConnResetError
	}
	addSeq, exists := cm.connAddSeq[connID]
	if !exists {
		cm.mu.RUnlock()
		return errx.ConnResetError
	}
	cm.mu.RUnlock()

	cm.strategyCheckPool.Submit(&strategyCheckRequest{conn: conn, addSeq: addSeq})
	return nil
}

func (cm *connManager) processStrategyCheck(req *strategyCheckRequest) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conn := req.conn
	handshakeInfo := conn.Context().HandshakeInfo()
	userID := handshakeInfo.GetUserID()
	platformID := handshakeInfo.GetPlatformID()
	connID := conn.ID()

	if _, exists := cm.connIndex[connID]; !exists {
		return
	}

	// 获取当前连接的序列号，如果已经被更新过则跳过
	if currentSeq, exists := cm.connAddSeq[connID]; exists && currentSeq > req.addSeq {
		return
	}

	switch cm.loginStrategyConfig.LoginStrategy {
	case loginstrategy.LoginStrategySingle:
		cm.checkSingleLoginAsync(userID, connID, req.addSeq)
	case loginstrategy.LoginStrategyReplace:
		cm.checkReplaceLoginAsync(userID, connID, req.addSeq)
	case loginstrategy.LoginStrategyReplaceSamePlatform:
		cm.checkReplaceSamePlatformLoginAsync(userID, platformID, connID, req.addSeq)
	case loginstrategy.LoginStrategyAllowMulti:
		fallthrough
	default:
		cm.checkAllowMultiLoginAsync(userID, platformID, connID, req.addSeq)
	}

	if cm.onAdd != nil {
		cm.onAdd(conn)
	}
}

func (cm *connManager) checkSingleLoginAsync(userID string, newConnID string, addSeq uint64) {
	conns := cm.connections[userID]
	if len(conns) <= 1 {
		return
	}

	for _, existingConn := range conns {
		if existingConn.ID() != newConnID {
			// 只有当现有连接的序列号小于当前请求的序列号时才踢掉
			if existingSeq, exists := cm.connAddSeq[existingConn.ID()]; !exists || existingSeq < addSeq {
				_ = existingConn.SendResponse(&Response{
					ReqIdentifier: WSKickOnlineMsg,
				})
				cm.removeLocked(existingConn.ID())
			}
		}
	}
}

func (cm *connManager) checkReplaceLoginAsync(userID string, newConnID string, addSeq uint64) {
	conns := cm.connections[userID]
	for _, existingConn := range conns {
		if existingConn.ID() != newConnID {
			if existingSeq, exists := cm.connAddSeq[existingConn.ID()]; !exists || existingSeq < addSeq {
				_ = existingConn.SendResponse(&Response{
					ReqIdentifier: WSKickOnlineMsg,
				})
				cm.removeLocked(existingConn.ID())
			}
		}
	}
}

func (cm *connManager) checkReplaceSamePlatformLoginAsync(userID string, platformID int32, newConnID string, addSeq uint64) {
	conns := cm.connections[userID]
	for _, existingConn := range conns {
		if existingConn.ID() != newConnID && existingConn.Context().HandshakeInfo().GetPlatformID() == platformID {
			if existingSeq, exists := cm.connAddSeq[existingConn.ID()]; !exists || existingSeq < addSeq {
				_ = existingConn.SendResponse(&Response{
					ReqIdentifier: WSKickOnlineMsg,
				})
				cm.removeLocked(existingConn.ID())
			}
		}
	}
}

func (cm *connManager) checkAllowMultiLoginAsync(userID string, platformID int32, newConnID string, addSeq uint64) {
	conns := cm.connections[userID]

	// 检查总连接数限制
	if int64(len(conns)) > cm.loginStrategyConfig.MaxConnPerUser {
		// 踢掉最旧的连接（保留新连接）
		for i := 0; i < len(conns)-int(cm.loginStrategyConfig.MaxConnPerUser); i++ {
			if conns[i].ID() != newConnID {
				if existingSeq, exists := cm.connAddSeq[conns[i].ID()]; !exists || existingSeq < addSeq {
					_ = conns[i].SendResponse(&Response{
						ReqIdentifier: WSKickOnlineMsg,
					})
					cm.removeLocked(conns[i].ID())
				}
			}
		}
	}

	// 检查同平台连接数限制
	conns = cm.connections[userID] //重新获取更新后的连接列表
	samePlatformConns := make([]Connection, 0)
	for _, c := range conns {
		if c.Context().HandshakeInfo().GetPlatformID() == platformID {
			samePlatformConns = append(samePlatformConns, c)
		}
	}

	if int64(len(samePlatformConns)) > cm.loginStrategyConfig.MaxConnPerUserPerPlatform {
		// 踢掉同平台最旧的连接（保留新连接）
		for i := 0; i < len(samePlatformConns)-int(cm.loginStrategyConfig.MaxConnPerUserPerPlatform); i++ {
			if samePlatformConns[i].ID() != newConnID {
				if existingSeq, exists := cm.connAddSeq[samePlatformConns[i].ID()]; !exists || existingSeq < addSeq {
					_ = samePlatformConns[i].SendResponse(&Response{
						ReqIdentifier: WSKickOnlineMsg,
					})
					cm.removeLocked(samePlatformConns[i].ID())
				}
			}
		}
	}
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
	cm.connSeq++
	return cm.addDirectly(conn, cm.connSeq)
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
	cm.connSeq++
	return cm.addDirectly(conn, cm.connSeq)
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
	cm.connSeq++
	return cm.addDirectly(conn, cm.connSeq)
}

func (cm *connManager) handleAllowMultiLogin(userID string, platformID int32, conn Connection) error {
	existingConns, exists := cm.connections[userID]
	if !exists {
		cm.connSeq++
		return cm.addDirectly(conn, cm.connSeq)
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

	cm.connSeq++
	return cm.addDirectly(conn, cm.connSeq)
}

func (cm *connManager) Kick(connID string) error {
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
	delete(cm.connAddSeq, connID)

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

	if cm.connCount == 0 {
		cm.connSeq = 0
		cm.connAddSeq = make(map[string]uint64)
	}

	if cm.onRemove != nil {
		cm.onRemove(conn)
	}

	_ = conn.Close()

	metrics.ConnCloseCounter.Inc("normal")
	metrics.ActiveConnGauge.Set(float64(cm.connCount))
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
	cm.connSeq = 0
	cm.connAddSeq = make(map[string]uint64)

	metrics.ConnCloseCounter.Add(float64(closedCount), "server_stop")
	metrics.ActiveConnGauge.Set(0)
}
