package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo_SucceedsImmediately(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_RetriesUntilSuccess(t *testing.T) {
	calls := 0
	var seen []uint
	err := Do(context.Background(), func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	}, Options{
		InitialInterval: time.Millisecond,
		MaxInterval:     2 * time.Millisecond,
		Notify:          func(attempt uint, _ time.Duration, _ error) { seen = append(seen, attempt) },
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	assert.Equal(t, []uint{1, 2}, seen, "Notify 按重试次数回调")
}

func TestDo_BudgetExhausted(t *testing.T) {
	start := time.Now()
	err := Do(context.Background(), func(context.Context) error {
		return errors.New("always fails")
	}, Options{
		InitialInterval: time.Millisecond,
		MaxInterval:     time.Millisecond,
		MaxElapsed:      30 * time.Millisecond,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "always fails")
	assert.ErrorContains(t, err, "retry exhausted")
	assert.Less(t, time.Since(start), 2*time.Second, "预算驱动而非无限重试")
}

func TestDo_MaxTries(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(context.Context) error {
		calls++
		return errors.New("boom")
	}, Options{
		InitialInterval: time.Millisecond,
		MaxElapsed:      time.Minute, // 预算放宽，专测次数闸
		MaxTries:        3,
	})
	require.Error(t, err)
	assert.Equal(t, 3, calls, "MaxTries 硬限尝试次数")
}

func TestDo_PermanentErrorStopsEarly(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(context.Context) error {
		calls++
		return backoff.Permanent(errors.New("fatal config"))
	}, Options{InitialInterval: time.Millisecond})
	require.Error(t, err)
	assert.Equal(t, 1, calls, "Permanent 错误立即终止")
}

func TestDo_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	err := Do(ctx, func(ctx context.Context) error {
		cancel() // 首次失败即取消
		return errors.New("first")
	}, Options{InitialInterval: 10 * time.Millisecond})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLogNotify_FieldsShaped(t *testing.T) {
	// 只验证不 panic（logger 全局 nop 兜底）；字段形状由调用方日志断言。
	assert.NotPanics(t, func() {
		LogNotify("test-scope")(1, time.Millisecond, errors.New("x"))
	})
}
