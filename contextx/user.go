package contextx

import (
	"context"
)

// Private key types to avoid collisions.
type (
	userIDKeyType         struct{}
	userNameKeyType       struct{}
	userPermissionsKeyType struct{}
)

var (
	userIDKey          = userIDKeyType{}
	userNameKey        = userNameKeyType{}
	userPermissionsKey = userPermissionsKeyType{}
)

// WithUserID injects the user ID into context.
func WithUserID(ctx context.Context, userID int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userIDKey, userID)
}

// UserID extracts the user ID from context. Returns 0 if not present.
func UserID(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	userID, ok := ctx.Value(userIDKey).(int64)
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

// WithUserPermissions injects the user permission list into context.
func WithUserPermissions(ctx context.Context, perms []string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userPermissionsKey, perms)
}

// UserPermissions extracts the user permission list from context. Returns nil if not present.
func UserPermissions(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	perms, ok := ctx.Value(userPermissionsKey).([]string)
	if !ok {
		return nil
	}
	return perms
}
