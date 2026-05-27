package gosafe

import (
	"context"
	"runtime/debug"

	"github.com/imxw/gopkg/logger"
)

// Go starts a goroutine with panic recovery and logging.
func Go(fn func()) {
	go func() {
		defer recoverPanic()
		fn()
	}()
}

// GoWithContext starts a goroutine with context propagation, panic recovery and logging.
func GoWithContext(ctx context.Context, fn func(ctx context.Context)) {
	go func() {
		defer recoverPanic()
		fn(ctx)
	}()
}

func recoverPanic() {
	if r := recover(); r != nil {
		logger.Errorw("goroutine panic recovered",
			"error", r,
			"stack", string(debug.Stack()),
		)
	}
}
