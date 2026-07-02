package webhook

import (
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/webhooks"
)

const (
	WebhookDeliveryExpire = 24 * time.Hour // 24小时过期
)

const (
	WebhookDelivery = "webhook:delivery:" //存储投递记录
)

func getWebhookDeliveryKey(recordID string) string {
	return WebhookDelivery + recordID
}

func toModelDelivery(record *webhooks.DeliveryRecord) *model.WebhookDelivery {
	return &model.WebhookDelivery{
		ID:           record.ID,
		EventID:      record.EventID,
		EventType:    string(record.EventType),
		EventPayload: record.EventPayload,
		WebhookURL:   record.WebhookURL,
		Status:       string(record.Status),
		StatusCode:   record.StatusCode,
		AttemptCount: record.AttemptCount,
		LastAttempt:  record.LastAttempt,
		NextAttempt:  record.NextAttempt,
		ErrorMessage: record.ErrorMessage,
		Response:     record.Response,
		Duration:     record.Duration,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

func toDeliveryRecord(record *model.WebhookDelivery) *webhooks.DeliveryRecord {
	return &webhooks.DeliveryRecord{
		ID:           record.ID,
		EventID:      record.EventID,
		EventType:    webhooks.EventType(record.EventType),
		EventPayload: record.EventPayload,
		WebhookURL:   record.WebhookURL,
		Status:       webhooks.DeliveryStatus(record.Status),
		StatusCode:   record.StatusCode,
		AttemptCount: record.AttemptCount,
		LastAttempt:  record.LastAttempt,
		NextAttempt:  record.NextAttempt,
		ErrorMessage: record.ErrorMessage,
		Response:     record.Response,
		Duration:     record.Duration,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}
