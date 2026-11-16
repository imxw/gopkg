package gosafe

import (
	"sync"
	"testing"
	"time"
)

// GoNamed：panic 被恢复（不崩进程），且调用方 fn 正常路径照常执行。
func TestGoNamed_RecoversPanicAndRuns(t *testing.T) {
	var ran sync.WaitGroup
	ran.Add(2)

	done := make(chan struct{})
	go func() {
		defer close(done)
		GoNamed("test:worker", func() {
			ran.Done()
		})
		GoNamed("test:panicker", func() {
			ran.Done()
			panic("boom in named goroutine")
		})
		// 两个 goroutine 的事件都落地后再等一小拍，确保 recover 路径走完。
		time.Sleep(50 * time.Millisecond)
	}()

	ran.Wait()
	<-done
}

// recoverPanic：真实 panic 经 defer 进入时被吞掉（空名/带名两种形态
// 都不外泄）；无 panic 时直接返回。
func TestRecoverPanic_SwallowsRealPanic(t *testing.T) {
	run := func(name string) (recovered any) {
		defer func() {
			recovered = recover() // 测试哨兵：recoverPanic 若失守则此处非 nil
		}()
		defer recoverPanic(name)
		panic("seed-" + name)
	}
	if r := run(""); r != nil {
		t.Fatalf("recoverPanic(\"\") leaked panic: %v", r)
	}
	if r := run("labeled"); r != nil {
		t.Fatalf("recoverPanic(labeled) leaked panic: %v", r)
	}
}
