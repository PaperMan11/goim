package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/PaperMan11/goim/pkg/webhooks"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type WebhookMongoStore struct {
	mongoClient *mon.Model
	redisClient *redis.Redis
}

var _ webhooks.DeliveryRepository = (*WebhookMongoStore)(nil)

func NewWebhookMongoStore(mongoClient *mon.Model, redisClient *redis.Redis) *WebhookMongoStore {
	return &WebhookMongoStore{
		mongoClient: mongoClient,
		redisClient: redisClient,
	}
}

func (w *WebhookMongoStore) Save(record *webhooks.DeliveryRecord) error {
	_, err := w.mongoClient.Collection.InsertOne(context.Background(), record)
	if err != nil {
		return err
	}

	bytes, err := bson.Marshal(record)
	if err == nil {
		w.redisClient.Setex(getWebhookDeliveryKey(record.ID), string(bytes), int(WebhookDeliveryExpire.Seconds()))
	}

	return nil
}

func (w *WebhookMongoStore) Update(record *webhooks.DeliveryRecord) error {
	err := w.mongoClient.FindOneAndReplace(context.Background(), nil, bson.M{"_id": record.ID}, record)
	if err != nil {
		return fmt.Errorf("record not found")
	}

	w.redisClient.Del(getWebhookDeliveryKey(record.ID))

	return nil
}

func (w *WebhookMongoStore) Get(id string) (*webhooks.DeliveryRecord, error) {
	key := getWebhookDeliveryKey(id)

	val, err := w.redisClient.Get(key)
	if err == nil {
		var record webhooks.DeliveryRecord
		if err := bson.Unmarshal([]byte(val), &record); err == nil {
			return &record, nil
		}
	}

	var mr webhooks.DeliveryRecord
	err = w.mongoClient.FindOne(context.Background(), &mr, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("record not found")
	}

	bytes, marshalErr := bson.Marshal(&mr)
	if marshalErr == nil {
		w.redisClient.Setex(key, string(bytes), int(WebhookDeliveryExpire.Seconds()))
	}

	return &mr, nil
}

func (w *WebhookMongoStore) GetByEventID(eventID string) ([]*webhooks.DeliveryRecord, error) {
	var mrs []*webhooks.DeliveryRecord
	err := w.mongoClient.Find(context.Background(), &mrs, bson.M{"eventId": eventID})
	if err != nil {
		return nil, err
	}

	var records []*webhooks.DeliveryRecord
	for _, mr := range mrs {
		records = append(records, mr)
	}

	return records, nil
}

func (w *WebhookMongoStore) GetPending(limit int) ([]*webhooks.DeliveryRecord, error) {
	var mrs []*webhooks.DeliveryRecord
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
		records = append(records, mr)
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

	w.redisClient.Del(getWebhookDeliveryKey(id))

	return nil
}
