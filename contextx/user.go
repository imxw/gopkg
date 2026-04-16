package contextx

import (
	"context"
)

// Private key types to avoid collisions.
type (
	userIDKeyType   struct{}
	userNameKeyType struct{}
)

var (
	userIDKey   = userIDKeyType{}
	userNameKey = userNameKeyType{}
)

// WithUserID injects the user ID into context.
func WithUserID(ctx context.Context, userID uint64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userIDKey, userID)
}

// UserID extracts the user ID from context. Returns 0 if not present.
func UserID(ctx context.Context) uint64 {
	if ctx == nil {
		return 0
	}
	userID, ok := ctx.Value(userIDKey).(uint64)
	if !ok {
		return 0
	}
	return userID
}

// WithUserName injects the user name into context.
func WithUserName(ctx context.Context, userName string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if userName == "" {
		return ctx
	}
	return context.WithValue(ctx, userNameKey, userName)
}

// UserName extracts the user name from context. Returns "" if not present.
func UserName(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	userName, ok := ctx.Value(userNameKey).(string)
	if !ok {
		return ""
	}
	return userName
}
