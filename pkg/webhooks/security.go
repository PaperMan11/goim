package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/PaperMan11/goim/pkg/utils/convert"
	"github.com/PaperMan11/goim/pkg/utils/timex"
)

const (
	// SignatureHeader 签名请求头名称
	SignatureHeader = "X-Webhook-Signature"
	// TimestampHeader 时间戳请求头名称
	TimestampHeader = "X-Webhook-Timestamp"
	// EventIDHeader 事件ID请求头名称
	EventIDHeader = "X-Webhook-Event-ID"
	// EventTypeHeader 事件类型请求头名称
	EventTypeHeader = "X-Webhook-Event-Type"
	// RetryCountHeader 重试次数请求头名称
	RetryCountHeader = "X-Webhook-Retry-Count"
)

// Signer 签名器
type Signer struct {
	secret string
}

// NewSigner 创建签名器
func NewSigner(secret string) *Signer {
	return &Signer{
		secret: secret,
	}
}

// Sign 生成签名
// 格式: sha256=hex(hmac_sha256(secret, timestamp + payload))
func (s *Signer) Sign(timestamp int64, payload []byte) string {
	if s.secret == "" {
		return ""
	}

	// 计算 HMAC-SHA256
	h := hmac.New(sha256.New, []byte(s.secret))
	h.Write([]byte(strconv.FormatInt(timestamp, 10)))
	h.Write(payload)
	signature := hex.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("sha256=%s", signature)
}

// Verify 验证签名
func (s *Signer) Verify(signature string, timestamp int64, payload []byte) bool {
	if s.secret == "" {
		// 如果没有密钥，则不验证签名
		return true
	}

	expectedSignature := s.Sign(timestamp, payload)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// VerifyWithTimestamp 验证签名并检查时间戳
func (s *Signer) VerifyWithTimestamp(signature string, timestamp int64, payload []byte, maxAge time.Duration) bool {
	// 检查时间戳是否过期
	if maxAge > 0 {
		now := timex.UnixMilli()
		if now-timestamp > maxAge.Milliseconds() {
			return false
		}
	}

	return s.Verify(signature, timestamp, payload)
}

// GenerateHeaders 生成 webhook 请求头
func (s *Signer) GenerateHeaders(event *WebhookEvent) map[string]string {
	headers := make(map[string]string)

	// 添加事件ID
	headers[EventIDHeader] = event.EventID

	// 添加事件类型
	headers[EventTypeHeader] = string(event.EventType)

	// 添加时间戳
	timestamp := timex.UnixMilli()
	headers[TimestampHeader] = convert.ToString(timestamp)

	// 添加重试次数
	headers[RetryCountHeader] = convert.ToString(event.RetryCount)

	// 生成签名
	if s.secret != "" {
		payload, err := json.Marshal(event)
		if err == nil {
			signature := s.Sign(timestamp, payload)
			headers[SignatureHeader] = signature
		}
	}

	return headers
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableSignature bool          // 是否启用签名验证
	EnableTimestamp bool          // 是否启用时间戳验证
	MaxTimestampAge time.Duration // 时间戳最大有效期
	AllowedIPs      []string      // 允许的IP地址列表
	AllowedOrigins  []string      // 允许的来源列表
}

// DefaultSecurityConfig 默认安全配置
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EnableSignature: true,
		EnableTimestamp: true,
		MaxTimestampAge: 5 * time.Minute,
		AllowedIPs:      nil,
		AllowedOrigins:  nil,
	}
}

// SecurityValidator 安全验证器
type SecurityValidator struct {
	config *SecurityConfig
	signer *Signer
}

// NewSecurityValidator 创建安全验证器
func NewSecurityValidator(config *SecurityConfig, secret string) *SecurityValidator {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &SecurityValidator{
		config: config,
		signer: NewSigner(secret),
	}
}

// Validate 验证 webhook 请求
func (v *SecurityValidator) Validate(headers map[string]string, payload []byte) error {
	// 验证签名
	if v.config.EnableSignature {
		signature := headers[SignatureHeader]
		timestampStr := headers[TimestampHeader]

		if signature == "" {
			return fmt.Errorf("missing signature header")
		}

		var timestamp int64
		if timestampStr != "" {
			timestamp = convert.ToInt64(timestampStr)
			if timestamp == 0 {
				return fmt.Errorf("invalid timestamp format")
			}
		}

		// 验证签名
		if v.config.EnableTimestamp && v.config.MaxTimestampAge > 0 {
			if !v.signer.VerifyWithTimestamp(signature, timestamp, payload, v.config.MaxTimestampAge) {
				return fmt.Errorf("signature verification failed or timestamp expired")
			}
		} else {
			if !v.signer.Verify(signature, timestamp, payload) {
				return fmt.Errorf("signature verification failed")
			}
		}
	}

	return nil
}

// ValidateIP 验证IP地址
func (v *SecurityValidator) ValidateIP(ip string) error {
	if !v.config.EnableTimestamp {
		return nil
	}

	if len(v.config.AllowedIPs) == 0 {
		return nil
	}

	for _, allowedIP := range v.config.AllowedIPs {
		if ip == allowedIP {
			return nil
		}
	}

	return fmt.Errorf("IP address not allowed: %s", ip)
}

// ValidateOrigin 验证来源
func (v *SecurityValidator) ValidateOrigin(origin string) error {
	if len(v.config.AllowedOrigins) == 0 {
		return nil
	}

	for _, allowedOrigin := range v.config.AllowedOrigins {
		if origin == allowedOrigin {
			return nil
		}
	}

	return fmt.Errorf("origin not allowed: %s", origin)
}
