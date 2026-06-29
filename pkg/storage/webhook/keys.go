package webhook

import "time"

const (
	WebhookDeliveryExpire = 24 * time.Hour // 24小时过期
)

const (
	WebhookDelivery = "webhook:delivery:" //存储投递记录
)

func getWebhookDeliveryKey(recordID string) string {
	return WebhookDelivery + recordID
}
