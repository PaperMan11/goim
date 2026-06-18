package internal

import (
	"errors"
	"testing"
	"time"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/stretchr/testify/assert"
)

type MockConn struct {
	id           string
	context      *mockConnContext
	closed       bool
	sendMessages [][]byte
	connError    error
}

type mockConnContext struct {
	info *handshakeInfo
}

func (m *mockConnContext) HandshakeInfo() HandshakeInfo {
	return m.info
}

func (m *mockConnContext) Close() error {
	return nil
}

func (m *mockConnContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (m *mockConnContext) Done() <-chan struct{} {
	return nil
}

func (m *mockConnContext) Err() error {
	return nil
}

func (m *mockConnContext) Value(key interface{}) interface{} {
	return nil
}

func (m *mockConnContext) RemoteAddr() string {
	return "127.0.0.1:12345"
}

func (m *mockConnContext) ParseHandshakeRequest() error {
	return nil
}

func NewMockConn(id, userID string, platformID int32, deviceID string) *MockConn {
	return &MockConn{
		id: id,
		context: &mockConnContext{
			info: &handshakeInfo{
				UserID:     userID,
				PlatformID: platformID,
				DeviceID:   deviceID,
			},
		},
	}
}

func (m *MockConn) ID() string {
	return m.id
}

func (m *MockConn) Context() ConnContext {
	return m.context
}

func (m *MockConn) Send(message []byte) error {
	m.sendMessages = append(m.sendMessages, message)
	return nil
}

func (m *MockConn) Close() error {
	m.closed = true
	return nil
}

func (m *MockConn) Start() {
}

func (m *MockConn) IsClosed() bool {
	return m.closed
}

func (m *MockConn) Error() error {
	return m.connError
}

func (m *MockConn) Server() WsServer {
	return nil
}

func (m *MockConn) SendResponse(resp *Response) error {
	return nil
}

func TestConnManager_SingleLoginStrategy(t *testing.T) {
	config := &WsServerConfig{
		LoginStrategyConfig: LoginStrategyConfig{
			LoginStrategy:             LoginStrategySingle,
			MaxConnPerUser:            10,
			MaxConnPerUserPerPlatform: 10,
		},
		MaxConns: 100,
	}
	connManager := NewConnManager(config.MaxConns, config.LoginStrategyConfig)

	userID := "user1"
	conn1 := NewMockConn("conn1", userID, 1, "device1")

	err := connManager.Add(conn1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), connManager.Count())

	conn2 := NewMockConn("conn2", userID, 2, "device2")
	err = connManager.Add(conn2)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errx.ConnResetError))
	assert.Equal(t, int64(1), connManager.Count())
}

func TestConnManager_ReplaceLoginStrategy(t *testing.T) {
	config := &WsServerConfig{
		LoginStrategyConfig: LoginStrategyConfig{
			LoginStrategy:             LoginStrategyReplace,
			MaxConnPerUser:            10,
			MaxConnPerUserPerPlatform: 10,
		},
		MaxConns: 100,
	}
	connManager := NewConnManager(config.MaxConns, config.LoginStrategyConfig)

	userID := "user1"
	conn1 := NewMockConn("conn1", userID, 1, "device1")
	conn2 := NewMockConn("conn2", userID, 2, "device2")
	conn3 := NewMockConn("conn3", userID, 3, "device3")

	err := connManager.Add(conn1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), connManager.Count())

	err = connManager.Add(conn2)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), connManager.Count())
	assert.True(t, conn1.IsClosed())
	assert.False(t, conn2.IsClosed())

	err = connManager.Add(conn3)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), connManager.Count())
	assert.True(t, conn2.IsClosed())
	assert.False(t, conn3.IsClosed())
}

func TestConnManager_ReplaceSamePlatformLoginStrategy(t *testing.T) {
	config := &WsServerConfig{
		LoginStrategyConfig: LoginStrategyConfig{
			LoginStrategy:             LoginStrategyReplaceSamePlatform,
			MaxConnPerUser:            10,
			MaxConnPerUserPerPlatform: 10,
		},
		MaxConns: 100,
	}
	connManager := NewConnManager(config.MaxConns, config.LoginStrategyConfig)

	userID := "user1"
	conn1 := NewMockConn("conn1", userID, 1, "device1")
	conn2 := NewMockConn("conn2", userID, 1, "device2")
	conn3 := NewMockConn("conn3", userID, 2, "device3")
	conn4 := NewMockConn("conn4", userID, 2, "device4")

	err := connManager.Add(conn1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), connManager.Count())

	err = connManager.Add(conn2)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), connManager.Count())
	assert.True(t, conn1.IsClosed())
	assert.False(t, conn2.IsClosed())

	err = connManager.Add(conn3)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), connManager.Count())
	assert.False(t, conn2.IsClosed())
	assert.False(t, conn3.IsClosed())

	err = connManager.Add(conn4)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), connManager.Count())
	assert.True(t, conn3.IsClosed())
	assert.False(t, conn2.IsClosed())
	assert.False(t, conn4.IsClosed())
}

