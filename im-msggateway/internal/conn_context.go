package internal

import (
	"context"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// ConnContext 连接上下文接口，封装 WebSocket 连接的元数据和上下文信息
type ConnContext interface {
	context.Context
	RemoteAddr() string
	Close() error
	HandshakeInfo() HandshakeInfo
}

// HandshakeInfo 握手信息接口，包含客户端连接时的元数据
type HandshakeInfo interface {
	GetPlatformID() int32
	GetDeviceID() string
	GetSendResponse() bool
	GetIsBackground() bool
	GetOperationID() string
	GetCompression() string
	GetSDKType() string
	GetSDKVersion() string
	GetToken() string
	GetUserID() string
}

type handshakeInfo struct {
	UserID       string `form:"userID"`
	Token        string `form:"token"`
	PlatformID   int32  `form:"platformID"`
	DeviceID     string `form:"deviceID"`
	SendResponse bool   `form:"sendResponse"`
	IsBackground bool   `form:"isBackground"` // 是否为后台连接
	OperationID  string `form:"operationID"`
	Compression  string `form:"compression"`
	SDKType      string `form:"sdkType"`
	SDKVersion   string `form:"sdkVersion"`
}

func (c *handshakeInfo) GetPlatformID() int32 {
	return c.PlatformID
}

func (c *handshakeInfo) GetDeviceID() string {
	return c.DeviceID
}

func (c *handshakeInfo) GetSendResponse() bool {
	return c.SendResponse
}

func (c *handshakeInfo) GetIsBackground() bool {
	return c.IsBackground
}

func (c *handshakeInfo) GetOperationID() string {
	return c.OperationID
}

func (c *handshakeInfo) GetCompression() string {
	return c.Compression
}

func (c *handshakeInfo) GetSDKType() string {
	return c.SDKType
}

func (c *handshakeInfo) GetSDKVersion() string {
	return c.SDKVersion
}

func (c *handshakeInfo) GetToken() string {
	return c.Token
}

func (c *handshakeInfo) GetUserID() string {
	return c.UserID
}

type connContext struct {
	w          http.ResponseWriter
	r          *http.Request
	remoteAddr string
	info       *handshakeInfo
	ctx        context.Context
	cancel     context.CancelFunc
}

func newConnContext(w http.ResponseWriter, r *http.Request) *connContext {
	remoteAddr := httpx.GetRemoteAddr(r)
	ctx, cancel := context.WithCancel(context.Background())
	return &connContext{
		w:          w,
		r:          r,
		remoteAddr: remoteAddr,
		info:       &handshakeInfo{},
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (c *connContext) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *connContext) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *connContext) Err() error {
	return c.ctx.Err()
}

func (c *connContext) Value(key any) any {
	return c.ctx.Value(key)
}

// Close 关闭 context，用于 WebSocket 连接关闭时调用
func (c *connContext) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

func (c *connContext) RemoteAddr() string {
	return c.remoteAddr
}

// ParseHandshakeRequest 解析握手请求参数，返回解析错误
func (c *connContext) ParseHandshakeRequest() error {
	return httpx.Parse(c.r, c.info)
}

func (c *connContext) HandshakeInfo() HandshakeInfo {
	return c.info
}
