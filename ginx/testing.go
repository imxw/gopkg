package ginx

import "github.com/gin-gonic/gin"

// SetupRouter creates a test-mode Gin Engine with Recovery middleware.
func SetupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	return r
}
