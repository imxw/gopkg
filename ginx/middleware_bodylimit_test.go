package ginx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bodyLimitRouter(maxBytes int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimit(maxBytes))
	r.POST("/upload", func(c *gin.Context) {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"size": buf.Len()})
	})
	return r
}

func TestBodyLimit_RejectsByContentLength(t *testing.T) {
	r := bodyLimitRouter(16)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(strings.Repeat("x", 32)))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Content-Length 超限直接 400")
}

func TestBodyLimit_ChunkedBodyCappedByMaxBytesReader(t *testing.T) {
	// 无 Content-Length（chunked）：第一道 ContentLength 检查放行，
	// MaxBytesReader 在读取侧硬闸——超出容量后 Body 读取返回
	// *http.MaxBytesError（是否向客户端呈现 4xx 由 handler 决定，
	// 中间件保证「读不到静默截断之外的东西」）。
	var readErr error
	r := bodyLimitRouter(16)
	r.POST("/probe", func(c *gin.Context) {
		buf := new(bytes.Buffer)
		_, readErr = buf.ReadFrom(c.Request.Body)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(strings.Repeat("x", 32)))
	req.ContentLength = 0
	req.Header.Set("Transfer-Encoding", "chunked")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Error(t, readErr, "超出容量的体读取必须报错（不静默截断）")
	var maxErr *http.MaxBytesError
	assert.ErrorAs(t, readErr, &maxErr)
	assert.Equal(t, int64(16), maxErr.Limit)
}

func TestBodyLimit_UnderLimitPasses(t *testing.T) {
	r := bodyLimitRouter(16)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("hello"))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"size":5`, "限内体完整到达 handler")
}
