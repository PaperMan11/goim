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
	AddDelListener(obj string, getDelKey func(key string, value any) string)
	Close() error
}

type CacheConfig struct {
	Expire time.Duration `json:",default=5m"`
	Limit  int           `json:",default=1000"`
	Name   string        `json:",default=rpccache"`
	Topic  string        `json:",default=rpccache"`
}

type localCache struct {
	cfg            CacheConfig
	cache          *collection.Cache
	ctx            context.Context
	cancel         context.CancelFunc
	redis          redis.UniversalClient
	mu             sync.Mutex
	running        bool
	observers      sync.Map
	closed         bool
	getDelKeyFuncs map[string][]func(string, any) string // 构造del key方法
}

type CacheEvent struct {
	Action string `json:"action"`
	Key    string `json:"key"`
	Value  any    `json:"value,omitempty"`
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
		cfg:            cfg,
		cache:          cache,
		redis:          redis,
		ctx:            ctx,
		cancel:         cancel,
		running:        false,
		getDelKeyFuncs: make(map[string][]func(string, any) string),
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
	var event CacheEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return
	}

	switch event.Action {
	case "del":
		objs := c.getDelKeyFuncs[event.Key]
		for _, getDelKey := range objs {
			delKey := getDelKey(event.Key, event.Value)
			c.Del(delKey)
		}
	default:
		return
	}
}

func (c *localCache) AddDelListener(obj string, getDelKey func(key string, value any) string) {
	c.getDelKeyFuncs[obj] = append(c.getDelKeyFuncs[obj], getDelKey)
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
