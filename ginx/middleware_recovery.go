package ginx

import (
	"net"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
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
				httpRequest, _ := httputil.DumpRequest(c.Request, false)

				if brokenPipe {
					logger.CtxErrorw(ctx, "broken pipe",
						"error", panicErr,
						"request", string(httpRequest),
					)
					c.Error(panicErr.(error)) //nolint:errcheck
					c.Abort()
					return
				}

				logger.CtxErrorw(ctx, "panic recovered",
					"panic_time", time.Now(),
					"error", panicErr,
					"request", string(httpRequest),
					"stack", string(debug.Stack()),
				)
				Error(c, errorx.ErrInternal.WithMessage("服务器内部异常，请稍后重试"))
			}
		}()
		c.Next()
	}
}

func isBrokenPipeError(err any) bool {
	if ne, ok := err.(*net.OpError); ok {
		if se, ok := ne.Err.(*os.SyscallError); ok {
			s := strings.ToLower(se.Error())
			return strings.Contains(s, "broken pipe") ||
				strings.Contains(s, "connection reset by peer")
		}
	}
	return false
}
