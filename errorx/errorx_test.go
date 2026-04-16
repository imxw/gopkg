package errorx

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	e := New(1001, 400, "bad request")
	assert.Equal(t, 1001, e.BusinessCode)
	assert.Equal(t, 400, e.HTTPCode)
	assert.Equal(t, "bad request", e.Message)
	assert.Nil(t, e.RawErr)
}

func TestError_String(t *testing.T) {
	e := New(1001, 400, "bad request")
	s := e.Error()
	assert.Contains(t, s, "biz_code=1001")
	assert.Contains(t, s, "http_code=400")
	assert.Contains(t, s, "msg=bad request")
}

func TestError_StringWithRawErr(t *testing.T) {
	e := New(1001, 400, "bad request").Wrap(errors.New("inner"))
	s := e.Error()
	assert.Contains(t, s, "raw_err=inner")
}

func TestError_StringWithExtra(t *testing.T) {
	e := New(1001, 400, "bad request").WithExtra("field", "name")
	s := e.Error()
	assert.Contains(t, s, "extra=map[field:name]")
}

func TestWrap_Immutable(t *testing.T) {
	base := New(1001, 400, "original")
	wrapped := base.Wrap(errors.New("inner"))

	assert.Nil(t, base.RawErr, "original should not be modified")
	assert.Equal(t, "inner", wrapped.RawErr.Error())
	assert.Equal(t, base.BusinessCode, wrapped.BusinessCode)
}

func TestWithMessage_Immutable(t *testing.T) {
	base := New(1001, 400, "original")
	updated := base.WithMessage("updated")

	assert.Equal(t, "original", base.Message, "original should not be modified")
	assert.Equal(t, "updated", updated.Message)
}

func TestWithExtra_Immutable(t *testing.T) {
	base := New(1001, 400, "original")
	withExtra := base.WithExtra("key", "val")

	assert.Nil(t, base.Extra, "original should not be modified")
	assert.Equal(t, "val", withExtra.Extra["key"])
}

func TestWithExtra_ChainDoesNotShareMap(t *testing.T) {
	base := New(1001, 400, "original")
	e1 := base.WithExtra("k1", "v1")
	e2 := e1.WithExtra("k2", "v2")

	assert.Equal(t, "v1", e2.Extra["k1"], "chained extra should preserve previous")
	assert.Equal(t, "v2", e2.Extra["k2"])
	_, hasK2 := e1.Extra["k2"]
	assert.False(t, hasK2, "e1 should not have k2 from e2")
}

func TestData_Basic(t *testing.T) {
	e := New(1001, 400, "bad request")
	data := e.Data()
	assert.Equal(t, 1001, data["code"])
	assert.Equal(t, "bad request", data["msg"])
}

func TestData_WithExtra(t *testing.T) {
	e := New(1001, 400, "bad request").WithExtra("field", "name")
	data := e.Data()
	assert.Equal(t, "name", data["field"])
}

func TestFromError_Nil(t *testing.T) {
	assert.Nil(t, FromError(nil))
}

func TestFromError_ErrorX(t *testing.T) {
	e := New(1001, 400, "bad request")
	result := FromError(e)
	assert.Equal(t, e, result)
}

func TestFromError_StandardError(t *testing.T) {
	result := FromError(errors.New("std error"))
	assert.Equal(t, CodeInternal, result.BusinessCode)
	assert.Equal(t, http.StatusInternalServerError, result.HTTPCode)
	assert.NotNil(t, result.RawErr)
}

func TestFromError_WrappedErrorX(t *testing.T) {
	base := New(1001, 400, "bad request")
	wrapped := base.Wrap(errors.New("inner"))
	result := FromError(wrapped)
	assert.Equal(t, 1001, result.BusinessCode)
}

func TestUnwrap(t *testing.T) {
	inner := errors.New("inner")
	e := New(1001, 400, "outer").Wrap(inner)
	assert.Equal(t, inner, errors.Unwrap(e))
}

func TestErrorsIs_WrappedInner(t *testing.T) {
	// errors.Is follows Unwrap chain — the RawErr is the inner error
	inner := errors.New("inner")
	wrapped := ErrNotFound.Wrap(inner)
	assert.True(t, errors.Is(wrapped, inner))
}

func TestErrorsIs_SamePointer(t *testing.T) {
	// errors.Is uses pointer equality by default (no Is() method)
	e := ErrNotFound
	assert.True(t, errors.Is(e, ErrNotFound))
}

func TestErrorsAs(t *testing.T) {
	err := New(1001, 400, "base").Wrap(errors.New("inner"))
	var ex *ErrorX
	assert.True(t, errors.As(err, &ex))
	assert.Equal(t, 1001, ex.BusinessCode)
}

func TestStatus(t *testing.T) {
	e := New(1001, 400, "bad request")
	assert.Equal(t, 400, e.Status())
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *ErrorX
		code int
		http int
	}{
		{"OK", OK, CodeSuccess, 200},
		{"ErrInternal", ErrInternal, CodeInternal, 500},
		{"ErrNotFound", ErrNotFound, CodeNotFound, 404},
		{"ErrInvalidArgument", ErrInvalidArgument, CodeInvalidArgument, 400},
		{"ErrUnauthorized", ErrUnauthorized, CodeUnauthorized, 401},
		{"ErrForbidden", ErrForbidden, CodeForbidden, 403},
		{"ErrConflict", ErrConflict, CodeConflict, 409},
		{"ErrTooManyRequests", ErrTooManyRequests, CodeTooManyRequests, 429},
		{"ErrTokenExpired", ErrTokenExpired, CodeTokenExpired, 401},
		{"ErrTokenInvalid", ErrTokenInvalid, CodeTokenInvalid, 401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.BusinessCode)
			assert.Equal(t, tt.http, tt.err.HTTPCode)
		})
	}
}

func TestWithTraceID(t *testing.T) {
	e := New(1001, 400, "err").WithTraceID("trace-123")
	assert.Equal(t, "trace-123", e.TraceID)
	// Original immutable
	orig := New(1001, 400, "err")
	assert.Equal(t, "", orig.TraceID)
}

func TestChainedOperations(t *testing.T) {
	e := ErrNotFound.
		WithMessage("user not found").
		WithExtra("user_id", "123").
		Wrap(errors.New("query failed"))

	assert.Equal(t, CodeNotFound, e.BusinessCode)
	assert.Equal(t, "user not found", e.Message)
	assert.Equal(t, "123", e.Extra["user_id"])
	require.NotNil(t, e.RawErr)
	assert.Equal(t, "query failed", e.RawErr.Error())
}
