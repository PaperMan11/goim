package msgcontent

import (
	"encoding/json"
	"fmt"
)

// ========== Quote（引用消息，contentType=114）==========

// QuoteContent 引用消息的 content JSON 结构。
// 服务端需要解析此结构来：
//   1. 读取 quoteMessage.seq，回源查被引用的原始消息，判断是否已被撤回
//   2. 若已撤回，把引用体改写为撤回通知
//
// JSON 示例：
//
//	{
//	  "text": "你说得对",
//	  "quoteMessage": {
//	    "clientMsgID": "xxx",
//	    "seq": 42,
//	    "contentType": 101,
//	    "content": "{\"text\":\"明天开会\"}",
//	    "sendID": "userA",
//	    "sendTime": 1720000000
//	  },
//	  "messageEntityList": [...]
//	}
type QuoteContent struct {
	Text              string          `json:"text,omitempty"`
	QuoteMessage      *QuoteMessage   `json:"quoteMessage,omitempty"`
	MessageEntityList json.RawMessage `json:"messageEntityList,omitempty"`
}

// QuoteMessage 引用体里嵌入的被引用消息摘要。
// 客户端可以多带字段（如 senderNickname、ex），服务端只关心这些。
type QuoteMessage struct {
	ClientMsgID string `json:"clientMsgID,omitempty"`
	Seq         int64  `json:"seq,omitempty"`
	ContentType int32  `json:"contentType,omitempty"`
	Content     string `json:"content,omitempty"`
	SendID      string `json:"sendID,omitempty"`
	SendTime    int64  `json:"sendTime,omitempty"`
}

// ParseQuote 把 contentType=114 的消息 content 解析成 QuoteContent。
// 返回 error 代表 JSON 格式完全不符合预期（字段缺失则用零值，不报错）。
func ParseQuote(content string) (*QuoteContent, error) {
	if content == "" {
		return nil, fmt.Errorf("msgcontent: ParseQuote 空 content")
	}
	var q QuoteContent
	if err := json.Unmarshal([]byte(content), &q); err != nil {
		return nil, fmt.Errorf("msgcontent: ParseQuote JSON 解析失败: %w", err)
	}
	return &q, nil
}

// Marshal 把 QuoteContent 序列化回 JSON 字符串，用于改写后回写 Msg.Content。
func (q *QuoteContent) Marshal() (string, error) {
	data, err := json.Marshal(q)
	if err != nil {
		return "", fmt.Errorf("msgcontent: QuoteContent.Marshal 失败: %w", err)
	}
	return string(data), nil
}

// QuoteMessage 序列化后的常见兼容补丁：
// 历史上部分客户端把空对象 {} 以 base64("e30=") 编码存在 content 里。
// 调用方在改写引用体前可以调用 FixQuoteEmptyObjectEncoding 做一次清理。
func (q *QuoteContent) FixQuoteEmptyObjectEncoding() bool {
	if q.QuoteMessage != nil && q.QuoteMessage.Content == "e30=" {
		q.QuoteMessage.Content = "{}"
		return true
	}
	return false
}
