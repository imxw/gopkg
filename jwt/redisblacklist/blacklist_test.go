package redisblacklist

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TokenBlacklist 语义面：add/contains/remove 往返 + 过期 token 不入黑
// （TTL≤0 即已过期，无锁定意义）+ 空 jti 防御 + 自定义前缀。miniredis
// 内存实现承接 Redis 协议，无需容器。

func newTestBlacklist(t *testing.T) (*TokenBlacklist, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	return New(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "test:bl"), mr
}

func TestBlacklist_AddContainsRemove(t *testing.T) {
	ctx := context.Background()
	tb, _ := newTestBlacklist(t)
	expires := time.Now().Add(time.Hour)

	if err := tb.AddToken(ctx, "jti-1", expires); err != nil {
		t.Fatalf("add: %v", err)
	}
	contains, err := tb.Contains(ctx, "jti-1")
	if err != nil {
		t.Fatalf("contains: %v", err)
	}
	if !contains {
		t.Fatal("added jti should be blacklisted")
	}

	if err := tb.Remove(ctx, "jti-1"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	contains, err = tb.Contains(ctx, "jti-1")
	if err != nil {
		t.Fatalf("contains after remove: %v", err)
	}
	if contains {
		t.Fatal("removed jti should not be blacklisted")
	}
}

func TestBlacklist_UnknownJTI_NotContained(t *testing.T) {
	ctx := context.Background()
	tb, _ := newTestBlacklist(t)
	contains, err := tb.Contains(ctx, "never-added")
	if err != nil {
		t.Fatalf("contains: %v", err)
	}
	if contains {
		t.Fatal("unknown jti should not be blacklisted")
	}
}

// 已过期 token 不入黑：TTL ≤ 0 时 Set 会被 Redis 拒绝（或即刻消失），
// 语义上「过期即失效」，黑名单无意义——直接跳过且不报错。
func TestBlacklist_ExpiredToken_Skipped(t *testing.T) {
	ctx := context.Background()
	tb, _ := newTestBlacklist(t)
	if err := tb.AddToken(ctx, "expired", time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("expired add should be a no-op success, got %v", err)
	}
	contains, err := tb.Contains(ctx, "expired")
	if err != nil {
		t.Fatalf("contains: %v", err)
	}
	if contains {
		t.Fatal("expired token should not be blacklisted")
	}
}

func TestBlacklist_EmptyJTI_Defensive(t *testing.T) {
	ctx := context.Background()
	tb, _ := newTestBlacklist(t)
	if err := tb.AddToken(ctx, "", time.Now().Add(time.Hour)); err == nil {
		t.Fatal("empty jti add should be rejected")
	}
	contains, err := tb.Contains(ctx, "")
	if err != nil {
		t.Fatalf("empty jti contains should be a clean false, got %v", err)
	}
	if contains {
		t.Fatal("empty jti should never match")
	}
	if err := tb.Remove(ctx, ""); err != nil {
		t.Fatalf("empty jti remove should be a no-op success, got %v", err)
	}
}

func TestNew_DefaultPrefix(t *testing.T) {
	mr := miniredis.RunT(t)
	tb := New(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "")
	if tb.prefix != "token_blacklist" {
		t.Fatalf("default prefix = %q, want token_blacklist", tb.prefix)
	}
	custom := New(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "x")
	if custom.prefix != "x" {
		t.Fatalf("custom prefix = %q, want x", custom.prefix)
	}
}
