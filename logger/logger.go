package logger

import (
	"log"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// -------------------------- 核心定义 --------------------------
type ctxKeyLogger struct{}

var (
	logCtxKey      = ctxKeyLogger{}
	globalLogger   *zap.SugaredLogger
	initOnce       sync.Once
	initFired      atomic.Bool // Once 是否已消费（禁用态判定：fired && globalLogger==nil = 显式禁用）
	globalLoggerMu sync.RWMutex
	defaultLevel   = zapcore.InfoLevel
	atomicLevel    = zap.NewAtomicLevelAt(defaultLevel)
)

// initOnce 全局初始化互斥（L() 兜底与显式 Init 共用同一把——二者竞争
// 「谁先建出 globalLogger」；Once 体内绝不可再经 L()/Init() 触碰本 Once，
// 同 goroutine 嵌套 Do = 自死锁）。
// initLogger Once 体的实际构建逻辑（L() 兜底与 Init 共用；调用方保证
// Once 互斥已就位）。
func initLogger(opts ...func(*Options)) {
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

	// 6. 生产环境提示（直写 globalLogger——此处经 L() 会重入 Once 自死锁）
	if options.Env == "production" {
		globalLogger.Infof("logger initialized in production mode | level: %s | encoding: %s",
			options.Level, options.Encoding)
	}
}

// Init 初始化全局 Logger（入口方法）。
//
// ⚠️ 语义：进程内**只有首次调用生效**（sync.Once），再次调用静默 no-op
// ——测试里替换 Logger 用 SetLogger，不要重复 Init。宿主程序应在启动
// 期显式 Init 一次。
func Init(opts ...func(*Options)) {
	initOnce.Do(func() {
		initFired.Store(true)
		initLogger(opts...)
	})
}

// L 返回全局 SugaredLogger（核心快捷方法，兜底初始化）。
//
// nil 语义三分：① 从未初始化 → 兜底完整构建；② SetLogger(nil) 显式禁用
// → 永久 no-op sugar（禁用态被兜底重新接管 = 「禁用日志」语义破坏，且
// 修复前返回 nil sugar 曾让调用方 SIGSEGV）；③ 正常已初始化 → 直返。
func L() *zap.SugaredLogger {
	globalLoggerMu.RLock()
	l := globalLogger
	globalLoggerMu.RUnlock()
	if l != nil {
		return l
	}

	if initFired.Load() {
		// Once 已消费且 globalLogger 为 nil = 显式禁用：保持禁用。
		return zap.NewNop().Sugar()
	}

	// 从未初始化：兜底完整构建（与显式 Init 同 Once 互斥；Once 体内
	// 绝不可再经 L()/Init() 触碰本 Once——同 goroutine 嵌套 Do = 自死锁）。
	initOnce.Do(func() {
		initFired.Store(true)
		initLogger()
	})
	globalLoggerMu.RLock()
	l = globalLogger
	globalLoggerMu.RUnlock()
	if l == nil {
		return zap.NewNop().Sugar() // 竞态兜底：并发 SetLogger(nil) 抢先禁用
	}
	return l
}

// SetLevel 动态调整全局日志级别
func SetLevel(level zapcore.Level) {
	atomicLevel.SetLevel(level) // 直接修改全局级别控制器
	L().Infow("log level updated", StringField("new_level", level.String()))
}

// GetLevel 获取当前日志级别
func GetLevel() zapcore.Level {
	return atomicLevel.Level()
}
