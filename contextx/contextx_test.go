package contextx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCtx_Nil(t *testing.T) {
	ctx := GetCtx(nil)
	assert.Equal(t, context.Background(), ctx)
}

func TestGetCtx_NonNil(t *testing.T) {
	orig := context.Background()
	ctx := GetCtx(orig)
	assert.Equal(t, orig, ctx)
}

func TestUserID_RoundTrip(t *testing.T) {
	ctx := WithUserID(context.Background(), 42)
	assert.Equal(t, uint64(42), UserID(ctx))
}

func TestUserID_Zero(t *testing.T) {
	assert.Equal(t, uint64(0), UserID(context.Background()))
}

func TestUserID_Nil(t *testing.T) {
	assert.Equal(t, uint64(0), UserID(nil))
}

func TestWithUserID_Nil(t *testing.T) {
	ctx := WithUserID(nil, 1)
	assert.Equal(t, uint64(1), UserID(ctx))
}

func TestUserName_RoundTrip(t *testing.T) {
	ctx := WithUserName(context.Background(), "admin")
	assert.Equal(t, "admin", UserName(ctx))
}

func TestUserName_Empty(t *testing.T) {
	ctx := WithUserName(context.Background(), "")
	assert.Equal(t, "", UserName(ctx))
}

func TestUserName_Nil(t *testing.T) {
	assert.Equal(t, "", UserName(nil))
}

func TestTraceID_RoundTrip(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-123")
	assert.Equal(t, "trace-123", TraceID(ctx))
}

func TestTraceID_Empty(t *testing.T) {
	ctx := WithTraceID(context.Background(), "")
	assert.Equal(t, "", TraceID(ctx))
}

func TestTraceID_Nil(t *testing.T) {
	assert.Equal(t, "", TraceID(nil))
}

func TestCombinedContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithUserID(ctx, 99)
	ctx = WithUserName(ctx, "testuser")
	ctx = WithTraceID(ctx, "trace-abc")

	assert.Equal(t, uint64(99), UserID(ctx))
	assert.Equal(t, "testuser", UserName(ctx))
	assert.Equal(t, "trace-abc", TraceID(ctx))
}
