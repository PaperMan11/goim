package errx

import (
	"errors"
	"strings"
	"testing"
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
