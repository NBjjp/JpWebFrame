package frame

import (
	"errors"
	"fmt"
	"github.com/NBjjp/JpWebFrame/jperror"
	"net/http"
	"runtime"
	"strings"
)

//打印错误堆栈详细信息
//runtime.Caller()报告当前go调用栈所执行的函数的文件和行号信息
//skip 需要跳过的栈帧数量
func DetailErr(err any) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:])
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%v\n", err))
	for _, pc := range pcs[0:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		sb.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return sb.String()
}
func Recovery(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				err2 := err.(error)
				if err2 != nil {
					var jpError *jperror.JpError
					if errors.As(err2, &jpError) {
						jpError.ExecResult()
						return
					}
				}

				ctx.Logger.Error(DetailErr(err))
				ctx.Fail(http.StatusInternalServerError, "Internal Serval Error")
			}
		}()
		next(ctx)
	}
}
