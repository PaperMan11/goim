package errx

import (
	"errors"
	"fmt"
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
	var errInfo *ErrInfo
	if !errors.As(err, &errInfo) {
		errInfo = InternalError.WrapWithError(err)
	}
	return errInfo
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
		Message:   err.Error(),
		WrapMsg:   "",
		WrapError: e,
	}
}

func (e *ErrInfo) Unwrap() error {
	return e.WrapError
}
