package logger

import (
	"context"

	"go.uber.org/zap"
)

// WithFields 向上下文添加日志字段（封装 zap.Field，对外无感知）
func WithFields(ctx context.Context, fields ...interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background() // 空 Context 兜底
	}
	// 转换封装的字段为 zap.Field
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		if field, ok := f.(zap.Field); ok {
			zapFields = append(zapFields, field)
		}
	}
	// 从上下文获取 Logger，追加字段后重新注入
	baseLogger := fromContext(ctx).Desugar()
	newLogger := baseLogger.With(zapFields...).Sugar()
	return context.WithValue(ctx, logCtxKey, newLogger)
}

// fromContext 从上下文提取 Logger（空 Context 兜底）
func fromContext(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil {
		return L() // 空 Context 返回全局 Logger
	}
	if l, ok := ctx.Value(logCtxKey).(*zap.SugaredLogger); ok && l != nil {
		return l
	}
	return L()
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	return WithFields(ctx, StringField("trace_id", traceID))
}
