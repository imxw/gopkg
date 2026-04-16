package logger

import (
	"context"
	"strings"

	"go.uber.org/zap"
)

// -------------------------- 全局 Logger 辅助方法 --------------------------
// SetLogger 替换全局 Logger（仅测试用，生产级禁用）
// 示例：logger.SetLogger(zap.NewExample())
// 如果传入 nil，会将全局 logger 重置为 nil（禁用日志）
func SetLogger(l *zap.Logger) {
	globalLoggerMu.Lock()
	defer globalLoggerMu.Unlock()
	if l == nil {
		globalLogger = nil
		return
	}
	globalLogger = l.Sugar()
}

// Sync 同步日志缓冲区（优雅退出必备）
// 调用时机：程序退出前（如 main 函数 defer logger.Sync()）
func Sync() error {
	globalLoggerMu.RLock()
	l := globalLogger
	globalLoggerMu.RUnlock()
	if l == nil {
		return nil
	}

	// 执行原始 Sync 操作
	err := l.Sync()
	if err == nil {
		return nil
	}

	// ========== 核心优化：过滤无需关注的同步错误 ==========
	// 兼容 macOS/Linux：stdout/stderr 的不兼容错误
	errMsg := err.Error()
	skipErrors := []string{
		"sync /dev/stdout: inappropriate ioctl for device", // macOS/Linux 终端
		"sync /dev/stdout: invalid argument",               // 部分系统别名
		"sync /dev/stderr: inappropriate ioctl for device", // 包含 stderr
		"sync /dev/stderr: invalid argument",
		"The handle is invalid", // Windows 兼容
	}

	// 匹配到跳过的错误，返回 nil（视为成功）
	for _, skipMsg := range skipErrors {
		if strings.Contains(errMsg, skipMsg) {
			return nil
		}
	}

	// 非跳过错误：返回真实错误（如文件同步失败）
	return err
}

// -------------------------- 无上下文快捷日志方法（对齐 Zap 原生） --------------------------
// Debug 纯参数日志（无消息前缀，适配 Zap Debug(args ...interface{})）
// 示例：logger.Debug("failed to get hostname", err)
func Debug(args ...interface{}) {
	L().Debug(args...) // 直接透传参数，无转换
}

// Debugw 结构化日志（消息+键值对，适配 Zap Debugw(msg string, kv ...interface{})）
// 示例：logger.Debugw("get hostname failed", "error", err, "agent_id", "123")
func Debugw(msg string, kv ...interface{}) {
	L().Debugw(msg, kv...)
}

// Debugf 格式化日志（适配 Zap Debugf(format string, args ...interface{})）
// 示例：logger.Debugf("failed to get hostname: %v", err)
func Debugf(format string, args ...interface{}) {
	L().Debugf(format, args...)
}

// Info 纯参数日志
// 示例：logger.Info("agent started", "pid", os.Getpid())
func Info(args ...interface{}) {
	L().Info(args...)
}

// Infow 结构化日志（消息+键值对）
// 示例：logger.Infow("agent started", "pid", os.Getpid(), "hostname", "node-1")
func Infow(msg string, kv ...interface{}) {
	L().Infow(msg, kv...)
}

// Infof 格式化日志
// 示例：logger.Infof("agent started (pid: %d)", os.Getpid())
func Infof(format string, args ...interface{}) {
	L().Infof(format, args...)
}

// Warn 纯参数日志
// 示例：logger.Warn("low disk space", 80)
func Warn(args ...interface{}) {
	L().Warn(args...)
}

// Warnw 结构化日志（消息+键值对）
// 示例：logger.Warnw("low disk space", "usage", 80, "path", "/")
func Warnw(msg string, kv ...interface{}) {
	L().Warnw(msg, kv...)
}

// Warnf 格式化日志
// 示例：logger.Warnf("low disk space: %d%% used", 80)
func Warnf(format string, args ...interface{}) {
	L().Warnf(format, args...)
}

// Error 纯参数日志
// 示例：logger.Error("exec command failed", err)
func Error(args ...interface{}) {
	L().Error(args...)
}

