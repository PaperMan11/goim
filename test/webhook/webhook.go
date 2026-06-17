package main

import (
	"fmt"
	"time"

	"github.com/PaperMan11/goim/pkg/webhooks"
)

// // MemoryDeliveryRepository 内存投递记录仓库
type MemoryDeliveryRepository struct {
	records map[string]*webhooks.DeliveryRecord
}

func NewMemoryDeliveryRepository() *MemoryDeliveryRepository {
	return &MemoryDeliveryRepository{
		records: make(map[string]*webhooks.DeliveryRecord),
	}
}

func (r *MemoryDeliveryRepository) Save(record *webhooks.DeliveryRecord) error {
	r.records[record.ID] = record
	return nil
}

func (r *MemoryDeliveryRepository) Update(record *webhooks.DeliveryRecord) error {
	r.records[record.ID] = record
	return nil
}

func (r *MemoryDeliveryRepository) Get(id string) (*webhooks.DeliveryRecord, error) {
	record, exists := r.records[id]
	if !exists {
		return nil, fmt.Errorf("record not found")
	}
	return record, nil
}

func (r *MemoryDeliveryRepository) GetByEventID(eventID string) ([]*webhooks.DeliveryRecord, error) {
	var records []*webhooks.DeliveryRecord
	for _, record := range r.records {
		if record.EventID == eventID {
			records = append(records, record)
		}
	}
	return records, nil
}

func (r *MemoryDeliveryRepository) GetPending(limit int) ([]*webhooks.DeliveryRecord, error) {
	var records []*webhooks.DeliveryRecord
	for _, record := range r.records {
		if record.Status == webhooks.DeliveryStatusRetrying {
			records = append(records, record)
			if len(records) >= limit {
				break
			}
		}
	}
	return records, nil
}

func (r *MemoryDeliveryRepository) Delete(id string) error {
	delete(r.records, id)
	return nil
}

func sendUserEvent() {
	deliveryRepo := NewMemoryDeliveryRepository()
	manager := webhooks.NewManager(deliveryRepo, 5)
	manager.Start()
	defer manager.Stop()
	manager.AddWebhook(&webhooks.WebhookConfig{
		URL:           "http://localhost:8080/api/webhooks",
		Secret:        webhookSecret,
		Timeout:       5 * time.Second,
		MaxRetries:    0,
		RetryInterval: 0,
		Enabled:       true,
		Events: []webhooks.EventType{
			webhooks.EventUserOnline,
			webhooks.EventUserOffline,
		},
	})

	onlineEvent := webhooks.NewUserOnlineEvent(&webhooks.UserEventData{
		UserID:       "user_001",
		Nickname:     "Alice",
		PlatformID:   1,
		OnlineStatus: 1,
		DeviceID:     "device_001",
		Extra: map[string]string{
			"ip": "192.168.1.1",
		},
	})
	offlineEvent := webhooks.NewUserOfflineEvent(&webhooks.UserEventData{
		UserID:       "user_002",
		Nickname:     "Bob",
		PlatformID:   1,
		OnlineStatus: 0,
		DeviceID:     "device_002",
		Extra: map[string]string{
			"ip": "192.168.1.2",
		},
	})
	manager.Dispatch(onlineEvent)
	manager.Dispatch(offlineEvent)
}
