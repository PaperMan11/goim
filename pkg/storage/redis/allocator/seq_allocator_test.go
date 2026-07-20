package allocator

import (
	"context"
	"errors"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) goredis.UniversalClient {
	redisHost := "192.168.241.128:6379"
	redisPass := "123456"
	r := goredis.NewClient(&goredis.Options{
		Addr:     redisHost,
		Password: redisPass,
		DB:       0,
	})

	ctx := context.Background()
	if err := r.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s: %v", redisHost, err)
	}

	t.Cleanup(func() {
		iter := r.Scan(ctx, 0, "goim:msg:seq:*", 0).Iterator()
		var keys []string
		for iter.Next(ctx) {
			keys = append(keys, iter.Val())
		}
		if len(keys) > 0 {
			_ = r.Del(ctx, keys...).Err()
		}
		_ = r.Close()
	})

	return r
}

func setupAllocator(t *testing.T, options ...RedisSeqAllocatorOption) *RedisSeqAllocator {
	r := setupTestRedis(t)
	allocator, err := NewRedisSeqAllocator(r, options...)
	if err != nil {
		t.Fatalf("NewRedisSeqAllocator failed: %v", err)
	}
	return allocator
}

func TestRedisSeqAllocator_Allocate(t *testing.T) {
	allocator := setupAllocator(t)
	ctx := context.Background()
	conversationID := "test-conv-1"

	seq1, err := allocator.Allocate(ctx, conversationID)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if seq1 != 1 {
		t.Errorf("Expected seq 1, got %d", seq1)
	}

	seq2, err := allocator.Allocate(ctx, conversationID)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if seq2 != 2 {
		t.Errorf("Expected seq 2, got %d", seq2)
	}

	seq3, err := allocator.Allocate(ctx, conversationID)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if seq3 != 3 {
		t.Errorf("Expected seq 3, got %d", seq3)
	}
}

func TestRedisSeqAllocator_AllocateBatch(t *testing.T) {
	allocator := setupAllocator(t)
	ctx := context.Background()
	conversationID := "test-conv-2"

	start, end, err := allocator.AllocateBatch(ctx, conversationID, 10)
	if err != nil {
		t.Fatalf("AllocateBatch failed: %v", err)
	}
	if start != 1 {
		t.Errorf("Expected start 1, got %d", start)
	}
	if end != 10 {
		t.Errorf("Expected end 10, got %d", end)
	}

	start2, end2, err := allocator.AllocateBatch(ctx, conversationID, 5)
	if err != nil {
		t.Fatalf("AllocateBatch failed: %v", err)
	}
	if start2 != 11 {
		t.Errorf("Expected start 11, got %d", start2)
	}
	if end2 != 15 {
		t.Errorf("Expected end 15, got %d", end2)
	}
}

func TestRedisSeqAllocator_AllocateBatchCrossPool(t *testing.T) {
	maxSeq := int64(0)
	allocator := setupAllocator(t,
		WithPoolSize(5),
		WithGetMaxSeqFn(func(ctx context.Context, conversationID string) (int64, error) {
			return maxSeq, nil
		}),
	)
	ctx := context.Background()
	conversationID := "test-conv-3"

	start1, end1, err := allocator.AllocateBatch(ctx, conversationID, 3)
	if err != nil {
		t.Fatalf("AllocateBatch failed: %v", err)
	}
	if start1 != 1 || end1 != 3 {
		t.Errorf("Expected [1,3], got [%d,%d]", start1, end1)
	}
	maxSeq = 3

	start2, end2, err := allocator.AllocateBatch(ctx, conversationID, 5)
	if err != nil {
		t.Fatalf("AllocateBatch failed: %v", err)
	}
	if start2 != 4 || end2 != 8 {
		t.Errorf("Expected [4,8], got [%d,%d]", start2, end2)
	}
	maxSeq = 8

	start3, end3, err := allocator.AllocateBatch(ctx, conversationID, 3)
	if err != nil {
		t.Fatalf("AllocateBatch failed: %v", err)
	}
	if start3 != 9 || end3 != 11 {
		t.Errorf("Expected [9,11], got [%d,%d]", start3, end3)
	}
}

func TestRedisSeqAllocator_GetCurrent(t *testing.T) {
	allocator := setupAllocator(t)
	ctx := context.Background()
	conversationID := "test-conv-4"

	allocator.Allocate(ctx, conversationID)
	allocator.Allocate(ctx, conversationID)
	allocator.Allocate(ctx, conversationID)

	curr, err := allocator.GetCurrent(ctx, conversationID)
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if curr != 3 {
		t.Errorf("Expected curr 3, got %d", curr)
	}
}

