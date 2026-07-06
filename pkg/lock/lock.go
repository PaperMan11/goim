package lock

import (
	"context"
	"errors"
	"time"
)

var (
	ErrLockNotObtained = errors.New("lock not obtained")
	ErrLockTimeout     = errors.New("lock timeout")
	ErrRetryExhausted  = errors.New("retry exhausted")
)

type RetryStrategy func(attempt int) time.Duration

func NoRetry() RetryStrategy {
	return func(attempt int) time.Duration {
		if attempt == 0 {
			return 0
		}
		return -1
	}
}

func LinearBackoff(backoff time.Duration) RetryStrategy {
	return func(attempt int) time.Duration {
		if attempt == 0 {
			return 0
		}
		return backoff
	}
}

func ExponentialBackoff(min, max time.Duration) RetryStrategy {
	return func(attempt int) time.Duration {
		if attempt == 0 {
			return 0
		}
		duration := min
		for i := 0; i < attempt; i++ {
			if duration > max/2 {
				return max
			}
			duration *= 2
		}
		if duration > max {
			return max
		}
		return duration
	}
}

type Locker interface {
	ExecWithLock(ctx context.Context, key string, fn func() error) error
	TryLock(ctx context.Context, key string) (bool, error)
	Unlock(ctx context.Context, key string) error
}

type Options struct {
	DefaultTTL    time.Duration
	RetryStrategy RetryStrategy
	MaxRetries    int
}

type Option func(*Options)

func WithDefaultTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.DefaultTTL = ttl
	}
}

func WithRetryStrategy(strategy RetryStrategy) Option {
	return func(o *Options) {
		o.RetryStrategy = strategy
	}
}

func WithMaxRetries(maxRetries int) Option {
	return func(o *Options) {
		o.MaxRetries = maxRetries
	}
}

func NewOptions(opts ...Option) *Options {
	o := &Options{
		DefaultTTL:    30 * time.Second,
		RetryStrategy: ExponentialBackoff(50*time.Millisecond, 2*time.Second),
		MaxRetries:    10,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
