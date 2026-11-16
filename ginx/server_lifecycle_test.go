package ginx

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// freePort 领一个空闲端口（测试并发下有微小竞争窗口，失败即换）。
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	require.NoError(t, err)
	return ":" + port
}

// signalSelf 向自身发终止信号——signal.NotifyContext 拦在默认终止之前，
// Serve 的等待腿被唤醒而测试进程存活（标准自信号测试法）。
func signalSelf(t *testing.T) {
	t.Helper()
	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGTERM))
}

func TestServe_HealthEndpointAndCleanShutdown(t *testing.T) {
	addr := freePort(t)
	engine := newTestEngine()
	served := make(chan struct{})

	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(engine, ServeConfig{
			Addr:            addr,
			ShutdownTimeout: 2 * time.Second,
			HealthPath:      "/healthz",
			OnHealthy:       func() { close(served) },
		})
	}()

	<-served // listen 已落地（OnHealthy 在启动日志后调用）

	resp, err := http.Get("http://127.0.0.1" + addr + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `"status":"ok"`)

	signalSelf(t)
	select {
	case err := <-errCh:
		assert.NoError(t, err, "SIGTERM 路径应干净返回 nil")
	case <-time.After(5 * time.Second):
		t.Fatal("Serve 未在 5s 内响应 SIGTERM")
	}
}

func TestServe_ListenFailureReturnsError(t *testing.T) {
	// 先占住端口。
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	_, port, _ := net.SplitHostPort(l.Addr().String())

	err = Serve(newTestEngine(), ServeConfig{Addr: "127.0.0.1:" + port})
	require.Error(t, err)
	assert.ErrorContains(t, err, "http listen")
}

func TestServe_HealthRouteEmpty_NoRoute(t *testing.T) {
	addr := freePort(t)
	engine := newTestEngine()
	served := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(engine, ServeConfig{
			Addr:            addr,
			ShutdownTimeout: time.Second,
			OnHealthy:       func() { close(served) },
		})
	}()

	<-served
	// 空 HealthPath 不挂路由：探针应 404。
	resp, err := http.Get("http://127.0.0.1" + addr + "/healthz")
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	signalSelf(t)
	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve 未在 5s 内响应 SIGTERM")
	}
}

// 外部 ctx 模式（v0.12.1）：调用方自有信号 ctx——Serve 不重复注册信号，
// ctx 取消即排水；ReadTimeout/WriteTimeout 透传 http.Server。
func TestServe_ExternalContext(t *testing.T) {
	addr := freePort(t)
	engine := newTestEngine()
	served := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	srvCfg := ServeConfig{
		Addr:            addr,
		ShutdownTimeout: 2 * time.Second,
		HealthPath:      "/healthz",
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		Context:         ctx,
		OnHealthy:       func() { close(served) },
	}
	errCh := make(chan error, 1)
	go func() { errCh <- Serve(engine, srvCfg) }()

	<-served
	resp, err := http.Get("http://127.0.0.1" + addr + "/healthz")
	require.NoError(t, err)
	resp.Body.Close()

	cancel() // 外部取消（替代信号）
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("外部 ctx 取消未触发排水")
	}
}
