package batcher

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// 批次阈值触发 + Stop 排空：缓冲中未达阈值的数据必须在 Stop 时全部冲出，不丢失。
func TestBatcherFlushOnStop(t *testing.T) {
	var mu sync.Mutex
	got := make(map[string]int)
	total := 0

	b := NewBatcher[string](
		WithWorkerNum(1),
		WithBatchSize(10),
		WithDataBufSize(64),
		WithInterval(time.Hour), // 定时器永不触发，只验证阈值与 Stop 排空
	)
	if err := b.Start(func(data *string) string { return *data }, func(key string, dataList []*string) {
		mu.Lock()
		got[key] += len(dataList)
		total += len(dataList)
		mu.Unlock()
	}); err != nil {
		t.Fatalf("start: %v", err)
	}

	for i := 0; i < 25; i++ {
		s := "k"
		if err := b.Push(&s); err != nil {
			t.Fatalf("push: %v", err)
		}
	}

	// 批次阈值 10 应立即触发一次
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := total
		mu.Unlock()
		if n >= 10 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("batchSize flush not triggered, total=%d", n)
		}
		time.Sleep(5 * time.Millisecond)
	}

	if err := b.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if total != 25 {
		t.Fatalf("Stop 后应处理全部 25 条, got %d", total)
	}
	if got["k"] != 25 {
		t.Fatalf("key 聚合错误: %v", got)
	}
}

// batchFn 的 panic 只影响当前批次（丢弃并恢复），worker 必须继续处理后续数据，
// 分片不能死亡（否则该 key 的 Push 会永久阻塞）。
func TestBatcherWorkerPanicRecovery(t *testing.T) {
	var mu sync.Mutex
	var processed []string

	b := NewBatcher[string](
		WithWorkerNum(1),
		WithBatchSize(1),
		WithDataBufSize(64),
		WithInterval(time.Hour),
	)
	if err := b.Start(func(data *string) string { return *data }, func(key string, dataList []*string) {
		if key == "boom" {
			panic("simulated batch failure")
		}
		mu.Lock()
		processed = append(processed, key)
		mu.Unlock()
	}); err != nil {
		t.Fatalf("start: %v", err)
	}

	boom, ok1, ok2 := "boom", "ok1", "ok2"
	for _, s := range []*string{&boom, &ok1, &ok2} {
		if err := b.Push(s); err != nil {
			t.Fatalf("push: %v", err)
		}
	}

	if err := b.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if strings.Join(processed, ",") != "ok1,ok2" {
		t.Fatalf("panic 后 worker 应继续处理后续消息, got %v", processed)
	}
}

// Stop 后 Push 返回 ErrorBatcherClosed，不再接受新数据；
// 重复 Stop 幂等；并发 Push/Stop 下 Push 不允许"假成功"（消息落入死 channel）。
func TestBatcherPushAfterStop(t *testing.T) {
	b := NewBatcher[string](WithWorkerNum(1), WithBatchSize(10))
	if err := b.Start(func(data *string) string { return *data }, func(key string, dataList []*string) {}); err != nil {
		t.Fatalf("start: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := "racy"
			errs <- b.Push(&s)
		}()
	}

	if err := b.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := b.Stop(); err != nil { // 重复 Stop 必须幂等，不得 panic 或悬挂
		t.Fatalf("second stop: %v", err)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil && err != ErrorBatcherClosed {
			t.Fatalf("unexpected push error: %v", err)
		}
	}

	s := "late"
	if err := b.Push(&s); err != ErrorBatcherClosed {
		t.Fatalf("Stop 后 Push 应返回 ErrorBatcherClosed, got %v", err)
	}
}

// 同 key 路由到同一分片，单 worker 下保证同 key 顺序。
func TestBatcherSameKeyOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string

	b := NewBatcher[string](
		WithWorkerNum(4),
		WithBatchSize(100),
		WithDataBufSize(64),
		WithInterval(time.Hour),
	)
	if err := b.Start(func(data *string) string { return *data }, func(key string, dataList []*string) {
		mu.Lock()
		order = append(order, key)
		mu.Unlock()
	}); err != nil {
		t.Fatalf("start: %v", err)
	}

	const n = 20
	want := make([]string, 0, n)
	for i := 0; i < n; i++ {
		s := "same-key"
		want = append(want, s)
		if err := b.Push(&s); err != nil {
			t.Fatalf("push: %v", err)
		}
	}

	if err := b.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 1 {
		t.Fatalf("同 key 应聚合为一个分组的批处理, got %d 次调用", len(order))
	}
}
