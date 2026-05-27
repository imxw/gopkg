package ginx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/errorx"
)

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
