package errx

import (
	"errors"
	"fmt"

	"github.com/PaperMan11/goim/pkg/protocol/errinfo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrInfo 错误信息结构，支持包装原始错误
type ErrInfo struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	WrapMsg   string `json:"-"`
	WrapError error  `json:"-"`
}

func NewErrInfo(code int, message string) *ErrInfo {
	return &ErrInfo{
		Code:    code,
		Message: message,
	}
}

func ParseError(err error) *ErrInfo {
	if err == nil {
		return nil
	}

	// 进程内直接返回的 *ErrInfo（如 API 层中间件、本地 Wrap），保持原样
	var errInfo *ErrInfo
	if errors.As(err, &errInfo) {
		return errInfo
	}

	// RPC 调用返回的 gRPC status 错误
	if st, ok := status.FromError(err); ok {
		// 服务端返回的 *ErrInfo 会通过 details 携带业务错误码，优先还原
		for _, detail := range st.Details() {
			if d, ok := detail.(*errinfo.ErrDetail); ok {
				msg := d.GetMessage()
				if msg == "" {
					msg = st.Message()
				}
				return &ErrInfo{
					Code:    int(d.GetCode()),
					Message: msg,
				}
			}
		}

		// 普通 gRPC 错误：只返回 desc 中的真实报错信息，
		// 不暴露 "rpc error: code = ... desc = ..." 框架前缀
		return &ErrInfo{
			Code:    ErrCodeInternalError,
			Message: statusMessage(st, err),
		}
	}

	return InternalError.WrapWithError(err)
}

func statusMessage(st *status.Status, err error) string {
	if msg := st.Message(); msg != "" {
		return msg
	}
	return err.Error()
}

// GRPCStatus 让 *ErrInfo 在跨越 gRPC 边界时保留业务错误码。
// gRPC 服务端直接返回 *ErrInfo 时，gRPC 核心会自动调用本方法，
// 客户端可通过 status.Details 还原为原始错误。
func (e *ErrInfo) GRPCStatus() *status.Status {
	msg := e.Error()
	st := status.New(codes.Unknown, msg)
	detailed, err := st.WithDetails(&errinfo.ErrDetail{
		Code:    uint32(e.Code),
		Message: msg,
	})
	if err != nil {
		return st
	}
	return detailed
}

func (e *ErrInfo) Error() string {
	var result string
	if e.WrapError != nil {
		result = e.WrapError.Error()
	}
	if e.Message != "" {
		if result != "" {
			result = fmt.Sprintf("%s, %s", result, e.Message)
		} else {
			result = e.Message
		}
	}
	if e.WrapMsg != "" {
		if result != "" {
			result = fmt.Sprintf("%s, %s", result, e.WrapMsg)
		} else {
			result = e.WrapMsg
		}
	}
	return result
}

func (e *ErrInfo) Wrap(msg string) *ErrInfo {
	return &ErrInfo{
		Code:      e.Code,
		Message:   "",
		WrapMsg:   msg,
		WrapError: e,
	}
}

func (e *ErrInfo) WrapWithError(err error) *ErrInfo {
	return &ErrInfo{
		Code:      e.Code,
		Message:   flatMessage(err),
		WrapMsg:   "",
		WrapError: e,
	}
}

func (e *ErrInfo) Unwrap() error {
	return e.WrapError
}

// flatMessage 返回错误的对外文本：剥离 gRPC 错误的
// "rpc error: code = ... desc = ..." 框架前缀，只保留真实描述。
func flatMessage(err error) string {
	if err == nil {
		return ""
	}

	var errInfo *ErrInfo
	if errors.As(err, &errInfo) {
		return errInfo.Error()
	}

	if st, ok := status.FromError(err); ok {
		if msg := st.Message(); msg != "" {
			return msg
		}
	}

	return err.Error()
}
