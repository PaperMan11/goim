package batcher

import "time"

func WithWorkerNum(num int) BatcherOption {
	return func(cfg *Config) {
		cfg.workerNum = num
	}
}

func WithBatchSize(size int) BatcherOption {
	return func(cfg *Config) {
		cfg.batchSize = size
	}
}

func WithInterval(interval time.Duration) BatcherOption {
	return func(cfg *Config) {
		cfg.interval = interval
	}
}

func WithDataBufSize(size int) BatcherOption {
	return func(cfg *Config) {
		cfg.dataBufSize = size
	}
}
