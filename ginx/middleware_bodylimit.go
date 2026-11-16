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
			// 413（RFC 9110 payload too large）——此前借用 ErrInvalidArgument
			// 出 400，客户端无法区分「体太大」与「参数错」。
			Error(c, errorx.ErrContentTooLarge)
			c.Abort()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
