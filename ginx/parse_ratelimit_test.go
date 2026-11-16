package ginx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Parse 族：参数解析 + 400 信封 ----

func probeEngine(t *testing.T, path string, h gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	r := newTestEngine()
	r.GET(path, h)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestParseIDParam(t *testing.T) {
	w := probeEngine(t, "/i/:id", func(c *gin.Context) {
		id, ok := ParseIDParam(c, "id", "无效的标识")
		if !ok {
			return
		}
		c.JSON(200, gin.H{"id": id})
	})
	// 经 pathTypesDirective 无参形式不便造合法 id——直接两种探测：
	// 这里路径无参命中 404 前置检查；合法形态在下方单独造。
	_ = w

	r := newTestEngine()
	var got int64
	var ok bool
	r.GET("/i/:id", func(c *gin.Context) {
		got, ok = ParseIDParam(c, "id", "无效的标识")
		if ok {
			c.Status(200)
		}
	})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/i/42", nil))
	assert.True(t, ok)
	assert.Equal(t, int64(42), got)
	assert.Equal(t, 200, w2.Code)

	ok = false
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest("GET", "/i/abc", nil))
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w3.Code)
	assert.Contains(t, w3.Body.String(), "无效的标识")
}

func TestParseIDQuery(t *testing.T) {
	r := newTestEngine()
	var got int64
	var ok bool
	r.GET("/q", func(c *gin.Context) {
		got, ok = ParseIDQuery(c, "projectId", "无效的项目")
		if ok {
			c.Status(200)
		}
	})

	for _, tc := range []struct {
		query  string
		wantOK bool
		wantID int64
	}{
		{"", true, 0},            // 缺省 = 无过滤哨兵
		{"?projectId=", true, 0}, // 空串同缺省
		{"?projectId=7", true, 7},
		{"?projectId=0", true, 0}, // 0 合法（调用方语义同无过滤）
	} {
		ok = false
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/q"+tc.query, nil))
		assert.True(t, ok, tc.query)
		assert.Equal(t, tc.wantID, got, tc.query)
		assert.Equal(t, 200, w.Code, tc.query)
	}

	ok = true
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/q?projectId=-1", nil))
	assert.False(t, ok, "负值拒绝")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	ok = true
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/q?projectId=abc", nil))
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestParseUUIDParam(t *testing.T) {
	r := newTestEngine()
	var got [16]byte
	var ok bool
	r.GET("/u/:id", func(c *gin.Context) {
		u, o := ParseUUIDParam(c, "id", "无效的 UUID")
		got, ok = u, o
		if ok {
			c.Status(200)
		}
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/u/550e8400-e29b-41d4-a716-446655440000", nil))
	assert.True(t, ok)
	assert.Equal(t, 200, w.Code)
	assert.NotEqual(t, [16]byte{}, got)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/u/not-a-uuid", nil))
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Equal(t, uuid.Nil, uuid.UUID(got))
}

// ---- 限流中间件：miniredis 承载 redis_rate 协议 ----

func newTestLimiter(t *testing.T, prefix string) (*RedisLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	return NewRedisLimiter(redis.NewClient(&redis.Options{Addr: mr.Addr()}), prefix), mr
}

func TestRateLimit_AllowedAndDenied(t *testing.T) {
	lim, _ := newTestLimiter(t, "test:rl")

	r := newTestEngine()
	r.GET("/p", RateLimit(lim, RateLimitConfig{
		KeyFn:   func(c *gin.Context) string { return "pat-1" },
		LimitFn: func(c *gin.Context) (int64, int64) { return 3, 3 }, // 3/min
	}), func(c *gin.Context) {
		c.Status(200)
	})

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/p", nil))
		assert.Equal(t, 200, w.Code, "第 %d 次", i+1)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/p", nil))
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "超档 429")
	assert.Contains(t, w.Body.String(), "TOO_MANY_REQUESTS")
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	// 第 4 次 remaining=0。
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimit_KeysAreIndependent(t *testing.T) {
	lim, _ := newTestLimiter(t, "test:rl")
	r := newTestEngine()
	r.GET("/p", RateLimit(lim, RateLimitConfig{
		KeyFn:   func(c *gin.Context) string { return c.GetHeader("X-Key") },
		LimitFn: func(c *gin.Context) (int64, int64) { return 1, 1 },
	}), func(c *gin.Context) { c.Status(200) })

	// 直接构造两个 key 的请求。
	do := func(key string) int {
		req := httptest.NewRequest("GET", "/p", nil)
		req.Header.Set("X-Key", key)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	assert.Equal(t, 200, do("a"))
	assert.Equal(t, 429, do("a"), "key a 已耗尽")
	assert.Equal(t, 200, do("b"), "key b 独立计数")
}

func TestRateLimit_SkipPaths(t *testing.T) {
	lim, _ := newTestLimiter(t, "test:rl")
	r := newTestEngine()
	r.GET("/p", RateLimit(lim, RateLimitConfig{
		KeyFn:   func(c *gin.Context) string { return "" }, // 空键跳过
		LimitFn: func(c *gin.Context) (int64, int64) { return 0, 0 },
	}), func(c *gin.Context) { c.Status(200) })

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/p", nil))
		assert.Equal(t, 200, w.Code, "空键/零档位恒放行")
	}
}

func TestRateLimit_FailOpenAndClosed(t *testing.T) {
	// 存储故障：limiter 指向已关闭的 redis。
	mr := miniredis.RunT(t)
	lim := NewRedisLimiter(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "test:rl")
	mr.Close()
	time.Sleep(50 * time.Millisecond)

	build := func(failOpen bool) *gin.Engine {
		r := newTestEngine()
		r.GET("/p", RateLimit(lim, RateLimitConfig{
			KeyFn:    func(c *gin.Context) string { return "k" },
			LimitFn:  func(c *gin.Context) (int64, int64) { return 5, 5 },
			FailOpen: failOpen,
		}), func(c *gin.Context) { c.Status(200) })
		return r
	}

	// 计数不可用 fail-open 放行（D48f 默认语义）。
	w := httptest.NewRecorder()
	build(true).ServeHTTP(w, httptest.NewRequest("GET", "/p", nil))
	assert.Equal(t, 200, w.Code)

	// fail-closed 500。
	w2 := httptest.NewRecorder()
	build(false).ServeHTTP(w2, httptest.NewRequest("GET", "/p", nil))
	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}

func TestRedisLimiter_TakeValidation(t *testing.T) {
	lim, _ := newTestLimiter(t, "test:rl")
	_, _, _, err := lim.Take(context.Background(), "k", 0, 5)
	require.Error(t, err, "capacity 必须 > 0")
	_, _, _, err = lim.Take(context.Background(), "k", 5, 0)
	require.Error(t, err, "refill 必须 > 0")
	allowed, _, _, err := lim.Take(context.Background(), "k", 5, 5)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.True(t, strings.Contains("x", "x")) // keep strings import honest
}
