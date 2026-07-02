package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/PaperMan11/goim/pkg/queue"
)

type KafkaProducer struct {
	topic    string
	producer *kafka.Writer
}

type ProducerOptions struct {
	// kafka.Writer options
	allowAutoTopicCreation bool
	balancer               kafka.Balancer

	// syncPush is used to enable sync push
	syncPush bool
}

type ProducerOption func(*ProducerOptions)

func MustNewProducer(cfg KafkaConfig, opts ...ProducerOption) *KafkaProducer {
	producer, err := NewProducer(cfg, opts...)
	if err != nil {
		logx.Must(err)
	}
	return producer
}

func NewProducer(cfg KafkaConfig, opts ...ProducerOption) (*KafkaProducer, error) {
	writerConfig := kafka.WriterConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	if len(cfg.Username) > 0 && len(cfg.Password) > 0 {
		writerConfig.Dialer = &kafka.Dialer{
			SASLMechanism: plain.Mechanism{
				Username: cfg.Username,
				Password: cfg.Password,
			},
		}
	}

	if len(cfg.CaFile) > 0 {
		caCert, err := os.ReadFile(cfg.CaFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrReadCAFile, err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("%w: invalid PEM format", ErrAppendCertToPEM)
		}

		if writerConfig.Dialer == nil {
			writerConfig.Dialer = &kafka.Dialer{}
		}
		writerConfig.Dialer.TLS = &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: cfg.InsecureTLS,
		}
	}

	producer := kafka.NewWriter(writerConfig)

	var options ProducerOptions
	for _, opt := range opts {
		opt(&options)
	}

	if options.balancer != nil {
		producer.Balancer = options.balancer
	}
	if options.syncPush {
		producer.BatchSize = 1
	}
	producer.AllowAutoTopicCreation = options.allowAutoTopicCreation

	return &KafkaProducer{
		topic:    cfg.Topic,
		producer: producer,
	}, nil
}

func (p *KafkaProducer) Push(ctx context.Context, value string) error {
	return p.PushWithKey(ctx, strconv.FormatInt(time.Now().UnixNano(), 10), value)
}

func (p *KafkaProducer) PushWithKey(ctx context.Context, key, value string) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
	}
	return p.pushMessage(ctx, msg)
}

func (p *KafkaProducer) pushMessage(ctx context.Context, msg kafka.Message) error {
	start := time.Now()
	msgSize := float64(len(msg.Value))

	err := p.producer.WriteMessages(ctx, msg)
	duration := time.Since(start).Seconds()

	kafkaProducerMessageSizeBytes.ObserveFloat(msgSize, p.topic)
	kafkaProducerSendDurationSeconds.ObserveFloat(duration, p.topic)

	if err != nil {
		kafkaProducerMessagesTotal.Inc(p.topic, "failed")
		kafkaProducerErrorsTotal.Inc(p.topic, "send_error")
		return err
	}

	kafkaProducerMessagesTotal.Inc(p.topic, "success")
	return nil
}

func (p *KafkaProducer) PushMessage(ctx context.Context, msg queue.Message) error {
	kafkaHeaders := make([]kafka.Header, 0, len(msg.Header()))
	for k, v := range msg.Header() {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   k,
			Value: v,
		})
	}

	kafkaMsg := kafka.Message{
		Key:     msg.Key(),
		Value:   msg.Value(),
		Headers: kafkaHeaders,
	}
	return p.pushMessage(ctx, kafkaMsg)
}

func (p *KafkaProducer) Close() error {
	return p.producer.Close()
}

func (p *KafkaProducer) Name() string {
	return p.topic
}

func WithAllowAutoTopicCreation(allowAutoTopicCreation bool) ProducerOption {
	return func(o *ProducerOptions) {
		o.allowAutoTopicCreation = allowAutoTopicCreation
	}
}

func WithBalancer(balancer kafka.Balancer) ProducerOption {
	return func(o *ProducerOptions) {
		o.balancer = balancer
	}
}

func WithSyncPush(syncPush bool) ProducerOption {
	return func(o *ProducerOptions) {
		o.syncPush = syncPush
	}
}
