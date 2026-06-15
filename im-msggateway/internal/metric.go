package internal

import "github.com/zeromicro/go-zero/core/metric"

// ========== Gauge 指标（瞬时值）==========

// 活跃连接数
var activeConnGauge = metric.NewGaugeVec(&metric.GaugeVecOpts{
	Name: "websocket_active_connections",
	Help: "Number of active WebSocket connections",
})

// ========== Counter 指标（累计值）==========

// 总连接数
var totalConnCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name: "websocket_connections_total",
	Help: "Total number of WebSocket connections established",
})

// 连接断开数
var connCloseCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name:   "websocket_disconnections_total",
	Help:   "Total number of WebSocket disconnections",
	Labels: []string{"reason"},
})

// 消息接收数
var msgReceivedCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name:   "websocket_messages_received_total",
	Help:   "Total number of messages received from clients",
	Labels: []string{"message_type"},
})

// 消息发送数
var msgSentCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name:   "websocket_messages_sent_total",
	Help:   "Total number of messages sent to clients",
	Labels: []string{"message_type"},
})

// 消息处理错误数
var msgHandleErrorCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name:   "websocket_message_errors_total",
	Help:   "Total number of message handling errors",
	Labels: []string{"error_type"},
})

// 认证失败数
var authFailedCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name:   "websocket_auth_failed_total",
	Help:   "Total number of authentication failures",
	Labels: []string{"reason"},
})

// 限流拒绝数
var rateLimitCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name: "websocket_rate_limited_total",
	Help: "Total number of rate limited requests",
})

// ========== 业务操作指标 ==========

// 业务操作计数
var businessOpCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name:   "websocket_business_ops_total",
	Help:   "Total number of business operations",
	Labels: []string{"operation"},
})

// 业务操作错误计数
var businessOpErrorCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name:   "websocket_business_errors_total",
	Help:   "Total number of business operation errors",
	Labels: []string{"operation"},
})

// 消息发送操作计数（业务层）
var sendMsgCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name:   "websocket_send_msg_total",
	Help:   "Total number of messages sent",
	Labels: []string{"msg_type"},
})

// 消息拉取操作计数
var pullMsgCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name:   "websocket_pull_msg_total",
	Help:   "Total number of message pull operations",
	Labels: []string{"pull_type"},
})

// 用户在线状态订阅计数
var subscribeCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Name:   "websocket_subscribe_total",
	Help:   "Total number of subscription operations",
	Labels: []string{"type"},
})

// ========== Histogram 指标（分布统计）==========

// 消息处理耗时（毫秒）
var msgHandleDurationHistogram = metric.NewHistogramVec(&metric.HistogramVecOpts{
	Name:    "websocket_message_handle_duration_ms",
	Help:    "Duration of message handling in milliseconds",
	Buckets: []float64{1, 5, 10, 50, 100, 500, 1000},
})

// 消息大小分布（字节）
var msgSizeHistogram = metric.NewHistogramVec(&metric.HistogramVecOpts{
	Name:    "websocket_message_size_bytes",
	Help:    "Size of messages in bytes",
	Buckets: []float64{64, 256, 1024, 4096, 16384, 65536},
})

// 业务操作耗时（毫秒）
var businessOpDurationHistogram = metric.NewHistogramVec(&metric.HistogramVecOpts{
	Name:    "websocket_business_op_duration_ms",
	Help:    "Duration of business operations in milliseconds",
	Buckets: []float64{1, 5, 10, 50, 100, 500, 1000},
	Labels:  []string{"operation"},
})
