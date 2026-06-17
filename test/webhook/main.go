package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/PaperMan11/goim/pkg/webhooks"
)

const webhookSecret = "webhook-secret-key"

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Received webhook request header: %v\n", r.Header)
	// 1. 验证签名
	signature := r.Header.Get("X-Webhook-Signature")
	timestamp := r.Header.Get("X-Webhook-Timestamp")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// 验证时间戳（防止重放攻击）
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || time.Now().UnixMilli()-ts > 5*60*1000 {
		fmt.Printf("Invalid timestamp: %s\n", timestamp)
		http.Error(w, "Invalid timestamp", http.StatusBadRequest)
		return
	}

	// 计算并验证签名
	if webhookSecret != "" {
		expectedSignature := calculateSignature(ts, body, webhookSecret)
		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			fmt.Printf("Invalid signature: %s, expected: %s\n", signature, expectedSignature)
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// 2. 解析事件数据
	var event webhooks.WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		fmt.Printf("Failed to parse event: %v\n", err)
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}
	fmt.Printf("Received webhook event: %+v\n", event)

	var eventData webhooks.UserEventData
	bytes, _ := json.Marshal(event.Data)
	if err := json.Unmarshal(bytes, &eventData); err != nil {
		fmt.Printf("Failed to parse event data: %v\n", err)
		http.Error(w, "Failed to parse event data", http.StatusBadRequest)
		return
	}

	// 3. 处理事件
	eventType := event.EventType
	switch eventType {
	case "user.online":
		handleUserOnline(eventData)
	case "user.offline":
		handleUserOffline(eventData)
	default:
		fmt.Printf("Unknown event type: %s\n", eventType)
	}

	// 4. 返回成功响应
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleUserOnline(event webhooks.UserEventData) {
	fmt.Printf("用户上线: userInfo: %+v\n", event)
}

func handleUserOffline(event webhooks.UserEventData) {
	fmt.Printf("用户下线: userInfo: %+v\n", event)
}

func calculateSignature(timestamp int64, payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(strconv.FormatInt(timestamp, 10)))
	h.Write(payload)
	return fmt.Sprintf("sha256=%s", hex.EncodeToString(h.Sum(nil)))
}

func main() {

	http.HandleFunc("/api/webhooks", handleWebhook)
	go func() {
		fmt.Println("Webhook server listening on :8080")
		http.ListenAndServe(":8080", nil)
	}()

	time.Sleep(time.Second)
	sendUserEvent()
	time.Sleep(3 * time.Second)
}
