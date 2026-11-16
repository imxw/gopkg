package ginx

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/logger"
)

// EngineConfig holds Gin engine configuration without depending on project Config.
type EngineConfig struct {
	Mode           string
	TrustedProxies []string
	MaxBodyBytes   int64
	CORS           CORSConfig
	// DisableSecurityHeaders 退出内建安全响应头（SecurityHeaders：CSP/
	// nosniff/X-Frame-Options/Referrer-Policy）。缺省开启（secure by
	// default）——零值 false 保持守卫在位；自服务 HTML 的消费方可退出
	// 后自行承担策略（下游重新 Set 同名 header 亦可覆盖单条）。
	DisableSecurityHeaders bool
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
// SecurityHeaders 排在链首：限流拒绝、CORS 拒绝、recovery 500 等一切
// 短路路径同样落安全头（被拒响应恰是最可能被利用的注入面）。
//
// Mode 白名单归一（大小写不敏感）：release（含空值，安全默认）→ release /
// debug / test；未知值告警后回落 release——此前只精确识别 "release"，
// 拼错或大小写差异会静默留在 gin debug 模式（启动横幅刷屏 + 性能损耗）。
func NewEngine(cfg EngineConfig) *gin.Engine {
	switch mode := strings.ToLower(strings.TrimSpace(cfg.Mode)); mode {
	case "", "release":
		gin.SetMode(gin.ReleaseMode)
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		logger.Warnw("ginx: unknown EngineConfig.Mode, falling back to release",
			logger.StringField("mode", cfg.Mode))
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.SetTrustedProxies(cfg.TrustedProxies)

	if !cfg.DisableSecurityHeaders {
		r.Use(SecurityHeaders())
	}
	r.Use(
		RequestID(),
		Recovery(),
		AccessLog(),
		BodyLimit(cfg.MaxBodyBytes),
		cors.New(buildCORSConfig(cfg)),
	)

	return r
}

func buildCORSConfig(cfg EngineConfig) cors.Config {
	corsCfg := cors.Config{
		AllowMethods: cfg.CORS.AllowMethods,
		AllowHeaders: cfg.CORS.AllowHeaders,
		MaxAge:       time.Duration(cfg.CORS.MaxAge) * time.Second,
	}
	if len(cfg.CORS.AllowOrigins) == 1 && cfg.CORS.AllowOrigins[0] == "*" {
		corsCfg.AllowAllOrigins = true
	} else {
		corsCfg.AllowOrigins = cfg.CORS.AllowOrigins
		corsCfg.AllowCredentials = true
	}
	return corsCfg
}
