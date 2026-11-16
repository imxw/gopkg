package ginx

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"

	"github.com/imxw/gopkg/errorx"
	"github.com/imxw/gopkg/logger"
)

// RedisLimiter 分布式限流（GCRA / leaky bucket，go-redis 官方生态
// redis_rate 封装）。消费方按凭据类型解析 key 与档位，本类型只管计数。
type RedisLimiter struct {
	limiter *redis_rate.Limiter
	prefix  string
}

// NewRedisLimiter wraps a Redis client (single/cluster/sentinel) with a key
// prefix ("myapp:rl" → key "myapp:rl:<key>").
func NewRedisLimiter(rdb redis.UniversalClient, prefix string) *RedisLimiter {
	return &RedisLimiter{
		limiter: redis_rate.NewLimiter(rdb),
		prefix:  prefix,
	}
}

// Take consumes one token from the bucket for key.
//
// capacity = bucket size (burst ceiling); refillPerMin = steady rate per
// minute. Both in "hits". At steady state the rate equals refillPerMin
// exactly while still allowing an initial full-minute burst (CI 并发起手
// 不该被掐).
//
// Returns allowed=false with remaining=0 and retryAfter = time until the
// next token.
func (l *RedisLimiter) Take(ctx context.Context, key string, capacity, refillPerMin int64) (allowed bool, remaining int64, retryAfter time.Duration, err error) {
	if capacity <= 0 || refillPerMin <= 0 {
		return false, 0, 0, fmt.Errorf("take rate limit token: capacity and refill must be positive (got %d/%d)", capacity, refillPerMin)
	}
	res, err := l.limiter.AllowN(ctx, l.prefix+":"+key, redis_rate.Limit{
		Rate:   int(refillPerMin),
		Period: time.Minute,
		Burst:  int(capacity),
	}, 1)
	if err != nil {
		return false, 0, 0, fmt.Errorf("take rate limit token: %w", err)
	}
	if res.Allowed == 0 {
		return false, int64(res.Remaining), res.RetryAfter, nil
	}
	return true, int64(res.Remaining), 0, nil
}

// RateLimitConfig wires the per-request limiter policy. KeyFn and LimitFn
// are caller-owned (凭据类型分档是业务面)；resolver 返回 false 跳过限流
// （未鉴权路径、白名单身份等）。
type RateLimitConfig struct {
	// KeyFn 限流维度键（凭据 ID / 来源 IP…）；空串跳过。
	KeyFn func(c *gin.Context) string
	// LimitFn 档位解析：capacity（突发）+ refillPerMin（稳态每分钟）。
	LimitFn func(c *gin.Context) (capacity, refillPerMin int64)
	// FailOpen 计数存储不可用时放行（可用性优先）——false 则 fail-closed
	// 返回 500。D48f 先例：限流是保命闸不是锁，默认 fail-open。
	FailOpen bool
}

// RateLimit 分布式限流中间件：超档出 429 信封（errorx.ErrTooManyRequests）
// + Retry-After / X-RateLimit-Remaining 头；计数存储故障按 FailOpen 放行
// 并落 WARN。
func RateLimit(limiter *RedisLimiter, cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := cfg.KeyFn(c)
		if key == "" || cfg.LimitFn == nil {
			c.Next()
			return
		}
		capacity, refill := cfg.LimitFn(c)
		if capacity <= 0 || refill <= 0 {
			c.Next()
			return
		}

		allowed, remaining, retryAfter, err := limiter.Take(c.Request.Context(), key, capacity, refill)
		if err != nil {
			if cfg.FailOpen {
				logger.CtxWarnw(c.Request.Context(), "rate limiter unavailable, fail-open",
					logger.StringField("key", key), logger.ErrorField(err))
				c.Next()
				return
			}
			InternalError(c, "rate limiter unavailable")
			c.Abort()
			return
		}

		if remaining >= 0 {
			c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		}
		if !allowed {
			if retryAfter > 0 {
				c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())+1))
			}
			Error(c, errorx.ErrTooManyRequests.WithMessage(
				fmt.Sprintf("请求过于频繁，请在 %ds 后重试", max(1, int(retryAfter.Seconds())))))
			c.Abort()
			return
		}
		c.Next()
	}
}
