package logger

import (
	"log"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// -------------------------- 核心定义 --------------------------
type ctxKeyLogger struct{}

var (
	logCtxKey        = ctxKeyLogger{}
	globalLogger     *zap.SugaredLogger
	globalLoggerOnce sync.Once
	globalLoggerMu   sync.RWMutex
	defaultLevel     = zapcore.InfoLevel
	atomicLevel      = zap.NewAtomicLevelAt(defaultLevel)
)

// Init 初始化全局 Logger（入口方法，调用 options/builder 逻辑）
func Init(opts ...func(*Options)) {
	globalLoggerOnce.Do(func() {
		// 1. 合并配置
		options := DefaultOptions()
		for _, opt := range opts {
			opt(&options)
		}

		// 2. 配置校验
		if err := validateOptions(options); err != nil {
			log.Fatalf("[logger] invalid options: %v", err)
		}

		// 3. 环境适配
		options = adaptEnv(options)

		// 4. 构建 Logger
		zapLogger, err := buildZapLogger(options)
		if err != nil {
			log.Fatalf("[logger] failed to build logger: %v", err)
		}

		// 5. 转为 SugaredLogger
		globalLoggerMu.Lock()
		globalLogger = zapLogger.Sugar()
		globalLoggerMu.Unlock()

		// 6. 生产环境提示
		if options.Env == "production" {
			L().Infof("logger initialized in production mode | level: %s | encoding: %s",
				options.Level, options.Encoding)
		}
	})
}

// L 返回全局 SugaredLogger（核心快捷方法，兜底初始化）
func L() *zap.SugaredLogger {
	globalLoggerMu.RLock()
	l := globalLogger
	globalLoggerMu.RUnlock()
	if l == nil {
		Init()
		globalLoggerMu.RLock()
		l = globalLogger
		globalLoggerMu.RUnlock()
	}
	return l
}

// SetLevel 动态调整全局日志级别
func SetLevel(level zapcore.Level) {
	atomicLevel.SetLevel(level) // 直接修改全局级别控制器
	L().Infow("log level updated", "new_level", level.String())
}

// GetLevel 获取当前日志级别
func GetLevel() zapcore.Level {
	return atomicLevel.Level()
}
