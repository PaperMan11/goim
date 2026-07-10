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

func TestLocalCache_SetGetDel(t *testing.T) {
	cache := setupLocalCache(t, nil)

	key := "test-key-setget"
	value := "test-value-setget"

	cache.Set(key, value)
	retrieved, ok := cache.Get(key)
	assert.True(t, ok)
	assert.Equal(t, value, retrieved)

	cache.Del(key)
	retrieved, ok = cache.Get(key)
	assert.False(t, ok)
	assert.Nil(t, retrieved)
}

func TestLocalCache_SetWithExpire(t *testing.T) {
	cache := setupLocalCache(t, nil)

	key := "test-key-expire"
	value := "test-value-expire"
	expire := 2 * time.Second

	cache.SetWithExpire(key, value, expire)
	retrieved, ok := cache.Get(key)
	assert.True(t, ok)
	assert.Equal(t, value, retrieved)

	time.Sleep(3 * time.Second)

	retrieved, ok = cache.Get(key)
	assert.False(t, ok)
	assert.Nil(t, retrieved)
}

func TestLocalCache_HandleEvent(t *testing.T) {
	cache := setupLocalCache(t, nil)

	key1 := "test-key-1"
	key2 := "test-key-2"
	key3 := "test-key-3"

	cache.Set(key1, "value1")
	cache.Set(key2, "value2")
	cache.Set(key3, "value3")

	_, ok := cache.Get(key1)
	assert.True(t, ok)
	_, ok = cache.Get(key2)
	assert.True(t, ok)
	_, ok = cache.Get(key3)
	assert.True(t, ok)

	keysToDelete := []string{key1, key3}
	payload, _ := json.Marshal(keysToDelete)
	cache.handleEvent(string(payload))

	_, ok = cache.Get(key1)
	assert.False(t, ok)

	_, ok = cache.Get(key2)
	assert.True(t, ok)

	_, ok = cache.Get(key3)
	assert.False(t, ok)
}

func TestLocalCache_HandleEvent_InvalidPayload(t *testing.T) {
	cache := setupLocalCache(t, nil)

	key := "test-key-valid"
	cache.Set(key, "value")

	invalidPayloads := []string{
		`{"action": "del", "key": "test"}`,
		`"not-an-array"`,
		`123`,
		`invalid-json`,
	}

	for _, payload := range invalidPayloads {
		cache.handleEvent(payload)
	}

	_, ok := cache.Get(key)
	assert.True(t, ok)
}

func TestLocalCache_HandleEvent_EmptyArray(t *testing.T) {
	cache := setupLocalCache(t, nil)

	key := "test-key-empty"
	cache.Set(key, "value")

	payload, _ := json.Marshal([]string{})
	cache.handleEvent(string(payload))

	_, ok := cache.Get(key)
	assert.True(t, ok)
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

func TestLocalCache_Publish_NoRedis(t *testing.T) {
	cache := setupLocalCache(t, nil)

	key1 := "test-key-publish-1"
	key2 := "test-key-publish-2"

	cache.Set(key1, "value1")
	cache.Set(key2, "value2")

	err := cache.PublishDelete([]string{key1})
	assert.NoError(t, err)

	_, ok := cache.Get(key1)
	assert.False(t, ok)

	_, ok = cache.Get(key2)
	assert.True(t, ok)
}

func TestLocalCache_Publish_WithRedis(t *testing.T) {
	r := setupTestRedis(t)
	cache1 := setupLocalCache(t, r)
	cache2 := setupLocalCache(t, r)
	cleanupRedisTopic(t, r, cache1.cfg.Topic)

	key1 := "test-key-publish-redis-1"
	key2 := "test-key-publish-redis-2"

	cache1.Set(key1, "value1")
	cache1.Set(key2, "value2")

	cache2.Set(key1, "value1")
	cache2.Set(key2, "value2")

	cache1.Start()
	cache2.Start()

	time.Sleep(200 * time.Millisecond)

	err := cache1.PublishDelete([]string{key1})
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	_, ok := cache1.Get(key1)
	assert.False(t, ok)
	_, ok = cache1.Get(key2)
	assert.True(t, ok)

	_, ok = cache2.Get(key1)
	assert.False(t, ok)
	_, ok = cache2.Get(key2)
	assert.True(t, ok)
}

func TestLocalCache_Publish_EmptyKeys(t *testing.T) {
	cache := setupLocalCache(t, nil)

	key := "test-key-empty-publish"
	cache.Set(key, "value")

	err := cache.PublishDelete([]string{})
	assert.NoError(t, err)

	_, ok := cache.Get(key)
	assert.True(t, ok)
}

func TestLocalCache_PubSub(t *testing.T) {
	r := setupTestRedis(t)
	cache := setupLocalCache(t, r)
	cleanupRedisTopic(t, r, cache.cfg.Topic)

	key1 := "test-key-pubsub-1"
	key2 := "test-key-pubsub-2"

	cache.Set(key1, "value1")
	cache.Set(key2, "value2")

	cache.Start()

	time.Sleep(200 * time.Millisecond)

	keysToDelete := []string{key1}
	payload, _ := json.Marshal(keysToDelete)
	_ = r.Publish(context.Background(), cache.cfg.Topic, payload).Err()

	time.Sleep(200 * time.Millisecond)

	retrieved, ok := cache.Get(key1)
	assert.False(t, ok)
	assert.Nil(t, retrieved)

	retrieved, ok = cache.Get(key2)
	assert.True(t, ok)
	assert.Equal(t, "value2", retrieved)
}
