package offlnepush

import (
	"context"

	"github.com/PaperMan11/goim/im-push/internal/offlnepush/fcm"
	"github.com/PaperMan11/goim/im-push/internal/offlnepush/getui"
	"github.com/PaperMan11/goim/im-push/internal/offlnepush/jpush"
	"github.com/PaperMan11/goim/im-push/internal/offlnepush/options"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
)

const (
	EnableGeTui = "geTui"
	EnableFCM   = "fcm"
	EnableJPush = "jpush"
)

// Config 离线推送配置，内嵌到主 Config 的 OfflinePush 字段
type Config struct {
	MaxConcurrentWorkers int
	Enable               string
	PushSound            string
	BadgeCount           int
	GeTui                getui.GeTuiConfig
	FCM                  fcm.FCMConfig
	JPush                jpush.JpushConfig
}

// OfflinePusher 离线推送接口
// Push 按 tokens（已归一化的 device token 列表）向第三方推送服务发起批量推送
type OfflinePusher interface {
	Push(ctx context.Context, tokens []string, title string, content string, opts ...options.Option) error
}

// NewOfflinePusher 根据 Config.Enable 创建对应平台的离线推送器
// 返回 nil 表示未启用离线推送，调用方应做 nil 检查
func NewOfflinePusher(cfg *Config) OfflinePusher {
	if cfg == nil || cfg.Enable == "" {
		return nil
	}
	switch cfg.Enable {
	case EnableGeTui:
		return getui.NewGeTuiPusher(&cfg.GeTui)
	case EnableFCM:
		return fcm.NewFCMPusher(&cfg.FCM)
	case EnableJPush:
		return jpush.NewJPushPusher(&cfg.JPush)
	default:
		return nil
	}
}

// tokenPlatformID 约定：客户端通过 SetUserClientConfig 注册 token 时，
// key 格式为 push_token_<platformID>。这里统一约定：
//
//	platformID = constant.AndroidPlatformID (2) → 走 JPush/GeTui/FCM 的 Android 通道
//	platformID = constant.IOSPlatformID (1)   → 走 JPush/GeTui/FCM 的 iOS 通道
//
// 每个平台 Pusher 自身会根据 OfflinePush.Config.IOSPush.Production 决定用生产还是沙盒环境
func tokenKey(platformID int32) string {
	return "push_token_" + constant.PlatformIDToName(int(platformID))
}
