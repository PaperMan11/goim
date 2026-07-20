package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"github.com/PaperMan11/goim/pkg/queue"
	"github.com/PaperMan11/goim/pkg/utils/timex"
)

var (
	defaultCommitInterval = time.Duration(0)
	defaultQueueCapacity  = 100
	defaultMaxWait        = 10 * time.Second
)

type KafkaConsumer struct {
	cfg          KafkaConfig
	consumer     *kafka.Reader
	handler      queue.ConsumeHandler
	ctx          context.Context
	cancel       context.CancelFunc
	msgChan      chan queue.Message
	producerWg   sync.WaitGroup
	consumerWg   sync.WaitGroup
	running      bool
	mu           sync.Mutex
	errorHandler ConsumeErrorHandler
	options      ConsumerOptions
	commitRunner *threading.StableRunner[queue.Message, queue.Message]
}

type ConsumeErrorHandler func(ctx context.Context, msg queue.Message, err error)

type ConsumerOptions struct {
	commitInterval time.Duration
	queueCapacity  int
	maxWait        time.Duration
	errorHandler   ConsumeErrorHandler
	manualCommit   bool
}

type ConsumerOption func(*ConsumerOptions)

var defaultErrorHandler = func(ctx context.Context, msg queue.Message, err error) {
	logx.Errorf("KafkaConsumer: %v, msg key: %s, value: %s", err, msg.Key(), msg.Value())
}

func MustNewConsumer(cfg KafkaConfig, opts ...ConsumerOption) *KafkaConsumer {
	consumer, err := NewConsumer(cfg, opts...)
	if err != nil {
		logx.Must(err)
	}
	return consumer
}

func NewConsumer(cfg KafkaConfig, opts ...ConsumerOption) (*KafkaConsumer, error) {
	var offset int64
	if cfg.Offset == OffsetFirst {
		offset = kafka.FirstOffset
	} else {
		offset = kafka.LastOffset
	}

	var options ConsumerOptions
	for _, opt := range opts {
		opt(&options)
	}

	if options.commitInterval == 0 {
		options.commitInterval = defaultCommitInterval
	}

	if options.queueCapacity == 0 {
		options.queueCapacity = defaultQueueCapacity
	}

	if options.maxWait == 0 {
		options.maxWait = defaultMaxWait
	}

	if options.errorHandler == nil {
		options.errorHandler = defaultErrorHandler
	}

	readerConfig := kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		Topic:          cfg.Topic,
		StartOffset:    offset,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		MaxWait:        options.maxWait,
		CommitInterval: options.commitInterval,
		QueueCapacity:  options.queueCapacity,
	}

	if len(cfg.Username) > 0 && len(cfg.Password) > 0 {
		readerConfig.Dialer = &kafka.Dialer{
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

		if readerConfig.Dialer == nil {
			readerConfig.Dialer = &kafka.Dialer{}
		}
		readerConfig.Dialer.TLS = &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: cfg.InsecureTLS,
		}
	}

	consumer := kafka.NewReader(readerConfig)
	ctx, cancel := context.WithCancel(context.Background())

	c := &KafkaConsumer{
		cfg:      cfg,
		consumer: consumer,
		ctx:      ctx,
		cancel:   cancel,
		msgChan:  make(chan queue.Message, options.queueCapacity),
		options:  options,
	}

	if cfg.OrderCommit {
		c.commitRunner = threading.NewStableRunner[queue.Message, queue.Message](func(msg queue.Message) queue.Message {
			err := c.consumerOne(c.ctx, msg)
			if err != nil && c.errorHandler != nil {
				c.errorHandler(c.ctx, msg, err)
			}
			return msg
		})
	}

	return c, nil
}

func (c *KafkaConsumer) Subscribe(handler queue.ConsumeHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.handler != nil {
		return ErrHandlerAlreadySubscribed
	}
	c.handler = handler
	return nil
}

func (c *KafkaConsumer) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return ErrConsumerAlreadyRunning
	}
	if c.handler == nil {
		c.mu.Unlock()
		return ErrNoHandlerSubscribed
	}
	c.running = true
	c.mu.Unlock()

	logx.Infof("Kafka consumer starting, topic: %s, group: %s", c.cfg.Topic, c.cfg.GroupID)

	if c.cfg.OrderCommit {
		c.producerWg.Add(1)
		go c.fetchLoop(func(msg queue.Message) {
			err := c.commitRunner.Push(msg)
			if err != nil {
				logx.Errorf("Error pushing message to commit runner: %v", err)
			}
		})

		c.consumerWg.Add(1)
		go c.commitInOrder()
	} else {
		processorCount := c.cfg.Processors
		if processorCount <= 0 {
			processorCount = 1
		}

		for i := 0; i < processorCount; i++ {
			c.consumerWg.Add(1)
			go c.consumeLoop()
		}

		c.producerWg.Add(1)
		for i := 0; i < c.cfg.Consumers; i++ {
			c.producerWg.Add(1)
			go c.fetchLoop(func(msg queue.Message) {
				select {
				case <-c.ctx.Done():
					return
				case c.msgChan <- msg:
				}
			})
		}
	}
	return nil
}

