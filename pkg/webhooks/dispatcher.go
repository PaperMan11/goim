package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

// Dispatcher webhook 事件分发器
type Dispatcher struct {
	configManager *ConfigManager
	deliveryRepo  DeliveryRepository
	retryManager  *RetryManager
	eventQueue    chan *WebhookEvent
	workerCount   int
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewDispatcher 创建事件分发器
func NewDispatcher(
	configManager *ConfigManager,
	deliveryRepo DeliveryRepository,
	retryManager *RetryManager,
	workerCount int,
) *Dispatcher {
	return &Dispatcher{
		configManager: configManager,
		deliveryRepo:  deliveryRepo,
		retryManager:  retryManager,
		eventQueue:    make(chan *WebhookEvent, 10000),
		workerCount:   workerCount,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动分发器
func (d *Dispatcher) Start() {
	for i := 0; i < d.workerCount; i++ {
		d.wg.Add(1)
		go d.worker()
	}

	// 启动重试管理器
	if d.retryManager != nil {
		d.retryManager.Start()
		// 加载待重试的记录
		d.retryManager.LoadPendingRetry()
	}

	logx.Infof("Webhook dispatcher started with %d workers", d.workerCount)
}

// Stop 停止分发器
func (d *Dispatcher) Stop() {
	logx.Info("Stopping webhook dispatcher...")

	close(d.stopChan)
	d.wg.Wait()

	// 停止重试管理器
	if d.retryManager != nil {
		d.retryManager.Stop()
	}

	logx.Info("Webhook dispatcher stopped")
}

// worker 分发工作协程
func (d *Dispatcher) worker() {
	defer d.wg.Done()

	for {
		select {
		case <-d.stopChan:
			return
		case event := <-d.eventQueue:
			d.processEvent(event)
		}
	}
}

// processEvent 处理事件
func (d *Dispatcher) processEvent(event *WebhookEvent) {
	// 获取订阅该事件的所有 webhook
	webhooks := d.configManager.GetWebhooksByEvent(event.EventType)
	if len(webhooks) == 0 {
		logx.Debugf("No webhooks subscribed to event: %s", event.EventType)
		return
	}

	logx.Infof("Processing event %s (%s) to %d webhooks", event.EventID, event.EventType, len(webhooks))

	// 更新指标
	webhookEventTotal.Inc(string(event.EventType), "processed")
	webhookEventPending.Inc()

	// 并发发送到所有 webhook
	var wg sync.WaitGroup
	for _, webhookConfig := range webhooks {
		wg.Add(1)
		go func(config *WebhookConfig) {
			defer wg.Done()
			d.deliverEvent(context.Background(), event, config)
		}(webhookConfig)
	}

	wg.Wait()

	// 更新指标
	webhookEventPending.Dec()
}

// deliverEvent 投递事件到指定的 webhook
func (d *Dispatcher) deliverEvent(ctx context.Context, event *WebhookEvent, config *WebhookConfig) {
	// 序列化事件数据
	eventPayload, err := json.Marshal(event)
	if err != nil {
		logx.Errorf("Failed to marshal event payload: %v", err)
		return
	}

	// 创建投递记录
	nowTime := time.Now()
	record := &DeliveryRecord{
		ID:           uuid.New().String(),
		EventID:      event.EventID,
		EventType:    event.EventType,
		EventPayload: string(eventPayload),
		WebhookURL:   config.URL,
		Status:       DeliveryStatusSending,
		AttemptCount: 0,
		LastAttempt:  nowTime,
		CreatedAt:    nowTime,
		UpdatedAt:    nowTime,
	}

	// 保存投递记录
	if d.deliveryRepo != nil {
		if err := d.deliveryRepo.Save(record); err != nil {
			logx.Errorf("Failed to save delivery record: %v", err)
			return
		}
	}

	// 创建发送器
	sender := NewSender(config, d.deliveryRepo)
	sender.SetRecord(record)

	// 发送事件
	startTime := time.Now()
	resp, err := sender.Send(ctx, event)
	duration := time.Since(startTime)

	// 更新投递记录
	record.StatusCode = resp.StatusCode
	record.Response = resp.Body
	record.Duration = duration
	record.UpdatedAt = time.Now()

	// 更新指标
	webhookDeliveryTotal.Inc(config.URL, "total", string(event.EventType))
	webhookDeliveryDuration.ObserveFloat(duration.Seconds(), config.URL, string(event.EventType))

	if err != nil {
		// 发送失败
		record.Status = DeliveryStatusFailed
		record.ErrorMessage = err.Error()

		webhookEventTotal.Inc(string(event.EventType), "failed")
		webhookDeliveryTotal.Inc(config.URL, "failed", string(event.EventType))

		logx.Errorf("Failed to deliver event %s to %s: %v", event.EventID, config.URL, err)

		// 如果启用了重试，加入重试队列
		if d.retryManager != nil && config.MaxRetries > 0 {
			record.Status = DeliveryStatusRetrying
			record.NextAttempt = time.Now().Add(config.RetryInterval)
			webhookEventRetrying.Inc()

			d.retryManager.ScheduleRetry(record)
		} else {
			record.Status = DeliveryStatusAbandoned
			webhookEventAbandoned.Inc()
		}
	} else if resp.Success {
		// 发送成功
		record.Status = DeliveryStatusSuccess

		webhookEventTotal.Inc(string(event.EventType), "success")
		webhookDeliveryTotal.Inc(config.URL, "success", string(event.EventType))

		logx.Infof("Successfully delivered event %s to %s (status: %d, duration: %v)",
			event.EventID, config.URL, resp.StatusCode, duration)
	} else {
		// HTTP 状态码非 2xx
		record.Status = DeliveryStatusFailed
		record.ErrorMessage = resp.Error

		webhookEventTotal.Inc(string(event.EventType), "failed")
		webhookDeliveryTotal.Inc(config.URL, "failed", string(event.EventType))

		logx.Errorf("Failed to deliver event %s to %s (status: %d, error: %s)",
			event.EventID, config.URL, resp.StatusCode, resp.Error)

		// 如果启用了重试，加入重试队列
		if d.retryManager != nil && config.MaxRetries > 0 {
			record.Status = DeliveryStatusRetrying
			record.NextAttempt = time.Now().Add(config.RetryInterval)
			webhookEventRetrying.Inc()

			d.retryManager.ScheduleRetry(record)
		} else {
			record.Status = DeliveryStatusAbandoned
			webhookEventAbandoned.Inc()
		}
	}

	// 更新投递记录
	if d.deliveryRepo != nil {
		if err := d.deliveryRepo.Update(record); err != nil {
			logx.Errorf("Failed to update delivery record: %v", err)
		}
	}
}

// Dispatch 分发事件（异步）
func (d *Dispatcher) Dispatch(event *WebhookEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// 设置事件ID
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}

	// 设置时间戳
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}

	select {
	case d.eventQueue <- event:
		return nil
	default:
		return fmt.Errorf("event queue is full")
	}
}

// DispatchSync 分发事件（同步等待）
func (d *Dispatcher) DispatchSync(ctx context.Context, event *WebhookEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// 设置事件ID
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}

	// 设置时间戳
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}

	// 获取订阅该事件的所有 webhook
	webhooks := d.configManager.GetWebhooksByEvent(event.EventType)
	if len(webhooks) == 0 {
		return fmt.Errorf("no webhooks subscribed to event: %s", event.EventType)
	}

	// 并发发送到所有 webhook
	var wg sync.WaitGroup
	errChan := make(chan error, len(webhooks))

	for _, webhookConfig := range webhooks {
		wg.Add(1)
		go func(config *WebhookConfig) {
			defer wg.Done()
			sender := NewSender(config, d.deliveryRepo)
			resp, err := sender.Send(ctx, event)
			if err != nil {
				errChan <- err
			}
			logx.Debugf("Successfully delivered event %s to %s (status: %d, res: %v, duration: %v)",
				event.EventID, config.URL, resp.StatusCode, err, resp.Duration)
		}(webhookConfig)
	}

	wg.Wait()
	close(errChan)
	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to deliver to %d webhooks: %v", len(errors), errors)
	}

	return nil
}

// GetQueueSize 获取队列大小
func (d *Dispatcher) GetQueueSize() int {
	return len(d.eventQueue)
}
