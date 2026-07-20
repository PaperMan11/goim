package webhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/webhooks"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type WebhookMongoStore struct {
	mongoClient *mon.Model
	redisClient goredis.UniversalClient
}

var _ webhooks.DeliveryRepository = (*WebhookMongoStore)(nil)

func NewWebhookMongoStore(mongoClient *mon.Model, redisClient goredis.UniversalClient) *WebhookMongoStore {
	return &WebhookMongoStore{
		mongoClient: mongoClient,
		redisClient: redisClient,
	}
}

func (w *WebhookMongoStore) Save(record *webhooks.DeliveryRecord) error {
	modelRecord := toModelDelivery(record)
	_, err := w.mongoClient.Collection.InsertOne(context.Background(), modelRecord)
	if err != nil {
		return err
	}

	if w.redisClient != nil {
		bytes, err := bson.Marshal(record)
		if err == nil {
			_ = w.redisClient.Set(context.Background(), getWebhookDeliveryKey(record.ID), string(bytes), WebhookDeliveryExpire).Err()
		}
	}

	return nil
}

func (w *WebhookMongoStore) Update(record *webhooks.DeliveryRecord) error {
	modelRecord := toModelDelivery(record)
	err := w.mongoClient.FindOneAndReplace(context.Background(), nil, bson.M{"_id": record.ID}, modelRecord)
	if err != nil {
		return fmt.Errorf("record not found")
	}

	if w.redisClient != nil {
		_ = w.redisClient.Del(context.Background(), getWebhookDeliveryKey(record.ID)).Err()
	}

	return nil
}

func (w *WebhookMongoStore) Get(id string) (*webhooks.DeliveryRecord, error) {
	key := getWebhookDeliveryKey(id)

	if w.redisClient != nil {
		val, err := w.redisClient.Get(context.Background(), key).Result()
		if err == nil {
			var record webhooks.DeliveryRecord
			if err := bson.Unmarshal([]byte(val), &record); err == nil {
				return &record, nil
			}
		} else if !errors.Is(err, goredis.Nil) {
			_ = w.redisClient.Del(context.Background(), key).Err()
		}
	}

	var mr model.WebhookDelivery
	err := w.mongoClient.FindOne(context.Background(), &mr, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("record not found")
	}

	if w.redisClient != nil {
		bytes, marshalErr := bson.Marshal(&mr)
		if marshalErr == nil {
			_ = w.redisClient.Set(context.Background(), key, string(bytes), WebhookDeliveryExpire).Err()
		}
	}

	return toDeliveryRecord(&mr), nil
}

func (w *WebhookMongoStore) GetByEventID(eventID string) ([]*webhooks.DeliveryRecord, error) {
	var mrs []*model.WebhookDelivery
	err := w.mongoClient.Find(context.Background(), &mrs, bson.M{"event_id": eventID})
	if err != nil {
		return nil, err
	}

	var records []*webhooks.DeliveryRecord
	for _, mr := range mrs {
		records = append(records, toDeliveryRecord(mr))
	}

	return records, nil
}

func (w *WebhookMongoStore) GetPending(limit int) ([]*webhooks.DeliveryRecord, error) {
	var mrs []*model.WebhookDelivery
	now := time.Now()
	err := w.mongoClient.Find(context.Background(), &mrs, bson.M{
		"status":      webhooks.DeliveryStatusRetrying,
		"nextAttempt": bson.M{"$lte": now},
	})
	if err != nil {
		return nil, err
	}

	var records []*webhooks.DeliveryRecord
	for _, mr := range mrs {
		records = append(records, toDeliveryRecord(mr))
		if len(records) >= limit {
			break
		}
	}

	return records, nil
}

func (w *WebhookMongoStore) Delete(id string) error {
	_, err := w.mongoClient.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return err
	}

	if w.redisClient != nil {
		_ = w.redisClient.Del(context.Background(), getWebhookDeliveryKey(id)).Err()
	}

	return nil
}
