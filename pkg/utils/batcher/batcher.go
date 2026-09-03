package batcher

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	DefaultWorkerNum   = 10              // 默认工作线程数
	DefaultBatchSize   = 100             // 默认批次大小
	DefaultInterval    = 3 * time.Second // 默认间隔时间
	DefaultDataBufSize = 1000            // 默认数据缓冲区大小
)

var (
	ErrorBatcherClosed = errors.New("batcher is closed")
	ErrorBatcherFull   = errors.New("batcher is full")
	ErrorBatcherEmpty  = errors.New("batcher is empty")
)

type Config struct {
	workerNum   int           // 工作线程数
	batchSize   int           // 批次大小
	interval    time.Duration // 单位：秒
	dataBufSize int           // 数据缓冲区大小
}

type BatcherOption func(*Config)

// ResetFunc 用于重置对象状态的函数
type ResetFunc[T any] func(T)

type Batcher[T any] struct {
	cfg       Config         // 配置
	dataCh    chan T         // 数据通道（改为指针类型）
	doneCh    chan struct{}  // 完成通道
	wg        sync.WaitGroup // 等待组，用于等待所有工作线程完成
	pool      sync.Pool      // 对象池
	resetFunc ResetFunc[T]   // 对象重置函数
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
		dataCh: make(chan T, cfg.dataBufSize),
		doneCh: make(chan struct{}, 1),
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

// Get 从对象池获取获取一个对象
func (b *Batcher[T]) Get() T {
	return b.pool.Get().(T)
}

// Put 将对象放回对象池
func (b *Batcher[T]) Put(data T) {
	if b.resetFunc != nil {
		b.resetFunc(data)
	}
	b.pool.Put(data)
}

func (b *Batcher[T]) Push(data T) error {
	select {
	case <-b.doneCh:
		return ErrorBatcherClosed
	case b.dataCh <- data:
		return nil
	}
}

func (b *Batcher[T]) PushImmediately(data T) error {
	select {
	case <-b.doneCh:
		return ErrorBatcherClosed
	case b.dataCh <- data:
		return nil
	default:
		return ErrorBatcherFull
	}
}

func (b *Batcher[T]) Start(keyExtractor func(T) string, batchFn func(key string, dataList []T)) error {
	b.wg.Add(b.cfg.workerNum)
	for i := 0; i < b.cfg.workerNum; i++ {
		go func(j int) {
			defer func() {
				b.wg.Done()
				r := recover()
				if r != nil {
					fmt.Printf("工作线程 %d panic: %v\n", j, r)
				}
			}()

			b.worker(keyExtractor, batchFn)
		}(i)
	}
	return nil
}

func (b *Batcher[T]) worker(keyExtractor func(T) string, batchFn func(key string, dataList []T)) {
	ticker := time.NewTicker(b.cfg.interval)
	defer ticker.Stop()

	// 按分类键分组存储数据，key为分类键（如房间ID），value为该组的数据列表
	dataGroups := make(map[string][]T)

	for {
		select {
		case <-b.doneCh:
			// 如果收到停止信号，先处理剩余数据
			if len(dataGroups) > 0 {
				for key, dataList := range dataGroups {
					batchFn(key, dataList)
				}
				// 处理完成后将对象放回对象池
				for _, items := range dataGroups {
					for _, item := range items {
						b.Put(item)
					}
				}
			}
			return
		case data := <-b.dataCh:
			// 提取分类键（如房间ID）
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
				clear(dataGroups)
			}
		case <-ticker.C:
			// 定时触发批量处理
			if len(dataGroups) > 0 {
				for key, dataList := range dataGroups {
					batchFn(key, dataList)
				}
				// 处理完成后将对象放回对象池
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
