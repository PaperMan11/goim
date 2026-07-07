package captcha

import (
	"github.com/mojocn/base64Captcha"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	RedisPrefix = "captcha:"
	RedisExpire = 60
)

type RedisStore struct {
	redisClient *redis.Redis
	keyPrefix   string
	expire      int
}

var _ base64Captcha.Store = (*RedisStore)(nil)

func NewRedisStore(redisClient *redis.Redis, keyPrefix string, expire int) *RedisStore {
	if keyPrefix == "" {
		keyPrefix = RedisPrefix
	}
	if expire <= 0 {
		expire = RedisExpire
	}
	return &RedisStore{
		redisClient: redisClient,
		keyPrefix:   keyPrefix,
		expire:      expire,
	}
}

func (s *RedisStore) getKey(id string) string {
	return s.keyPrefix + id
}

func (s *RedisStore) Set(id string, value string) error {
	return s.redisClient.Setex(s.getKey(id), value, s.expire)
}
func (s *RedisStore) Get(id string, clear bool) string {
	get, _ := s.redisClient.Get(s.keyPrefix + id)
	if clear {
		_, _ = s.redisClient.Del(s.getKey(id))
	}
	return get
}
func (s *RedisStore) Verify(id string, answer string, clear bool) bool {
	return s.Get(id, clear) == answer
}