func (c *KafkaConsumer) fetchLoop(handle func(msg queue.Message)) {
	defer c.producerWg.Done()

	for {
		select {
		case <-c.ctx.Done():
			logx.Info("Fetch loop stopped")
			return
		default:
			start := timex.Now()
			msg, err := c.consumer.FetchMessage(c.ctx)
			duration := time.Since(start).Seconds()

			kafkaConsumerFetchDurationSeconds.ObserveFloat(duration, c.cfg.Topic, c.cfg.GroupID)

			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
					logx.Info("Consumer closed")
					return
				}
				if errors.Is(err, context.Canceled) {
					logx.Info("Consumer context canceled")
					return
				}
				logx.Errorf("Error on reading message: %v", err)
				kafkaConsumerErrorsTotal.Inc(c.cfg.Topic, c.cfg.GroupID, "fetch_error")
				time.Sleep(time.Second)
				continue
			}

			wrappedMessage := &WrappedMessage{
				Message:  msg,
				Consumer: c.consumer,
			}
			handle(wrappedMessage)
		}
	}
}

func (c *KafkaConsumer) consumeLoop() {
	defer c.consumerWg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.msgChan:
			kafkaConsumerMessageQueueSize.Set(float64(len(c.msgChan)), c.cfg.Topic, c.cfg.GroupID)

			err := c.consumerOne(c.ctx, msg)
			if err != nil {
				if c.errorHandler != nil {
					c.errorHandler(c.ctx, msg, err)
				}
				logx.Errorf("Error handling message: %v", err)
				if !c.cfg.ForceCommit {
					continue
				}
			}

			if !c.options.manualCommit {
				if err := msg.Commit(c.ctx); err != nil {
					kafkaConsumerCommitErrorsTotal.Inc(c.cfg.Topic, c.cfg.GroupID)
					logx.Errorf("Commit failed: %v", err)
				}
			}
		}
	}
}

func (c *KafkaConsumer) commitInOrder() {
	defer c.consumerWg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			msg, err := c.commitRunner.Get()
			if err != nil {
				logx.Error(err)
				return
			}

			if err := msg.Commit(c.ctx); err != nil {
				kafkaConsumerCommitErrorsTotal.Inc(c.cfg.Topic, c.cfg.GroupID)
				logx.Errorf("commit failed, message: %v, error: %v", msg, err)
			}
		}
	}
}

func (c *KafkaConsumer) consumerOne(ctx context.Context, message queue.Message) error {
	msgSize := float64(len(message.Value()))
	kafkaConsumerMessageSizeBytes.ObserveFloat(msgSize, c.cfg.Topic, c.cfg.GroupID)

	start := timex.Now()
	err := c.handler(ctx, message)
	duration := time.Since(start).Seconds()

	kafkaConsumerProcessDurationSeconds.ObserveFloat(duration, c.cfg.Topic, c.cfg.GroupID)

	if err != nil {
		kafkaConsumerMessagesTotal.Inc(c.cfg.Topic, c.cfg.GroupID, "failed")
		kafkaConsumerErrorsTotal.Inc(c.cfg.Topic, c.cfg.GroupID, "process_error")
	} else {
		kafkaConsumerMessagesTotal.Inc(c.cfg.Topic, c.cfg.GroupID, "success")
	}
	return err
}

func (c *KafkaConsumer) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = false
	c.mu.Unlock()

	logx.Infof("Stopping Kafka consumer, topic: %s", c.cfg.Topic)

	c.cancel()
	c.producerWg.Wait()
	close(c.msgChan)
	c.consumerWg.Wait()

	c.commitRunner.Wait()

	return c.consumer.Close()
}

func (c *KafkaConsumer) Name() string {
	return c.cfg.Topic
}

func WithCommitInterval(commitInterval time.Duration) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.commitInterval = commitInterval
	}
}

func WithQueueCapacity(queueCapacity int) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.queueCapacity = queueCapacity
	}
}

func WithMaxWait(maxWait time.Duration) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.maxWait = maxWait
	}
}

func WithErrorHandler(errorHandler ConsumeErrorHandler) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.errorHandler = errorHandler
	}
}

func WithManualCommit(manualCommit bool) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.manualCommit = manualCommit
	}
}
