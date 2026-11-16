package gosafe

import (
	"context"
	"runtime/debug"

	"github.com/imxw/gopkg/logger"
)

// Go starts a goroutine with panic recovery and logging.
func Go(fn func()) {
	go func() {
		defer recoverPanic("")
		fn()
	}()
}

// GoNamed behaves like Go but tags the goroutine with a name so panic logs
// identify the offending worker (e.g. "scheduler:job-cleanup"). The name is
// for observability only — no dedup or lifecycle semantics attached.
func GoNamed(name string, fn func()) {
	go func() {
		defer recoverPanic(name)
		fn()
	}()
}

// GoWithContext starts a goroutine with context propagation, panic recovery and logging.
func GoWithContext(ctx context.Context, fn func(ctx context.Context)) {
	go func() {
		defer recoverPanic("")
		fn(ctx)
	}()
}

func recoverPanic(name string) {
	if r := recover(); r != nil {
		fields := []any{
			logger.AnyField("error", r),
			logger.StringField("stack", string(debug.Stack())),
		}
		if name != "" {
			fields = append(fields, logger.StringField("goroutine", name))
		}
		logger.Errorw("goroutine panic recovered", fields...)
	}
}
