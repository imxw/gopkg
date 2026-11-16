package contextx

import (
	"context"
)

// WithUserID injects the user ID into context. Zero/negative IDs are
// skipped (consistent with WithRequestID/WithUserName: an invalid value is
// indistinguishable from "not injected" on read — use HasUserID for the
// explicit check instead of a silent zero).
func WithUserID(ctx context.Context, userID int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if userID <= 0 {
		return ctx
	}
	return context.WithValue(ctx, userIDKey, userID)
}

// UserID extracts the user ID from context. Returns 0 if not present.
// Prefer HasUserID + UserID, or ginx.RequireUserID at the HTTP boundary,
// when zero would silently flow into domain calls.
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

// HasUserID reports whether a valid (positive) user ID is present —
// the explicit "authenticated?" check that a silent 0 cannot express.
func HasUserID(ctx context.Context) bool {
	return UserID(ctx) > 0
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

// WithOrgID injects the organization ID into context (multi-org session
// context). Zero/negative IDs are skipped (same normalization as WithUserID).
func WithOrgID(ctx context.Context, orgID int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if orgID <= 0 {
		return ctx
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

// HasOrgID reports whether a valid (positive) org ID is present.
func HasOrgID(ctx context.Context) bool {
	return OrgID(ctx) > 0
}