// Errorw 结构化日志（消息+键值对）
// 示例：logger.Errorw("exec command failed", "error", err, "cmd", "ls -l")
func Errorw(msg string, kv ...interface{}) {
	L().Errorw(msg, kv...)
}

// Errorf 格式化日志
// 示例：logger.Errorf("exec command failed: %v", err)
func Errorf(format string, args ...interface{}) {
	L().Errorf(format, args...)
}

// Fatal 纯参数日志（执行后退出程序）
// 示例：logger.Fatal("config load failed", err)
func Fatal(args ...interface{}) {
	L().Fatal(args...)
}

// Fatalw 结构化日志（消息+键值对，执行后退出程序）
// 示例：logger.Fatalw("config load failed", "error", err, "path", "config.yaml")
func Fatalw(msg string, kv ...interface{}) {
	L().Fatalw(msg, kv...)
}

// Fatalf 格式化日志（执行后退出程序）
// 示例：logger.Fatalf("config load failed: %v", err)
func Fatalf(format string, args ...interface{}) {
	L().Fatalf(format, args...)
}

// -------------------------- 带上下文快捷日志方法（Ctx 前缀，对齐 Zap） --------------------------
// CtxDebug 带上下文的纯参数日志
// 示例：logger.CtxDebug(ctx, "get request", req)
func CtxDebug(ctx context.Context, args ...interface{}) {
	fromContext(ctx).Debug(args...)
}

// CtxDebugw 带上下文的结构化日志（消息+键值对）
// 示例：logger.CtxDebugw(ctx, "get request", "path", "/api/v1/agent")
func CtxDebugw(ctx context.Context, msg string, kv ...interface{}) {
	fromContext(ctx).Debugw(msg, kv...)
}

// CtxDebugf 带上下文的格式化日志
// 示例：logger.CtxDebugf(ctx, "get request: %s", req.Path)
func CtxDebugf(ctx context.Context, format string, args ...interface{}) {
	fromContext(ctx).Debugf(format, args...)
}

// CtxInfo 带上下文的纯参数日志
func CtxInfo(ctx context.Context, args ...interface{}) {
	fromContext(ctx).Info(args...)
}

// CtxInfow 带上下文的结构化日志（消息+键值对）
func CtxInfow(ctx context.Context, msg string, kv ...interface{}) {
	fromContext(ctx).Infow(msg, kv...)
}

// CtxInfof 带上下文的格式化日志
func CtxInfof(ctx context.Context, format string, args ...interface{}) {
	fromContext(ctx).Infof(format, args...)
}

// CtxWarn 带上下文的纯参数日志
func CtxWarn(ctx context.Context, args ...interface{}) {
	fromContext(ctx).Warn(args...)
}

// CtxWarnw 带上下文的结构化日志（消息+键值对）
func CtxWarnw(ctx context.Context, msg string, kv ...interface{}) {
	fromContext(ctx).Warnw(msg, kv...)
}

// CtxWarnf 带上下文的格式化日志
func CtxWarnf(ctx context.Context, format string, args ...interface{}) {
	fromContext(ctx).Warnf(format, args...)
}

// CtxError 带上下文的纯参数日志
func CtxError(ctx context.Context, args ...interface{}) {
	fromContext(ctx).Error(args...)
}

// CtxErrorw 带上下文的结构化日志（消息+键值对）
func CtxErrorw(ctx context.Context, msg string, kv ...interface{}) {
	fromContext(ctx).Errorw(msg, kv...)
}

// CtxErrorf 带上下文的格式化日志
func CtxErrorf(ctx context.Context, format string, args ...interface{}) {
	fromContext(ctx).Errorf(format, args...)
}

// CtxFatal 带上下文的纯参数日志（执行后退出程序）
func CtxFatal(ctx context.Context, args ...interface{}) {
	fromContext(ctx).Fatal(args...)
}

// CtxFatalw 带上下文的结构化日志（消息+键值对，执行后退出程序）
func CtxFatalw(ctx context.Context, msg string, kv ...interface{}) {
	fromContext(ctx).Fatalw(msg, kv...)
}

// CtxFatalf 带上下文的格式化日志（执行后退出程序）
func CtxFatalf(ctx context.Context, format string, args ...interface{}) {
	fromContext(ctx).Fatalf(format, args...)
}
