package ginx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SecurityHeaders 安全响应头：XSS 纵深的最低成本一层——CSP 兜住注入面
// （default-src 'none' 对 API/JSON 响应零副作用），frame-ancestors 收口
// 点击劫持，nosniff 防内容嗅探错判。放在引擎链最前：错误响应、限流拒
// 绝、CORS 拒绝等一切短路路径同样落头。
//
// 需要不同策略的消费方（如自 ginx 服务 HTML 文档）在下游中间件/处理器
// 里对同名 header 重新 Set 即可覆盖——本中间件刻意只 Set 不 Abort。

func newHeaderTestEngine(cfg EngineConfig) *gin.Engine { //nolint:unparam // 便于按档位扩展用例
	gin.SetMode(gin.TestMode)
	return NewEngine(cfg)
}

func TestSecurityHeaders_SetOnSuccess(t *testing.T) {
	r := newHeaderTestEngine(DefaultEngineConfig())
	r.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
		w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
}

// 短路路径全覆盖：CORS 拒绝（403）发生在引擎内建链中，若本中间件不在
// 其之前，被拒响应将裸奔——这正是它必须排在链首的原因。
func TestSecurityHeaders_SetOnCORSReject(t *testing.T) {
	cfg := DefaultEngineConfig()
	cfg.CORS.AllowOrigins = []string{"https://allowed.example"}
	r := newHeaderTestEngine(cfg)
	r.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "https://evil.example")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"),
		"CORS 拒绝的 403 同样落安全头（本中间件须先于 CORS）")
}

func TestSecurityHeaders_SetOnPanicRecovery(t *testing.T) {
	r := newHeaderTestEngine(DefaultEngineConfig())
	r.GET("/boom", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"),
		"recovery 500 同样落安全头")
}

func TestSecurityHeaders_Disable(t *testing.T) {
	cfg := DefaultEngineConfig()
	cfg.DisableSecurityHeaders = true
	r := newHeaderTestEngine(cfg)
	r.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	assert.Empty(t, w.Header().Get("Content-Security-Policy"),
		"自服务 HTML 的消费方可显式退出（退出后自行承担响应头策略）")
}
