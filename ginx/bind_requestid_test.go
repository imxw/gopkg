package ginx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/imxw/gopkg/errorx"
)

// ---- RequestID：入站头传播 + 响应头回写 + RequestIDFrom 单一读路 ----

func requestIDRoundTrip(t *testing.T, inbound string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	cfg := DefaultEngineConfig()
	cfg.Mode = gin.TestMode
	r := NewEngine(cfg)
	r.GET("/probe", RequestID(), func(c *gin.Context) {
		c.String(http.StatusOK, RequestIDFrom(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if inbound != "" {
		req.Header.Set("X-Request-Id", inbound)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	return w, w.Header().Get("X-Request-Id")
}

func TestRequestID_InboundHeaderPropagated(t *testing.T) {
	w, echoed := requestIDRoundTrip(t, "gateway-req-1234567890")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gateway-req-1234567890", echoed, "合法入站头须原样传播且回写响应头")
	assert.Equal(t, "gateway-req-1234567890", w.Body.String(), "RequestIDFrom 读到的与回写一致")
}

func TestRequestID_NoInboundHeaderGenerates(t *testing.T) {
	w, echoed := requestIDRoundTrip(t, "")
	assert.Len(t, echoed, 36, "无入站头生成 UUID")
	assert.NotEmpty(t, w.Body.String())
}

func TestRequestID_IllegalInboundHeaderRejected(t *testing.T) {
	for _, bad := range []string{
		"short",                  // <8
		strings.Repeat("a", 129), // >128
		"has space-inside-1234",  // 非法字符
		"injection\u202e-123456", // 控制字符
	} {
		_, echoed := requestIDRoundTrip(t, bad)
		assert.Len(t, echoed, 36, "非法入站头 %q 须拒绝并生成 UUID", bad)
	}
}

func TestSanitizeRequestID_Matrix(t *testing.T) {
	assert.Equal(t, "abc-123_ok", sanitizeRequestID("  abc-123_ok  "))
	assert.Equal(t, "", sanitizeRequestID(""))
	assert.Equal(t, "", sanitizeRequestID("1234567"))
	assert.Equal(t, strings.Repeat("a", 128), sanitizeRequestID(strings.Repeat("a", 128)))
	assert.Equal(t, "", sanitizeRequestID(strings.Repeat("a", 129)))
	assert.Equal(t, "", sanitizeRequestID("bad*char-1234"))
}

// ---- ErrorWithDetails：错误腿结构化 details 载荷 ----

func TestErrorWithDetails_PayloadEmitted(t *testing.T) {
	c, w := newTestContext()

	ErrorWithDetails(c, errorx.ErrInvalidArgument, map[string]any{
		"errors": []string{"name required"},
		"warns":  []string{"legacy field"},
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(errorx.CodeInvalidArgument), body["code"])
	details, ok := body["details"].(map[string]any)
	require.True(t, ok, "details 须为对象")
	assert.Equal(t, []any{"name required"}, details["errors"])
	assert.Equal(t, []any{"legacy field"}, details["warns"])
}

func TestError_PlainOmitsDetails(t *testing.T) {
	c, w := newTestContext()
	Error(c, errorx.ErrInvalidArgument)
	assert.NotContains(t, w.Body.String(), "details")
}

// ---- 便捷族空 msg 守卫：空串保留哨兵默认消息（同族语义统一）----

func TestConvenienceHelpers_EmptyMsgKeepsDefault(t *testing.T) {
	cases := []struct {
		name     string
		call     func(c *gin.Context)
		wantMsg  string
		wantCode int
	}{
		{"BadRequest", func(c *gin.Context) { BadRequest(c, "") }, "Argument verification failed", http.StatusBadRequest},
		{"InternalError", func(c *gin.Context) { InternalError(c, "") }, "Internal server error", http.StatusInternalServerError},
		{"Unauthorized", func(c *gin.Context) { Unauthorized(c, "") }, "Unauthenticated", http.StatusUnauthorized},
		{"Forbidden", func(c *gin.Context) { Forbidden(c, "") }, "Permission denied", http.StatusForbidden},
		{"NotFound", func(c *gin.Context) { NotFound(c, "") }, "Resource not found", http.StatusNotFound},
		{"TokenExpired", func(c *gin.Context) { TokenExpired(c, "") }, "Token expired, please refresh", http.StatusUnauthorized},
		{"TokenInvalid", func(c *gin.Context) { TokenInvalid(c, "") }, "Token invalid, please re-authenticate", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := newTestContext()
			tc.call(c)
			assert.Equal(t, tc.wantCode, w.Code)
			var body map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.wantMsg, body["msg"], "%s 空 msg 须保留哨兵默认消息", tc.name)
		})
	}
}

// ---- Bind 族：一步绑定 + 失败自动出 400 信封 ----

type bindProbe struct {
	Name string `json:"name" binding:"required"`
	Age  int    `json:"age" binding:"gte=0"`
}

func bindEngine(bind func(*gin.Context, any) bool, setup func(*http.Request)) (*httptest.ResponseRecorder, bool, bindProbe) {
	var got bindProbe
	var ok bool
	r := newTestEngine()
	r.POST("/p", func(c *gin.Context) {
		ok = bind(c, &got)
	})
	req := httptest.NewRequest(http.MethodPost, "/p", nil)
	if setup != nil {
		setup(req)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, ok, got
}

func TestBindJSON_SetsFields(t *testing.T) {
	r := newTestEngine()
	var got bindProbe
	var ok bool
	r.POST("/p", func(c *gin.Context) {
		ok = BindJSON(c, &got)
	})
	req := httptest.NewRequest(http.MethodPost, "/p", strings.NewReader(`{"name":"n","age":3}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.True(t, ok)
	assert.Equal(t, "n", got.Name)
	assert.Equal(t, 3, got.Age)
}

func TestBindJSON_FailureWrites400(t *testing.T) {
	w, ok, _ := bindEngine(BindJSON, func(r *http.Request) {
		r.Header.Set("Content-Type", "application/json")
		r.Body = http.NoBody
	})
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotEmpty(t, body["msg"])
}

func TestBindQuery_SuccessAndFailure(t *testing.T) {
	type probe struct {
		Tag string `form:"tag" binding:"required"`
	}
	run := func(query string) (*httptest.ResponseRecorder, bool, probe) {
		var got probe
		var ok bool
		r := newTestEngine()
		r.GET("/q", func(c *gin.Context) {
			ok = BindQuery(c, &got)
		})
		req := httptest.NewRequest(http.MethodGet, "/q"+query, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w, ok, got
	}

	w, ok, got := run("?tag=x")
	assert.True(t, ok)
	assert.Equal(t, "x", got.Tag)
	assert.Equal(t, http.StatusOK, w.Code)

	w2, ok2, _ := run("")
	assert.False(t, ok2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestBindUri_SuccessAndFailure(t *testing.T) {
	type probe struct {
		ID int64 `uri:"id"`
	}
	var got probe
	var ok bool
	r := newTestEngine()
	r.GET("/u/:id", func(c *gin.Context) {
		ok = BindUri(c, &got)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/u/42", nil))

	assert.True(t, ok)
	assert.Equal(t, int64(42), got.ID)
}

// ---- RequireUserID / RequireOrgID：缺身份显式 401，而非静默 0 ----

func TestRequireUserID_MissingWrites401(t *testing.T) {
	c, w := newTestContext()
	id, ok := RequireUserID(c)
	assert.Zero(t, id)
	assert.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireOrgID_MissingWrites401(t *testing.T) {
	c, w := newTestContext()
	id, ok := RequireOrgID(c)
	assert.Zero(t, id)
	assert.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// newTestEngine 测试引擎：默认配置换 test 模式（零 CORS 配置会 panic）。
func newTestEngine() *gin.Engine {
	cfg := DefaultEngineConfig()
	cfg.Mode = gin.TestMode
	return NewEngine(cfg)
}
