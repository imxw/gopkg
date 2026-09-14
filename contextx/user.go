package contextx

import (
	"context"
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

// WithOrgID injects the organization ID into context (multi-org session context).
func WithOrgID(ctx context.Context, orgID int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, orgIDKey, orgID)
}

// OrgID extracts the organization ID from context. Returns 0 if not present.
// Authorization chains that must scope by org should treat 0 as
// "no org context" and fail closed rather than fall back to a default org.
func OrgID(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	orgID, ok := ctx.Value(orgIDKey).(int64)
	if !ok {
		return 0
	}
	return orgID
}
