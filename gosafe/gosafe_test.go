package gosafe

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGo_RecoversPanic(t *testing.T) {
	var ran atomic.Bool
	Go(func() {
		defer func() { ran.Store(true) }()
		panic("test panic")
	})
	time.Sleep(100 * time.Millisecond)
	assert.True(t, ran.Load(), "function should have run and recovered")
}

func TestGoWithContext_PropagatesContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var gotCtx context.Context
	GoWithContext(ctx, func(c context.Context) {
		gotCtx = c
	})
	time.Sleep(100 * time.Millisecond)

	assert.NotNil(t, gotCtx)
	assert.Equal(t, ctx, gotCtx)
}

func TestGo_NormalExecution(t *testing.T) {
	var result atomic.Int32
	Go(func() {
		result.Store(42)
	})
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(42), result.Load())
}
