// Package retry wraps github.com/cenkalti/backoff/v5 with an opinionated
// default policy: exponential backoff with full jitter, bounded by max
// elapsed time — the safe default for retrying transient failures
// (reconnects, flaky downstreams). Callers that need raw control can use
// cenkalti/backoff/v5 directly; this package is the quick path.
package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v5"

	"github.com/imxw/gopkg/logger"
)

// Defaults: 500ms 起步（本地抖动即够跨退避级差）、上限 30s（单次等待不
// 超过半分钟）、总预算 1min（调用方可用 WithMaxElapsed 放宽到分钟级）。
const (
	DefaultInitialInterval = 500 * time.Millisecond
	DefaultMaxInterval     = 30 * time.Second
	DefaultMaxElapsed      = time.Minute
)

// Options retry policy knobs. Zero value = default policy.
type Options struct {
	// InitialInterval 第一次重试前的基准间隔（full jitter 实际睡眠
	// ∈ [0, interval]）。
	InitialInterval time.Duration
	// MaxInterval 单次退避上限。
	MaxInterval time.Duration
	// MaxElapsed 总预算（含等待与执行）：超时即停止并返回最后一次错误。
	// ≤0 表示不限（仅受 ctx 约束）。
	MaxElapsed time.Duration
	// MaxTries 最大尝试次数（含首次）：≤0 = 不限次数。
	MaxTries uint
	// Notify 每次重试前的回调（可选）：落日志/指标；返回不再重试的错误
	// 无法经此表达——用永久错误语义（cenkalti backoff.Permanent）。
	Notify func(attempt uint, delay time.Duration, err error)
}

func (o Options) withDefaults() Options {
	if o.InitialInterval <= 0 {
		o.InitialInterval = DefaultInitialInterval
	}
	if o.MaxInterval <= 0 {
		o.MaxInterval = DefaultMaxInterval
	}
	if o.MaxElapsed <= 0 {
		o.MaxElapsed = DefaultMaxElapsed
	}
	return o
}

// Do retries op until it succeeds, the policy budget is exhausted, or ctx
// is done. Returns the last error (wrapped with attempt count). Permanent
// failures: wrap the error with backoff.Permanent inside op to stop early.
//
// Default policy: exponential full jitter, 500ms→30s per attempt, 1min total.
func Do(ctx context.Context, op func(ctx context.Context) error, opts ...Options) error {
	o := Options{}
	if len(opts) > 0 {
		o = opts[0]
	}
	o = o.withDefaults()

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = o.InitialInterval
	bo.MaxInterval = o.MaxInterval

	bopts := []backoff.RetryOption{
		backoff.WithBackOff(bo),
		backoff.WithMaxElapsedTime(o.MaxElapsed),
	}
	if o.MaxTries > 0 {
		bopts = append(bopts, backoff.WithMaxTries(o.MaxTries))
	}
	if o.Notify != nil {
		attempt := uint(0)
		bopts = append(bopts, backoff.WithNotify(func(err error, delay time.Duration) {
			attempt++
			o.Notify(attempt, delay, err)
		}))
	}

	// cenkalti v5 的 Operation 无 ctx 形参（取消经 Retry 的 ctx 传导）；
	// T 取 bool 显式具型避免推断失败。
	_, err := backoff.Retry(ctx, func() (bool, error) {
		return true, op(ctx)
	}, bopts...)
	if err != nil {
		return fmt.Errorf("retry exhausted: %w", err)
	}
	return nil
}

// LogNotify 是 Notify 的日志快捷实现（attempt/delay/error 结构化落日志）。
func LogNotify(scope string) func(attempt uint, delay time.Duration, err error) {
	return func(attempt uint, delay time.Duration, err error) {
		logger.Warnw("retry attempt failed",
			logger.StringField("scope", scope),
			logger.Uint64Field("attempt", uint64(attempt)),
			logger.AnyField("next_delay_ms", delay.Milliseconds()),
			logger.AnyField("error", err),
		)
	}
}
