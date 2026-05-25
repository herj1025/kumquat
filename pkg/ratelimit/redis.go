package ratelimit

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter 基于 Redis 的固定窗口限流器。
//
// 使用 INCR + EXPIRE 实现，配合 Lua 脚本保证原子性。
// 适用于分布式场景下的速率限制。
type RedisLimiter struct {
	client    redis.UniversalClient
	rate      int
	window    time.Duration
	keyPrefix string

	allowed atomic.Int64
	denied  atomic.Int64
}

// NewRedisLimiter 创建 Redis 限流器。
//
//	client Redis 客户端（支持单机、哨兵、集群模式）
//	rate   窗口内允许的最大请求数
//	window 时间窗口，如 time.Second、time.Minute
func NewRedisLimiter(client redis.UniversalClient, rate int, window time.Duration) *RedisLimiter {
	return &RedisLimiter{
		client:    client,
		rate:      rate,
		window:    window,
		keyPrefix: "ratelimit:",
	}
}

// rateLimitScript 原子检查并递增计数
//
// KEYS[1] — 限流 key
// ARGV[1] — 窗口上限
// ARGV[2] — 窗口秒数
// 返回 {allowed(0/1), remaining}
var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local current = redis.call("GET", key)
if current and tonumber(current) >= limit then
    return {0, 0}
end

local count = redis.call("INCR", key)
if count == 1 then
    redis.call("EXPIRE", key, window)
end

local remaining = limit - count
if remaining < 0 then
    remaining = 0
end

return {1, remaining}
`)

// Allow 检查 key 是否允许通过。
func (r *RedisLimiter) Allow(ctx context.Context, key string) (AllowResult, error) {
	vals, err := rateLimitScript.Run(ctx, r.client,
		[]string{r.keyPrefix + key},
		r.rate, int(r.window.Seconds()),
	).Int64Slice()
	if err != nil {
		return AllowResult{}, err
	}

	allowed := vals[0] == 1
	remaining := int(vals[1])

	if allowed {
		r.allowed.Add(1)
	} else {
		r.denied.Add(1)
	}

	return AllowResult{
		Allowed:   allowed,
		Remaining: remaining,
		Limit:     r.rate,
	}, nil
}

// Wait 轮询等待直到 key 被允许或 ctx 超时。
func (r *RedisLimiter) Wait(ctx context.Context, key string) error {
	const (
		initialDelay = 50 * time.Millisecond
		maxDelay     = 500 * time.Millisecond
	)

	delay := initialDelay
	for {
		result, err := r.Allow(ctx, key)
		if err != nil {
			return err
		}
		if result.Allowed {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}

// Allowed 返回累计通过的请求数。
func (r *RedisLimiter) Allowed() int64 { return r.allowed.Load() }

// Denied 返回累计被限流的请求数。
func (r *RedisLimiter) Denied() int64 { return r.denied.Load() }
