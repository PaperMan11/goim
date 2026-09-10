package middlewares

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

var errTest = errors.New("test error")

// 正常请求：返回 200，body 为 pong，logger 走 Infof 分支不阻塞响应。
func TestLogger_NormalRequest(t *testing.T) {
	r := gin.New()
	r.Use(Logger())
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pong", w.Body.String())
}

// 路径与方法被正确记录（通过响应行为间接验证中间件透传 c.Next）。
func TestLogger_PassesThroughMethodAndPath(t *testing.T) {
	r := gin.New()
	r.Use(Logger())
	r.POST("/api/v1/echo", func(c *gin.Context) {
		assert.Equal(t, http.MethodPost, c.Request.Method)
		assert.Equal(t, "/api/v1/echo", c.Request.URL.Path)
		c.String(http.StatusCreated, "created")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/echo", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// handler 往 c.Errors 写入错误：logger 走 Errorf 分支，但响应仍由 handler 控制。
func TestLogger_WithErrors(t *testing.T) {
	r := gin.New()
	r.Use(Logger())
	r.GET("/err", func(c *gin.Context) {
		_ = c.Error(errTest)
		c.String(http.StatusBadRequest, "bad")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "bad", w.Body.String())
}

// 顺序注册 Logger、Recovery，handler panic 后仍返回 500，且不向上抛出。
func TestLoggerAndRecovery_PanicReturns500(t *testing.T) {
	r := gin.New()
	r.Use(Logger(), Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	assert.NotPanics(t, func() {
		r.ServeHTTP(w, req)
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
