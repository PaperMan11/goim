package batcher

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"time"
)

const (
	DefaultWorkerNum   = 10              // 默认工作线程数
	DefaultBatchSize   = 100             // 默认批次大小
	DefaultInterval    = 3 * time.Second // 默认间隔时间
	DefaultDataBufSize = 1000            // 默认每个分片的数据缓冲区大小
)

var (
	ErrorBatcherClosed = errors.New("batcher is closed")
	ErrorBatcherFull   = errors.New("batcher is full")
	ErrorBatcherEmpty  = errors.New("batcher is empty")
)

type Config struct {
	workerNum   int           // 工作线程数（即分片数）
	batchSize   int           // 批次大小
	interval    time.Duration // 定时触发间隔
	dataBufSize int           // 每个分片的数据缓冲区大小
}

type BatcherOption func(*Config)

// ResetFunc 用于重置对象状态的函数
type ResetFunc[T any] func(*T)

type Batcher[T any] struct {
	cfg          Config          // 配置
	dataChs      []chan *T       // 按 key hash 分片的通道，保证同 key 路由到同一 worker
	doneCh       chan struct{}   // 完成通道
	wg           sync.WaitGroup  // 等待组，用于等待所有工作线程完成
	pool         sync.Pool       // 对象池
	resetFunc    ResetFunc[T]    // 对象重置函数
	keyExtractor func(*T) string // 在 Start 时设置，Push 时用于计算分片
}

func NewBatcher[T any](opts ...BatcherOption) *Batcher[T] {
	cfg := Config{
		workerNum:   DefaultWorkerNum,
		batchSize:   DefaultBatchSize,
		interval:    DefaultInterval,
		dataBufSize: DefaultDataBufSize,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	b := &Batcher[T]{
		cfg:    cfg,
		doneCh: make(chan struct{}, 1),
	}

	// 初始化分片通道
	b.dataChs = make([]chan *T, cfg.workerNum)
	for i := range b.dataChs {
		b.dataChs[i] = make(chan *T, cfg.dataBufSize)
	}

	// 初始化对象池
	b.pool = sync.Pool{
		New: func() interface{} {
			return new(T)
		},
	}

	return b
}

// SetResetFunc 设置对象重置函数
func (b *Batcher[T]) SetResetFunc(fn ResetFunc[T]) {
	b.resetFunc = fn
}

// Get 从对象池获取一个对象
func (b *Batcher[T]) Get() *T {
	return b.pool.Get().(*T)
}

// Put 将对象放回对象池
func (b *Batcher[T]) Put(data *T) {
	if b.resetFunc != nil {
		b.resetFunc(data)
	}
	b.pool.Put(data)
}

// shardIndex 根据 key 的 hash 值选择分片索引，保证同 key 路由到同一 worker。
func (b *Batcher[T]) shardIndex(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % len(b.dataChs)
}

func (b *Batcher[T]) Push(data *T) error {
	// 若 keyExtractor 已设置，按 key hash 路由到固定分片；
	// 否则随机选择一个分片（Start 之前调用 Push 的场景）。
	var idx int
	if b.keyExtractor != nil {
		idx = b.shardIndex(b.keyExtractor(data))
	} else {
		idx = 0
	}

	select {
	case <-b.doneCh:
		return ErrorBatcherClosed
	case b.dataChs[idx] <- data:
		return nil
	}
}

func (b *Batcher[T]) PushImmediately(data *T) error {
	var idx int
	if b.keyExtractor != nil {
		idx = b.shardIndex(b.keyExtractor(data))
	} else {
		idx = 0
	}

	select {
	case <-b.doneCh:
		return ErrorBatcherClosed
	case b.dataChs[idx] <- data:
		return nil
	default:
		return ErrorBatcherFull
	}
}

func (b *Batcher[T]) Start(keyExtractor func(*T) string, batchFn func(key string, dataList []*T)) error {
	b.keyExtractor = keyExtractor

	b.wg.Add(b.cfg.workerNum)
	for i := 0; i < b.cfg.workerNum; i++ {
		go func(j int, ch <-chan *T) {
			defer func() {
				b.wg.Done()
				r := recover()
				if r != nil {
					fmt.Printf("batcher worker %d panic: %v\n", j, r)
				}
			}()

			b.worker(ch, keyExtractor, batchFn)
		}(i, b.dataChs[i])
	}
	return nil
}

func (b *Batcher[T]) worker(ch <-chan *T, keyExtractor func(*T) string, batchFn func(key string, dataList []*T)) {
	ticker := time.NewTicker(b.cfg.interval)
	defer ticker.Stop()

	// 按分类键分组存储数据，key 为分类键（如会话 ID），value 为该组的数据列表
	dataGroups := make(map[string][]*T)

	for {
		select {
		case <-b.doneCh:
			// 收到停止信号，先处理剩余数据
			if len(dataGroups) > 0 {
				for key, dataList := range dataGroups {
					batchFn(key, dataList)
				}
				for _, items := range dataGroups {
					for _, item := range items {
						b.Put(item)
					}
				}
				clear(dataGroups)
			}
			return
		case data := <-ch:
			// 提取分类键（如会话 ID）
			key := keyExtractor(data)
			// 将数据添加到对应的分组中
			dataGroups[key] = append(dataGroups[key], data)

			// 如果当前批次大小达到阈值，立即处理
			if len(dataGroups[key]) >= b.cfg.batchSize {
				batchFn(key, dataGroups[key])
				// 处理完成后将对象放回对象池
				for _, item := range dataGroups[key] {
					b.Put(item)
				}
				delete(dataGroups, key)
			}
		case <-ticker.C:
			// 定时触发批量处理
			if len(dataGroups) > 0 {
				for key, dataList := range dataGroups {
					batchFn(key, dataList)
				}
				for _, items := range dataGroups {
					for _, item := range items {
						b.Put(item)
					}
				}
				clear(dataGroups)
			}
		}
	}
}

func (b *Batcher[T]) Stop() error {
	b.doneCh <- struct{}{}
	close(b.doneCh)
	b.wg.Wait()
	return nil
}
