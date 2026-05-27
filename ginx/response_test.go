package ginx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/imxw/gopkg/errorx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func TestError_NilErr(t *testing.T) {
	c, w := newTestContext()

	Error(c, nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(errorx.CodeInternal), body["code"])
}

func TestError_KnownErr(t *testing.T) {
	c, w := newTestContext()

	Error(c, errorx.ErrNotFound)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(errorx.CodeNotFound), body["code"])
}

func TestSuccess_NoData(t *testing.T) {
	c, w := newTestContext()

	SuccessNoData(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPaginationSuccess_NilList(t *testing.T) {
	c, w := newTestContext()

	PaginationSuccess[int](c, nil, 0, 1, 10)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	data := body["data"].(map[string]any)
	list := data["list"]
	assert.NotNil(t, list, "list should not be null")
}

func TestPaginationSuccess_WithItems(t *testing.T) {
	c, w := newTestContext()

	PaginationSuccess(c, []string{"a", "b"}, 2, 1, 10)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(2), data["total"])
}

func TestBadRequest(t *testing.T) {
	c, w := newTestContext()

	BadRequest(c, "参数错误")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForbidden(t *testing.T) {
	c, w := newTestContext()

	Forbidden(c, "权限不足")

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUnauthorized(t *testing.T) {
	c, w := newTestContext()

	Unauthorized(c, "未认证")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNotFound(t *testing.T) {
	c, w := newTestContext()

	NotFound(c, "资源不存在")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTokenExpired(t *testing.T) {
	c, w := newTestContext()

	TokenExpired(c, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(errorx.CodeTokenExpired), body["code"])
}

func TestTokenInvalid(t *testing.T) {
	c, w := newTestContext()

	TokenInvalid(c, "无效token")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(errorx.CodeTokenInvalid), body["code"])
}
