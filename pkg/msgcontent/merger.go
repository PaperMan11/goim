package msgcontent

import (
	"encoding/json"
	"fmt"
)

// ========== Merger（合并转发，contentType=107）==========

// MergerContent 合并转发消息的 content JSON 结构。
// 合并转发把多条原消息打包成一条新消息发出，服务端需要知道哪些原消息被合并进来了，
// 以便做撤回联动、统计、会话过滤等。
//
// JSON 示例（不同客户端实现有差异，这是最通用的结构）：
//
//	{
//	  "title": "聊天记录",
//	  "abstractList": ["userA: 明天开会", "userB: 好的"],
//	  "messageIDList": ["msgID1", "msgID2", "msgID3"],
//	  "messageEntityList": [...]
//	}
//
// 说明：
//   - messageIDList 和 msgSeqList 是两种常见客户端实现，优先识别 messageIDList
//   - 客户端可以只给其中一种；服务端处理时需要同时兼容
type MergerContent struct {
	Title             string          `json:"title,omitempty"`
	AbstractList      []string        `json:"abstractList,omitempty"`
	MessageIDList     []string        `json:"messageIDList,omitempty"`
	MsgSeqList        []int64         `json:"msgSeqList,omitempty"`
	MessageEntityList json.RawMessage `json:"messageEntityList,omitempty"`
}

// ParseMerger 把 contentType=107 的消息 content 解析成 MergerContent。
func ParseMerger(content string) (*MergerContent, error) {
	if content == "" {
		return nil, fmt.Errorf("msgcontent: ParseMerger 空 content")
	}
	var m MergerContent
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		return nil, fmt.Errorf("msgcontent: ParseMerger JSON 解析失败: %w", err)
	}
	return &m, nil
}
