package ginx

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/imxw/gopkg/contextx"
	"github.com/imxw/gopkg/logger"
)

// RequestID generates a unique trace ID for each request and injects it
// into both the context and the logger.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := uuid.NewString()
		ctx := contextx.WithTraceID(c.Request.Context(), traceID)
		ctx = logger.WithTraceID(ctx, traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
