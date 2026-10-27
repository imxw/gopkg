package logger

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 禁用语义与 Once 死锁回归（2026-10-27 事故钉子）：
//
// 1. SetLogger(nil) 后再打日志曾 SIGSEGV——L() 兜底 Init() 走同一把
//    sync.Once（已消费即 no-op），返回 nil sugar 给调用方解引用；
// 2. L() 兜底调 Init() 曾自死锁——L() 的 Once.Do 内调 Init()，后者
//    又 Do 同一把 Once（同 goroutine 嵌套 = 永久等自己的锁）。
//
// 修复后契约：SetLogger(nil) = 禁用态（L() 返 no-op sugar，打日志
// 静默）；未 Init 且未禁用时 L() 兜底完整初始化。

func resetLoggerState() {
	globalLoggerMu.Lock()
	globalLogger = nil
	initOnce = sync.Once{}
	initFired.Store(false)
	globalLoggerMu.Unlock()
}

func TestSetLogger_Nil_DisabledStaysDisabled(t *testing.T) {
	resetLoggerState()
	SetLogger(nil)
	assert.Nil(t, globalLogger, "显式禁用后全局保持 nil")

	// 关键腿：禁用态下经 L() 取到的必须是 no-op sugar（非 nil、可打日志），
	// 而不是 nil（SIGSEGV）也不是重新初始化的 stdout logger。
	l := L()
	require.NotNil(t, l)
	assert.NotPanics(t, func() { l.Infow("disabled logger stays silent", "k", "v") })
	assert.Nil(t, globalLogger, "禁用态不得被兜底初始化重新接管")
}

func TestL_FallbackInit_DoesNotDeadlock(t *testing.T) {
	resetLoggerState()
	// 修复前：L() → Once.Do(Init) → Init 内部 Once.Do（同一把）→ 自死锁。
	// 修复后：Once 体内不再触碰 Once（生产提示直写 globalLogger），
	// 本调用若死锁将由测试超时暴露。
	var l *zap.SugaredLogger
	var err error
	done := make(chan struct{})
	go func() {
		defer close(done)
		l = L()
		_ = err
	}()
	select {
	case <-done:
	case <-timeoutAfter(5):
		t.Fatal("L() fallback init deadlocked (Once re-entry)")
	}
	require.NotNil(t, l)
	assert.NotPanics(t, func() { l.Info("fallback initialized") })
	assert.NotNil(t, globalLogger, "兜底应建出全局 logger")
}

func TestInit_ReInitIsNoop(t *testing.T) {
	resetLoggerState()
	Init()
	first := globalLogger
	require.NotNil(t, first)
	Init()
	assert.Same(t, first, globalLogger, "二次 Init 必须是 no-op（同一实例）")
}

func timeoutAfter(seconds int) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		<-time.After(time.Duration(seconds) * time.Second)
		close(ch)
	}()
	return ch
}
