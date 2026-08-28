package metrics

import "github.com/zeromicro/go-zero/core/metric"

var (
	// ========== Producer 指标 ==========

	KafkaProducerMessagesTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Name:   "kafka_producer_messages_total",
			Help:   "Total number of messages produced by Kafka producer",
			Labels: []string{"topic", "status"},
		},
	)

	KafkaProducerErrorsTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Name:   "kafka_producer_errors_total",
			Help:   "Total number of errors occurred during message production",
			Labels: []string{"topic", "error_type"},
		},
	)

	KafkaProducerMessageSizeBytes = metric.NewHistogramVec(
		&metric.HistogramVecOpts{
			Name:    "kafka_producer_message_size_bytes",
			Help:    "Size of messages produced by Kafka producer",
			Buckets: []float64{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576},
			Labels:  []string{"topic"},
		},
	)

	KafkaProducerSendDurationSeconds = metric.NewHistogramVec(
		&metric.HistogramVecOpts{
			Name:    "kafka_producer_send_duration_seconds",
			Help:    "Duration of sending messages to Kafka",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
			Labels:  []string{"topic"},
		},
	)

	// ========== Consumer 指标 ==========

	KafkaConsumerMessagesTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Name:   "kafka_consumer_messages_total",
			Help:   "Total number of messages consumed by Kafka consumer",
			Labels: []string{"topic", "group_id", "status"},
		},
	)

	KafkaConsumerErrorsTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Name:   "kafka_consumer_errors_total",
			Help:   "Total number of errors occurred during message consumption",
			Labels: []string{"topic", "group_id", "error_type"},
		},
	)

	KafkaConsumerCommitErrorsTotal = metric.NewCounterVec(
		&metric.CounterVecOpts{
			Name:   "kafka_consumer_commit_errors_total",
			Help:   "Total number of commit errors occurred during message consumption",
			Labels: []string{"topic", "group_id"},
		},
	)

	KafkaConsumerMessageQueueSize = metric.NewGaugeVec(
		&metric.GaugeVecOpts{
			Name:   "kafka_consumer_message_queue_size",
			Help:   "Number of messages waiting in the consumer queue",
			Labels: []string{"topic", "group_id"},
		},
	)

	KafkaConsumerMessageSizeBytes = metric.NewHistogramVec(
		&metric.HistogramVecOpts{
			Name:    "kafka_consumer_message_size_bytes",
			Help:    "Size of messages consumed by Kafka consumer",
			Buckets: []float64{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576},
			Labels:  []string{"topic", "group_id"},
		},
	)

	KafkaConsumerProcessDurationSeconds = metric.NewHistogramVec(
		&metric.HistogramVecOpts{
			Name:    "kafka_consumer_process_duration_seconds",
			Help:    "Duration of processing consumed messages",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
			Labels:  []string{"topic", "group_id"},
		},
	)

	KafkaConsumerFetchDurationSeconds = metric.NewHistogramVec(
		&metric.HistogramVecOpts{
			Name:    "kafka_consumer_fetch_duration_seconds",
			Help:    "Duration of fetching messages from Kafka",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
			Labels:  []string{"topic", "group_id"},
		},
	)
)
