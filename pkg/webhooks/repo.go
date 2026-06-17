package webhooks

import (
	"fmt"
)

// DeliveryRepository 投递记录仓库接口
type DeliveryRepository interface {
	Save(record *DeliveryRecord) error
	Update(record *DeliveryRecord) error
	Get(id string) (*DeliveryRecord, error)
	GetByEventID(eventID string) ([]*DeliveryRecord, error)
	GetPending(limit int) ([]*DeliveryRecord, error)
	Delete(id string) error
}

// MemoryDeliveryRepository 内存投递记录仓库（用于测试）
type MemoryDeliveryRepository struct {
	records map[string]*DeliveryRecord
}

func NewMemoryDeliveryRepository() *MemoryDeliveryRepository {
	return &MemoryDeliveryRepository{
		records: make(map[string]*DeliveryRecord),
	}
}

func (r *MemoryDeliveryRepository) Save(record *DeliveryRecord) error {
	r.records[record.ID] = record
	return nil
}

func (r *MemoryDeliveryRepository) Update(record *DeliveryRecord) error {
	r.records[record.ID] = record
	return nil
}

func (r *MemoryDeliveryRepository) Get(id string) (*DeliveryRecord, error) {
	record, exists := r.records[id]
	if !exists {
		return nil, fmt.Errorf("record not found")
	}
	return record, nil
}

func (r *MemoryDeliveryRepository) GetByEventID(eventID string) ([]*DeliveryRecord, error) {
	var records []*DeliveryRecord
	for _, record := range r.records {
		if record.EventID == eventID {
			records = append(records, record)
		}
	}
	return records, nil
}

func (r *MemoryDeliveryRepository) GetPending(limit int) ([]*DeliveryRecord, error) {
	var records []*DeliveryRecord
	for _, record := range r.records {
		if record.Status == DeliveryStatusRetrying {
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
