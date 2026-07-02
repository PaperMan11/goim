package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"

	"github.com/PaperMan11/goim/pkg/queue"
)

type WrappedMessage struct {
	kafka.Message
	Consumer *kafka.Reader
}

var _ queue.Message = (*WrappedMessage)(nil)

func (m *WrappedMessage) Key() []byte {
	return m.Message.Key
}

func (m *WrappedMessage) Value() []byte {
	return m.Message.Value
}

func (m *WrappedMessage) Header() map[string][]byte {
	headers := make(map[string][]byte)
	for _, header := range m.Message.Headers {
		headers[header.Key] = header.Value
	}
	return headers
}

func (m *WrappedMessage) Commit(ctx context.Context) error {
	return m.Consumer.CommitMessages(ctx, m.Message)
}
