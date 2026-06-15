package internal

import (
	"sync"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
)

type ConnManager interface {
	Add(conn Connection) error
	Remove(connID string) error
	Get(connID string) (Connection, error)
	GetAll() []Connection
	Count() int64
	Broadcast(message []byte)
	SendTo(connID string, message []byte) error
	CloseAll()
}

type connManager struct {
	connections map[string]Connection
	mu          sync.RWMutex
	maxConns    int64
	connCount   int64
}

func NewConnManager(maxConns int64) *connManager {
	return &connManager{
		connections: make(map[string]Connection),
		maxConns:    maxConns,
	}
}

func (cm *connManager) Add(conn Connection) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.connCount >= cm.maxConns {
		return errx.ConnOverMaxNumLimit
	}

	if _, exists := cm.connections[conn.ID()]; exists {
		return errx.ConnResetError
	}

	cm.connections[conn.ID()] = conn
	cm.connCount++

	totalConnCounter.Inc()
	activeConnGauge.Set(float64(cm.connCount))
	return nil
}

func (cm *connManager) Remove(connID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, exists := cm.connections[connID]; exists {
		delete(cm.connections, connID)
		cm.connCount--
		_ = conn.Close()

		connCloseCounter.Inc("normal")
		activeConnGauge.Set(float64(cm.connCount))
	}
	return nil
}

func (cm *connManager) Get(connID string) (Connection, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.connections[connID]
	if !exists {
		return nil, errx.ConnResetError
	}
	return conn, nil
}

func (cm *connManager) GetAll() []Connection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conns := make([]Connection, 0, len(cm.connections))
	for _, conn := range cm.connections {
		conns = append(conns, conn)
	}
	return conns
}

func (cm *connManager) Count() int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.connCount
}

func (cm *connManager) Broadcast(message []byte) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, conn := range cm.connections {
		conn.Send(message)
	}
}

func (cm *connManager) SendTo(connID string, message []byte) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.connections[connID]
	if !exists {
		return errx.ConnResetError
	}
	conn.Send(message)
	return nil
}

func (cm *connManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	closedCount := int64(0)
	for _, conn := range cm.connections {
		_ = conn.Close()
		closedCount++
	}
	cm.connections = make(map[string]Connection)
	cm.connCount = 0

	connCloseCounter.Add(float64(closedCount), "server_stop")
	activeConnGauge.Set(0)
}
