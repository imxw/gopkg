package ginx

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/logger"
)

// sensitiveQueryKeys 凭据类 query 参数名片段（大小写不敏感子串匹配）：
// 值在访问日志中替换为 ***（键保留供排障）。这是纵深而非安全边界——
// 正确姿势仍是「凭据不进 query」（见消费方 WS 鉴权约定），本脱敏保证
// 约定被破例时日志不放大泄露。
var sensitiveQueryKeys = []string{
	"token", "secret", "password", "passwd", "pwd", "apikey", "api_key",
	"authorization", "credential", "signature", "sig", "jwt", "ticket",
}

// sanitizeQuery 脱敏 query 中的凭据类参数值：逐段检查 key，命中敏感片段
// 则值替换为 ***，其余原样保留（段序不重排，方便与原始 URL 对账）。
func sanitizeQuery(raw string) string {
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, "&")
	for i, p := range parts {
		key := p
		if eq := strings.IndexByte(p, '='); eq >= 0 {
			key = p[:eq]
		}
		if key == "" {
			continue
		}
		lk := strings.ToLower(key)
		if unescaped, err := url.QueryUnescape(lk); err == nil {
			lk = unescaped
		}
		for _, s := range sensitiveQueryKeys {
			if strings.Contains(lk, s) {
				if eq := strings.IndexByte(p, '='); eq >= 0 {
					parts[i] = p[:eq+1] + "***"
				} else {
					parts[i] = p + "=***"
				}
				break
			}
		}
	}
	return strings.Join(parts, "&")
}

// AccessLog logs each HTTP request with structured fields and selects log
// level based on the response status code. query 中的凭据类参数值先脱敏
// 再落日志（见 sanitizeQuery）。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		ctx := c.Request.Context()
		method := c.Request.Method
		path := c.Request.URL.Path
		query := sanitizeQuery(c.Request.URL.RawQuery)
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		logger.CtxDebugw(ctx, "HTTP request incoming",
			logger.StringField("method", method),
			logger.StringField("path", path),
			logger.StringField("query", query),
			logger.StringField("client_ip", clientIP),
			logger.StringField("user_agent", userAgent),
		)

		c.Next()

		costMs := time.Since(startTime).Milliseconds()
		httpCode := c.Writer.Status()
		logFields := []interface{}{
			logger.StringField("method", method),
			logger.StringField("path", path),
			logger.StringField("query", query),
			logger.StringField("client_ip", clientIP),
			logger.StringField("user_agent", userAgent),
			logger.IntField("http_code", httpCode),
			logger.Int64Field("cost_ms", costMs),
		}

		switch {
		case httpCode >= 200 && httpCode < 400:
			logger.CtxInfow(ctx, "HTTP request completed", logFields...)
		case httpCode >= 400 && httpCode < 500:
			logger.CtxWarnw(ctx, "HTTP request client error", logFields...)
		default:
			logger.CtxErrorw(ctx, "HTTP request server error", logFields...)
		}
	}
}
