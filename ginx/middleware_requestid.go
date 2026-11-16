package ginx

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/imxw/gopkg/contextx"
	"github.com/imxw/gopkg/logger"
)

// RequestID 为每个请求解析追踪 ID 并注入 context 与 logger。
//
// 传播规则：入站 X-Request-Id 头经合法性校验（8-128 位字母数字/-/_）后
// 沿用——LB/网关传入的关联 ID 跨服务串联不断链；无合法入站头时生成
// UUID。无论来源，解析结果都回写 X-Request-Id 响应头，客户端可凭它
// 对账日志。
//
// 信任语义：接受入站头意味着客户端可自称请求 ID（日志冒认，不影响
// 完整性）；部署在公网直接暴露且不接受这一点时，请在入口网关强制重写。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := sanitizeRequestID(c.GetHeader("X-Request-Id"))
		if traceID == "" {
			traceID = uuid.NewString()
		}
		ctx := contextx.WithRequestID(c.Request.Context(), traceID)
		ctx = logger.WithRequestID(ctx, traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set("X-Request-Id", traceID)
		c.Next()
	}
}

// RequestIDFrom returns the request ID injected by the RequestID middleware.
// This is the single read path — reading the header name from gin keys
// (c.GetString("X-Request-Id")) is always empty: the value lives in the
// request context, not in c.Keys.
func RequestIDFrom(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return contextx.RequestID(c.Request.Context())
}

// sanitizeRequestID 校验入站请求 ID：8-128 字符、字母数字与 -/_ 之外
// 一律拒绝（防日志注入/超长垃圾），不合法返回空串走生成路径。
func sanitizeRequestID(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) < 8 || len(raw) > 128 {
		return ""
	}
	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '-' || r == '_':
		default:
			return ""
		}
	}
	return raw
}
