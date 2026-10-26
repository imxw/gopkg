package ginx

import "github.com/gin-gonic/gin"

// securityHeadersPolicy 默认 CSP 策略：对 API/JSON 响应零副作用（浏览器
// 只对文档型响应执行 most 指令），frame-ancestors 收口点击劫持，base-uri
// 兜 HTML 型响应。自服务 HTML 的消费方在下游重新 Set 同名 header 即可
// 覆盖（本中间件只 Set 不 Abort），或经 EngineConfig.DisableSecurityHeaders
// 整体退出后自行承担策略。
const securityHeadersPolicy = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"

// SecurityHeaders 安全响应头：XSS 纵深的最低成本一层。必须排在引擎链
// 最前（NewEngine 已如此装配）——限流拒绝、CORS 拒绝、recovery 500 等
// 一切短路路径同样落头，被拒响应恰恰是最可能被利用的注入面。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Content-Security-Policy", securityHeadersPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		c.Next()
	}
}
