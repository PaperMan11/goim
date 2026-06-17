# IM Webhooks 系统

一个功能完整的 IM Webhooks 系统，支持多种 IM 事件的实时通知和投递。

## 功能特性

- **事件类型丰富**：支持消息、用户、好友、群组、会话、推送等多种 IM 事件
- **配置管理**：灵活的 webhook 配置管理，支持动态添加、删除、启用、禁用
- **安全机制**：HMAC-SHA256 签名验证，时间戳防重放攻击
- **重试机制**：指数退避重试策略，支持自定义重试次数和间隔
- **异步分发**：高并发事件分发，支持队列缓冲
- **投递记录**：完整的投递记录追踪，支持状态查询
- **指标监控**：实时统计事件和投递指标

## 文件结构

```
pkg/webhooks/
├── types.go          # 事件类型和数据结构定义
├── config.go         # 配置管理器
├── security.go       # 安全验证和签名机制
├── sender.go         # webhook 发送器和重试机制
├── dispatcher.go     # 事件分发器
├── webhooks.go       # 主入口和管理器
├── repo.go           # 投递记录仓库接口和内存实现
└── example_test.go   # 使用示例
```

## 快速开始

### 1. 创建 webhook 管理器

```go
import "github.com/PaperMan11/goim/pkg/webhooks"

// 创建投递记录仓库（需要实现 DeliveryRepository 接口）
deliveryRepo := webhooks.NewMemoryDeliveryRepository()

// 创建 webhook 管理器（5 个工作协程）
manager := webhooks.NewManager(deliveryRepo, 5)

// 启动管理器
manager.Start()
defer manager.Stop()
```

### 2. 添加 webhook 配置

```go
webhookConfig := &webhooks.WebhookConfig{
    URL:           "https://example.com/webhook",
    Secret:        "your-secret-key",  // 用于签名验证
    Timeout:       10 * time.Second,
    MaxRetries:    3,
    RetryInterval: 5 * time.Second,
    Enabled:       true,
    Events: []webhooks.EventType{
        webhooks.EventMessageSent,
        webhooks.EventUserOnline,
        webhooks.EventUserOffline,
    },
    Headers: map[string]string{
        "X-Custom-Header": "custom-value",
    },
}

if err := manager.AddWebhook(webhookConfig); err != nil {
    log.Fatalf("Failed to add webhook: %v", err)
}
```

### 3. 分发事件

```go
// 创建消息事件
messageEvent := webhooks.NewMessageEvent(&webhooks.MessageEventData{
    MessageID:      "msg_001",
    ServerMsgID:    "server_msg_001",
    ClientMsgID:    "client_msg_001",
    SenderID:       "user_001",
    SenderNickname: "Alice",
    ReceiverID:     "user_002",
    ContentType:    101, // 文本消息
    Content:        "Hello, World!",
    SessionType:    1,  // 单聊
    SendTime:       time.Now().Unix(),
    Seq:            1,
    PlatformID:     1,
})

// 异步分发事件
if err := manager.Dispatch(messageEvent); err != nil {
    log.Printf("Failed to dispatch event: %v", err)
}

// 或者同步分发事件（等待所有 webhook 响应）
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := manager.DispatchSync(ctx, messageEvent); err != nil {
    log.Printf("Failed to dispatch event synchronously: %v", err)
}
```

### 4. 查看指标

```go
metrics := manager.GetMetrics()
fmt.Printf("Total events: %d\n", metrics.TotalEvents)
fmt.Printf("Success events: %d\n", metrics.SuccessEvents)
fmt.Printf("Failed events: %d\n", metrics.FailedEvents)
fmt.Printf("Total deliveries: %d\n", metrics.TotalDeliveries)
fmt.Printf("Success deliveries: %d\n", metrics.SuccessDeliveries)
fmt.Printf("Failed deliveries: %d\n", metrics.FailedDeliveries)
```

## 支持的事件类型

### 消息事件
- `message.sent` - 消息发送成功
- `message.received` - 消息接收
- `message.revoked` - 消息撤回
- `message.deleted` - 消息删除
- `message.read` - 消息已读
- `message.reaction_added` - 消息表情回应添加
- `message.reaction_removed` - 消息表情回应删除

### 用户事件
- `user.online` - 用户上线
- `user.offline` - 用户下线
- `user.kicked` - 用户被踢
- `user.info_updated` - 用户信息更新
- `user.status_changed` - 用户状态变更

### 好友事件
- `friend.application_received` - 收到好友申请
- `friend.application_approved` - 好友申请已同意
- `friend.application_rejected` - 好友申请已拒绝
- `friend.added` - 已添加好友
- `friend.deleted` - 已删除好友
- `friend.black_added` - 已加入黑名单
- `friend.black_deleted` - 已移出黑名单

