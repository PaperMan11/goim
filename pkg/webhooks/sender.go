package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// Sender webhook 发送器
type Sender struct {
	client       *http.Client
	config       *WebhookConfig
	signer       *Signer
	deliveryRepo DeliveryRepository
	record       *DeliveryRecord
}

// NewSender 创建 webhook 发送器
func NewSender(config *WebhookConfig, deliveryRepo DeliveryRepository) *Sender {
	return &Sender{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		config:       config,
		signer:       NewSigner(config.Secret),
		deliveryRepo: deliveryRepo,
	}
}

// SetRecord 设置投递记录（用于记录重试次数）
func (s *Sender) SetRecord(record *DeliveryRecord) {
	s.record = record
}

// Send 发送 webhook 事件
func (s *Sender) Send(ctx context.Context, event *WebhookEvent) (*WebhookResponse, error) {
	return s.sendOnce(ctx, event)
}

// SendWithRetry 发送 webhook 事件（带重试）
func (s *Sender) SendWithRetry(ctx context.Context, event *WebhookEvent, maxRetries int) (*WebhookResponse, error) {
	var lastResponse *WebhookResponse
	var lastError error

	attempt := 0
	if s.record != nil {
		attempt = s.record.AttemptCount
	}
	for ; attempt <= maxRetries; attempt++ {
		// 更新重试次数
		event.RetryCount = attempt
		event.IsRetry = attempt > 0

		// 更新投递记录中的尝试次数
		if s.record != nil {
			s.record.AttemptCount = attempt
			s.record.LastAttempt = time.Now()
			if s.deliveryRepo != nil {
				s.deliveryRepo.Update(s.record)
			}
		}

		// 发送请求
		resp, err := s.sendOnce(ctx, event)
		if err != nil {
			lastError = err
			lastResponse = resp

			// 如果是最后一次尝试，返回错误
			if attempt == maxRetries {
				break
			}

			// 计算退避时间（指数退避）
			backoff := s.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}

		// 请求成功
		return resp, nil
	}

	return lastResponse, lastError
}

// sendOnce 发送一次 webhook 请求
func (s *Sender) sendOnce(ctx context.Context, event *WebhookEvent) (*WebhookResponse, error) {
	startTime := time.Now()

	// 序列化事件数据
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoIM-Webhook/1.0")

	// 添加自定义请求头
	for key, value := range s.config.Headers {
		req.Header.Set(key, value)
	}

	// 添加安全相关请求头
	if s.signer != nil {
		securityHeaders := s.signer.GenerateHeaders(event)
		for key, value := range securityHeaders {
			req.Header.Set(key, value)
		}
	}

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 构造响应
	response := &WebhookResponse{
		StatusCode: resp.StatusCode,
		Success:    resp.StatusCode >= 200 && resp.StatusCode < 300,
		Error:      "",
		Headers:    make(map[string]string),
		Body:       string(body),
		Duration:   time.Since(startTime),
	}

	// 复制响应头
	for key, values := range resp.Header {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}

	// 检查响应状态
	if !response.Success {
		response.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return response, nil
}

// calculateBackoff 计算退避时间（指数退避）
func (s *Sender) calculateBackoff(attempt int) time.Duration {
	// 基础延迟时间
	baseDelay := s.config.RetryInterval

	// 指数退避：delay = baseDelay * (2 ^ attempt)
	delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))

	// 添加随机抖动（±25%）
	jitter := time.Duration(float64(delay) * 0.25 * (float64(2*time.Now().UnixNano()%1000)/1000.0 - 0.5))

	return delay + jitter
}

