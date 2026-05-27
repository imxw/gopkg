package ginx

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/logger"
)

// AccessLog logs each HTTP request with structured fields and selects log
// level based on the response status code.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		ctx := c.Request.Context()
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		logger.CtxDebugw(ctx, "HTTP request incoming",
			"method", method,
			"path", path,
			"query", query,
			"client_ip", clientIP,
			"user_agent", userAgent,
		)

		c.Next()

		costMs := time.Since(startTime).Milliseconds()
		httpCode := c.Writer.Status()
		logFields := []interface{}{
			"method", method,
			"path", path,
			"query", query,
			"client_ip", clientIP,
			"user_agent", userAgent,
			"http_code", httpCode,
			"cost_ms", costMs,
		}

		switch {
		case httpCode >= 200 && httpCode < 400:
			logger.CtxInfow(ctx, "HTTP request completed", logFields...)
		case httpCode >= 400 && httpCode < 500:
			logger.CtxWarnw(ctx, "HTTP request client error", logFields...)
		default:
			logger.CtxErrorw(ctx, "HTTP request server error", logFields...)
		}
	}
}
