package internal

import (
	"context"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"net/http"
	"time"

	"github.com/PaperMan11/goim/pkg/mcontext"
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
	// 客户端 IP 在建立连接时即可确定，尽早注入，供鉴权 RPC 与日志追踪使用。
	// 注意：不能派生自 r.Context()，ServeHTTP 返回后请求 context 会被取消，
	// 而 WebSocket 连接生命周期远长于 HTTP 握手请求，因此基于 context.Background() 自建。
	ctx = mcontext.SetClientIPInContext(ctx, remoteAddr)
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

// ParseHandshakeRequest 解析握手请求参数，并把传输层即可信的信息注入连接级 context。
// 此时用户身份尚未鉴权、连接 ID 尚未生成，二者分别由
// SetAuthenticatedUser（鉴权通过后）与 SetConnID（连接创建后）注入。
func (c *connContext) ParseHandshakeRequest() error {
	if err := httpx.Parse(c.r, c.info); err != nil {
		return err
	}
	c.injectHandshakeContext()
	return nil
}

// injectHandshakeContext 注入握手参数：operationID/token/platform。
// 这些值是下游 ParseToken RPC 的 gRPC metadata 与全链路日志追踪所依赖的字段
// （见 pkg/rpcinterceptors/clientinterceptors/context_interceptor.go）。
// 全部注入发生在连接启动前（readPump/writePump 协程创建、connManager.Add 之前）
func (c *connContext) injectHandshakeContext() {
	operationID := c.info.OperationID
	if operationID == "" {
		// 客户端未传时按内部统一格式补齐：<source>:<uuid>
		operationID = mcontext.NewOperationID("msggateway")
	}
	c.ctx = mcontext.SetOperationIDInContext(c.ctx, operationID)
	c.ctx = mcontext.SetTokenInContext(c.ctx, c.info.Token)
	c.ctx = mcontext.SetOpUserPlatformInContext(c.ctx, constant.PlatformID2Name[int(c.info.PlatformID)])
}

// SetAuthenticatedUser 鉴权通过（或未开启鉴权直接放行）后注入可信用户身份。
// 不能在 ParseHandshakeRequest 阶段注入：EnableAuth=true 时该 userID 仅为客户端自报，
// 尚未经 ParseToken 校验，提前注入会让未鉴权身份随 RPC metadata 透传到下游服务。
func (c *connContext) SetAuthenticatedUser(userID string) {
	c.ctx = mcontext.SetOpUserIDInContext(c.ctx, userID)
}

// SetConnID 在 WebSocket 连接对象创建后注入连接 ID。
// 必须在 connManager.Add 之前调用：Add 会把连接交给登录策略 worker 处理，
// 其中触发的 onConnAdd 上线 RPC 通过 conn.Context() 读取该字段。
func (c *connContext) SetConnID(connID string) {
	c.ctx = mcontext.SetConnIDInContext(c.ctx, connID)
}

func (c *connContext) HandshakeInfo() HandshakeInfo {
	return c.info
}

// WithMessageOperationID 基于连接级 context 派生单条消息的处理 context。
//
// 派生自连接 context 而非 context.Background()，因此：
//   - 继承连接的取消信号：连接断开/Close 时，该消息在途的下游 RPC 一并取消；
//   - 继承连接级全部可信值（opUserID/token/platform/connID/clientIP 等），
//     这些会经 gRPC client interceptor 透传到下游 metadata；
//   - 仅覆盖 operationID 为本条消息携带的值，实现消息级全链路追踪，
//     且不回写连接级 context，每条消息各自派生、无并发竞争。
//
// operationID 为空时直接返回连接 context 本身，沿用握手级 operationID。
// 注意：不要在此覆盖 opUserID —— 身份以鉴权后的连接级值为准，
// EnableAuth 时 Handle 入口已保证 req.SenderID 与鉴权用户一致。
func WithMessageOperationID(connCtx ConnContext, operationID string) context.Context {
	if operationID == "" {
		return connCtx
	}
	return mcontext.SetOperationIDInContext(connCtx, operationID)
}
