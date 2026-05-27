package ginx

import "github.com/gin-gonic/gin"

// RegisterFunc is the function signature for route registration.
type RegisterFunc func(apiGroup *gin.RouterGroup)

// RegisterProtected registers protected routes (authentication middleware applied by caller).
func RegisterProtected(apiGroup *gin.RouterGroup, registers ...RegisterFunc) {
	for _, reg := range registers {
		if reg != nil {
			reg(apiGroup)
		}
	}
}
