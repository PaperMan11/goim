package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"github.com/PaperMan11/goim/pkg/metrics"
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
	shardChans   []chan queue.Message
	producerWg   sync.WaitGroup
	// 并发消费下提交位点必须按分区连续推进，见 commitMessage
	commitMu    sync.Mutex
	commitState map[int]*partitionCommitState
	// commitFn 位点提交动作，测试可注入；默认 reader.CommitMessages
	commitFn func(ctx context.Context, msgs ...kafka.Message) error
	consumerWg   sync.WaitGroup
	running      bool
	mu           sync.Mutex
	errorHandler ConsumeErrorHandler
	options      ConsumerOptions
	commitRunner *threading.StableRunner[queue.Message, queue.Message]
}

// partitionCommitState 维护单个分区的连续提交进度。
type partitionCommitState struct {
	// next 下一个待提交的连续位置（即下一条待消费的位点）
	next int64
	// done 已处理完成、但尚未纳入连续提交范围的位点集合
	done map[int64]struct{}
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
		commitFn: consumer.CommitMessages,
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

	logx.Infof("Kafka consumer starting, topic: %s, group: %s, mode: %s",
		c.cfg.Topic, c.cfg.GroupID, c.modeName())

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
	} else if c.cfg.KeySharding {
		// key 分发模式：同 key 消息固定路由到同一 consumeLoop 串行处理
		c.startSharded()
	} else {
		processorCount := c.cfg.Processors
		if processorCount <= 0 {
			processorCount = 1
		}

		for i := 0; i < processorCount; i++ {
			c.consumerWg.Add(1)
			go c.consumeLoop()
		}

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

// modeName 返回当前消费模式的可读名称，用于启动日志。
func (c *KafkaConsumer) modeName() string {
	switch {
	case c.cfg.OrderCommit:
		return "order-commit (parallel process, ordered commit)"
	case c.cfg.KeySharding:
		shardNum := c.cfg.Processors
		if shardNum <= 0 {
			shardNum = 1
		}
		return fmt.Sprintf("key-sharding (consumers: %d, shards: %d)", c.cfg.Consumers, shardNum)
	default:
		return fmt.Sprintf("shared-channel (consumers: %d, processors: %d)", c.cfg.Consumers, c.cfg.Processors)
	}
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

			metrics.KafkaConsumerFetchDurationSeconds.ObserveFloat(duration, c.cfg.Topic, c.cfg.GroupID)

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
				metrics.KafkaConsumerErrorsTotal.Inc(c.cfg.Topic, c.cfg.GroupID, "fetch_error")
				time.Sleep(time.Second)
				continue
			}

			wrappedMessage := &WrappedMessage{
				Message:  msg,
				Consumer: c.consumer,
			}
			c.trackFetched(wrappedMessage)
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
			metrics.KafkaConsumerMessageQueueSize.Set(float64(len(c.msgChan)), c.cfg.Topic, c.cfg.GroupID)

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
				c.commitMessage(c.ctx, msg)
			}
		}
	}
}

// startSharded 启动 key 分发模式：fetchLoop 按 msg key hash 分发到固定分片，
// 每个分片由独立的 consumeLoop 串行消费，保证同 key（如同会话）消息严格有序。
// 分片数取 Processors，总并发度与共享 channel 模式一致。
func (c *KafkaConsumer) startSharded() {
	shardNum := c.cfg.Processors
	if shardNum <= 0 {
		shardNum = 1
	}

	c.shardChans = make([]chan queue.Message, shardNum)
	for i := range c.shardChans {
		c.shardChans[i] = make(chan queue.Message, c.options.queueCapacity)
		c.consumerWg.Add(1)
		go c.shardConsumeLoop(c.shardChans[i])
	}

	for i := 0; i < c.cfg.Consumers; i++ {
		c.producerWg.Add(1)
		go c.fetchLoop(c.dispatchToShard)
	}
}

func (c *KafkaConsumer) dispatchToShard(msg queue.Message) {
	select {
	case <-c.ctx.Done():
		return
	case c.shardChans[c.shardIndex(msg)] <- msg:
	}
}

func (c *KafkaConsumer) shardIndex(msg queue.Message) int {
	key := msg.Key()
	if len(key) == 0 {
		// 无 key 消息按内容散列，避免全部堆积到 0 号分片
		key = msg.Value()
	}
	h := fnv.New32a()
	h.Write(key)
	return int(h.Sum32()) % len(c.shardChans)
}

func (c *KafkaConsumer) shardConsumeLoop(ch <-chan queue.Message) {
	defer c.consumerWg.Done()

	// 退出由 Stop 关闭分片通道驱动：fetchLoop 已先停止，range 会先处理完
	// 缓冲中的剩余消息再退出，保证停机时不丢已拉取的消息。
	for msg := range ch {
		var total int
		for _, shard := range c.shardChans {
			total += len(shard)
		}
		metrics.KafkaConsumerMessageQueueSize.Set(float64(total), c.cfg.Topic, c.cfg.GroupID)

		// 停机 drain 阶段 c.ctx 已取消，使用独立 ctx 保证剩余消息的 RPC 与 commit 仍可执行
		ctx := context.Background()
		err := c.consumerOne(ctx, msg)
		if err != nil {
			if c.errorHandler != nil {
				c.errorHandler(ctx, msg, err)
			}
			logx.Errorf("Error handling message: %v", err)
			if !c.cfg.ForceCommit {
				continue
			}
		}

		if !c.options.manualCommit {
			c.commitMessage(ctx, msg)
		}
	}
}

