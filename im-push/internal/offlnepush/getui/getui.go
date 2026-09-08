package getui

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

type GeTuiConfig struct {
	PushUrl      string
	MasterSecret string
	AppKey       string
	Intent       string
	ChannelID    string
	ChannelName  string
}

// GeTuiPusher 个推实现
// 官方文档：https://docs.getui.com/getui/server/rest_v2/push/
type GeTuiPusher struct {
	cfg *GeTuiConfig
}

func NewGeTuiPusher(cfg *GeTuiConfig) *GeTuiPusher {
	return &GeTuiPusher{
		cfg: cfg,
	}
}

func (p *GeTuiPusher) Push(ctx context.Context, tokens []string, title string, content string, opts ...options.Option) error {
	if len(tokens) == 0 {
		return nil
	}

	opt := &options.Options{}
	for _, fn := range opts {
		fn(opt)
	}

	// 构造 BasicAuth 认证头（AppKey:MasterSecret 的 base64）
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(p.cfg.AppKey+":"+p.cfg.MasterSecret))

	// 个推 v2 REST API：单条推送对应单个 cid，批量调用需循环（或走 list-push-api 但限制 100 以内）
	// 这里使用 tolist 方式，一次请求推送多个 cid
	payload := p.buildListPayload(tokens, title, content, opt)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal getui payload: %w", err)
	}

	_, err = httpclient.PostJSON(p.cfg.PushUrl, bodyBytes,
		httpclient.WithHeader("Authorization", authHeader),
		httpclient.WithTTL(10*time.Second),
	)
	if err != nil {
		if replyErr, ok := httpclient.ToReplyErr(err); ok {
			logx.Errorf("getui push failed, status: %d, body: %s, tokens: %d", replyErr.StatusCode(), string(replyErr.Body()), len(tokens))
			return fmt.Errorf("getui push failed, status: %d, body: %s", replyErr.StatusCode(), string(replyErr.Body()))
		}
		return fmt.Errorf("getui http do: %w", err)
	}

	logx.Infof("getui push success, tokens: %d", len(tokens))
	return nil
}

// buildListPayload 构造个推 list-push 格式 payload
func (p *GeTuiPusher) buildListPayload(tokens []string, title, content string, opt *options.Options) map[string]any {
	payload := map[string]any{
		"audience": map[string]any{
			"cid": tokens,
		},
		"settings": map[string]any{
			"ttl": 3600000, // ms
		},
		"payload": map[string]any{
			"aps": map[string]any{
				"alert": map[string]any{
					"title": title,
					"body":  content,
				},
				"sound": "default",
			},
			"android": map[string]any{
				"notification": map[string]any{
					"title":  title,
					"body":   content,
					"intent": p.cfg.Intent,
				},
			},
		},
	}

	if opt.IOSPushSound != "" {
		if aps, ok := payload["payload"].(map[string]any)["aps"].(map[string]any); ok {
			aps["sound"] = opt.IOSPushSound
		}
	}

	if opt.Signal != nil && opt.Signal.ClientMsgID != "" {
		payload["payload"].(map[string]any)["ios"] = map[string]any{
			"clientMsgID": opt.Signal.ClientMsgID,
		}
	}

	if opt.Ex != "" {
		payload["payload"].(map[string]any)["payload"] = opt.Ex
	}

	return payload
}
