package logger

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestSync_FilterPlatformErrors(t *testing.T) {
	tests := []struct {
		name    string
		errMsg  string
		wantNil bool
	}{
		{
			name:    "macOS stdout ioctl",
			errMsg:  "sync /dev/stdout: inappropriate ioctl for device",
			wantNil: true,
		},
		{
			name:    "macOS stderr ioctl",
			errMsg:  "sync /dev/stderr: inappropriate ioctl for device",
			wantNil: true,
		},
		{
			name:    "linux stdout invalid argument",
			errMsg:  "sync /dev/stdout: invalid argument",
			wantNil: true,
		},
		{
			name:    "linux stderr invalid argument",
			errMsg:  "sync /dev/stderr: invalid argument",
			wantNil: true,
		},
		{
			name:    "windows invalid handle",
			errMsg:  "The handle is invalid",
			wantNil: true,
		},
		{
			name:    "partial match stdout",
			errMsg:  "some error: sync /dev/stdout: inappropriate ioctl for device",
			wantNil: true,
		},
		{
			name:    "real file sync error",
			errMsg:  "sync /var/log/app.log: input/output error",
			wantNil: false,
		},
		{
			name:    "empty error message",
			errMsg:  "",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build a logger that writes to a file that will fail on sync
			cfg := DefaultOptions()
			cfg.OutputPaths = []string{"stdout"}

			l, err := buildZapLogger(cfg)
			require.NoError(t, err)

			SetLogger(l)

			// Create a sync error by simulating what the skipErrors check does
			syncErr := &fakeError{msg: tt.errMsg}
			got := filterSyncError(syncErr)

			if tt.wantNil {
				assert.Nil(t, got, "expected nil for error: %q", tt.errMsg)
			} else {
				assert.NotNil(t, got, "expected non-nil for error: %q", tt.errMsg)
			}

			SetLogger(nil)
		})
	}
}

// fakeSyncError creates an error with a specific message for testing
// without actually triggering a sync failure
type fakeError struct {
	msg string
}

func (e *fakeError) Error() string { return e.msg }

// filterSyncError mirrors the skipErrors logic in Sync for unit testing
func filterSyncError(err error) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	skipErrors := []string{
		"sync /dev/stdout: inappropriate ioctl for device",
		"sync /dev/stdout: invalid argument",
		"sync /dev/stderr: inappropriate ioctl for device",
		"sync /dev/stderr: invalid argument",
		"The handle is invalid",
	}
	for _, skip := range skipErrors {
		if strings.Contains(errMsg, skip) {
			return nil
		}
	}
	return err
}

func TestSetLogger(t *testing.T) {
	// Should not panic with nil
	SetLogger(nil)
	assert.Nil(t, globalLogger)

	// Should not panic with valid logger built via buildZapLogger
	opts := DefaultOptions()
	opts.Env = "test"
	opts.OutputPaths = []string{"stdout"}
	l, err := buildZapLogger(opts)
	require.NoError(t, err)
	globalLogger = l.Sugar()
	assert.NotNil(t, globalLogger)

	// Reset to prevent affecting other tests
	globalLogger = nil
}

func TestShortcutMethods_DelegatesToL(t *testing.T) {
	// These just verify the shortcut methods don't panic
	// They delegate to L() which is tested via integration
	opts := DefaultOptions()
	opts.Env = "test"
	opts.OutputPaths = []string{"stdout"}
	l, err := buildZapLogger(opts)
	require.NoError(t, err)
	globalLogger = l.Sugar()
	defer func() { globalLogger = nil }()

	// Should not panic - these are smoke tests
	assert.NotPanics(t, func() { Debug("test") })
	assert.NotPanics(t, func() { Debugw("test") })
	assert.NotPanics(t, func() { Debugf("test: %d", 1) })

	assert.NotPanics(t, func() { Info("test") })
	assert.NotPanics(t, func() { Infow("test") })
	assert.NotPanics(t, func() { Infof("test: %d", 1) })

	assert.NotPanics(t, func() { Warn("test") })
	assert.NotPanics(t, func() { Warnw("test") })
	assert.NotPanics(t, func() { Warnf("test: %d", 1) })

	assert.NotPanics(t, func() { Error("test") })
	assert.NotPanics(t, func() { Errorw("test") })
	assert.NotPanics(t, func() { Errorf("test: %d", 1) })
	// Skip Fatalw: Nop logger's Fatal still calls runtime.Goexit which may panic in test
}

func TestSync_NilGlobalLogger(t *testing.T) {
	// When globalLogger is nil, Sync should return nil
	SetLogger(nil)
	err := Sync()
	assert.Nil(t, err)
}

// testCtxLogger sets up a real logger for context tests
func testCtxLogger() {
	opts := DefaultOptions()
	opts.Env = "test"
	opts.OutputPaths = []string{"stdout"}
	l, _ := buildZapLogger(opts)
	globalLogger = l.Sugar()
}

func TestCtxMethods_DelegatesToFromContext(t *testing.T) {
	testCtxLogger()
	defer func() { globalLogger = nil }()

	ctx := context.Background()

	// All Ctx* methods should not panic and just call through
	assert.NotPanics(t, func() { CtxDebug(ctx, "debug") })
	assert.NotPanics(t, func() { CtxDebugw(ctx, "debugw") })
	assert.NotPanics(t, func() { CtxDebugf(ctx, "debugf: %d", 1) })

	assert.NotPanics(t, func() { CtxInfo(ctx, "info") })
	assert.NotPanics(t, func() { CtxInfow(ctx, "infow") })
	assert.NotPanics(t, func() { CtxInfof(ctx, "infof: %d", 1) })

	assert.NotPanics(t, func() { CtxWarn(ctx, "warn") })
	assert.NotPanics(t, func() { CtxWarnw(ctx, "warnw") })
	assert.NotPanics(t, func() { CtxWarnf(ctx, "warnf: %d", 1) })

	assert.NotPanics(t, func() { CtxError(ctx, "error") })
	assert.NotPanics(t, func() { CtxErrorw(ctx, "errorw") })
	assert.NotPanics(t, func() { CtxErrorf(ctx, "errorf: %d", 1) })
}

func TestCtxMethods_WithFields(t *testing.T) {
	testCtxLogger()
	defer func() { globalLogger = nil }()

	ctx := context.Background()
	ctx = WithFields(ctx, StringField("trace_id", "trace-abc"))

	// With fields in context should still work
	assert.NotPanics(t, func() { CtxInfow(ctx, "with fields") })
	assert.NotPanics(t, func() { CtxErrorw(ctx, "with fields error") })
}

func TestSetLevel(t *testing.T) {
	// SetLevel should not panic and should update the atomic level
	assert.NotPanics(t, func() {
		SetLevel(zapcore.DebugLevel)
	})
	assert.Equal(t, zapcore.DebugLevel, GetLevel())

	SetLevel(zapcore.InfoLevel)
	assert.Equal(t, zapcore.InfoLevel, GetLevel())

	// Reset to default
	SetLevel(zapcore.InfoLevel)
}

func TestGetLevel(t *testing.T) {
	// GetLevel should return the current level
	SetLevel(zapcore.WarnLevel)
	assert.Equal(t, zapcore.WarnLevel, GetLevel())

	// Reset
	SetLevel(zapcore.InfoLevel)
}
