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
	DefaultTimeout     = 5 * time.Second // 默认超时时间
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

type Batcher[T any] struct {
	cfg          Config          // 配置
	dataChs      []chan *T       // 按 key hash 分片的通道，保证同 key 路由到同一 worker
	doneCh       chan struct{}   // 完成通道
	wg           sync.WaitGroup  // 等待组，用于等待所有工作线程完成
	keyExtractor func(*T) string // 在 Start 时设置，Push 时用于计算分片

	mu     sync.RWMutex // 保护 closed 与 Push 的投递：保证"检查关闭"与"进入 channel"原子
	closed bool
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

	return b
}

// shardIndex 根据 key 的 hash 值选择分片索引，保证同 key 路由到同一 worker。
func (b *Batcher[T]) shardIndex(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % len(b.dataChs)
}

func (b *Batcher[T]) Push(data *T) error {
	// 若 keyExtractor 已设置，按 key hash 路由到固定分片；
	// 否则路由到 0 号分片（Start 之前调用 Push 的场景）。
	var idx int
	if b.keyExtractor != nil {
		idx = b.shardIndex(b.keyExtractor(data))
	}

	// 不能用 select 与 doneCh 竞争判断关闭：Stop 后两个 case 可能同时就绪，
	// select 随机选择会让消息以"成功"假象落入无人消费的死 channel。
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return ErrorBatcherClosed
	}

	timer := time.NewTimer(DefaultTimeout)
	defer timer.Stop()

	select {
	case b.dataChs[idx] <- data:
	case <-timer.C:
		return ErrorBatcherFull
	}
	return nil
}

func (b *Batcher[T]) Start(keyExtractor func(*T) string, batchFn func(key string, dataList []*T)) error {
	b.keyExtractor = keyExtractor

	b.wg.Add(b.cfg.workerNum)
	for i := 0; i < b.cfg.workerNum; i++ {
		go func(j int, ch <-chan *T) {
			// 兜底防线：正常情况下 batchFn 的 panic 已在 handle 内恢复，不会走到这里；
			// 若 keyExtractor 等处发生 panic，保证 wg 不悬挂并记录现场。
			defer func() {
				b.wg.Done()
				if r := recover(); r != nil {
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

	// handle 恢复单次批次处理中的 panic：worker 继续运行，
	// 避免一个坏批次永久杀死分片导致该 key 的 Push 死锁。失败批次直接丢弃。
	handle := func(key string, dataList []*T) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("batcher worker panic, key: %s, batchSize: %d, err: %v\n", key, len(dataList), r)
			}
		}()
		batchFn(key, dataList)
	}

	flush := func() {
		for key, dataList := range dataGroups {
			handle(key, dataList)
		}
		clear(dataGroups)
	}

	for {
		select {
		case <-b.doneCh:
			// 停机：先排空分片缓冲中尚未取出的消息（与 Push 并发窗口极小，
			// 调用方应在 Stop 前停止上游投递），再处理残余分组，避免停机丢数据。
			for {
				select {
				case data := <-ch:
					key := keyExtractor(data)
					dataGroups[key] = append(dataGroups[key], data)
					if len(dataGroups[key]) >= b.cfg.batchSize {
						handle(key, dataGroups[key])
						delete(dataGroups, key)
					}
				default:
					flush()
					return
				}
			}
		case data := <-ch:
			// 提取分类键（如会话 ID），将数据添加到对应的分组中
			key := keyExtractor(data)
			dataGroups[key] = append(dataGroups[key], data)

			// 如果当前批次大小达到阈值，立即处理
			if len(dataGroups[key]) >= b.cfg.batchSize {
				handle(key, dataGroups[key])
				delete(dataGroups, key)
			}
		case <-ticker.C:
			// 定时触发批量处理
			if len(dataGroups) > 0 {
				flush()
			}
		}
	}
}

func (b *Batcher[T]) Stop() error {
	// 先关闭入口再停 worker：closed 置位后 Push 无法再进入 channel，
	// 与 worker 的缓冲排空互斥，停机不丢数据；重复 Stop 幂等。
	b.mu.Lock()
	if !b.closed {
		b.closed = true
		close(b.doneCh)
	}
	b.mu.Unlock()

	b.wg.Wait()
	return nil
}
