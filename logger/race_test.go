package logger

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestSetLogger_Nil(t *testing.T) {
	// Reset state for test
	globalLoggerMu.Lock()
	globalLogger = nil
	globalLoggerOnce = sync.Once{}
	globalLoggerMu.Unlock()

	SetLogger(nil)
	assert.Nil(t, globalLogger)
}

func TestSetLogger_Replace(t *testing.T) {
	// Reset state for test
	globalLoggerMu.Lock()
	globalLogger = nil
	globalLoggerOnce = sync.Once{}
	globalLoggerMu.Unlock()

	testLogger := zap.NewExample()
	SetLogger(testLogger)
	assert.NotNil(t, globalLogger)
}

func TestL_FallbackInit(t *testing.T) {
	// Reset state for test
	globalLoggerMu.Lock()
	globalLogger = nil
	globalLoggerOnce = sync.Once{}
	globalLoggerMu.Unlock()

	l := L()
	assert.NotNil(t, l)
}

func TestSetLogger_ConcurrentAccess(t *testing.T) {
	// This test would fail with -race before the fix
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = L()
		}()
		go func() {
			defer wg.Done()
			SetLogger(zap.NewExample())
		}()
		go func() {
			defer wg.Done()
			SetLogger(nil)
		}()
	}
	wg.Wait()
}
