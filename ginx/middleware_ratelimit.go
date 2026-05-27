package ginx

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/imxw/gopkg/errorx"
	"github.com/imxw/gopkg/logger"
)

// RateLimitConfig configures fixed-window rate limiting.
type RateLimitConfig struct {
	Window      time.Duration
	MaxRequests int
	KeyPrefix   string
}

// RateLimiter provides Redis-based rate limiting per client IP.
type RateLimiter struct {
	rdb  *redis.Client
	conf RateLimitConfig
}

// NewRateLimiter creates a RateLimiter backed by Redis.
func NewRateLimiter(rdb *redis.Client, conf RateLimitConfig) *RateLimiter {
	return &RateLimiter{rdb: rdb, conf: conf}
}

// Middleware returns gin middleware that rate-limits by client IP.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	script := redis.NewScript(`
		local count = redis.call("INCR", KEYS[1])
		if count == 1 then
			redis.call("EXPIRE", KEYS[1], ARGV[1])
		end
		return count
	`)
	return func(c *gin.Context) {
		key := fmt.Sprintf("%s:%s", rl.conf.KeyPrefix, c.ClientIP())
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		count, err := script.Run(ctx, rl.rdb, []string{key}, int(rl.conf.Window.Seconds())).Int64()
		if err != nil {
			logger.CtxWarnw(c.Request.Context(), "rate limiter redis error", "error", err)
			c.Next()
			return
		}
		if count > int64(rl.conf.MaxRequests) {
			Error(c, errorx.ErrTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}
