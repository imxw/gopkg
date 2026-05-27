package ginx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/errorx"
)

// MaxBytesReader returns middleware that wraps the request body with
// http.MaxBytesReader to prevent oversized payloads.
func MaxBytesReader(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

// BodyLimit returns middleware that rejects requests whose Content-Length
// exceeds maxBytes and caps the body reader. Defaults to 2MB.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = 2 << 20
	}
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			Error(c, errorx.ErrInvalidArgument.WithMessage("请求体过大"))
			c.Abort()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
