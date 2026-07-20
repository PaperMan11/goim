package webhooks

import (
	"context"
	"net/http"

	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Manager webhook 管理器
type Manager struct {
	configManager *ConfigManager
	dispatcher    *Dispatcher
}

// NewManager 创建 webhook 管理器
func NewManager(deliveryRepo DeliveryRepository, workerCount int) *Manager {
	configManager := NewConfigManager()

	// 创建重试管理器
	retryManager := NewRetryManager(configManager, deliveryRepo, workerCount)

	// 创建分发器
	dispatcher := NewDispatcher(configManager, deliveryRepo, retryManager, workerCount)

	return &Manager{
		configManager: configManager,
		dispatcher:    dispatcher,
	}
}

// Start 启动 webhook 管理器
func (m *Manager) Start() {
	m.dispatcher.Start()
}

// Stop 停止 webhook 管理器
func (m *Manager) Stop() {
	m.dispatcher.Stop()
}

// AddWebhook 添加 webhook 配置
func (m *Manager) AddWebhook(config *WebhookConfig) error {
	return m.configManager.AddWebhook(config)
}

// RemoveWebhook 移除 webhook 配置
func (m *Manager) RemoveWebhook(url string) {
	m.configManager.RemoveWebhook(url)
}

// GetWebhook 获取 webhook 配置
func (m *Manager) GetWebhook(url string) (*WebhookConfig, bool) {
	return m.configManager.GetWebhook(url)
}

// GetAllWebhooks 获取所有 webhook 配置
func (m *Manager) GetAllWebhooks() []*WebhookConfig {
	return m.configManager.GetAllWebhooks()
}

// Dispatch 分发事件（异步）
func (m *Manager) Dispatch(event *WebhookEvent) error {
	return m.dispatcher.Dispatch(event)
}

// DispatchSync 分发事件（同步等待）
func (m *Manager) DispatchSync(ctx context.Context, event *WebhookEvent) error {
	return m.dispatcher.DispatchSync(ctx, event)
}

// GetQueueSize 获取队列大小
func (m *Manager) GetQueueSize() int {
	return m.dispatcher.GetQueueSize()
}

// PrometheusHandler 返回 Prometheus metrics HTTP handler
func (m *Manager) PrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// EnableWebhook 启用 webhook
func (m *Manager) EnableWebhook(url string) error {
	return m.configManager.EnableWebhook(url)
}

// DisableWebhook 禁用 webhook
func (m *Manager) DisableWebhook(url string) error {
	return m.configManager.DisableWebhook(url)
}

// NewMessageEvent 创建消息事件
func NewMessageEvent(messageData *MessageEventData) *WebhookEvent {
	return &WebhookEvent{
		EventType: EventMessageSent,
		Timestamp: timex.UnixMilli(),
		Data: map[string]interface{}{
			"messageId":      messageData.MessageID,
			"serverMsgId":    messageData.ServerMsgID,
			"clientMsgId":    messageData.ClientMsgID,
			"senderId":       messageData.SenderID,
			"senderNickname": messageData.SenderNickname,
			"receiverId":     messageData.ReceiverID,
			"groupId":        messageData.GroupID,
			"contentType":    messageData.ContentType,
			"content":        messageData.Content,
			"sessionType":    messageData.SessionType,
			"sendTime":       messageData.SendTime,
			"seq":            messageData.Seq,
			"platformId":     messageData.PlatformID,
			"ex":             messageData.Ex,
			"atUserList":     messageData.AtUserList,
			"offlinePush":    messageData.OfflinePush,
		},
	}
}

// NewUserOnlineEvent 创建用户上线事件
func NewUserOnlineEvent(userData *UserEventData) *WebhookEvent {
	return &WebhookEvent{
		EventType: EventUserOnline,
		Timestamp: timex.UnixMilli(),
		Data: map[string]interface{}{
			"userId":       userData.UserID,
			"nickname":     userData.Nickname,
			"faceUrl":      userData.FaceURL,
			"platformId":   userData.PlatformID,
			"onlineStatus": userData.OnlineStatus,
			"deviceId":     userData.DeviceID,
			"extra":        userData.Extra,
		},
	}
}

// NewUserOfflineEvent 创建用户下线事件
func NewUserOfflineEvent(userData *UserEventData) *WebhookEvent {
	return &WebhookEvent{
		EventType: EventUserOffline,
		Timestamp: timex.UnixMilli(),
		Data: map[string]interface{}{
			"userId":       userData.UserID,
			"nickname":     userData.Nickname,
			"faceUrl":      userData.FaceURL,
			"platformId":   userData.PlatformID,
			"onlineStatus": userData.OnlineStatus,
			"deviceId":     userData.DeviceID,
			"extra":        userData.Extra,
		},
	}
}

// NewFriendAddedEvent 创建添加好友事件
func NewFriendAddedEvent(friendData *FriendEventData) *WebhookEvent {
	return &WebhookEvent{
		EventType: EventFriendAdded,
		Timestamp: timex.UnixMilli(),
		Data: map[string]interface{}{
			"ownerUserId":    friendData.OwnerUserID,
			"friendUserId":   friendData.FriendUserID,
			"friendNickname": friendData.FriendNickname,
			"friendFaceUrl":  friendData.FriendFaceURL,
			"remark":         friendData.Remark,
			"source":         friendData.Source,
			"operatorUserId": friendData.OperatorUserID,
			"handleResult":   friendData.HandleResult,
			"reqMsg":         friendData.ReqMsg,
			"extra":          friendData.Extra,
		},
	}
}

// NewGroupCreatedEvent 创建群组事件
func NewGroupCreatedEvent(groupData *GroupEventData) *WebhookEvent {
	return &WebhookEvent{
		EventType: EventGroupCreated,
		Timestamp: timex.UnixMilli(),
		Data: map[string]interface{}{
			"groupId":         groupData.GroupID,
			"groupName":       groupData.GroupName,
			"groupFaceUrl":    groupData.GroupFaceURL,
			"groupType":       groupData.GroupType,
			"ownerUserId":     groupData.OwnerUserID,
			"memberCount":     groupData.MemberCount,
			"introduction":    groupData.Introduction,
			"notification":    groupData.Notification,
			"memberUserId":    groupData.MemberUserID,
			"memberNickname":  groupData.MemberNickname,
			"memberRoleLevel": groupData.MemberRoleLevel,
			"operatorUserId":  groupData.OperatorUserID,
			"reason":          groupData.Reason,
			"extra":           groupData.Extra,
		},
	}
}

// NewConversationUpdatedEvent 创建会话更新事件
func NewConversationUpdatedEvent(conversationData *ConversationEventData) *WebhookEvent {
	return &WebhookEvent{
		EventType: EventConversationUpdated,
		Timestamp: timex.UnixMilli(),
		Data: map[string]interface{}{
			"conversationId":   conversationData.ConversationID,
			"conversationType": conversationData.ConversationType,
			"userId":           conversationData.UserID,
			"groupId":          conversationData.GroupID,
			"showName":         conversationData.ShowName,
			"faceUrl":          conversationData.FaceURL,
			"recvMsgOpt":       conversationData.RecvMsgOpt,
			"unreadCount":      conversationData.UnreadCount,
			"latestMsg":        conversationData.LatestMsg,
			"latestMsgTime":    conversationData.LatestMsgTime,
			"isPinned":         conversationData.IsPinned,
			"isPrivateChat":    conversationData.IsPrivateChat,
			"extra":            conversationData.Extra,
		},
	}
}
