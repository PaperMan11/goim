package errx

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrInfo_Wrap_Chain(t *testing.T) {
	baseErr := NewErrInfo(ErrCodeInternalError, "internal server error")

	chainErr := baseErr.Wrap("level1").Wrap("level2").Wrap("level3")

	errorStr := chainErr.Error()
	if !strings.Contains(errorStr, "level1") {
		t.Error("expected error string to contain 'level1'")
	}
	if !strings.Contains(errorStr, "level2") {
		t.Error("expected error string to contain 'level2'")
	}
	if !strings.Contains(errorStr, "level3") {
		t.Error("expected error string to contain 'level3'")
	}
	t.Log(errorStr)
}

func TestErrInfo_Unwrap(t *testing.T) {
	baseErr := NewErrInfo(ErrCodeInternalError, "internal server error")
	wrapped := baseErr.Wrap("wrapped")

	unwrapped := wrapped.Unwrap()
	if unwrapped != baseErr {
		t.Error("expected unwrapped to be baseErr")
	}
}

func TestErrInfo_Unwrap_Chain(t *testing.T) {
	baseErr := NewErrInfo(ErrCodeInternalError, "internal server error")
	chainErr := baseErr.Wrap("level1").Wrap("level2").Wrap("level3")

	current := chainErr
	for i := 3; i >= 1; i-- {
		unwrapped := current.Unwrap()
		if unwrapped == nil {
			t.Fatalf("expected unwrapped to be non-nil at level %d", i)
		}
		current = unwrapped.(*ErrInfo)
	}

	if current != baseErr {
		t.Error("expected final unwrapped to be baseErr")
	}
}

func TestErrInfo_Error_ChainFormat(t *testing.T) {
	baseErr := NewErrInfo(ErrCodeInternalError, "base error")
	chainErr := baseErr.Wrap("level1").WrapWithError(errors.New("level2")).WrapWithError(errors.New("level3")).Wrap("level4").Wrap("level5").WrapWithError(errors.New("level6"))

	errorStr := chainErr.Error()
	t.Log(errorStr)
	t.Log(errors.As(chainErr, &baseErr))
}

func TestParseError_PlainGRPCError_StripsPrefix(t *testing.T) {
	// 模拟 RPC 返回的普通错误：errors.New("group request not found")
	rpcErr := status.Error(codes.Unknown, "group request not found")

	errInfo := ParseError(rpcErr)
	if errInfo.Code != ErrCodeInternalError {
		t.Errorf("expected code %d, got %d", ErrCodeInternalError, errInfo.Code)
	}
	if errInfo.Error() != "group request not found" {
		t.Errorf("expected clean desc, got %q", errInfo.Error())
	}
	if strings.Contains(errInfo.Error(), "rpc error") {
		t.Errorf("message must not contain rpc error prefix, got %q", errInfo.Error())
	}
}

func TestParseError_ErrxRoundTripThroughGRPC(t *testing.T) {
	// 服务端逻辑返回的自定义 errx（含 Wrap 上下文）
	serverErr := ArgsError.Wrap("userID and groupID are required")

	// gRPC 核心通过 GRPCStatus() 转成 status error 发给客户端，
	// 等价于 grpc server 内部调用 status.FromError(serverErr)
	rpcErr := serverErr.GRPCStatus().Err()

	errInfo := ParseError(rpcErr)
	if errInfo.Code != ErrCodeArgsError {
		t.Errorf("expected code %d, got %d", ErrCodeArgsError, errInfo.Code)
	}
	if !strings.Contains(errInfo.Error(), "input parameter error") ||
		!strings.Contains(errInfo.Error(), "userID and groupID are required") {
		t.Errorf("expected full chain message, got %q", errInfo.Error())
	}
	if strings.Contains(errInfo.Error(), "rpc error") {
		t.Errorf("message must not contain rpc error prefix, got %q", errInfo.Error())
	}
}

func TestParseError_DirectErrInfo(t *testing.T) {
	errInfo := ParseError(TokenNotExistError)
	if errInfo != TokenNotExistError {
		t.Error("expected the original *ErrInfo to be returned")
	}
}

func TestWrapWithError_GRPCError_StripsPrefix(t *testing.T) {
	rpcErr := status.Error(codes.Unavailable, `connection error: desc = "transport: dial refused"`)
	errInfo := TokenNotValidYetError.WrapWithError(rpcErr)

	if !strings.HasPrefix(errInfo.Error(), "token is not valid yet") {
		t.Errorf("expected base message first, got %q", errInfo.Error())
	}
	if strings.Contains(errInfo.Error(), "rpc error: code") {
		t.Errorf("wrapped message must not contain rpc prefix, got %q", errInfo.Error())
	}
	if !strings.Contains(errInfo.Error(), "dial refused") {
		t.Errorf("expected grpc desc to be kept, got %q", errInfo.Error())
	}
}

func TestParseError_Nil(t *testing.T) {
	if errInfo := ParseError(nil); errInfo != nil {
		t.Errorf("expected nil, got %v", errInfo)
	}
}
