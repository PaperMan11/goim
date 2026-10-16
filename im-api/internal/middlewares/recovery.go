package middlewares

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logc"
)

const (
	// stackDepth 单次采集的最大栈帧深度
	stackDepth = 64
	// stackMaxFrames 日志中最多输出的业务栈帧数量，避免调用栈过长
	stackMaxFrames = 16
	// stackSkip 从 recover 调用点起跳过的栈帧数（runtime.Callers 语义），
	// 与 gin recovery 的 stackSkip 一致，用于跳过 recover 闭包与 runtime.gopanic
	stackSkip = 3
)

// Recovery 捕获 panic，记录精简后的调用栈并返回 500。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logc.Errorf(c.Request.Context(), "panic recovered: %v\n%s", r, compactStack(stackSkip))
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

// compactStack 采集调用栈并压缩为紧凑格式，每行仅输出「函数 文件:行号」。
// 与 gin recovery 直接打印完整栈不同，这里跳过 panic 机制与框架内部帧，
// 只保留业务栈帧，并限制最大帧数，避免日志中的调用栈过长。
func compactStack(skip int) string {
	pcs := make([]uintptr, stackDepth)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return ""
	}

	frames := runtime.CallersFrames(pcs[:n])
	var b strings.Builder
	started := false
	for written := 0; written < stackMaxFrames; {
		frame, more := frames.Next()
		if !started {
			// panic 机制与框架内部帧不输出，定位到业务代码后才开始记录
			if isRuntimeFrame(frame.Function) || isFrameworkFrame(frame.Function) {
				if !more {
					return b.String()
				}
				continue
			}
			started = true
		} else if isRuntimeFrame(frame.Function) || isFrameworkFrame(frame.Function) {
			// 回到运行时/框架层，说明业务调用链已结束
			break
		}

		fmt.Fprintf(&b, "%s %s:%d\n", shortFuncName(frame.Function), filepath.Base(frame.File), frame.Line)
		written++
		if !more {
			break
		}
	}
	return b.String()
}

// isRuntimeFrame 判断是否为运行时/panic 机制相关栈帧
func isRuntimeFrame(name string) bool {
	return strings.HasPrefix(name, "runtime.") || strings.HasPrefix(name, "runtime/")
}

// isFrameworkFrame 判断是否为 Web 框架内部栈帧（gin 或 Recovery 自身）。
// Recovery 被内联时闭包符号会带上调用方的名字（如 main.Recovery.func1），
// 因此这里用 Contains 匹配其闭包名。
func isFrameworkFrame(name string) bool {
	return strings.HasPrefix(name, "github.com/gin-gonic/gin.") ||
		strings.Contains(name, ".Recovery.func")
}

// shortFuncName 去掉函数名中的模块路径，仅保留「包.函数」，便于阅读
func shortFuncName(name string) string {
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	return strings.ReplaceAll(name, "·", ".")
}
