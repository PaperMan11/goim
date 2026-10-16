package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/stretchr/testify/assert"
)

func newTestConnContext(t *testing.T, target string, remoteAddr string) *connContext {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.RemoteAddr = remoteAddr
	return newConnContext(httptest.NewRecorder(), r)
}

// go-zero 的 httpx.Parse 要求非指针字段全部出现，模拟真实客户端的完整握手参数。
// overrides 用于覆盖/置空单个参数（值为空串表示显式传空）。
func handshakeQuery(overrides map[string]string) string {
	q := url.Values{}
	q.Set("userID", "u1")
	q.Set("token", "tok-1")
	q.Set("platformID", "5")
	q.Set("deviceID", "dev-1")
	q.Set("sendResponse", "true")
	q.Set("isBackground", "false")
	q.Set("operationID", "op-client-1")
	q.Set("compression", "none")
	q.Set("sdkType", "normal")
	q.Set("sdkVersion", "1.0.0")
	for k, v := range overrides {
		q.Set(k, v)
	}
	return "/ws?" + q.Encode()
}

// 握手参数解析后，传输层可信信息应注入 context
func TestConnContext_ParseHandshakeRequest_InjectsTransportInfo(t *testing.T) {
	cc := newTestConnContext(t, handshakeQuery(nil), "10.0.0.1:54321")

	err := cc.ParseHandshakeRequest()
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.1:54321", mcontext.GetClientIPFromContext(cc))
	assert.Equal(t, "op-client-1", mcontext.GetOperationIDFromContext(cc))
	assert.Equal(t, "tok-1", mcontext.GetTokenFromContext(cc))
	assert.Equal(t, "5", mcontext.GetOpUserPlatformFromContext(cc))
}

// operationID 为空时（如非 HTTP 入口构造的连接）按约定补齐为 msggateway:<uuid>。
// 注：HTTP 握手入口经 go-zero httpx.Parse，空 operationID 会被判定为参数缺失，
// 该兜底主要防御直接调用注入逻辑的场景。
func TestConnContext_ParseHandshakeRequest_GeneratesOperationID(t *testing.T) {
	cc := newTestConnContext(t, handshakeQuery(nil), "10.0.0.1:54321")
	cc.info.OperationID = ""
	cc.injectHandshakeContext()

	opID := mcontext.GetOperationIDFromContext(cc)
	assert.True(t, strings.HasPrefix(opID, "msggateway:"), "expect msggateway: prefix, got %q", opID)
}

// 必填参数缺失时返回错误
func TestConnContext_ParseHandshakeRequest_MissingRequired(t *testing.T) {
	cc := newTestConnContext(t, "/ws?userID=u1&platformID=1", "10.0.0.1:54321")
	err := cc.ParseHandshakeRequest()
	assert.Error(t, err)
}

// 鉴权通过后注入可信用户身份；连接创建后注入 connID
func TestConnContext_AuthUserAndConnID(t *testing.T) {
	cc := newTestConnContext(t, handshakeQuery(nil), "10.0.0.1:54321")
	assert.NoError(t, cc.ParseHandshakeRequest())

	// 鉴权前不应有 opUserID，避免未校验身份透传下游
	assert.Empty(t, mcontext.GetOpUserIDFromContext(cc))
	assert.Empty(t, mcontext.GetConnIDFromContext(cc))

	cc.SetAuthenticatedUser("u1")
	cc.SetConnID("10.0.0.1:54321-1750000000000")
	assert.Equal(t, "u1", mcontext.GetOpUserIDFromContext(cc))
	assert.Equal(t, "10.0.0.1:54321-1750000000000", mcontext.GetConnIDFromContext(cc))
}

// Close 取消 context，连接级 RPC/协程可感知退出
func TestConnContext_CloseCancelsContext(t *testing.T) {
	cc := newTestConnContext(t, handshakeQuery(nil), "10.0.0.1:54321")
	assert.NoError(t, cc.ParseHandshakeRequest())
	assert.NoError(t, cc.Close())
	assert.Same(t, context.Canceled, cc.Err())
}

// 消息级派生：operationID 覆盖为消息值，连接级可信值与取消信号继承，连接级 ctx 不被回写
func TestConnContext_WithMessageOperationID(t *testing.T) {
	cc := newTestConnContext(t, handshakeQuery(nil), "10.0.0.1:54321")
	assert.NoError(t, cc.ParseHandshakeRequest())
	cc.SetAuthenticatedUser("u1")
	cc.SetConnID("conn-1")

	msgCtx := WithMessageOperationID(cc, "op-msg-1")

	// operationID 被覆盖
	assert.Equal(t, "op-msg-1", mcontext.GetOperationIDFromContext(msgCtx))
	// 连接级可信值全部继承
	assert.Equal(t, "u1", mcontext.GetOpUserIDFromContext(msgCtx))
	assert.Equal(t, "conn-1", mcontext.GetConnIDFromContext(msgCtx))
	assert.Equal(t, "tok-1", mcontext.GetTokenFromContext(msgCtx))
	assert.Equal(t, "5", mcontext.GetOpUserPlatformFromContext(msgCtx))
	assert.Equal(t, "10.0.0.1:54321", mcontext.GetClientIPFromContext(msgCtx))
	// 连接级 context 保持握手级 operationID，不被消息污染
	assert.Equal(t, "op-client-1", mcontext.GetOperationIDFromContext(cc))

	// 连接断开级联取消消息级 ctx（在途 RPC 可感知）
	assert.NoError(t, cc.Close())
	assert.Same(t, context.Canceled, msgCtx.Err())
}

// 消息未携带 operationID 时直接沿用连接级 context
func TestConnContext_WithMessageOperationID_Empty(t *testing.T) {
	cc := newTestConnContext(t, handshakeQuery(nil), "10.0.0.1:54321")
	assert.NoError(t, cc.ParseHandshakeRequest())

	msgCtx := WithMessageOperationID(cc, "")
	assert.Same(t, context.Context(cc), msgCtx)
	assert.Equal(t, "op-client-1", mcontext.GetOperationIDFromContext(msgCtx))
}
