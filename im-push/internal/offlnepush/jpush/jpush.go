package jpush

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/PaperMan11/goim/im-push/internal/offlnepush/options"
	"github.com/PaperMan11/goim/pkg/utils/httpclient"
	"github.com/zeromicro/go-zero/core/logx"
)

type JpushConfig struct {
	AppKey       string
	MasterSecret string
	PushURL      string
	PushIntent   string
}

// JPushPusher 极光推送实现
// 官方文档：https://docs.jiguang.cn/jpush/server/push/rest_api_v3_push
type JPushPusher struct {
	cfg *JpushConfig
}

func NewJPushPusher(cfg *JpushConfig) *JPushPusher {
	return &JPushPusher{cfg: cfg}
}

// Push 批量推送
// JPush 支持 registration_id / alias / tag 三种目标方式，这里采用 registration_id（即 device token）
func (p *JPushPusher) Push(ctx context.Context, tokens []string, title string, content string, opts ...options.Option) error {
	if len(tokens) == 0 {
		return nil
	}

	opt := &options.Options{}
	for _, fn := range opts {
		fn(opt)
	}

	// 构造 basicAuth 认证头（AppKey:MasterSecret 的 base64）
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(p.cfg.AppKey+":"+p.cfg.MasterSecret))

	// JPush v3 REST API 的 audience 是 registration_ids 数组
	// iOS 的 token 走 apns_production / apns_sandbox；Android 的走 android
	payload := p.buildPayload(tokens, title, content, opt)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal jpush payload: %w", err)
	}

	respBody, err := httpclient.PostJSON(p.cfg.PushURL, bodyBytes,
		httpclient.WithHeader("Authorization", authHeader),
		httpclient.WithTTL(10*time.Second),
	)
	if err != nil {
		if replyErr, ok := httpclient.ToReplyErr(err); ok {
			logx.Errorf("jpush push failed, status: %d, body: %s, tokens: %d", replyErr.StatusCode(), string(replyErr.Body()), len(tokens))
			return fmt.Errorf("jpush push failed, status: %d, body: %s", replyErr.StatusCode(), string(replyErr.Body()))
		}
		return fmt.Errorf("jpush http do: %w", err)
	}

	logx.Infof("jpush push success, tokens: %d, resp: %s", len(tokens), string(respBody))
	return nil
}

// buildPayload 构造 JPush v3 推送 payload
// 参考：https://docs.jiguang.cn/jpush/server/push/rest_api_v3_push#推送-推送全平台
func (p *JPushPusher) buildPayload(tokens []string, title, content string, opt *options.Options) map[string]any {
	// 先简单地把所有 token 都投给 registration_id（JPush 会自动根据 token 对应的平台分发）
	payload := map[string]any{
		"platform":     "all",
		"audience":     map[string]any{"registration_id": tokens},
		"notification": map[string]any{"alert": title},
		"message": map[string]any{
			"msg_content": content,
			"extras":      map[string]any{},
		},
	}

	if opt.IOSPushSound != "" {
		if notif, ok := payload["notification"].(map[string]any); ok {
			notif["ios"] = map[string]any{
				"alert": title,
				"sound": opt.IOSPushSound,
				"badge": "+1",
			}
		}
	}

	if opt.Signal != nil && opt.Signal.ClientMsgID != "" {
		payload["message"].(map[string]any)["title"] = title
		payload["message"].(map[string]any)["extras"].(map[string]any)["clientMsgID"] = opt.Signal.ClientMsgID
	}

	if opt.Ex != "" {
		payload["message"].(map[string]any)["extras"].(map[string]any)["content"] = opt.Ex
	}

	return payload
}
