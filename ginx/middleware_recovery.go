package ginx

import (
	"errors"
	"fmt"
	"net"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/errorx"
	"github.com/imxw/gopkg/logger"
)

// Recovery catches panics, logs the error with stack trace, and returns a
// unified 500 response. Broken-pipe errors are logged at ERROR level
// without a stack trace.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				ctx := c.Request.Context()
				brokenPipe := isBrokenPipeError(panicErr)

				var httpRequest string
				if dump, err := httputil.DumpRequest(c.Request, false); err == nil {
					httpRequest = string(dump)
				} else {
					httpRequest = fmt.Sprintf("method=%s path=%s (dump failed: %v)", c.Request.Method, c.Request.URL.Path, err)
				}

				if brokenPipe {
					logger.CtxErrorw(ctx, "broken pipe",
						"error", panicErr,
						"request", httpRequest,
					)
					c.Error(fmt.Errorf("%v", panicErr))
					c.Abort()
					return
				}

				logger.CtxErrorw(ctx, "panic recovered",
					"panic_time", time.Now(),
					"error", panicErr,
					"request", httpRequest,
					"stack", string(debug.Stack()),
				)
				Error(c, errorx.ErrInternal.WithMessage("服务器内部异常，请稍后重试"))
			}
		}()
		c.Next()
	}
}

func isBrokenPipeError(err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}
	opErr, ok := errors.AsType[*net.OpError](e)
	if !ok || opErr == nil {
		return false
	}
	sysErr, ok := errors.AsType[*os.SyscallError](opErr.Err)
	if !ok || sysErr == nil {
		return false
	}
	return sysErr.Err == syscall.EPIPE || sysErr.Err == syscall.ECONNRESET
}
