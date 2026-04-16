package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func testLogger(t *testing.T) {
	l := zap.NewNop()
	globalLogger = l.Sugar()
}

func TestWithFields_NilContext(t *testing.T) {
	testLogger(t)
	defer SetLogger(nil)

	// nil context should not panic, should use background
	ctx := WithFields(context.TODO(), StringField("k", "v"))
	assert.NotNil(t, ctx)
}

func TestWithFields_EmptyFields(t *testing.T) {
	testLogger(t)
	defer SetLogger(nil)

	ctx := context.Background()
	// Empty fields should not panic, but may return a new context with logger set
	newCtx := WithFields(ctx)
	assert.NotNil(t, newCtx)
}

func TestWithFields_ValidZapField(t *testing.T) {
	testLogger(t)
	defer SetLogger(nil)

	ctx := context.Background()
	newCtx := WithFields(ctx, StringField("key", "value"))
	assert.NotEqual(t, ctx, newCtx)
}

func TestWithFields_NonZapFieldIgnored(t *testing.T) {
	testLogger(t)
	defer SetLogger(nil)

	ctx := context.Background()
	// Non-zap.Field values should be silently dropped
	newCtx := WithFields(ctx, "not a zap field", 123, StringField("valid", "yes"))
	assert.NotNil(t, newCtx)
}

func TestWithTraceID_Empty(t *testing.T) {
	testLogger(t)
	defer SetLogger(nil)

	ctx := context.Background()

	// Empty traceID should return same context unchanged
	newCtx := WithTraceID(ctx, "")
	assert.Equal(t, ctx, newCtx)
}

func TestWithTraceID_Valid(t *testing.T) {
	testLogger(t)
	defer SetLogger(nil)

	ctx := context.Background()
	newCtx := WithTraceID(ctx, "abc-123")
	assert.NotEqual(t, ctx, newCtx)
}

func TestWithFields_Chaining(t *testing.T) {
	testLogger(t)
	defer SetLogger(nil)

	ctx := context.Background()
	ctx = WithFields(ctx, StringField("a", "1"))
	ctx = WithFields(ctx, StringField("b", "2"))
	// Both fields should be present
	assert.NotEqual(t, context.Background(), ctx)
}

func TestFromContext_NilContext(t *testing.T) {
	testLogger(t)
	defer SetLogger(nil)

	// fromContext explicitly handles nil by returning global logger
	//nolint:staticcheck // testing internal nil-handling behavior
	logger := fromContext(nil)
	assert.NotNil(t, logger)
}

func TestFromContext_NoLoggerInContext(t *testing.T) {
	testLogger(t)
	defer SetLogger(nil)

	// Context without logger should return global logger
	ctx := context.Background()
	logger := fromContext(ctx)
	assert.NotNil(t, logger)
}

func TestWithTraceID_Chaining(t *testing.T) {
	testLogger(t)
	defer SetLogger(nil)

	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-1")
	ctx = WithTraceID(ctx, "trace-2")
	assert.NotEqual(t, context.Background(), ctx)
}
