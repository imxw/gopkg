package ginx

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// 覆盖缺口补测：响应便捷族（SuccessWithMessage/InternalError/…）、
// Recovery 的 broken-pipe 分类、isBrokenPipeError 判定矩阵。

func TestResponseConvenienceHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// SuccessWithMessage：自定义 msg 腿。
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	SuccessWithMessage(c, gin.H{"ok": true}, "已受理")
	assert.Contains(t, w.Body.String(), `"msg":"已受理"`)

	// 错误便捷族逐个钉状态码（都走 Error 统一信封）。
	cases := []struct {
		name    string
		call    func(*gin.Context)
		status  int
		errName string
	}{
		{"internal", func(c *gin.Context) { InternalError(c, "boom") }, 500, "INTERNAL_ERROR"},
		{"unauthorized", func(c *gin.Context) { Unauthorized(c, "no") }, 401, "UNAUTHORIZED"},
		{"tokenExpired", func(c *gin.Context) { TokenExpired(c, "old") }, 401, "TOKEN_EXPIRED"},
		{"tokenInvalid", func(c *gin.Context) { TokenInvalid(c, "") }, 401, "TOKEN_INVALID"},
		{"forbidden", func(c *gin.Context) { Forbidden(c, "no") }, 403, "FORBIDDEN"},
		{"notFound", func(c *gin.Context) { NotFound(c, "gone") }, 404, "NOT_FOUND"},
		{"badRequest", func(c *gin.Context) { BadRequest(c, "bad") }, 400, "INVALID_ARGUMENT"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		tc.call(c)
		assert.Equal(t, tc.status, w.Code, tc.name)
		assert.Contains(t, w.Body.String(), `"error":"`+tc.errName+`"`, tc.name)
	}

	// TokenExpired/TokenInvalid 的空 msg 腿（不覆盖 WithMessage 分支）：
	// 上面 tokenInvalid 用例已传空串，这里补 tokenExpired 空串对照。
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	TokenExpired(c2, "")
	assert.Equal(t, 401, w2.Code)
}

// broken-pipe 分类：客户端提前断连（EPIPE/ECONNRESET）不打堆栈、按 ERROR
// 落「broken pipe」；其余 panic 走 500 + 堆栈。
func TestRecovery_BrokenPipeVsPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(Recovery())
		r.GET("/boom", func(c *gin.Context) { panic("boom") })
		r.GET("/pipe", func(c *gin.Context) {
			panic(&net.OpError{Op: "write", Err: &os.SyscallError{Syscall: "write", Err: syscall.EPIPE}})
		})
		return r
	}

	w := httptest.NewRecorder()
	newRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	w2 := httptest.NewRecorder()
	newRouter().ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/pipe", nil))
	// broken-pipe 腿：吞 panic 不再上抛（客户端已断，写 500 也无人收），
	// 状态停在默认 200/半途——契约是「不 crash + ERROR 日志」。
	assert.Equal(t, http.StatusOK, w2.Code, "broken pipe 不写 500（对端已断）")
}

func TestIsBrokenPipeError_Matrix(t *testing.T) {
	assert.False(t, isBrokenPipeError("not an error"))
	assert.False(t, isBrokenPipeError(errors.New("plain")))
	assert.False(t, isBrokenPipeError(&net.OpError{Op: "read", Err: errors.New("other")}))

	pipe := &net.OpError{Op: "write", Err: &os.SyscallError{Syscall: "write", Err: syscall.EPIPE}}
	assert.True(t, isBrokenPipeError(pipe))
	reset := &net.OpError{Op: "write", Err: &os.SyscallError{Syscall: "write", Err: syscall.ECONNRESET}}
	assert.True(t, isBrokenPipeError(reset))
}

// SetupRouter（testing.go）：测试模式引擎带 Recovery。
func TestSetupRouter_HasRecovery(t *testing.T) {
	r := SetupRouter()
	w := httptest.NewRecorder()
	r.GET("/boom", func(c *gin.Context) { panic("x") })
	assert.NotPanics(t, func() {
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
