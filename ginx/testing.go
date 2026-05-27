package ginx

import "github.com/gin-gonic/gin"

// SetupRouter creates a test-mode Gin Engine with the package's Recovery middleware.
func SetupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	return r
}
