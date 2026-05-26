package errorx

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	e := New(1001, 400, "BAD_REQUEST", "bad request")
	assert.Equal(t, 1001, e.BusinessCode)
	assert.Equal(t, 400, e.Status)
	assert.Equal(t, "BAD_REQUEST", e.Name)
	assert.Equal(t, "bad request", e.Message)
	assert.Nil(t, e.RawErr)
}

func TestError_String(t *testing.T) {
	e := New(1001, 400, "BAD_REQUEST", "bad request")
	s := e.Error()
	assert.Contains(t, s, "biz_code=1001")
	assert.Contains(t, s, "status=400")
	assert.Contains(t, s, "msg=bad request")
}

func TestError_StringWithRawErr(t *testing.T) {
	e := New(1001, 400, "BAD_REQUEST", "bad request").Wrap(errors.New("inner"))
	s := e.Error()
	assert.Contains(t, s, "raw_err=inner")
}

func TestError_StringWithExtra(t *testing.T) {
	e := New(1001, 400, "BAD_REQUEST", "bad request").WithExtra("field", "name")
	s := e.Error()
	assert.Contains(t, s, "extra=map[field:name]")
}

func TestWrap_Immutable(t *testing.T) {
	base := New(1001, 400, "BAD_REQUEST", "original")
	wrapped := base.Wrap(errors.New("inner"))

	assert.Nil(t, base.RawErr, "original should not be modified")
	assert.Equal(t, "inner", wrapped.RawErr.Error())
	assert.Equal(t, base.BusinessCode, wrapped.BusinessCode)
}

func TestWithMessage_Immutable(t *testing.T) {
	base := New(1001, 400, "BAD_REQUEST", "original")
	updated := base.WithMessage("updated")

	assert.Equal(t, "original", base.Message, "original should not be modified")
	assert.Equal(t, "updated", updated.Message)
}

func TestWithExtra_Immutable(t *testing.T) {
	base := New(1001, 400, "BAD_REQUEST", "original")
	withExtra := base.WithExtra("key", "val")

	assert.Nil(t, base.Extra, "original should not be modified")
	assert.Equal(t, "val", withExtra.Extra["key"])
}

func TestWithExtra_ChainDoesNotShareMap(t *testing.T) {
	base := New(1001, 400, "BAD_REQUEST", "original")
	e1 := base.WithExtra("k1", "v1")
	e2 := e1.WithExtra("k2", "v2")

	assert.Equal(t, "v1", e2.Extra["k1"], "chained extra should preserve previous")
	assert.Equal(t, "v2", e2.Extra["k2"])
	_, hasK2 := e1.Extra["k2"]
	assert.False(t, hasK2, "e1 should not have k2 from e2")
}

func TestData_Basic(t *testing.T) {
	e := New(1001, 400, "BAD_REQUEST", "bad request")
	data := e.Data()
	assert.Equal(t, 1001, data["code"])
	assert.Equal(t, "BAD_REQUEST", data["name"])
	assert.Equal(t, "bad request", data["msg"])
}

func TestData_WithExtra(t *testing.T) {
	e := New(1001, 400, "BAD_REQUEST", "bad request").WithExtra("field", "name")
	data := e.Data()
	assert.Equal(t, "name", data["field"])
}

func TestFromError_Nil(t *testing.T) {
	assert.Nil(t, FromError(nil))
}

func TestFromError_ErrorX(t *testing.T) {
	e := New(1001, 400, "BAD_REQUEST", "bad request")
	result := FromError(e)
	assert.Equal(t, e, result)
}

func TestFromError_StandardError(t *testing.T) {
	result := FromError(errors.New("std error"))
	assert.Equal(t, CodeInternal, result.BusinessCode)
	assert.Equal(t, http.StatusInternalServerError, result.Status)
	assert.NotNil(t, result.RawErr)
}

func TestFromError_WrappedErrorX(t *testing.T) {
	base := New(1001, 400, "BAD_REQUEST", "bad request")
	wrapped := base.Wrap(errors.New("inner"))
	result := FromError(wrapped)
	assert.Equal(t, 1001, result.BusinessCode)
}

func TestUnwrap(t *testing.T) {
	inner := errors.New("inner")
	e := New(1001, 400, "BAD_REQUEST", "outer").Wrap(inner)
	assert.Equal(t, inner, errors.Unwrap(e))
}

func TestIs_ValueSemantic(t *testing.T) {
	derived := ErrNotFound.WithMessage("user not found")
	assert.True(t, errors.Is(derived, ErrNotFound))
}

func TestIs_DifferentCode(t *testing.T) {
	assert.False(t, errors.Is(ErrNotFound, ErrInternal))
}

func TestIs_NonErrorX(t *testing.T) {
	assert.False(t, errors.Is(ErrNotFound, errors.New("some error")))
}

func TestErrorsIs_WrappedInner(t *testing.T) {
	inner := errors.New("inner")
	wrapped := ErrNotFound.Wrap(inner)
	assert.True(t, errors.Is(wrapped, inner))
}

func TestErrorsAs(t *testing.T) {
	err := New(1001, 400, "BAD_REQUEST", "base").Wrap(errors.New("inner"))
	var ex *ErrorX
	assert.True(t, errors.As(err, &ex))
	assert.Equal(t, 1001, ex.BusinessCode)
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     *ErrorX
		code    int
		status  int
		errName string
	}{
		{"OK", OK, CodeSuccess, 200, "SUCCESS"},
		{"ErrInternal", ErrInternal, CodeInternal, 500, "INTERNAL_ERROR"},
		{"ErrNotFound", ErrNotFound, CodeNotFound, 404, "NOT_FOUND"},
		{"ErrInvalidArgument", ErrInvalidArgument, CodeInvalidArgument, 400, "INVALID_ARGUMENT"},
		{"ErrUnauthorized", ErrUnauthorized, CodeUnauthorized, 401, "UNAUTHORIZED"},
		{"ErrForbidden", ErrForbidden, CodeForbidden, 403, "FORBIDDEN"},
		{"ErrConflict", ErrConflict, CodeConflict, 409, "CONFLICT"},
		{"ErrTooManyRequests", ErrTooManyRequests, CodeTooManyRequests, 429, "TOO_MANY_REQUESTS"},
		{"ErrTokenExpired", ErrTokenExpired, CodeTokenExpired, 401, "TOKEN_EXPIRED"},
		{"ErrTokenInvalid", ErrTokenInvalid, CodeTokenInvalid, 401, "TOKEN_INVALID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.BusinessCode)
			assert.Equal(t, tt.status, tt.err.Status)
			assert.Equal(t, tt.errName, tt.err.Name)
		})
	}
}

func TestWithTraceID(t *testing.T) {
	e := New(1001, 400, "BAD_REQUEST", "err").WithTraceID("trace-123")
	assert.Equal(t, "trace-123", e.TraceID)
	orig := New(1001, 400, "BAD_REQUEST", "err")
	assert.Equal(t, "", orig.TraceID)
}

func TestChainedOperations(t *testing.T) {
	e := ErrNotFound.
		WithMessage("user not found").
		WithExtra("user_id", "123").
		Wrap(errors.New("query failed"))

	assert.Equal(t, CodeNotFound, e.BusinessCode)
	assert.Equal(t, "NOT_FOUND", e.Name)
	assert.Equal(t, "user not found", e.Message)
	assert.Equal(t, "123", e.Extra["user_id"])
	require.NotNil(t, e.RawErr)
	assert.Equal(t, "query failed", e.RawErr.Error())
}
