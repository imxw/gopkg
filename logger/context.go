package logger

import (
	"context"

	"go.uber.org/zap"
)

// WithFields 向上下文添加日志字段。v0.12 起签名收窄为 ...zap.Field——
// 旧 ...interface{} 会静默丢弃非 Field 值（WithFields(ctx, "k", "v") 整体
// 无声失效），类型在编译期不可查。字段一律经 StringField/IntField 等
// 构造器产生（本包 fields.go）。
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	if ctx == nil {
		ctx = context.Background() // 空 Context 兜底
	}
	// 从上下文获取 Logger，追加字段后重新注入
	baseLogger := fromContext(ctx).Desugar()
	newLogger := baseLogger.With(fields...).Sugar()
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

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		return ctx
	}
	return WithFields(ctx, StringField("request_id", requestID))
}
