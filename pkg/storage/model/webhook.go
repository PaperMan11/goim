package model

import "time"

// WebhookDelivery Webhook投递记录，存储webhook事件的投递状态和重试信息
type WebhookDelivery struct {
	ID           string        `bson:"_id"`           // 投递记录ID
	EventID      string        `bson:"event_id"`      // 事件ID
	EventType    string        `bson:"event_type"`    // 事件类型
	EventPayload string        `bson:"event_payload"` // 事件数据（JSON序列化）
	WebhookURL   string        `bson:"webhook_url"`   // webhook URL
	Status       string        `bson:"status"`        // 投递状态
	StatusCode   int           `bson:"status_code"`   // HTTP状态码
	AttemptCount int           `bson:"attempt_count"` // 尝试次数
	LastAttempt  time.Time     `bson:"last_attempt"`  // 最后尝试时间
	NextAttempt  time.Time     `bson:"next_attempt"`  // 下次尝试时间
	ErrorMessage string        `bson:"error_message"` // 错误信息
	Response     string        `bson:"response"`      // 响应内容
	Duration     time.Duration `bson:"duration"`      // 请求耗时
	CreatedAt    time.Time     `bson:"created_at"`    // 创建时间
	UpdatedAt    time.Time     `bson:"updated_at"`    // 更新时间
}

func (w *WebhookDelivery) CollectionName() string {
	return CollectionWebhookDelivery
}
