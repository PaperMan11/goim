package msgcontent

import (
	"encoding/json"
	"fmt"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
)

// ========== AtText（@消息，contentType=106）==========

// AtTextContent @消息的 content JSON 结构。
// 服务端需要解析此结构来：
//  1. 找出被 @ 的用户列表，定向推送通知
//  2. 在会话未读计数里把 @ 消息标为"未读且需要提醒"
//
// JSON 示例：
//
//	{
//	  "text": "@user_002 加油吧",
//	  "atUsers": [{"atUserID": "user_002", "subTop": "user_002"}]
//	}
//
// @所有人时 atUserID 固定为 constant.AtAllString（"AtAllTag"）。
type AtTextContent struct {
	Text              string          `json:"text,omitempty"`
	AtUsers           []AtUser        `json:"atUsers,omitempty"`
	MessageEntityList json.RawMessage `json:"messageEntityList,omitempty"`
}

// AtUser 单个被 @ 用户的引用信息。
// AtUserID 为被 @ 的用户ID，SubTop 为该 @ 条目在文本中的定位标记。
type AtUser struct {
	AtUserID string `json:"atUserID,omitempty"`
	SubTop   string `json:"subTop,omitempty"`
}

// IsAtEveryone 判断是否 @ 所有人（atUserID == constant.AtAllString）。
func (a *AtTextContent) IsAtEveryone() bool {
	if a == nil {
		return false
	}
	for _, u := range a.AtUsers {
		if u.AtUserID == constant.AtAllString {
			return true
		}
	}
	return false
}

// AtUserIDs 返回所有被 @ 到的 userID 列表（不包含 @所有人 标记）。
func (a *AtTextContent) AtUserIDs() []string {
	if a == nil {
		return nil
	}
	ids := make([]string, 0, len(a.AtUsers))
	for _, u := range a.AtUsers {
		if u.AtUserID != "" && u.AtUserID != constant.AtAllString {
			ids = append(ids, u.AtUserID)
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
