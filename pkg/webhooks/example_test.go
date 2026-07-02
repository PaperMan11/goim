package webhooks_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/PaperMan11/goim/pkg/webhooks"
)

// TestUsage 基本使用测试
func TestUsage(t *testing.T) {
	deliveryRepo := webhooks.NewMemoryDeliveryRepository()
	manager := webhooks.NewManager(deliveryRepo, 5)
	manager.Start()
	defer manager.Stop()

	webhookConfig := &webhooks.WebhookConfig{
		URL:           "https://example.com/webhook",
		Secret:        "your-secret-key",
		Timeout:       10 * time.Second,
		MaxRetries:    3,
		RetryInterval: 5 * time.Second,
		Enabled:       true,
		Events: []webhooks.EventType{
			webhooks.EventMessageSentBefore,
			webhooks.EventMessageSentAfter,
			webhooks.EventUserOnline,
			webhooks.EventUserOffline,
		},
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	if err := manager.AddWebhook(webhookConfig); err != nil {
		t.Fatalf("Failed to add webhook: %v", err)
	}

	messageEvent := webhooks.NewMessageSentBeforeEvent(&webhooks.MessageEventData{
		MessageID:      "msg_001",
		ServerMsgID:    "server_msg_001",
		ClientMsgID:    "client_msg_001",
		SenderID:       "user_001",
		SenderNickname: "Alice",
		ReceiverID:     "user_002",
		ContentType:    101,
		Content:        "Hello, World!",
		SessionType:    1,
		SendTime:       time.Now().Unix(),
		Seq:            1,
		PlatformID:     1,
	})

	if err := manager.Dispatch(messageEvent); err != nil {
		t.Logf("Failed to dispatch event: %v", err)
	}

	userOnlineEvent := webhooks.NewUserOnlineEvent(&webhooks.UserEventData{
		UserID:       "user_001",
		Nickname:     "Alice",
		FaceURL:      "https://example.com/avatar.jpg",
		PlatformID:   1,
		OnlineStatus: 1,
		DeviceID:     "device_001",
	})

	if err := manager.Dispatch(userOnlineEvent); err != nil {
		t.Logf("Failed to dispatch event: %v", err)
	}

	// time.Sleep(60 * time.Second)

	t.Log("Event dispatched successfully")
}

// TestSyncDispatch 同步分发测试
func TestSyncDispatch(t *testing.T) {
	deliveryRepo := webhooks.NewMemoryDeliveryRepository()
	manager := webhooks.NewManager(deliveryRepo, 5)

	webhookConfig := &webhooks.WebhookConfig{
		URL:           "https://example.com/webhook",
		Secret:        "your-secret-key",
		Timeout:       10 * time.Second,
		MaxRetries:    3,
		RetryInterval: 5 * time.Second,
		Enabled:       true,
		Events: []webhooks.EventType{
			webhooks.EventMessageSentBefore,
			webhooks.EventMessageSentAfter,
		},
	}

	if err := manager.AddWebhook(webhookConfig); err != nil {
		t.Fatalf("Failed to add webhook: %v", err)
	}

	messageEvent := webhooks.NewMessageSentBeforeEvent(&webhooks.MessageEventData{
		MessageID:      "msg_002",
		ServerMsgID:    "server_msg_002",
		ClientMsgID:    "client_msg_002",
		SenderID:       "user_001",
		SenderNickname: "Alice",
		ReceiverID:     "user_002",
		ContentType:    101,
		Content:        "Sync message",
		SessionType:    1,
		SendTime:       time.Now().Unix(),
		Seq:            2,
		PlatformID:     1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := manager.DispatchSync(ctx, messageEvent); err != nil {
		t.Logf("Failed to dispatch event synchronously: %v", err)
	} else {
		t.Log("Event dispatched successfully to all webhooks")
	}
}

// TestCustomEvent 自定义事件测试
func TestCustomEvent(t *testing.T) {
	deliveryRepo := webhooks.NewMemoryDeliveryRepository()
	manager := webhooks.NewManager(deliveryRepo, 5)
	manager.Start()
	defer manager.Stop()

	webhookConfig := &webhooks.WebhookConfig{
		URL:           "https://example.com/webhook",
		Secret:        "your-secret-key",
		Timeout:       10 * time.Second,
		MaxRetries:    3,
		RetryInterval: 5 * time.Second,
		Enabled:       true,
	}

	if err := manager.AddWebhook(webhookConfig); err != nil {
		t.Fatalf("Failed to add webhook: %v", err)
	}

	customEvent := &webhooks.WebhookEvent{
		EventType:   webhooks.EventType("custom.event"),
		EventID:     "custom_001",
		Timestamp:   time.Now().UnixMilli(),
		OperationID: "operation_001",
		Data: map[string]interface{}{
			"customField1": "value1",
			"customField2": 123,
			"customField3": map[string]interface{}{
				"nestedField": "nestedValue",
			},
		},
	}

	if err := manager.Dispatch(customEvent); err != nil {
		t.Logf("Failed to dispatch custom event: %v", err)
	}

	time.Sleep(2 * time.Second)
}

// TestWebhookManagement webhook 管理测试
func TestWebhookManagement(t *testing.T) {
	deliveryRepo := webhooks.NewMemoryDeliveryRepository()
	manager := webhooks.NewManager(deliveryRepo, 5)

	webhookConfigs := []*webhooks.WebhookConfig{
		{
			URL:           "https://webhook1.example.com",
			Secret:        "secret1",
			Timeout:       10 * time.Second,
			MaxRetries:    3,
			RetryInterval: 5 * time.Second,
			Enabled:       true,
			Events: []webhooks.EventType{
				webhooks.EventMessageSentBefore,
				webhooks.EventMessageSentAfter,
			},
		},
		{
			URL:           "https://webhook2.example.com",
			Secret:        "secret2",
			Timeout:       15 * time.Second,
			MaxRetries:    5,
			RetryInterval: 10 * time.Second,
			Enabled:       true,
			Events: []webhooks.EventType{
				webhooks.EventUserOnline,
				webhooks.EventUserOffline,
			},
		},
	}

	for _, webhook := range webhookConfigs {
		if err := manager.AddWebhook(webhook); err != nil {
			t.Logf("Failed to add webhook %s: %v", webhook.URL, err)
		}
	}

	allWebhooks := manager.GetAllWebhooks()
	t.Logf("Total webhooks: %d", len(allWebhooks))

	if err := manager.DisableWebhook("https://webhook1.example.com"); err != nil {
		t.Logf("Failed to disable webhook: %v", err)
	}

	if err := manager.EnableWebhook("https://webhook1.example.com"); err != nil {
		t.Logf("Failed to enable webhook: %v", err)
	}

	if webhook, exists := manager.GetWebhook("https://webhook1.example.com"); exists {
		t.Logf("Webhook status: %v", webhook.Enabled)
	}

	manager.RemoveWebhook("https://webhook2.example.com")
}

// TestSecurity 安全验证测试
func TestSecurity(t *testing.T) {
	signer := webhooks.NewSigner("your-secret-key")

	timestamp := time.Now().UnixMilli()
	payload := []byte(`{"eventType":"message.sent","eventId":"event_001"}`)
	signature := signer.Sign(timestamp, payload)

	t.Logf("Signature: %s", signature)

	isValid := signer.Verify(signature, timestamp, payload)
	t.Logf("Signature valid: %v", isValid)

	securityConfig := webhooks.DefaultSecurityConfig()
	validator := webhooks.NewSecurityValidator(securityConfig, "your-secret-key")

	headers := map[string]string{
		webhooks.SignatureHeader: signature,
		webhooks.TimestampHeader: fmt.Sprintf("%d", timestamp),
	}

	err := validator.Validate(headers, payload)
	if err != nil {
		t.Logf("Validation failed: %v", err)
	} else {
		t.Log("Validation passed")
	}
}