// RetryManager 重试管理器
type RetryManager struct {
	configManager *ConfigManager
	deliveryRepo  DeliveryRepository
	retryQueue    chan *DeliveryRecord
	workerCount   int
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewRetryManager 创建重试管理器
func NewRetryManager(
	configManager *ConfigManager,
	deliveryRepo DeliveryRepository,
	workerCount int,
) *RetryManager {
	return &RetryManager{
		configManager: configManager,
		deliveryRepo:  deliveryRepo,
		retryQueue:    make(chan *DeliveryRecord, 1000),
		workerCount:   workerCount,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动重试管理器
func (rm *RetryManager) Start() {
	for i := 0; i < rm.workerCount; i++ {
		rm.wg.Add(1)
		go rm.worker()
	}
}

// Stop 停止重试管理器
func (rm *RetryManager) Stop() {
	close(rm.stopChan)
	rm.wg.Wait()
}

// worker 重试工作协程
func (rm *RetryManager) worker() {
	defer rm.wg.Done()

	for {
		select {
		case <-rm.stopChan:
			return
		case record := <-rm.retryQueue:
			rm.processRetry(record)
		}
	}
}

// processRetry 处理重试
func (rm *RetryManager) processRetry(record *DeliveryRecord) {
	// 检查是否到达下次尝试时间
	if time.Now().Before(record.NextAttempt) {
		// 重新放入队列
		time.Sleep(time.Until(record.NextAttempt))
		rm.retryQueue <- record
		return
	}

	// 获取 webhook 配置
	config, exists := rm.configManager.GetWebhook(record.WebhookURL)
	if !exists || !config.Enabled {
		// webhook 不存在或已禁用，放弃重试
		record.Status = DeliveryStatusAbandoned
		record.ErrorMessage = "webhook not found or disabled"
		rm.deliveryRepo.Update(record)
		return
	}

	// 更新记录状态
	record.Status = DeliveryStatusSending
	record.LastAttempt = time.Now()
	rm.deliveryRepo.Update(record)

	// 检查是否超过最大重试次数
	if record.AttemptCount > config.MaxRetries {
		record.Status = DeliveryStatusAbandoned
		record.ErrorMessage = "max retries exceeded"
		rm.deliveryRepo.Update(record)
		webhookEventAbandoned.Inc()
		webhookEventRetrying.Dec()
		return
	}

	// 从投递记录中恢复事件数据
	var event *WebhookEvent
	if record.EventPayload != "" {
		if err := json.Unmarshal([]byte(record.EventPayload), &event); err != nil {
			logx.Errorf("Failed to unmarshal event payload: %v, recordID: %s", err, record.ID)
			record.Status = DeliveryStatusAbandoned
			record.ErrorMessage = "failed to unmarshal event payload: " + err.Error()
			rm.deliveryRepo.Update(record)
			return
		}
	} else {
		logx.Errorf("Event payload is empty, cannot retry, recordID: %s", record.ID)
		record.Status = DeliveryStatusAbandoned
		record.ErrorMessage = "event payload is empty"
		rm.deliveryRepo.Update(record)
		return
	}

	// 更新重试相关字段
	event.RetryCount = record.AttemptCount
	event.IsRetry = true

	// 发送请求
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	sender := NewSender(config, rm.deliveryRepo)
	sender.SetRecord(record)
	resp, err := sender.Send(ctx, event)
	logx.Debugf("Send retry event: %s, status: %s, statusCode: %d, nextAttempt: %d, retryCount: %d, maxRetries: %d", record.EventID, record.Status, record.StatusCode, record.NextAttempt.Unix(), record.AttemptCount, config.MaxRetries)
	if err != nil {
		// 发送失败，计算下次重试时间
		record.Status = DeliveryStatusRetrying
		record.ErrorMessage = err.Error()
		record.Response = ""
		record.NextAttempt = time.Now().Add(sender.calculateBackoff(record.AttemptCount))
		record.AttemptCount++
		rm.deliveryRepo.Update(record)

		webhookDeliveryTotal.Inc(record.WebhookURL, "retry_failed", string(record.EventType))
		webhookRetryCount.ObserveFloat(float64(record.AttemptCount), record.WebhookURL, string(record.EventType))

		// 重新放入重试队列
		rm.retryQueue <- record
		return
	}

	// 发送成功
	if resp.Success {
		record.Status = DeliveryStatusSuccess
		record.StatusCode = resp.StatusCode
		record.Response = resp.Body
		record.Duration = resp.Duration
		rm.deliveryRepo.Update(record)

		webhookDeliveryTotal.Inc(record.WebhookURL, "retry_success", string(record.EventType))
		webhookEventRetrying.Dec()
	} else {
		// HTTP 状态码非 2xx，视为失败
		record.Status = DeliveryStatusFailed
		record.StatusCode = resp.StatusCode
		record.ErrorMessage = resp.Error
		record.Response = resp.Body
		record.Duration = resp.Duration

		// 检查是否可以重试
		if record.AttemptCount < config.MaxRetries {
			record.Status = DeliveryStatusRetrying
			record.NextAttempt = time.Now().Add(sender.calculateBackoff(record.AttemptCount))
			record.AttemptCount++
			rm.retryQueue <- record

			webhookDeliveryTotal.Inc(record.WebhookURL, "retry_failed", string(record.EventType))
			webhookRetryCount.ObserveFloat(float64(record.AttemptCount), record.WebhookURL, string(record.EventType))
		} else {
			record.Status = DeliveryStatusAbandoned
			webhookEventAbandoned.Inc()
			webhookEventRetrying.Dec()
		}

		rm.deliveryRepo.Update(record)
	}
}

// ScheduleRetry 调度重试
func (rm *RetryManager) ScheduleRetry(record *DeliveryRecord) {
	select {
	case rm.retryQueue <- record:
	default:
		// 队列已满，记录日志
		logx.Errorf("[webhooks] retry queue is full, dropping record: %s", record.ID)
	}
}

// LoadPendingRetry 加载待重试的记录
func (rm *RetryManager) LoadPendingRetry() {
	records, err := rm.deliveryRepo.GetPending(100)
	if err != nil {
		logx.Errorf("[webhooks] failed to load pending records: %v", err)
		return
	}

	for _, record := range records {
		rm.ScheduleRetry(record)
	}
}
