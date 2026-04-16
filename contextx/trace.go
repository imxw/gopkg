package contextx

import (
	"context"
)

var traceIDKey = ctxKey{}

// -------------------------- 2. 全局TraceID相关（核心，你的项目刚需） --------------------------
// WithTraceID 向上下文注入trace_id，项目唯一的注入入口
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceID 从上下文读取trace_id，项目唯一的读取入口，兜底空字符串，永不panic
func TraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	traceID, ok := ctx.Value(traceIDKey).(string)
	if !ok {
		return ""
	}
	return traceID
}
