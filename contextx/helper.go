// Package contextx provides type-safe context value helpers for extracting user identity and trace information from request contexts.
package contextx

import (
	"context"
)

// ctxKey is the private key type for context values.
type ctxKey struct{}

// GetCtx returns the context or Background if nil.
func GetCtx(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
