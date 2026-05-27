package ginx

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/contextx"
)

// RequirePermission returns middleware that checks if the current user has
// the specified permission. Supports wildcard matching:
//   - "admin" matches "admin"
//   - "cmdb:*" matches "cmdb:read", "cmdb:write", etc.
func RequirePermission(required string) gin.HandlerFunc {
	return func(c *gin.Context) {
		perms := contextx.UserPermissions(c.Request.Context())
		if !hasPermission(perms, required) {
			Forbidden(c, "权限不足")
			c.Abort()
			return
		}
		c.Next()
	}
}

func hasPermission(perms []string, required string) bool {
	for _, p := range perms {
		if p == required || p == "*" {
			return true
		}
		if strings.HasSuffix(p, ":*") && strings.HasPrefix(required, p[:len(p)-1]) {
			return true
		}
	}
	return false
}