func TestRedisSeqAllocator_GetSeqRange(t *testing.T) {
	allocator := setupAllocator(t, WithPoolSize(100))
	ctx := context.Background()
	conversationID := "test-conv-5"

	allocator.Allocate(ctx, conversationID)
	allocator.Allocate(ctx, conversationID)

	curr, last, err := allocator.GetSeqRange(ctx, conversationID)
	if err != nil {
		t.Fatalf("GetSeqRange failed: %v", err)
	}
	if curr != 2 {
		t.Errorf("Expected curr 2, got %d", curr)
	}
	if last != 100 {
		t.Errorf("Expected last 100, got %d", last)
	}
}

func TestRedisSeqAllocator_SetAndGet(t *testing.T) {
	allocator := setupAllocator(t)
	ctx := context.Background()
	conversationID := "test-conv-6"

	err := allocator.Set(ctx, conversationID, 100)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	curr, err := allocator.GetCurrent(ctx, conversationID)
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if curr != 100 {
		t.Errorf("Expected curr 100, got %d", curr)
	}

	seq, err := allocator.Allocate(ctx, conversationID)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if seq != 101 {
		t.Errorf("Expected seq 101, got %d", seq)
	}
}

func TestRedisSeqAllocator_Reset(t *testing.T) {
	allocator := setupAllocator(t)
	ctx := context.Background()
	conversationID := "test-conv-7"

	allocator.Allocate(ctx, conversationID)
	allocator.Allocate(ctx, conversationID)

	err := allocator.Reset(ctx, conversationID)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	curr, err := allocator.GetCurrent(ctx, conversationID)
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if curr != 0 {
		t.Errorf("Expected curr 0 after reset, got %d", curr)
	}

	seq, err := allocator.Allocate(ctx, conversationID)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if seq != 1 {
		t.Errorf("Expected seq 1 after reset, got %d", seq)
	}
}

func TestRedisSeqAllocator_SyncFromDB(t *testing.T) {
	allocator := setupAllocator(t)
	ctx := context.Background()
	conversationID := "test-conv-8"

	err := allocator.SyncFromDB(ctx, conversationID, func(ctx context.Context, conversationID string) (int64, error) {
		return 50, nil
	})
	if err != nil {
		t.Fatalf("SyncFromDB failed: %v", err)
	}

	curr, err := allocator.GetCurrent(ctx, conversationID)
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if curr != 50 {
		t.Errorf("Expected curr 50, got %d", curr)
	}

	seq, err := allocator.Allocate(ctx, conversationID)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if seq != 51 {
		t.Errorf("Expected seq 51, got %d", seq)
	}
}

