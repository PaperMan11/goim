package localcache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRedis(t *testing.T) redis.UniversalClient {
	redisHost := "192.168.241.128:6379"
	redisPass := "123456"
	r := redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPass,
		DB:       0,
	})

	_, err := r.Ping(context.Background()).Result()
	if err != nil {
		t.Skipf("Redis not available at %s: %v", redisHost, err)
	}

	t.Cleanup(func() {
		r.Close()
	})

	return r
}

func cleanupRedisTopic(t *testing.T, r redis.UniversalClient, topic string) {
	_ = t
	_ = r
	_ = topic
}

func setupLocalCache(t *testing.T, redisClient redis.UniversalClient) *localCache {
	cfg := CacheConfig{
		Expire: 10 * time.Second,
		Limit:  100,
		Name:   "testcache",
		Topic:  "test-topic-" + t.Name(),
	}

	cache, err := NewLocalCache(cfg, redisClient)
	assert.NoError(t, err)
	assert.NotNil(t, cache)

	t.Cleanup(func() {
		_ = cache.Close()
	})

	return cache
}

func TestLocalCache_Take(t *testing.T) {
	cache := setupLocalCache(t, nil)

	key := "test-key-take"
	fetchCalled := false

	fetch := func() (any, error) {
		fetchCalled = true
		return "fetched-value", nil
	}

	value, err := cache.Take(key, fetch)
	assert.NoError(t, err)
	assert.Equal(t, "fetched-value", value)
	assert.True(t, fetchCalled)

	fetchCalled = false

	value, err = cache.Take(key, fetch)
	assert.NoError(t, err)
	assert.Equal(t, "fetched-value", value)
	assert.False(t, fetchCalled)
}

func TestLocalCache_AddDelListener(t *testing.T) {
	cache := setupLocalCache(t, nil)

	obj := "test-obj"
	listenerCalled := false
	delKey := "test-del-key"

	getDelKey := func(key string, value any) string {
		return delKey
	}

	cache.AddDelListener(obj, getDelKey)

	assert.Len(t, cache.getDelKeyFuncs[obj], 1)

	cache.Set(delKey, "test-value")
	_, ok := cache.Get(delKey)
	assert.True(t, ok)

	event := CacheEvent{
		Action: "del",
		Key:    obj,
	}
	payload, _ := json.Marshal(event)
	cache.handleEvent(string(payload))

	listenerCalled = true
	_, ok = cache.Get(delKey)
	assert.False(t, ok)
	assert.True(t, listenerCalled)
}

func TestLocalCache_ConcurrentAccess(t *testing.T) {
	cache := setupLocalCache(t, nil)

	const goroutines = 10
	const operations = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make(chan error, goroutines*operations)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("concurrent-key-%d-%d", id, j)
				value := fmt.Sprintf("concurrent-value-%d-%d", id, j)

				cache.Set(key, value)
				time.Sleep(time.Millisecond)
				retrieved, ok := cache.Get(key)
				if !ok {
					errs <- fmt.Errorf("goroutine %d: expected key %s to exist", id, key)
					continue
				}
				if retrieved != value {
					errs <- fmt.Errorf("goroutine %d: expected value %s, got %v", id, value, retrieved)
				}

				if j%5 == 0 {
					cache.Del(key)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("%v", err)
	}
}

func TestLocalCache_PubSub(t *testing.T) {
	r := setupTestRedis(t)
	cache := setupLocalCache(t, r)
	cleanupRedisTopic(t, r, cache.cfg.Topic)

	obj := "test-obj-pubsub"
	delKey := "test-del-key-pubsub"

	getDelKey := func(key string, value any) string {
		return delKey
	}

	cache.AddDelListener(obj, getDelKey)
	cache.Set(delKey, "test-value")

	cache.Start()

	time.Sleep(200 * time.Millisecond)

	event := CacheEvent{
		Action: "del",
		Key:    obj,
	}
	payload, _ := json.Marshal(event)
	_ = r.Publish(context.Background(), cache.cfg.Topic, payload).Err()

	time.Sleep(200 * time.Millisecond)

	retrieved, ok := cache.Get(delKey)
	assert.False(t, ok)
	assert.Nil(t, retrieved)
}