// trackFetched 在 fetch 时落定分区的提交基准位点：同分区 fetch 严格有序，
// 该分区首个 fetch 到的消息位点即为本进程的最小未提交位点。基准必须在 fetch
// 时创建而非首次提交时创建，否则若高位点先处理完（分片并发下常见），会以它
// 为基准越过尚未处理的消息提交位点，造成消息丢失。
func (c *KafkaConsumer) trackFetched(msg *WrappedMessage) {
	c.commitMu.Lock()
	defer c.commitMu.Unlock()

	if c.commitState == nil {
		c.commitState = make(map[int]*partitionCommitState)
	}
	if _, ok := c.commitState[msg.Partition]; !ok {
		c.commitState[msg.Partition] = &partitionCommitState{
			next: msg.Offset,
			done: make(map[int64]struct{}),
		}
	}
}

// commitMessage 提交消息位点。
//
// key 分发/共享 channel 模式下多条消息并发处理，完成顺序与 fetch 顺序不一致，
// 若各自直接提交 msg.Offset+1，后完成的小位点会覆盖先提交的大位点（broker 保存
// 最后写入值），位点回退导致重启后重复消费已处理的消息。因此按分区维护连续
// 处理进度，仅当位点连续推进时提交最大连续位置：提交值单调递增，且不会越过
// 尚未处理完成的消息。
//
// kafka-go 同步提交内部仅重试 3 次（100ms~5s 退避），rebalance 窗口内（重启/
// 发布时同组实例交替）可能全部失败，这里在应用层再做一轮退避重试，直到成功
// 或 reader 关闭。提交成功打 debug 日志，失败打 error 日志，便于核对位点。
func (c *KafkaConsumer) commitMessage(ctx context.Context, msg queue.Message) {
	const attempts = 5

	wm, ok := msg.(*WrappedMessage)
	if !ok {
		// 非标准消息类型，退化为直接提交自身位点
		c.commitWithRetry(ctx, attempts, func() error { return msg.Commit(ctx) }, -1, -1)
		return
	}

	c.commitMu.Lock()
	defer c.commitMu.Unlock()

	if c.commitState == nil {
		c.commitState = make(map[int]*partitionCommitState)
	}
	st := c.commitState[wm.Partition]
	if st == nil {
		// 正常已由 trackFetched 在 fetch 时创建，此处为兜底
		st = &partitionCommitState{next: wm.Offset, done: map[int64]struct{}{}}
		c.commitState[wm.Partition] = st
	}
	st.done[wm.Offset] = struct{}{}

	next := st.next
	for {
		if _, ok := st.done[next]; !ok {
			break
		}
		delete(st.done, next)
		next++
	}
	if next == st.next {
		// 连续进度未推进（后续位点尚未处理完成）：暂不提交，避免位点回退
		logx.Debugf("Kafka consumer commit skipped, waiting for contiguous progress, topic: %s, group: %s, partition: %d, offset: %d",
			c.cfg.Topic, c.cfg.GroupID, wm.Partition, wm.Offset)
		return
	}
	st.next = next

	target := wm.Message
	target.Offset = next - 1
	c.commitWithRetry(ctx, attempts, func() error {
		return c.commitFn(ctx, target)
	}, wm.Partition, next)
}

// commitWithRetry 对提交动作做应用层退避重试；partition/position 仅用于日志。
func (c *KafkaConsumer) commitWithRetry(ctx context.Context, attempts int, commit func() error, partition int, position int64) {
	for attempt := 1; attempt <= attempts; attempt++ {
		err := commit()
		if err == nil {
			logx.Debugf("Kafka consumer committed, topic: %s, group: %s, partition: %d, position: %d",
				c.cfg.Topic, c.cfg.GroupID, partition, position)
			return
		}
		if errors.Is(err, io.ErrClosedPipe) || errors.Is(err, context.Canceled) {
			// reader 已关闭或 ctx 取消：位点停在原地，重启后按未提交位置重新消费
			logx.Errorf("Kafka consumer commit aborted, topic: %s, group: %s, partition: %d, position: %d, err: %v",
				c.cfg.Topic, c.cfg.GroupID, partition, position, err)
			return
		}
		metrics.KafkaConsumerCommitErrorsTotal.Inc(c.cfg.Topic, c.cfg.GroupID)
		logx.Errorf("Kafka consumer commit failed (attempt %d/%d), topic: %s, group: %s, partition: %d, position: %d, err: %v",
			attempt, attempts, c.cfg.Topic, c.cfg.GroupID, partition, position, err)
		if attempt == attempts {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
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
				metrics.KafkaConsumerCommitErrorsTotal.Inc(c.cfg.Topic, c.cfg.GroupID)
				logx.Errorf("commit failed, message: %v, error: %v", msg, err)
			}
		}
	}
}

func (c *KafkaConsumer) consumerOne(ctx context.Context, message queue.Message) error {
	msgSize := float64(len(message.Value()))
	metrics.KafkaConsumerMessageSizeBytes.ObserveFloat(msgSize, c.cfg.Topic, c.cfg.GroupID)

	start := timex.Now()
	err := c.handler(ctx, message)
	duration := time.Since(start).Seconds()

	metrics.KafkaConsumerProcessDurationSeconds.ObserveFloat(duration, c.cfg.Topic, c.cfg.GroupID)

	if err != nil {
		metrics.KafkaConsumerMessagesTotal.Inc(c.cfg.Topic, c.cfg.GroupID, "failed")
		metrics.KafkaConsumerErrorsTotal.Inc(c.cfg.Topic, c.cfg.GroupID, "process_error")
	} else {
		metrics.KafkaConsumerMessagesTotal.Inc(c.cfg.Topic, c.cfg.GroupID, "success")
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
	if c.cfg.KeySharding {
		// fetchLoop 已停止，关闭分片通道让各 shardConsumeLoop 处理完剩余消息后退出
		for _, ch := range c.shardChans {
			close(ch)
		}
	} else {
		close(c.msgChan)
	}
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
