package localcache

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/collection"
)

type LocalCache interface {
	Start()
	Del(key string)
	Get(key string) (any, bool)
	Set(key string, value any)
	SetWithExpire(key string, value any, expire time.Duration)
	Take(key string, fetch func() (any, error)) (any, error)
	PublishDelete(keys []string) error
	Close() error
}

type CacheConfig struct {
	Expire time.Duration `json:",default=5m"`
	Limit  int           `json:",default=1000"`
	Name   string        `json:",default=rpccache"`
	Topic  string        `json:",default=rpccache"`
}

type localCache struct {
	cfg       CacheConfig
	cache     *collection.Cache
	ctx       context.Context
	cancel    context.CancelFunc
	redis     redis.UniversalClient
	mu        sync.Mutex
	running   bool
	observers sync.Map
	closed    bool
}

func NewLocalCache(cfg CacheConfig, redis redis.UniversalClient) (*localCache, error) {
	opts := []collection.CacheOption{}
	if cfg.Limit > 0 {
		opts = append(opts, collection.WithLimit(cfg.Limit))
	}
	if cfg.Name != "" {
		opts = append(opts, collection.WithName(cfg.Name))
	}
	cache, err := collection.NewCache(cfg.Expire, opts...)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &localCache{
		cfg:     cfg,
		cache:   cache,
		redis:   redis,
		ctx:     ctx,
		cancel:  cancel,
		running: false,
	}, nil
}

func MustNewLocalCache(cfg CacheConfig, redis redis.UniversalClient) *localCache {
	cache, err := NewLocalCache(cfg, redis)
	if err != nil {
		panic(err)
	}
	return cache
}

func (c *localCache) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running || c.closed {
		return
	}
	c.running = true
	if c.redis != nil {
		go c.runSubscriber()
	}
}

func (c *localCache) runSubscriber() {
	pubsub := c.redis.Subscribe(c.ctx, c.cfg.Topic)
	defer pubsub.Close()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-pubsub.Channel():
			if !ok {
				return
			}
			c.handleEvent(msg.Payload)
		}
	}
}

func (c *localCache) handleEvent(payload string) {
	var keys []string
	if err := json.Unmarshal([]byte(payload), &keys); err != nil {
		return
	}

	for _, key := range keys {
		c.cache.Del(key)
	}
}

func (c *localCache) Del(key string) {
	c.cache.Del(key)
}

func (c *localCache) Get(key string) (any, bool) {
	return c.cache.Get(key)
}

func (c *localCache) Set(key string, value any) {
	c.cache.Set(key, value)
}

func (c *localCache) SetWithExpire(key string, value any, expire time.Duration) {
	c.cache.SetWithExpire(key, value, expire)
}

func (c *localCache) Take(key string, fetch func() (any, error)) (any, error) {
	return c.cache.Take(key, fetch)
}

func (c *localCache) PublishDelete(keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	for _, key := range keys {
		c.cache.Del(key)
	}

	if c.redis == nil {
		return nil
	}

	payload, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	return c.redis.Publish(c.ctx, c.cfg.Topic, payload).Err()
}

func (c *localCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	c.running = false
	c.cancel()
	if c.redis != nil {
		return c.redis.Close()
	}
	return nil
}
