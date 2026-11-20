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
// JSON 示例：
//
//	{
//	  "title": "历史聊天记录:2条",
//	  "isCollapseBirthdayTime": 0,
//	  "messageEntityList": [
//	    {"senderNickname":"Bob","senderID":"user_002","text":"上季 go-conference","contentType":101,"time":1650000000000}
//	  ],
//	  "nicknameList": ["Bob", "Alice"],
//	  "platformIDList": [1, 2]
//	}
type MergerContent struct {
	Title                  string          `json:"title,omitempty"`
	IsCollapseBirthdayTime int32           `json:"isCollapseBirthdayTime,omitempty"`
	MessageEntityList      json.RawMessage `json:"messageEntityList,omitempty"`
	NicknameList           []string        `json:"nicknameList,omitempty"`
	PlatformIDList         []int32         `json:"platformIDList,omitempty"`
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
