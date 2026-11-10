package msgcontent

import (
	"encoding/json"
	"fmt"

	sdkws "github.com/PaperMan11/goim/pkg/protocol/sdkws"
)

// ========== Revoke（撤回通知，contentType=2101 / MsgRevokeNotification）==========

// RevokeNotificationContent 撤回通知消息的 content JSON 外层结构。
// 实际撤回详情在 Detail 字段里，格式为 MessageRevokedContent 的 JSON 字符串。
//
// 原始消息 content 示例：
//
//	{"detail":"{\"revokerID\":\"userB\",\"clientMsgID\":\"xxx\",\"revokeTime\":1720000000,\"seq\":42,...}"}
//
// 服务端需要解析这个结构来：
//   1. 展示撤回通知（谁撤回了什么消息）
//   2. 在 Quote 引用体被撤回时改写引用体
type RevokeNotificationContent struct {
	Detail string `json:"detail,omitempty"`
}

// ParseRevoke 解析撤回通知消息的 content 外层结构。
func ParseRevoke(content string) (*RevokeNotificationContent, error) {
	if content == "" {
		return nil, fmt.Errorf("msgcontent: ParseRevoke 空 content")
	}
	var r RevokeNotificationContent
	if err := json.Unmarshal([]byte(content), &r); err != nil {
		return nil, fmt.Errorf("msgcontent: ParseRevoke JSON 解析失败: %w", err)
	}
	return &r, nil
}

// ParseRevokeDetail 把外层 Detail 里的 JSON 再解析成 proto 定义的 MessageRevokedContent。
// 这样服务端直接拿到完整结构体做业务处理。
func ParseRevokeDetail(content string) (*sdkws.MessageRevokedContent, error) {
	outer, err := ParseRevoke(content)
	if err != nil {
		return nil, err
	}
	if outer.Detail == "" {
		return nil, fmt.Errorf("msgcontent: ParseRevokeDetail detail 字段为空")
	}
	var inner sdkws.MessageRevokedContent
	if err := json.Unmarshal([]byte(outer.Detail), &inner); err != nil {
		return nil, fmt.Errorf("msgcontent: ParseRevokeDetail detail JSON 解析失败: %w", err)
	}
	return &inner, nil
}
