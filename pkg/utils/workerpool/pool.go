package workerpool

import (
	"sync"
)

const (
	DefaultWorkerCount = 4
	DefaultBufferSize  = 1024
)

type WorkerPool[T any] struct {
	taskCh      chan T
	handler     func(T)
	workerCount int
	done        chan struct{}
	wg          sync.WaitGroup
}

func New[T any](handler func(T), workerCount int, bufferSize int) *WorkerPool[T] {
	if workerCount <= 0 {
		workerCount = DefaultWorkerCount
	}
	if bufferSize <= 0 {
		bufferSize = DefaultBufferSize
	}
	return &WorkerPool[T]{
		taskCh:      make(chan T, bufferSize),
		handler:     handler,
		workerCount: workerCount,
		done:        make(chan struct{}),
	}
}

func (wp *WorkerPool[T]) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

func (wp *WorkerPool[T]) Stop() {
	close(wp.done)
	close(wp.taskCh)
	wp.wg.Wait()
}

func (wp *WorkerPool[T]) worker() {
	defer wp.wg.Done()
	for {
		select {
		case <-wp.done:
			return
		case task, ok := <-wp.taskCh:
			if !ok {
				return
			}
			wp.handler(task)
		}
	}
}

func (wp *WorkerPool[T]) WorkerCount() int {
	return wp.workerCount
}

func (wp *WorkerPool[T]) BufferSize() int {
	return cap(wp.taskCh)
}

func (wp *WorkerPool[T]) Submit(task T) {
	wp.taskCh <- task
}

func (wp *WorkerPool[T]) TrySubmit(task T) bool {
	select {
	case wp.taskCh <- task:
		return true
	default:
		return false
	}
}
