package msgcontent

import (
	"encoding/json"
	"fmt"
)

// ========== AtText（@消息，contentType=106）==========

// AtTextContent @消息的 content JSON 结构。
// 服务端需要解析此结构来：
//   1. 找出被 @ 的用户列表，定向推送通知
//   2. 在会话未读计数里把 @ 消息标为"未读且需要提醒"
//
// JSON 示例：
//
//	{
//	  "text": "@所有人 @userA 明天开会",
//	  "atUsers": [
//	    {"userID": "userA", "nickname": "A", "offset": 3, "length": 6},
//	    {"userID": "everyone", "nickname": "@所有人", "offset": 0, "length": 4}
//	  ],
//	  "messageEntityList": [...]
//	}
type AtTextContent struct {
	Text              string          `json:"text,omitempty"`
	AtUsers           []AtUser        `json:"atUsers,omitempty"`
	MessageEntityList json.RawMessage `json:"messageEntityList,omitempty"`
}

// AtUser 单个被 @ 用户的引用信息。
// Offset/Length 指出该 @ 名字在 Text 里的位置，用于富文本高亮显示。
type AtUser struct {
	UserID   string `json:"userID,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	Offset   int32  `json:"offset,omitempty"`
	Length   int32  `json:"length,omitempty"`
}

// IsAtEveryone 判断是否 @ 所有人（userID="everyone" 是约定）。
func (a *AtTextContent) IsAtEveryone() bool {
	if a == nil {
		return false
	}
	for _, u := range a.AtUsers {
		if u.UserID == "everyone" {
			return true
		}
	}
	return false
}

// AtUserIDs 返回所有被 @ 到的 userID 列表（不包含 "everyone"）。
func (a *AtTextContent) AtUserIDs() []string {
	if a == nil {
		return nil
	}
	ids := make([]string, 0, len(a.AtUsers))
	for _, u := range a.AtUsers {
		if u.UserID != "" && u.UserID != "everyone" {
			ids = append(ids, u.UserID)
		}
	}
	return ids
}

// ParseAtText 把 contentType=106 的消息 content 解析成 AtTextContent。
func ParseAtText(content string) (*AtTextContent, error) {
	if content == "" {
		return nil, fmt.Errorf("msgcontent: ParseAtText 空 content")
	}
	var a AtTextContent
	if err := json.Unmarshal([]byte(content), &a); err != nil {
		return nil, fmt.Errorf("msgcontent: ParseAtText JSON 解析失败: %w", err)
	}
	return &a, nil
}
