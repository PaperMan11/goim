package queue

import (
	"context"
)

type Message interface {
	Key() []byte
	Value() []byte
	Header() map[string][]byte
	Commit(ctx context.Context) error // 手动提交消息
}

type Producer interface {
	Push(ctx context.Context, value string) error
	PushWithKey(ctx context.Context, key, value string) error
	// PushMessage(ctx context.Context, msg Message) error
	Close() error
	Name() string
}

type ConsumeHandler func(ctx context.Context, msg Message) error

type Consumer interface {
	Subscribe(handler ConsumeHandler) error
	Start() error
	Stop() error
	Name() string
}

type MessageQueue interface {
	Producer
	Consumer
}
