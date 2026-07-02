package kafka

import "github.com/zeromicro/go-zero/core/metric"

var (
	// ========== Producer 指标 ==========

	kafkaProducerMessagesTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Name:   "kafka_producer_messages_total",
			Help:   "Total number of messages produced by Kafka producer",
			Labels: []string{"topic", "status"},
		},
	)

	kafkaProducerErrorsTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Name:   "kafka_producer_errors_total",
			Help:   "Total number of errors occurred during message production",
			Labels: []string{"topic", "error_type"},
		},
	)

	kafkaProducerMessageSizeBytes = metric.NewHistogramVec(
		&metric.HistogramVecOpts{
			Name:    "kafka_producer_message_size_bytes",
			Help:    "Size of messages produced by Kafka producer",
			Buckets: []float64{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576},
			Labels:  []string{"topic"},
		},
	)

	kafkaProducerSendDurationSeconds = metric.NewHistogramVec(
		&metric.HistogramVecOpts{
			Name:    "kafka_producer_send_duration_seconds",
			Help:    "Duration of sending messages to Kafka",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
			Labels:  []string{"topic"},
		},
	)

	// ========== Consumer 指标 ==========

	kafkaConsumerMessagesTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Name:   "kafka_consumer_messages_total",
			Help:   "Total number of messages consumed by Kafka consumer",
			Labels: []string{"topic", "group_id", "status"},
		},
	)

	kafkaConsumerErrorsTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Name:   "kafka_consumer_errors_total",
			Help:   "Total number of errors occurred during message consumption",
			Labels: []string{"topic", "group_id", "error_type"},
		},
	)

	kafkaConsumerCommitErrorsTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Name:   "kafka_consumer_commit_errors_total",
			Help:   "Total number of commit errors occurred during message consumption",
			Labels: []string{"topic", "group_id"},
		},
	)

	kafkaConsumerMessageQueueSize = metric.NewGaugeVec(
		&metric.GaugeVecOpts{
			Name:   "kafka_consumer_message_queue_size",
			Help:   "Number of messages waiting in the consumer queue",
			Labels: []string{"topic", "group_id"},
		},
	)

	kafkaConsumerMessageSizeBytes = metric.NewHistogramVec(
		&metric.HistogramVecOpts{
			Name:    "kafka_consumer_message_size_bytes",
			Help:    "Size of messages consumed by Kafka consumer",
			Buckets: []float64{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576},
			Labels:  []string{"topic", "group_id"},
		},
	)

	kafkaConsumerProcessDurationSeconds = metric.NewHistogramVec(
		&metric.HistogramVecOpts{
			Name:    "kafka_consumer_process_duration_seconds",
			Help:    "Duration of processing consumed messages",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
			Labels:  []string{"topic", "group_id"},
		},
	)

	kafkaConsumerFetchDurationSeconds = metric.NewHistogramVec(
		&metric.HistogramVecOpts{
			Name:    "kafka_consumer_fetch_duration_seconds",
			Help:    "Duration of fetching messages from Kafka",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
			Labels:  []string{"topic", "group_id"},
		},
	)
)