### 群组事件
- `group.created` - 群组创建
- `group.info_updated` - 群组信息更新
- `group.member_joined` - 成员加入群
- `group.member_left` - 成员退出群
- `group.member_kicked` - 成员被踢
- `group.owner_transferred` - 群主转让
- `group.dismissed` - 群组解散
- `group.muted` - 群组禁言
- `group.unmuted` - 群组取消禁言
- `group.member_muted` - 成员禁言
- `group.member_unmuted` - 成员取消禁言
- `group.member_role_changed` - 成员角色变更

### 会话事件
- `conversation.created` - 会话创建
- `conversation.updated` - 会话更新
- `conversation.deleted` - 会话删除
- `conversation.unread_changed` - 会话未读数变更

### 推送事件
- `push.offline` - 离线推送
- `push.online` - 在线推送

## Webhook 请求格式

### 请求头

```
Content-Type: application/json
User-Agent: GoIM-Webhook/1.0
X-Webhook-Signature: sha256=xxx
X-Webhook-Timestamp: 1234567890
X-Webhook-Event-ID: event_001
X-Webhook-Event-Type: message.sent
X-Webhook-Retry-Count: 0
```

### 请求体

```json
{
  "eventType": "message.sent",
  "eventId": "event_001",
  "timestamp": 1234567890,
  "operationId": "operation_001",
  "retryCount": 0,
  "isRetry": false,
  "data": {
    "messageId": "msg_001",
    "serverMsgId": "server_msg_001",
    "clientMsgId": "client_msg_001",
    "senderId": "user_001",
    "senderNickname": "Alice",
    "receiverId": "user_002",
    "contentType": 101,
    "content": "Hello, World!",
    "sessionType": 1,
    "sendTime": 1234567890,
    "seq": 1,
    "platformId": 1
  }
}
```

## 签名验证

### 签名生成

签名格式：`sha256=hex(hmac_sha256(secret, timestamp + payload))`

```go
signer := webhooks.NewSigner("your-secret-key")
timestamp := time.Now().UnixMilli()
payload := []byte(`{"eventType":"message.sent",...}`)
signature := signer.Sign(timestamp, payload)
```

### 签名验证

```go
signer := webhooks.NewSigner("your-secret-key")
isValid := signer.Verify(signature, timestamp, payload)
```

### 安全验证器

```go
securityConfig := webhooks.DefaultSecurityConfig()
validator := webhooks.NewSecurityValidator(securityConfig, "your-secret-key")

headers := map[string]string{
    webhooks.SignatureHeader: signature,
    webhooks.TimestampHeader: fmt.Sprintf("%d", timestamp),
}

err := validator.Validate(headers, payload)
if err != nil {
    log.Printf("Validation failed: %v", err)
}
```

## 配置管理

### 添加 webhook

```go
err := manager.AddWebhook(webhookConfig)
```

### 移除 webhook

```go
manager.RemoveWebhook("https://example.com/webhook")
```

### 获取 webhook

```go
webhook, exists := manager.GetWebhook("https://example.com/webhook")
```

### 启用/禁用 webhook

```go
err := manager.EnableWebhook("https://example.com/webhook")
err := manager.DisableWebhook("https://example.com/webhook")
```

### 获取所有 webhook

```go
webhooks := manager.GetAllWebhooks()
```

## 投递记录

### 投递状态

- `pending` - 待投递
- `sending` - 投递中
- `success` - 投递成功
- `failed` - 投递失败
- `retrying` - 重试中
- `abandoned` - 已放弃

### 实现 DeliveryRepository 接口

```go
type DeliveryRepository interface {
    Save(record *DeliveryRecord) error
    Update(record *DeliveryRecord) error
    Get(id string) (*DeliveryRecord, error)
    GetByEventID(eventID string) ([]*DeliveryRecord, error)
    GetPending(limit int) ([]*DeliveryRecord, error)
    Delete(id string) error
}
```

## 重试机制

系统使用指数退避策略进行重试：

- 基础延迟：`RetryInterval`
- 退避公式：`delay = baseDelay * (2 ^ attempt)`
- 添加随机抖动：±25%

## 最佳实践

1. **设置合理的超时时间**：根据 webhook 服务器的响应时间设置合适的超时
2. **配置重试策略**：根据业务重要性设置重试次数和间隔
3. **使用签名验证**：启用签名验证确保请求的安全性
4. **监控投递指标**：定期检查投递成功率，及时发现问题
5. **实现持久化存储**：实现 `DeliveryRepository` 接口，将投递记录持久化到数据库
6. **处理高并发**：根据业务量调整工作协程数量和队列大小

## 依赖

- Go 1.16+
- github.com/google/uuid
- github.com/zeromicro/go-zero
