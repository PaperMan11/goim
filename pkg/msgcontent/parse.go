package msgcontent

import (
	"fmt"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
)

// ========== 统一入口 ==========

// Parse 按 contentType 分发到对应解析器，返回通用的 any 类型。
// 调用方通过 contentType 再做一次类型断言拿具体结构。
//
// 对于不需要服务端解析的 contentType（Text/Picture/Video/File/Card/Custom 等），
// 返回 nil, ErrUnknownContentType，表示"服务端不解析这个类型，交给客户端处理"。
func Parse(contentType int32, content string) (any, error) {
	switch contentType {
	case constant.Quote:
		return ParseQuote(content)
	case constant.MsgRevokeNotification:
		return ParseRevoke(content)
	case constant.AtText:
		return ParseAtText(content)
	case constant.Merger:
		return ParseMerger(content)
	default:
		return nil, ErrUnknownContentType
	}
}

// ErrUnknownContentType 表示该 contentType 的 content 由客户端定义，服务端不做解析。
var ErrUnknownContentType = fmt.Errorf("msgcontent: 此 contentType 不在服务端解析范围内")
