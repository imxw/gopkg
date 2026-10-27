package ginx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/imxw/gopkg/logger"
)

// AccessLog 的 query 脱敏契约：凭据类参数（token/secret/password 等）的
// **值**在日志中替换为 ***（键保留供排障），防止凭据经访问日志泄露——
// 与 WS 鉴权约定（禁止凭据进 query）配套的纵深：约定被破例时日志不放大。
// 普通参数原样保留。

func newObservedAccessLogEngine(t *testing.T) (*gin.Engine, *observer.ObservedLogs) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zapcore.DebugLevel)
	logger.SetLogger(zap.New(core))
	t.Cleanup(func() { logger.SetLogger(nil) })

	r := gin.New()
	r.Use(AccessLog())
	r.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r, logs
}

func queryField(t *testing.T, logs *observer.ObservedLogs, msg string) string {
	t.Helper()
	found := ""
	for _, e := range logs.All() {
		if e.Message != msg {
			continue
		}
		for _, f := range e.Context {
			if f.Key == "query" {
				found = f.String // 取最后一条（多次请求场景对齐最后一次）
			}
		}
	}
	if found == "" && !hasEntry(logs, msg) {
		t.Fatalf("log entry %q with query field not found", msg)
	}
	return found
}

func hasEntry(logs *observer.ObservedLogs, msg string) bool {
	for _, e := range logs.All() {
		if e.Message == msg {
			return true
		}
	}
	return false
}

func TestAccessLog_SanitizesSensitiveQueryParams(t *testing.T) {
	r, logs := newObservedAccessLogEngine(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok?x=1&access_token=abc.def&apikey=xyz&session=ok", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	q := queryField(t, logs, "HTTP request completed")
	assert.Contains(t, q, "x=1", "普通参数原样保留")
	assert.Contains(t, q, "session=ok")
	assert.Contains(t, q, "access_token=***", "凭据参数值替换为 ***（键保留）")
	assert.Contains(t, q, "apikey=***")
	assert.NotContains(t, q, "abc.def")
	assert.NotContains(t, q, "xyz")
}

func TestAccessLog_SanitizedOnBothLogLegs(t *testing.T) {
	r, logs := newObservedAccessLogEngine(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok?password=hunter2", nil)
	r.ServeHTTP(w, req)

	assert.NotContains(t, queryField(t, logs, "HTTP request incoming"), "hunter2")
	assert.NotContains(t, queryField(t, logs, "HTTP request completed"), "hunter2")
}

func TestAccessLog_EmptyAndCleanQueryUntouched(t *testing.T) {
	r, logs := newObservedAccessLogEngine(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok?page=1&size=20", nil))

	q := queryField(t, logs, "HTTP request completed")
	assert.Contains(t, q, "page=1&size=20", "无凭据参数时不改写")

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/ok", nil))
	assert.Empty(t, queryField(t, logs, "HTTP request completed"), "无 query 时保持空串")
}
