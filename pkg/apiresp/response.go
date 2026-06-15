package apiresp

import (
	"net/http"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Body 定义API响应体结构
type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func (b *Body) Marshal() []byte {
	marshal, _ := jsonx.Marshal(b)
	return marshal
}

// Success 构建成功响应
func Success(w http.ResponseWriter, data interface{}) {
	httpx.OkJson(w, &Body{
		Code: errx.Success.Code,
		Msg:  errx.Success.Message,
		Data: data,
	})
}

// SuccessWithMsg 构建带自定义消息的成功响应
func SuccessWithMsg(w http.ResponseWriter, msg string, data interface{}) {
	httpx.OkJson(w, &Body{
		Code: errx.Success.Code,
		Msg:  msg,
		Data: data,
	})
}

func Error(w http.ResponseWriter, err error) {
	errInfo := errx.ParseError(err)
	httpx.OkJson(w, &Body{
		Code: errInfo.Code,
		Msg:  errInfo.Message,
	})
}
