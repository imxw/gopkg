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
	redis  *redis.Client
	prefix string
}

// New creates a new TokenBlacklist with the given Redis client and key prefix.
func New(redisClient *redis.Client, prefix string) *TokenBlacklist {
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

// Compile-time check that TokenBlacklist satisfies jwt.Blacklist.
var _ jwt.Blacklist = (*TokenBlacklist)(nil)
