package redis

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/PaperMan11/goim/pkg/lock"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

func setupTestRedis(t *testing.T) *redis.Redis {
	redisHost := "192.168.241.128:6379"
	redisPass := "123456"
	r, err := redis.NewRedis(redis.RedisConf{
		Host: redisHost,
		Type: "node",
		Pass: redisPass,
	})
	if err != nil {
		t.Skipf("Redis not available at %s: %v", redisHost, err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		keys, _ := r.KeysCtx(ctx, "lock:*")
		if len(keys) > 0 {
			r.DelCtx(ctx, keys...)
		}
	})

	return r
}

func setupLocker(t *testing.T, options ...lock.Option) *RedisLocker {
	r := setupTestRedis(t)
	return NewRedisLocker(r, options...)
}

func TestRedisLocker_ExecWithLock(t *testing.T) {
	locker := setupLocker(t)
	ctx := context.Background()
	key := "lock:test-exec-with-lock"

	counter := 0
	err := locker.ExecWithLock(ctx, key, func() error {
		counter++
		return nil
	})
	if err != nil {
		t.Fatalf("ExecWithLock failed: %v", err)
	}
	if counter != 1 {
		t.Errorf("Expected counter 1, got %d", counter)
	}
}

func TestRedisLocker_ExecWithLockError(t *testing.T) {
	locker := setupLocker(t)
	ctx := context.Background()
	key := "lock:test-exec-with-lock-error"

	expectedErr := errors.New("business error")
	err := locker.ExecWithLock(ctx, key, func() error {
		return expectedErr
	})
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRedisLocker_ExecWithLockConcurrent(t *testing.T) {
	locker := setupLocker(t)
	ctx := context.Background()
	key := "lock:test-exec-with-lock-concurrent"

	const goroutines = 10
	counter := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = locker.ExecWithLock(ctx, key, func() error {
				mu.Lock()
				counter++
				mu.Unlock()
				time.Sleep(10 * time.Millisecond)
				return nil
			})
		}()
	}

	wg.Wait()
	if counter != goroutines {
		t.Errorf("Expected counter %d, got %d", goroutines, counter)
	}
}

func TestRedisLocker_TryLock(t *testing.T) {
	locker := setupLocker(t)
	ctx := context.Background()
	key := "lock:test-try-lock"

	locked, err := locker.TryLock(ctx, key)
	if err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}
	if !locked {
		t.Error("Expected locked to be true")
	}
}

func TestRedisLocker_TryLockAlreadyLocked(t *testing.T) {
	locker := setupLocker(t)
	ctx := context.Background()
	key := "lock:test-try-lock-already-locked"

	r := setupTestRedis(t)
	_ = r.SetCtx(ctx, key, "fake-lock-value")
	_ = r.ExpireCtx(ctx, key, 5)

	locked, err := locker.TryLock(ctx, key)
	if err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}
	if locked {
		t.Error("Expected locked to be false when lock already held")
	}
}

func TestRedisLocker_ExecWithLockRetryExhausted(t *testing.T) {
	locker := setupLocker(t,
		lock.WithMaxRetries(2),
		lock.WithRetryStrategy(lock.LinearBackoff(10*time.Millisecond)),
	)
	ctx := context.Background()
	key := "lock:test-retry-exhausted"

	r := setupTestRedis(t)
	_ = r.SetCtx(ctx, key, "fake-lock-value")
	_ = r.ExpireCtx(ctx, key, 5)

	err := locker.ExecWithLock(ctx, key, func() error {
		return nil
	})
	if err != lock.ErrRetryExhausted {
		t.Errorf("Expected ErrRetryExhausted, got %v", err)
	}
}

func TestRedisLocker_ExecWithLockContextCanceled(t *testing.T) {
	locker := setupLocker(t,
		lock.WithMaxRetries(100),
		lock.WithRetryStrategy(lock.LinearBackoff(10*time.Millisecond)),
	)
	key := "lock:test-ctx-canceled"

	r := setupTestRedis(t)
	ctxBg := context.Background()
	_ = r.SetCtx(ctxBg, key, "fake-lock-value")
	_ = r.ExpireCtx(ctxBg, key, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := locker.ExecWithLock(ctx, key, func() error {
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestRedisLocker_NoRetry(t *testing.T) {
	locker := setupLocker(t,
		lock.WithMaxRetries(0),
		lock.WithRetryStrategy(lock.NoRetry()),
	)
	ctx := context.Background()
	key := "lock:test-no-retry"

	r := setupTestRedis(t)
	_ = r.SetCtx(ctx, key, "fake-lock-value")
	_ = r.ExpireCtx(ctx, key, 5)

	err := locker.ExecWithLock(ctx, key, func() error {
		return nil
	})
	if err != lock.ErrRetryExhausted {
		t.Errorf("Expected ErrRetryExhausted, got %v", err)
	}
}

func TestRetryStrategies(t *testing.T) {
	testCases := []struct {
		name     string
		strategy lock.RetryStrategy
		attempt  int
		expected time.Duration
	}{
		{"NoRetry_first", lock.NoRetry(), 0, 0},
		{"NoRetry_retry", lock.NoRetry(), 1, -1},
		{"LinearBackoff_first", lock.LinearBackoff(100 * time.Millisecond), 0, 0},
		{"LinearBackoff_retry", lock.LinearBackoff(100 * time.Millisecond), 1, 100 * time.Millisecond},
		{"ExponentialBackoff_first", lock.ExponentialBackoff(50*time.Millisecond, 2*time.Second), 0, 0},
		{"ExponentialBackoff_retry1", lock.ExponentialBackoff(50*time.Millisecond, 2*time.Second), 1, 100 * time.Millisecond},
		{"ExponentialBackoff_retry2", lock.ExponentialBackoff(50*time.Millisecond, 2*time.Second), 2, 200 * time.Millisecond},
		{"ExponentialBackoff_max", lock.ExponentialBackoff(50*time.Millisecond, 2*time.Second), 100, 2 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.strategy(tc.attempt)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}
