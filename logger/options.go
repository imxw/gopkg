package logger

import (
	"errors"
	"fmt"
	"os"

	"go.uber.org/zap/zapcore"
)

// Options 日志配置选项（函数式选项模式）
type Options struct {
	Env         string            // 运行环境（production/development/test）
	Level       zapcore.Level     // 日志级别
	AddCaller   bool              // 是否显示调用位置
	TraceIDKey  string            // TraceID 字段名
	OutputPaths []string          // 输出路径（stdout/stderr/文件路径）
	ErrorOutput []string          // 错误输出路径
	Encoding    string            // 编码格式（json/console）
	Stacktrace  zapcore.Level     // 栈追踪级别
	LogRotation LogRotationConfig // 日志轮转配置
}

// LogRotationConfig 日志轮转配置
type LogRotationConfig struct {
	MaxSize    int  `json:"maxSize"`    // 单个文件最大大小(MB)
	MaxBackups int  `json:"maxBackups"` // 保留的备份文件数量
	MaxAge     int  `json:"maxAge"`     // 保留的天数
	Compress   bool `json:"compress"`    // 是否压缩备份文件
}

// DefaultOptions 默认配置（生产级兜底）
func DefaultOptions() Options {
	return Options{
		Env:         getEnv("APP_ENV", "development"),
		Level:       defaultLevel,
		AddCaller:   true,
		TraceIDKey:  "trace_id",
		OutputPaths: []string{"stdout"},
		ErrorOutput: []string{"stderr"},
		Encoding:    "console",
		Stacktrace:  zapcore.ErrorLevel,
		LogRotation: LogRotationConfig{
			MaxSize:    100,  // 100MB
			MaxBackups: 10,   // 保留10个备份
			MaxAge:     7,    // 保留7天
			Compress:   true, // 压缩备份
		},
	}
}

// validateOptions 配置校验（生产级鲁棒性）
func validateOptions(opts Options) error {
	// 级别合法性校验
	validLevels := map[zapcore.Level]bool{
		zapcore.DebugLevel:  true,
		zapcore.InfoLevel:   true,
		zapcore.WarnLevel:   true,
		zapcore.ErrorLevel:  true,
		zapcore.DPanicLevel: true,
		zapcore.PanicLevel:  true,
		zapcore.FatalLevel:  true,
	}
	if !validLevels[opts.Level] {
		return fmt.Errorf("invalid log level: %s", opts.Level)
	}
	// 补充输出路径校验
	if len(opts.OutputPaths) == 0 {
		return errors.New("output paths cannot be empty")
	}
	// 补充编码格式校验
	if opts.Encoding != "json" && opts.Encoding != "console" {
		return fmt.Errorf("invalid encoding: %s (only json/console supported)", opts.Encoding)
	}
	return nil
}

// adaptEnv 环境适配（生产/开发/测试环境自动优化）
func adaptEnv(opts Options) Options {
	switch opts.Env {
	case "production", "prod":
		opts.Encoding = "json"           // 生产环境强制 JSON
		atomicLevel.SetLevel(opts.Level) // 同步到全局控制器
		opts.Stacktrace = zapcore.ErrorLevel
	case "test":
		opts.Encoding = "console"
		opts.Level = zapcore.ErrorLevel
		atomicLevel.SetLevel(opts.Level)
		opts.OutputPaths = []string{"stdout"}
	default: // development
		opts.Encoding = "console"
		opts.Level = zapcore.DebugLevel
		atomicLevel.SetLevel(opts.Level)
	}
	return opts
}

// -------------------------- 内部工具 --------------------------
// getEnv 获取环境变量（兜底）
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
