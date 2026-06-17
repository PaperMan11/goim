package webhooks

import (
	"github.com/zeromicro/go-zero/core/metric"
)

// ========== Counter 指标（累计值）==========

// webhookEventTotal 事件处理总数，按事件类型和状态统计
// Labels:
//   - event_type: 事件类型（如 user.online, message.sent）
//   - status: 处理状态（processed/success/failed）
var webhookEventTotal = metric.NewCounterVec(
	&metric.CounterVecOpts{
		Name:   "webhook_events_total",
		Help:   "Total number of webhook events processed",
		Labels: []string{"event_type", "status"},
	},
)

// webhookEventAbandoned 已放弃事件总数（超过最大重试次数后放弃）
var webhookEventAbandoned = metric.NewCounterVec(
	&metric.CounterVecOpts{
		Name: "webhook_events_abandoned_total",
		Help: "Total number of webhook events abandoned after max retries",
	},
)

// webhookDeliveryTotal 投递总数，按 webhook URL、状态和事件类型统计
// Labels:
//   - webhook_url: webhook 目标 URL
//   - status: 投递状态（total/success/failed/retry_success/retry_failed）
//   - event_type: 事件类型
var webhookDeliveryTotal = metric.NewCounterVec(
	&metric.CounterVecOpts{
		Name:   "webhook_deliveries_total",
		Help:   "Total number of webhook deliveries",
		Labels: []string{"webhook_url", "status", "event_type"},
	},
)

// ========== Gauge 指标（瞬时值）==========

// webhookEventPending 待处理事件数（正在处理中的事件数量）
var webhookEventPending = metric.NewGaugeVec(
	&metric.GaugeVecOpts{
		Name: "webhook_events_pending",
		Help: "Number of webhook events pending processing",
	},
)

// webhookEventRetrying 重试中事件数（等待重试或正在重试的事件数量）
var webhookEventRetrying = metric.NewGaugeVec(
	&metric.GaugeVecOpts{
		Name: "webhook_events_retrying",
		Help: "Number of webhook events being retried",
	},
)

// ========== Histogram 指标（分布统计）==========

// webhookDeliveryDuration 投递耗时（秒），按 webhook URL 和事件类型统计
// Labels:
//   - webhook_url: webhook 目标 URL
//   - event_type: 事件类型
//
// Buckets: 0.001(1ms), 0.005(5ms), 0.01(10ms), 0.05(50ms), 0.1(100ms), 0.5(500ms), 1(1s), 5(5s), 10(10s)
var webhookDeliveryDuration = metric.NewHistogramVec(
	&metric.HistogramVecOpts{
		Name:    "webhook_delivery_duration_seconds",
		Help:    "Duration of webhook delivery requests",
		Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
		Labels:  []string{"webhook_url", "event_type"},
	},
)

// webhookRetryCount 重试次数分布，按 webhook URL 和事件类型统计
// Labels:
//   - webhook_url: webhook 目标 URL
//   - event_type: 事件类型
//
// Buckets: 1, 2, 3, 5, 10
var webhookRetryCount = metric.NewHistogramVec(
	&metric.HistogramVecOpts{
		Name:    "webhook_retry_count",
		Help:    "Number of retries for failed webhook deliveries",
		Buckets: []float64{1, 2, 3, 5, 10},
		Labels:  []string{"webhook_url", "event_type"},
	},
)
