package ginx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/imxw/gopkg/logger"
)

// ServeConfig configures the blocking HTTP server lifecycle.
type ServeConfig struct {
	// Addr listen address ("host:port", ":8080" …).
	Addr string
	// ShutdownTimeout bounds the drain window after SIGINT/SIGTERM.
	// ≤0 defaults to 10s.
	ShutdownTimeout time.Duration
	// HealthPath mounts a liveness endpoint returning 200 {"status":"ok"}
	// before any auth middleware applies (engine routes registered later
	// must not shadow it). Empty string = no auto route — mount your own
	// with checks (readiness) instead.
	HealthPath string
	// OnHealthy optional callback invoked right after listen succeeds
	// (startup log stamp, post-start wiring).
	OnHealthy func()
}

// Serve runs the engine with a standard graceful-shutdown lifecycle and
// blocks until SIGINT/SIGTERM: listen → (onHealthy) → wait signal →
// bounded Shutdown → return nil. This is the hand-rolled ~30-line idiom
// every Gin service used to copy — now one call.
//
// Errors: listen failure (address busy, bad addr) returns the wrapped
// error immediately; drain failure (Shutdown error) is returned after
// the timeout window.
func Serve(engine *gin.Engine, cfg ServeConfig) error {
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}
	if cfg.HealthPath != "" {
		engine.GET(cfg.HealthPath, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: engine,
	}

	// 通知型 ctx：首个 SIGINT/SIGTERM 即触发（二次信号走系统默认终止，
	// 调试期可用 kill -9 强杀；不静默吞第二次信号是排障友好语义）。
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	logger.Infow("http server listening",
		logger.StringField("addr", cfg.Addr),
		logger.StringField("shutdown_timeout", cfg.ShutdownTimeout.String()),
	)
	if cfg.OnHealthy != nil {
		cfg.OnHealthy()
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http listen: %w", err)
		}
		return nil // 竞态关闭（极小窗口），视为干净退出
	case <-ctx.Done():
		// 进入排水：先撤销信号注册（再次 Ctrl-C 走默认终止强杀）。
		stop()
	}

	logger.Infow("shutdown signal received, draining",
		logger.StringField("timeout", cfg.ShutdownTimeout.String()))
	drainCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(drainCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("http server stopped cleanly")
	return nil
}
