package webhooks

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ConfigManager webhook 配置管理器
type ConfigManager struct {
	mu       sync.RWMutex
	configs  map[string]*WebhookConfig // key: webhook URL
	defaults *WebhookConfig
}

// NewConfigManager 创建配置管理器
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configs: make(map[string]*WebhookConfig),
		defaults: &WebhookConfig{
			Timeout:       10 * time.Second,
			MaxRetries:    3,
			RetryInterval: 5 * time.Second,
			Enabled:       true,
			Headers:       make(map[string]string),
		},
	}
}

// AddWebhook 添加 webhook 配置
func (cm *ConfigManager) AddWebhook(config *WebhookConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.URL == "" {
		return fmt.Errorf("webhook URL cannot be empty")
	}

	// 设置默认值
	if config.Timeout == 0 {
		config.Timeout = cm.defaults.Timeout
	}
	// if config.MaxRetries == 0 {
	// 	config.MaxRetries = cm.defaults.MaxRetries
	// }
	// if config.RetryInterval == 0 {
	// 	config.RetryInterval = cm.defaults.RetryInterval
	// }
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.configs[config.URL] = config
	return nil
}

// RemoveWebhook 移除 webhook 配置
func (cm *ConfigManager) RemoveWebhook(url string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.configs, url)
}

// GetWebhook 获取 webhook 配置
func (cm *ConfigManager) GetWebhook(url string) (*WebhookConfig, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config, exists := cm.configs[url]
	if !exists {
		return nil, false
	}

	// 返回副本，避免外部修改
	configCopy := *config
	return &configCopy, true
}

// GetAllWebhooks 获取所有 webhook 配置
func (cm *ConfigManager) GetAllWebhooks() []*WebhookConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make([]*WebhookConfig, 0, len(cm.configs))
	for _, config := range cm.configs {
		configCopy := *config
		configs = append(configs, &configCopy)
	}

	return configs
}

// GetEnabledWebhooks 获取启用的 webhook 配置
func (cm *ConfigManager) GetEnabledWebhooks() []*WebhookConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make([]*WebhookConfig, 0)
	for _, config := range cm.configs {
		if config.Enabled {
			configCopy := *config
			configs = append(configs, &configCopy)
		}
	}

	return configs
}

// GetWebhooksByEvent 根据事件类型获取订阅的 webhook 配置
func (cm *ConfigManager) GetWebhooksByEvent(eventType EventType) []*WebhookConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make([]*WebhookConfig, 0)
	for _, config := range cm.configs {
		if !config.Enabled {
			continue
		}

		// // 如果未指定事件列表，则订阅所有事件
		// if len(config.Events) == 0 {
		// 	configCopy := *config
		// 	configs = append(configs, &configCopy)
		// 	continue
		// }

		// 检查是否订阅了该事件
		for _, event := range config.Events {
			if event == eventType {
				configCopy := *config
				configs = append(configs, &configCopy)
				break
			}
		}
	}

	return configs
}

// UpdateWebhook 更新 webhook 配置
func (cm *ConfigManager) UpdateWebhook(url string, config *WebhookConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.URL == "" {
		return fmt.Errorf("webhook URL cannot be empty")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.configs[url]; !exists {
		return fmt.Errorf("webhook not found: %s", url)
	}

	cm.configs[url] = config
	return nil
}

// EnableWebhook 启用 webhook
func (cm *ConfigManager) EnableWebhook(url string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	config, exists := cm.configs[url]
	if !exists {
		return fmt.Errorf("webhook not found: %s", url)
	}

	config.Enabled = true
	return nil
}

// DisableWebhook 禁用 webhook
func (cm *ConfigManager) DisableWebhook(url string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	config, exists := cm.configs[url]
	if !exists {
		return fmt.Errorf("webhook not found: %s", url)
	}

	config.Enabled = false
	return nil
}

// SetDefaultConfig 设置默认配置
func (cm *ConfigManager) SetDefaultConfig(config *WebhookConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.defaults = config
}

// GetDefaultConfig 获取默认配置
func (cm *ConfigManager) GetDefaultConfig() *WebhookConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configCopy := *cm.defaults
	return &configCopy
}

// LoadFromJSON 从 JSON 加载配置
func (cm *ConfigManager) LoadFromJSON(data []byte) error {
	var configs []*WebhookConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.configs = make(map[string]*WebhookConfig)
	for _, config := range configs {
		// 设置默认值
		if config.Timeout == 0 {
			config.Timeout = cm.defaults.Timeout
		}
		if config.MaxRetries == 0 {
			config.MaxRetries = cm.defaults.MaxRetries
		}
		if config.RetryInterval == 0 {
			config.RetryInterval = cm.defaults.RetryInterval
		}
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}

		cm.configs[config.URL] = config
	}

	return nil
}

// SaveToJSON 保存配置到 JSON
func (cm *ConfigManager) SaveToJSON() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make([]*WebhookConfig, 0, len(cm.configs))
	for _, config := range cm.configs {
		configs = append(configs, config)
	}

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return data, nil
}

// Clear 清空所有配置
func (cm *ConfigManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.configs = make(map[string]*WebhookConfig)
}

// Count 获取 webhook 数量
func (cm *ConfigManager) Count() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.configs)
}

// Exists 检查 webhook 是否存在
func (cm *ConfigManager) Exists(url string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	_, exists := cm.configs[url]
	return exists
}
