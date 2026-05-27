// Package ratelimit 提供限流器功能，支持内存和 Redis 两种实现。
package ratelimit

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AllowResult 限流检查结果。
type AllowResult struct {
	Allowed   bool // 是否允许请求通过
	Remaining int  // 当前 key 剩余可用令牌数
	Limit     int  // 令牌桶容量（burst）
}

// Limiter 限流器接口。
type Limiter interface {
	// Allow 检查是否允许请求通过，返回详细结果。
	Allow(ctx context.Context, key string) (AllowResult, error)
	// Wait 阻塞直到允许请求通过或 ctx 被取消。
	Wait(ctx context.Context, key string) error
}

var (
	// ErrEmptyKey 表示限流 key 为空。
	ErrEmptyKey = errors.New("rate limit key is empty")
)

// limiterConfig 中间件配置。
type limiterConfig struct {
	keyFunc        func(*gin.Context) string
	errorHandler   func(*gin.Context, error)
	denyHandler    func(*gin.Context, AllowResult)
	emptyKeyPolicy func(*gin.Context)
	enableHeaders  bool
}

// LimiterOption 中间件配置选项。
type LimiterOption func(*limiterConfig)

// WithKeyFunc 设置自定义限流 key 提取函数。
func WithKeyFunc(fn func(*gin.Context) string) LimiterOption {
	return func(c *limiterConfig) { c.keyFunc = fn }
}

// WithErrorHandler 设置内部错误时的处理函数（如 Redis 不可达）。
func WithErrorHandler(h func(*gin.Context, error)) LimiterOption {
	return func(c *limiterConfig) { c.errorHandler = h }
}

// WithDenyHandler 设置限流拒绝时的处理函数。
func WithDenyHandler(h func(*gin.Context, AllowResult)) LimiterOption {
	return func(c *limiterConfig) { c.denyHandler = h }
}

// WithEmptyKeyDeny 配置空 key 时拒绝请求（默认行为是跳过限流）。
func WithEmptyKeyDeny() LimiterOption {
	return func(c *limiterConfig) {
		c.emptyKeyPolicy = func(ctx *gin.Context) {
			c.errorHandler(ctx, ErrEmptyKey)
			ctx.Abort()
		}
	}
}

// WithDisableHeaders 禁用 X-RateLimit-* 响应头。
func WithDisableHeaders() LimiterOption {
	return func(c *limiterConfig) { c.enableHeaders = false }
}

func defaultConfig() limiterConfig {
	return limiterConfig{
		keyFunc: func(c *gin.Context) string { return c.ClientIP() },
		errorHandler: func(c *gin.Context, err error) {
			slog.Error("rate limit internal error", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    50001,
				"message": "Internal server error",
				"data":    nil,
			})
		},
		denyHandler: func(c *gin.Context, r AllowResult) {
			c.Header("Retry-After", strconv.Itoa(r.Limit))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "Rate limit exceeded",
				"data":    nil,
			})
		},
		emptyKeyPolicy: func(c *gin.Context) {
			c.Next()
		},
		enableHeaders: true,
	}
}

// Middleware 创建可配置的限流中间件。
func Middleware(limiter Limiter, opts ...LimiterOption) gin.HandlerFunc {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(c *gin.Context) {
		key := cfg.keyFunc(c)
		if key == "" {
			cfg.emptyKeyPolicy(c)
			return
		}

		result, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			cfg.errorHandler(c, err)
			c.Abort()
			return
		}

		if cfg.enableHeaders {
			c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		}

		if !result.Allowed {
			cfg.denyHandler(c, result)
			c.Abort()
			return
		}

		c.Next()
	}
}
