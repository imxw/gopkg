package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, "development", opts.Env)
	assert.Equal(t, zapcore.InfoLevel, opts.Level)
	assert.True(t, opts.AddCaller)
	assert.Equal(t, "trace_id", opts.TraceIDKey)
	assert.Equal(t, []string{"stdout"}, opts.OutputPaths)
	assert.Equal(t, []string{"stderr"}, opts.ErrorOutput)
	assert.Equal(t, "console", opts.Encoding)
	assert.Equal(t, zapcore.ErrorLevel, opts.Stacktrace)
	assert.Equal(t, 100, opts.LogRotation.MaxSize)
	assert.Equal(t, 10, opts.LogRotation.MaxBackups)
	assert.Equal(t, 7, opts.LogRotation.MaxAge)
	assert.True(t, opts.LogRotation.Compress)
}

func TestValidateOptions_Valid(t *testing.T) {
	opts := DefaultOptions()
	err := validateOptions(opts)
	assert.NoError(t, err)
}

func TestValidateOptions_InvalidEncoding(t *testing.T) {
	opts := DefaultOptions()
	opts.Encoding = "yaml" // invalid
	err := validateOptions(opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid encoding")
}

func TestValidateOptions_EmptyOutputPaths(t *testing.T) {
	opts := DefaultOptions()
	opts.OutputPaths = nil
	err := validateOptions(opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output paths cannot be empty")
}

func TestAdaptEnv_Production(t *testing.T) {
	opts := DefaultOptions()
	opts.Env = "production"
	got := adaptEnv(opts)

	assert.Equal(t, "json", got.Encoding)
	assert.Equal(t, zapcore.ErrorLevel, got.Stacktrace)
}

func TestAdaptEnv_Test(t *testing.T) {
	opts := DefaultOptions()
	opts.Env = "test"
	opts.Level = zapcore.DebugLevel
	got := adaptEnv(opts)

	assert.Equal(t, "console", got.Encoding)
	assert.Equal(t, zapcore.ErrorLevel, got.Level)
}

func TestAdaptEnv_Development(t *testing.T) {
	opts := DefaultOptions()
	opts.Env = "development"
	got := adaptEnv(opts)

	assert.Equal(t, "console", got.Encoding)
	assert.Equal(t, zapcore.DebugLevel, got.Level)
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback string
		want     string
	}{
		{"existing key", "PATH", "", "should not be empty"},
		{"non-existing key", "DOES_NOT_EXIST_123", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getEnv(tt.key, tt.fallback)
			if tt.key == "DOES_NOT_EXIST_123" {
				assert.Equal(t, tt.fallback, got)
			} else {
				assert.NotEmpty(t, got)
			}
		})
	}
}

func TestBuildZapLogger_Console(t *testing.T) {
	opts := DefaultOptions()
	opts.Env = "test"
	opts.OutputPaths = []string{"stdout"}
	opts.Encoding = "console"

	logger, err := buildZapLogger(opts)
	assert.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestBuildZapLogger_JSON(t *testing.T) {
	opts := DefaultOptions()
	opts.Env = "production"
	opts.OutputPaths = []string{"stdout"}
	opts.Encoding = "json"

	logger, err := buildZapLogger(opts)
	assert.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestBuildZapLogger_FileOutput(t *testing.T) {
	opts := DefaultOptions()
	opts.Env = "test"
	opts.OutputPaths = []string{"/tmp/test-logger.log"}

	logger, err := buildZapLogger(opts)
	assert.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestGetEncoder(t *testing.T) {
	opts := DefaultOptions()

	jsonEnc := getEncoder("json", encoderConfig(opts))
	assert.NotNil(t, jsonEnc)

	consoleEnc := getEncoder("console", encoderConfig(opts))
	assert.NotNil(t, consoleEnc)
}

func TestGetLevelEncoder(t *testing.T) {
	jsonLevelEnc := getLevelEncoder("json")
	assert.NotNil(t, jsonLevelEnc)

	consoleLevelEnc := getLevelEncoder("console")
	assert.NotNil(t, consoleLevelEnc)
}

func encoderConfig(opts Options) zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeLevel:    getLevelEncoder(opts.Encoding),
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
	}
}