func TestConnManager_AllowMultiLoginStrategy(t *testing.T) {
	config := &WsServerConfig{
		LoginStrategyConfig: LoginStrategyConfig{
			LoginStrategy:             LoginStrategyAllowMulti,
			MaxConnPerUser:            3,
			MaxConnPerUserPerPlatform: 2,
		},
		MaxConns: 100,
	}
	connManager := NewConnManager(config.MaxConns, config.LoginStrategyConfig)

	userID := "user1"

	conn1 := NewMockConn("conn1", userID, 1, "device1")
	err := connManager.Add(conn1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), connManager.Count())

	conn2 := NewMockConn("conn2", userID, 1, "device2")
	err = connManager.Add(conn2)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), connManager.Count())

	conn3 := NewMockConn("conn3", userID, 1, "device3")
	err = connManager.Add(conn3)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errx.ConnOverMaxNumLimit))
	assert.Equal(t, int64(2), connManager.Count())

	conn4 := NewMockConn("conn4", userID, 2, "device4")
	err = connManager.Add(conn4)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), connManager.Count())

	conn5 := NewMockConn("conn5", userID, 3, "device5")
	err = connManager.Add(conn5)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errx.ConnOverMaxNumLimit))
	assert.Equal(t, int64(3), connManager.Count())
}

func TestConnManager_MultipleUsers(t *testing.T) {
	config := &WsServerConfig{
		LoginStrategyConfig: LoginStrategyConfig{
			LoginStrategy:             LoginStrategyAllowMulti,
			MaxConnPerUser:            10,
			MaxConnPerUserPerPlatform: 10,
		},
		MaxConns: 100,
	}
	connManager := NewConnManager(config.MaxConns, config.LoginStrategyConfig)

	user1Conn1 := NewMockConn("user1_conn1", "user1", 1, "device1")
	user1Conn2 := NewMockConn("user1_conn2", "user1", 2, "device2")

	user2Conn1 := NewMockConn("user2_conn1", "user2", 1, "device1")
	user2Conn2 := NewMockConn("user2_conn2", "user2", 2, "device2")

	err := connManager.Add(user1Conn1)
	assert.NoError(t, err)
	err = connManager.Add(user1Conn2)
	assert.NoError(t, err)

	err = connManager.Add(user2Conn1)
	assert.NoError(t, err)
	err = connManager.Add(user2Conn2)
	assert.NoError(t, err)

	assert.Equal(t, int64(4), connManager.Count())
	assert.Equal(t, int64(2), connManager.CountByUserID("user1"))
	assert.Equal(t, int64(2), connManager.CountByUserID("user2"))

	user1Conn3 := NewMockConn("user1_conn3", "user1", 3, "device3")
	err = connManager.Add(user1Conn3)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), connManager.Count())
	assert.Equal(t, int64(3), connManager.CountByUserID("user1"))

	assert.Equal(t, int64(2), connManager.CountByUserID("user2"))
}

func TestConnManager_ConcurrentAccess(t *testing.T) {
	config := &WsServerConfig{
		LoginStrategyConfig: LoginStrategyConfig{
			LoginStrategy:             LoginStrategyAllowMulti,
			MaxConnPerUser:            100,
			MaxConnPerUserPerPlatform: 100,
		},
		MaxConns: 100,
	}
	connManager := NewConnManager(config.MaxConns, config.LoginStrategyConfig)

	userID := "user1"
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			connID := "conn_" + string(rune('0'+index))
			deviceID := "device_" + string(rune('0'+index))
			conn := NewMockConn(connID, userID, int32(index%3+1), deviceID)
			_ = connManager.Add(conn)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	count := connManager.Count()
	assert.True(t, count > 0, "Should have at least one connection")
	assert.True(t, count <= 10, "Should have at most 10 connections")
}

func BenchmarkConnManager_AddConnection(b *testing.B) {
	config := &WsServerConfig{
		LoginStrategyConfig: LoginStrategyConfig{
			LoginStrategy:             LoginStrategyAllowMulti,
			MaxConnPerUser:            10000,
			MaxConnPerUserPerPlatform: 10000,
		},
		MaxConns: 100000,
	}
	connManager := NewConnManager(config.MaxConns, config.LoginStrategyConfig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := "user" + string(rune('0'+(i%10)))
		conn := NewMockConn("conn"+string(rune('0'+(i%100))), userID, 1, "device"+string(rune('0'+(i%100))))
		_ = connManager.Add(conn)
	}
}