func TestRedisSeqAllocator_SyncFromDBWithError(t *testing.T) {
	allocator := setupAllocator(t)
	ctx := context.Background()
	conversationID := "test-conv-9"

	err := allocator.SyncFromDB(ctx, conversationID, func(ctx context.Context, conversationID string) (int64, error) {
		return 0, errors.New("db error")
	})
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestRedisSeqAllocator_AllocateWithDBFn(t *testing.T) {
	allocator := setupAllocator(t,
		WithPoolSize(10),
		WithGetMaxSeqFn(func(ctx context.Context, conversationID string) (int64, error) {
			return 100, nil
		}),
	)
	ctx := context.Background()
	conversationID := "test-conv-10"

	seq, err := allocator.Allocate(ctx, conversationID)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if seq != 101 {
		t.Errorf("Expected seq 101, got %d", seq)
	}

	for i := 0; i < 9; i++ {
		allocator.Allocate(ctx, conversationID)
	}

	curr, last, err := allocator.GetSeqRange(ctx, conversationID)
	if err != nil {
		t.Fatalf("GetSeqRange failed: %v", err)
	}
	if curr != 110 {
		t.Errorf("Expected curr 110, got %d", curr)
	}
	if last != 110 {
		t.Errorf("Expected last 110, got %d", last)
	}
}

func TestRedisSeqAllocator_MaxRetries(t *testing.T) {
	allocator := setupAllocator(t, WithMaxRetries(2), WithRetryInterval(10*time.Millisecond))
	ctx := context.Background()
	conversationID := "test-conv-lock"

	r := setupTestRedis(t)
	_ = r.HSet(ctx, "goim:msg:seq:"+conversationID, "LOCK", "12345").Err()
	_ = r.Expire(ctx, "goim:msg:seq:"+conversationID, 5*time.Second).Err()

	_, err := allocator.Allocate(ctx, conversationID)
	if err != ErrRetryExhausted {
		t.Errorf("Expected ErrRetryExhausted, got %v", err)
	}
}

func TestRedisSeqAllocator_DifferentConversations(t *testing.T) {
	allocator := setupAllocator(t)
	ctx := context.Background()

	seq1, err := allocator.Allocate(ctx, "conv-a")
	if err != nil {
		t.Fatalf("Allocate conv-a failed: %v", err)
	}
	if seq1 != 1 {
		t.Errorf("Expected seq 1 for conv-a, got %d", seq1)
	}

	seq2, err := allocator.Allocate(ctx, "conv-b")
	if err != nil {
		t.Fatalf("Allocate conv-b failed: %v", err)
	}
	if seq2 != 1 {
		t.Errorf("Expected seq 1 for conv-b, got %d", seq2)
	}

	seq3, err := allocator.Allocate(ctx, "conv-a")
	if err != nil {
		t.Fatalf("Allocate conv-a failed: %v", err)
	}
	if seq3 != 2 {
		t.Errorf("Expected seq 2 for conv-a, got %d", seq3)
	}
}

func TestRedisSeqAllocator_AllocateZeroCount(t *testing.T) {
	allocator := setupAllocator(t)
	ctx := context.Background()
	conversationID := "test-conv-zero"

	start, end, err := allocator.AllocateBatch(ctx, conversationID, 0)
	if err != nil {
		t.Fatalf("AllocateBatch with 0 count failed: %v", err)
	}
	if start != 0 || end != 0 {
		t.Errorf("Expected [0,0], got [%d,%d]", start, end)
	}

	start, end, err = allocator.AllocateBatch(ctx, conversationID, -1)
	if err != nil {
		t.Fatalf("AllocateBatch with negative count failed: %v", err)
	}
	if start != 0 || end != 0 {
		t.Errorf("Expected [0,0], got [%d,%d]", start, end)
	}
}

func TestRedisSeqAllocator_Performance(t *testing.T) {
	maxSeq := int64(0)
	allocator := setupAllocator(t,
		WithPoolSize(1000),
		WithGetMaxSeqFn(func(ctx context.Context, conversationID string) (int64, error) {
			return maxSeq, nil
		}),
	)
	ctx := context.Background()
	conversationID := "test-conv-perf"

	const iterations = 500
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := allocator.Allocate(ctx, conversationID)
		if err != nil {
			t.Fatalf("Allocate failed at iteration %d: %v", i, err)
		}
		if i+1 >= 1000 {
			maxSeq = int64(i + 1)
		}
	}
	duration := time.Since(start)

	t.Logf("Allocated %d sequences in %v (%.2f ops/ms)", iterations, duration, float64(iterations)/duration.Seconds()/1000)

	curr, err := allocator.GetCurrent(ctx, conversationID)
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if curr != iterations {
		t.Errorf("Expected curr %d, got %d", iterations, curr)
	}
}

func TestRedisSeqAllocator_AllocateBatchPerformance(t *testing.T) {
	maxSeq := int64(0)
	allocator := setupAllocator(t,
		WithPoolSize(1000),
		WithGetMaxSeqFn(func(ctx context.Context, conversationID string) (int64, error) {
			return maxSeq, nil
		}),
	)
	ctx := context.Background()
	conversationID := "test-conv-batch-perf"

	const total = 5000
	const batchSize = 100
	start := time.Now()
	currentAllocated := 0
	for i := 0; i < total; i += batchSize {
		remaining := total - i
		if remaining > batchSize {
			remaining = batchSize
		}
		_, _, err := allocator.AllocateBatch(ctx, conversationID, remaining)
		if err != nil {
			t.Fatalf("AllocateBatch failed at iteration %d: %v", i, err)
		}
		currentAllocated += remaining
		if currentAllocated >= 1000 {
			maxSeq = int64(currentAllocated)
		}
	}
	duration := time.Since(start)

	t.Logf("Allocated %d sequences in %v (%.2f ops/ms)", total, duration, float64(total)/duration.Seconds()/1000)

	curr, err := allocator.GetCurrent(ctx, conversationID)
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if curr != total {
		t.Errorf("Expected curr %d, got %d", total, curr)
	}
}
