// Package redisblacklist provides a Redis-backed JWT token blacklist.
package redisblacklist

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/imxw/gopkg/jwt"
)

// TokenBlacklist manages revoked tokens via Redis.
type TokenBlacklist struct {
	redis  redis.UniversalClient
	prefix string
}

// New creates a new TokenBlacklist with the given Redis client and key prefix.
// Accepts redis.UniversalClient so single-node, Cluster and Sentinel
// deployments all plug in.
func New(redisClient redis.UniversalClient, prefix string) *TokenBlacklist {
	if prefix == "" {
		prefix = "token_blacklist"
	}
	return &TokenBlacklist{
		redis:  redisClient,
		prefix: prefix,
	}
}

func (tb *TokenBlacklist) blacklistKey(jti string) string {
	return fmt.Sprintf("%s:%s", tb.prefix, jti)
}

// AddToken adds a token to the blacklist, computing TTL from its expiration time.
func (tb *TokenBlacklist) AddToken(ctx context.Context, jti string, expiresAt time.Time) error {
	if jti == "" {
		return fmt.Errorf("jti cannot be empty")
	}

	if expiresAt.Before(time.Now()) {
		return nil
	}

	key := tb.blacklistKey(jti)
	expiration := time.Until(expiresAt)
	err := tb.redis.Set(ctx, key, "1", expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}
	return nil
}

// AddIfAbsent 原子地把 jti 入黑（SET NX + TTL）：key 不存在时写入并返回
// absent=true（本调用完成占位）；已存在返回 absent=false（判定重放）。
// jwt.AtomicBlacklist 的 Redis 实现——刷新轮换经此消除 check-then-act 竞态。
func (tb *TokenBlacklist) AddIfAbsent(ctx context.Context, jti string, expiresAt time.Time) (bool, error) {
	if jti == "" {
		return false, fmt.Errorf("jti cannot be empty")
	}

	// 已过期的 token 无锁定意义：视为可占位（轮换路径上解析层已先拒过期 token）。
	if expiresAt.Before(time.Now()) {
		return true, nil
	}

	claimed, err := tb.redis.SetNX(ctx, tb.blacklistKey(jti), "1", time.Until(expiresAt)).Result()
	if err != nil {
		return false, fmt.Errorf("failed to claim token blacklist entry: %w", err)
	}
	return claimed, nil
}

// Contains checks whether a token is in the blacklist.
func (tb *TokenBlacklist) Contains(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}

	key := tb.blacklistKey(jti)
	result, err := tb.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return result > 0, nil
}

// Remove removes a token from the blacklist.
func (tb *TokenBlacklist) Remove(ctx context.Context, jti string) error {
	if jti == "" {
		return nil
	}

	key := tb.blacklistKey(jti)
	err := tb.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to remove token from blacklist: %w", err)
	}

	return nil
}

// Compile-time checks that TokenBlacklist satisfies jwt.Blacklist and the
// optional atomic rotation extension.
var (
	_ jwt.Blacklist       = (*TokenBlacklist)(nil)
	_ jwt.AtomicBlacklist = (*TokenBlacklist)(nil)
)
