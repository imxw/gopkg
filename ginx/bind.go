package ginx

import (
	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/contextx"
)

// BindJSON binds the JSON request body into obj; on failure it writes the
// 400 envelope itself and returns false — handlers collapse the two-step
// bind + error dance to one line:
//
//	if !ginx.BindJSON(c, &req) {
//		return
//	}
func BindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		BadRequest(c, err.Error())
		return false
	}
	return true
}

// BindQuery binds query-string parameters into obj (same failure semantics
// as BindJSON).
func BindQuery(c *gin.Context, obj any) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		BadRequest(c, err.Error())
		return false
	}
	return true
}

// BindUri binds path parameters into obj (same failure semantics as BindJSON).
func BindUri(c *gin.Context, obj any) bool {
	if err := c.ShouldBindUri(obj); err != nil {
		BadRequest(c, err.Error())
		return false
	}
	return true
}

// RequireUserID returns the authenticated caller's ID, writing a 401
// envelope and returning false when the identity is missing (whitelist
// routes, miswired middleware). contextx.UserID alone silently yields 0 —
// which is indistinguishable from a real ID of 0; route handlers that
// require identity should fail loudly instead.
func RequireUserID(c *gin.Context) (int64, bool) {
	id := contextx.UserID(c.Request.Context())
	if id <= 0 {
		Unauthorized(c, "authenticated user identity required")
		return 0, false
	}
	return id, true
}

// RequireOrgID behaves like RequireUserID for the org identity. OrgID is a
// fail-closed accessor by contract (0 = absent); this makes the refusal
// explicit at the HTTP boundary.
func RequireOrgID(c *gin.Context) (int64, bool) {
	id := contextx.OrgID(c.Request.Context())
	if id <= 0 {
		Unauthorized(c, "organization context required")
		return 0, false
	}
	return id, true
}
