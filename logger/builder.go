package logger

import (
	"os"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// buildZapLogger 构建 zap.Logger（生产级可扩展）
func buildZapLogger(opts Options) (*zap.Logger, error) {
	// 1. 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // 标准化时间
		EncodeLevel:    getLevelEncoder(opts.Encoding), // 级别编码器
		EncodeCaller:   zapcore.ShortCallerEncoder,     // 短路径调用位置
		EncodeDuration: zapcore.MillisDurationEncoder,  // 毫秒级耗时
	}

	// 2. 输出写入器
	writeSyncer := zapcore.NewMultiWriteSyncer(getWriteSyncers(opts.OutputPaths, opts.LogRotation)...)
	errorSyncer := zapcore.NewMultiWriteSyncer(getWriteSyncers(opts.ErrorOutput, opts.LogRotation)...)

	// 3. 核心配置
	core := zapcore.NewCore(
		getEncoder(opts.Encoding, encoderConfig),
		writeSyncer,
		atomicLevel,
	)

	// 4. 构建选项
	zapOpts := []zap.Option{
		zap.ErrorOutput(errorSyncer),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(opts.Stacktrace), // 栈追踪级别可配置
	}
	if opts.AddCaller {
		zapOpts = append(zapOpts, zap.AddCaller())
	}

	// 5. 构建并返回
	return zap.New(core, zapOpts...), nil
}

// -------------------------- 内部构建工具 --------------------------
// getEncoder 获取编码器（JSON/Console）
func getEncoder(encoding string, encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
	if encoding == "json" {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// getLevelEncoder 级别编码器（生产/开发适配）
func getLevelEncoder(encoding string) zapcore.LevelEncoder {
	if encoding == "json" {
		return zapcore.LowercaseLevelEncoder // JSON 用小写级别
	}
	return zapcore.CapitalColorLevelEncoder // 控制台用彩色大写
}

// getWriteSyncers 获取写入器（支持 stdout/stderr/文件）
func getWriteSyncers(paths []string, rotationConfig LogRotationConfig) []zapcore.WriteSyncer {
	syncers := make([]zapcore.WriteSyncer, 0, len(paths))
	for _, path := range paths {
		switch path {
		case "stdout":
			syncers = append(syncers, zapcore.AddSync(os.Stdout))
		case "stderr":
			syncers = append(syncers, zapcore.AddSync(os.Stderr))
		default:
			// 生产级文件写入（使用lumberjack进行日志轮转）
			lumberjackLogger := &lumberjack.Logger{
				Filename:   path,
				MaxSize:    rotationConfig.MaxSize,
				MaxBackups: rotationConfig.MaxBackups,
				MaxAge:     rotationConfig.MaxAge,
				Compress:   rotationConfig.Compress,
			}
			syncers = append(syncers, zapcore.AddSync(lumberjackLogger))
		}
	}
	return syncers
}
