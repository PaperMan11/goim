package fcm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/PaperMan11/goim/im-push/internal/offlnepush/options"
	"github.com/PaperMan11/goim/pkg/utils/httpclient"
	"github.com/zeromicro/go-zero/core/logx"
)

type FCMConfig struct {
	FilePath string
	AuthURL  string
}

// FCMPusher Firebase Cloud Messaging 实现
// 官方文档：https://firebase.google.com/docs/cloud-messaging/send-message
// 注意：FCM v1 API 需要 OAuth2 access token，token 获取需要用到 service account 的 private key
// 当前实现仅提供基础骨架，access token 建议使用 google.golang.org/api/option 配合 FilePath 字段管理
type FCMPusher struct {
	cfg *FCMConfig
}

func NewFCMPusher(cfg *FCMConfig) *FCMPusher {
	return &FCMPusher{
		cfg: cfg,
	}
}

// Push 批量推送
// FCM 的 token 即 registration token；v1 API 支持 "topic"、"condition"、"token" 三种 target
// 这里对每个 token 单独发一条（FCM 无原生批量 API，需配合 topic 或使用 deprecated batch API）
// 生产建议：token 数量多时按 500 个一组并发推送
func (p *FCMPusher) Push(ctx context.Context, tokens []string, title string, content string, opts ...options.Option) error {
	if len(tokens) == 0 {
		return nil
	}

	opt := &options.Options{}
	for _, fn := range opts {
		fn(opt)
	}

	// 获取 OAuth2 access token（实际应通过 credential.FilePath 从 service account 自动获取）
	accessToken, err := p.getAccessToken(ctx)
	if err != nil {
		logx.Errorf("fcm get access token failed: %v", err)
		// token 获取失败时跳过本次推送，避免直接 panic
		return fmt.Errorf("fcm get access token: %w", err)
	}

	// 串行发送（后续可改为 goroutine 并发 + MaxConcurrentWorkers 限制）
	for _, token := range tokens {
		if err := p.pushOne(ctx, accessToken, token, title, content, opt); err != nil {
			logx.Errorf("fcm push one failed, token: %s, err: %v", token, err)
			// 单条失败不影响其他 token
			continue
		}
	}

	logx.Infof("fcm push done, total tokens: %d", len(tokens))
	return nil
}

// pushOne 单条 FCM v1 API 推送
func (p *FCMPusher) pushOne(ctx context.Context, accessToken, token, title, content string, opt *options.Options) error {
	payload := map[string]any{
		"message": map[string]any{
			"token": token,
			"notification": map[string]any{
				"title": title,
				"body":  content,
			},
			"data": map[string]any{},
		},
	}

	if opt.Signal != nil && opt.Signal.ClientMsgID != "" {
		payload["message"].(map[string]any)["data"].(map[string]any)["clientMsgID"] = opt.Signal.ClientMsgID
	}
	if opt.Ex != "" {
		payload["message"].(map[string]any)["data"].(map[string]any)["content"] = opt.Ex
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = httpclient.PostJSON(p.cfg.AuthURL, bodyBytes,
		httpclient.WithHeader("Authorization", "Bearer "+accessToken),
		httpclient.WithTTL(10*time.Second),
	)
	if err != nil {
		if replyErr, ok := httpclient.ToReplyErr(err); ok {
			logx.Errorf("fcm push failed, status: %d, body: %s", replyErr.StatusCode(), string(replyErr.Body()))
			return fmt.Errorf("fcm push failed, status: %d", replyErr.StatusCode())
		}
		return err
	}
	return nil
}

// getAccessToken 获取 FCM OAuth2 access token
// 本实现仅返回占位符——生产环境请使用 google.golang.org/api/option 配合 service account JSON 自动管理
func (p *FCMPusher) getAccessToken(ctx context.Context) (string, error) {
	if p.cfg.FilePath == "" {
		return "", fmt.Errorf("fcm FilePath (service account json) not configured")
	}
	// TODO: 使用 google.golang.org/api/auth 包从 FilePath 加载 service account 并获取 access token
	// 占位返回——首次使用前必须补充
	return "", fmt.Errorf("fcm getAccessToken not implemented, FilePath=%s", p.cfg.FilePath)
}
