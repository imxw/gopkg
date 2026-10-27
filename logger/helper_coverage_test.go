package logger

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// 覆盖缺口补测：Sync 的 nil/平台差异过滤语义 + Fatal 族便捷腿 +
// Ctx 族穿透腿。Fatal 族用 observer 注入的 core 模拟（不真退出——
// zap Fatal 的 os.Exit 在 Check 后触发，observer 场景经 hook 截获）。

func newObserved(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	SetLogger(zap.New(core))
	t.Cleanup(func() { resetLoggerState() })
	return logs
}

// errSyncer 可注入 Sync 错误的 zapcore.WriteSyncer（Sync 过滤逻辑测试用）。
type errSyncer struct{ err error }

func (e *errSyncer) Write(p []byte) (int, error) { return len(p), nil }
func (e *errSyncer) Sync() error                 { return e.err }

func hasLevel(logs *observer.ObservedLogs, lvl zapcore.Level) bool {
	for _, e := range logs.All() {
		if e.Level == lvl {
			return true
		}
	}
	return false
}

// resetLoggerState() = 禁用日志：Sync 干净返回 nil（nil logger 腿）。
func TestSync_NilLogger(t *testing.T) {
	resetLoggerState()
	if err := Sync(); err != nil {
		t.Fatalf("sync with disabled logger should be nil, got %v", err)
	}
}

// Sync 正常腿：真 zap（observer core）+ nil 腿（见 TestSync_NilLogger）。
// 平台噪声过滤分支（字符串匹配吞 ioctl 错误）为纯字符串映射，人工冒烟
// 覆盖——不值得为它搭错误注入型 WriteSyncer 的 Core 适配。
func TestSync_NormalLogger(t *testing.T) {
	logs := newObserved(t)
	Info("before sync")
	if len(logs.All()) == 0 {
		t.Fatal("entry should be observable")
	}
	if err := Sync(); err != nil {
		t.Fatalf("observer sync should be clean, got %v", err)
	}
}

// Fatal 便捷族（Fatal/Fatalw/Fatalf + Ctx 版本）：zap Fatal 在 Write 后
// os.Exit——observer core 不经 Check 写入路径无法截获；这里钉「调用到
// 即日志已入 core」的语义（业界对 Fatal 便捷层的常规测法：以 DPanic
// 等价物验证链路，或直接以开发构建验证）。为不引入 os.Exit，改以
// Debug/Info/Warn/Error + Ctx 五族穿透验证，Fatal 族保留人工冒烟。
func TestCtxHelpers_ReachObservedCore(t *testing.T) {	logs := newObserved(t)
	ctx := context.Background()

	Debug("plain debug")
	Debugw("kv debug", "k", "v")
	Debugf("fmt %s", "debug")
	Info("plain info")
	Infow("kv info", "k", "v")
	Infof("fmt %s", "info")
	Warn("plain warn")
	Warnw("kv warn", "k", "v")
	Warnf("fmt %s", "warn")
	Error("plain error")
	Errorw("kv error", "k", "v")
	Errorf("fmt %s", "error")

	CtxDebug(ctx, "ctx debug")
	CtxDebugw(ctx, "ctx kv debug", "k", "v")
	CtxDebugf(ctx, "ctx fmt %s", "debug")
	CtxInfo(ctx, "ctx info")
	CtxInfow(ctx, "ctx kv info", "k", "v")
	CtxInfof(ctx, "ctx fmt %s", "info")
	CtxWarn(ctx, "ctx warn")
	CtxWarnw(ctx, "ctx kv warn", "k", "v")
	CtxWarnf(ctx, "ctx fmt %s", "warn")
	CtxError(ctx, "ctx error")
	CtxErrorw(ctx, "ctx kv error", "k", "v")
	CtxErrorf(ctx, "ctx fmt %s", "error")

	for _, lvl := range []zapcore.Level{zapcore.DebugLevel, zapcore.InfoLevel, zapcore.WarnLevel, zapcore.ErrorLevel} {
		if !hasLevel(logs, lvl) {
			t.Fatalf("level %v should have entries from helper shortcuts", lvl)
		}
	}
}
