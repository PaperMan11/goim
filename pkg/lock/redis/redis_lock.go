package redis

import (
	"context"
	"errors"
	"time"

	"github.com/PaperMan11/goim/pkg/lock"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	red "github.com/redis/go-redis/v9"
	zredis "github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	lockCommand = `if redis.call("GET", KEYS[1]) == ARGV[1] then
    redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
    return "OK"
else
    return redis.call("SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2])
end`

	unlockCommand = `if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end`

	randomLen = 16
)

type RedisLocker struct {
	redisClient   *zredis.Redis
	lockScript    *zredis.Script
	unlockScript  *zredis.Script
	defaultTTL    time.Duration
	retryStrategy lock.RetryStrategy
	maxRetries    int
}

func NewRedisLocker(redisClient *zredis.Redis, options ...lock.Option) *RedisLocker {
	opts := lock.NewOptions(options...)

	return &RedisLocker{
		redisClient:   redisClient,
		lockScript:    zredis.NewScript(lockCommand),
		unlockScript:  zredis.NewScript(unlockCommand),
		defaultTTL:    opts.DefaultTTL,
		retryStrategy: opts.RetryStrategy,
		maxRetries:    opts.MaxRetries,
	}
}

func (r *RedisLocker) ExecWithLock(ctx context.Context, key string, fn func() error) error {
	l, err := r.obtainLock(ctx, key)
	if err != nil {
		return err
	}
	defer l.release(ctx)

	return fn()
}

func (r *RedisLocker) TryLock(ctx context.Context, key string) (bool, error) {
	l, err := r.tryObtainOnce(ctx, key)
	if err != nil {
		return false, err
	}
	if l == nil {
		return false, nil
	}
	l.release(ctx)
	return true, nil
}

func (r *RedisLocker) Unlock(ctx context.Context, key string) error {
	return errors.New("use ExecWithLock or ExecWithLockAndReturn for safe unlock")
}

func (r *RedisLocker) obtainLock(ctx context.Context, key string) (*lockObj, error) {
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		l, err := r.tryObtainOnce(ctx, key)
		if err != nil {
			return nil, err
		}
		if l != nil {
			return l, nil
		}

		if attempt < r.maxRetries {
			backoff := r.retryStrategy(attempt)
			if backoff < 0 {
				return nil, lock.ErrLockNotObtained
			}
			time.Sleep(backoff)
		}
	}

	return nil, lock.ErrRetryExhausted
}

func (r *RedisLocker) tryObtainOnce(ctx context.Context, key string) (*lockObj, error) {
	id := randomStr(randomLen)
	ttl := int(r.defaultTTL.Milliseconds())

	result, err := r.redisClient.ScriptRunCtx(ctx, r.lockScript, []string{key}, []interface{}{id, ttl})
	if err != nil {
		if errors.Is(err, red.Nil) {
			return nil, nil
		}
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	reply, ok := result.(string)
	if ok && reply == "OK" {
		return &lockObj{
			redisClient:  r.redisClient,
			unlockScript: r.unlockScript,
			key:          key,
			id:           id,
		}, nil
	}

	return nil, nil
}

func randomStr(n int) string {
	return randx.AlphaString(n)
}

type lockObj struct {
	redisClient  *zredis.Redis
	unlockScript *zredis.Script
	key          string
	id           string
}

func (l *lockObj) release(ctx context.Context) {
	_, _ = l.redisClient.ScriptRunCtx(ctx, l.unlockScript, []string{l.key}, []interface{}{l.id})
}
