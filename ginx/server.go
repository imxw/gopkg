package ginx

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// EngineConfig holds Gin engine configuration without depending on project Config.
type EngineConfig struct {
	Mode           string
	TrustedProxies []string
	MaxBodyBytes   int64
	CORS           CORSConfig
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
	MaxAge       int // seconds
}

// DefaultEngineConfig returns sensible defaults.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		Mode:           "release",
		TrustedProxies: []string{"127.0.0.1", "::1"},
		MaxBodyBytes:   30 << 20, // 30MB
		CORS: CORSConfig{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
			MaxAge:       86400,
		},
	}
}

// NewEngine creates a Gin engine with global middleware (no routes).
func NewEngine(cfg EngineConfig) *gin.Engine {
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.SetTrustedProxies(cfg.TrustedProxies)

	r.Use(
		RequestID(),
		Recovery(),
		AccessLog(),
		MaxBytesReader(cfg.MaxBodyBytes),
		cors.New(buildCORSConfig(cfg)),
	)

	return r
}

func buildCORSConfig(cfg EngineConfig) cors.Config {
	corsCfg := cors.Config{
		AllowMethods: cfg.CORS.AllowMethods,
		AllowHeaders: cfg.CORS.AllowHeaders,
		MaxAge:       timeDuration(cfg.CORS.MaxAge),
	}
	if len(cfg.CORS.AllowOrigins) == 1 && cfg.CORS.AllowOrigins[0] == "*" {
		corsCfg.AllowAllOrigins = true
	} else {
		corsCfg.AllowOrigins = cfg.CORS.AllowOrigins
		corsCfg.AllowCredentials = true
	}
	return corsCfg
}

func timeDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
