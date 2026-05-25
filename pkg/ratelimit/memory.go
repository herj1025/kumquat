package ratelimit

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultCleanupInterval 默认空闲 limiter 清理间隔。
	DefaultCleanupInterval = 10 * time.Minute
	// DefaultMaxIdleDuration 默认限流器最大空闲时间，超过此时间会被清理。
	DefaultMaxIdleDuration = 30 * time.Minute
)

type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// MemoryLimiter 基于内存的令牌桶限流器。
//
// 使用方法:
//
//	limiter := NewMemoryLimiter(10, 20) // 每秒 10 个，burst 20
//	limiter.StartCleanup()              // 启动空闲清理
//	defer limiter.Stop()                // 停止清理
type MemoryLimiter struct {
	mu              sync.RWMutex
	entries         map[string]*limiterEntry
	rl              rate.Limit
	burst           int
	cleanupInterval time.Duration
	maxIdleDuration time.Duration

	allowed atomic.Int64
	denied  atomic.Int64

	stopCh  chan struct{}
	started atomic.Bool
}

// NewMemoryLimiter 创建内存限流器。
//
//	r     每秒允许的请求数（rate.Limit 类型）
//	burst 令牌桶容量
func NewMemoryLimiter(r rate.Limit, burst int) *MemoryLimiter {
	return &MemoryLimiter{
		entries:         make(map[string]*limiterEntry),
		rl:              r,
		burst:           burst,
		cleanupInterval: DefaultCleanupInterval,
		maxIdleDuration: DefaultMaxIdleDuration,
	}
}

// NewMemoryLimiterWithCleanup 创建内存限流器，并指定清理参数。
func NewMemoryLimiterWithCleanup(r rate.Limit, burst int, cleanupInterval, maxIdle time.Duration) *MemoryLimiter {
	return &MemoryLimiter{
		entries:         make(map[string]*limiterEntry),
		rl:              r,
		burst:           burst,
		cleanupInterval: cleanupInterval,
		maxIdleDuration: maxIdle,
	}
}

// Allow 检查 key 是否允许通过。
func (m *MemoryLimiter) Allow(_ context.Context, key string) (AllowResult, error) {
	entry := m.getOrCreate(key)
	entry.lastAccess = time.Now()

	ok := entry.limiter.Allow()
	remaining := int(math.Ceil(entry.limiter.Tokens()))
	if remaining < 0 {
		remaining = 0
	}

	if ok {
		m.allowed.Add(1)
	} else {
		m.denied.Add(1)
	}

	return AllowResult{
		Allowed:   ok,
		Remaining: remaining,
		Limit:     m.burst,
	}, nil
}

// Wait 阻塞直到 key 被允许通过或 ctx 被取消。
func (m *MemoryLimiter) Wait(ctx context.Context, key string) error {
	entry := m.getOrCreate(key)
	entry.lastAccess = time.Now()

	err := entry.limiter.Wait(ctx)
	if err == nil {
		m.allowed.Add(1)
	} else {
		m.denied.Add(1)
	}
	return err
}

// Allowed 返回累计通过的请求数。
func (m *MemoryLimiter) Allowed() int64 { return m.allowed.Load() }

// Denied 返回累计被限流的请求数。
func (m *MemoryLimiter) Denied() int64 { return m.denied.Load() }

// StartCleanup 启动空闲 limiter 的周期性清理。
func (m *MemoryLimiter) StartCleanup() {
	if !m.started.CompareAndSwap(false, true) {
		return
	}
	m.stopCh = make(chan struct{})
	go m.cleanupLoop()
}

// Stop 停止清理 goroutine。
func (m *MemoryLimiter) Stop() {
	if m.started.CompareAndSwap(true, false) {
		close(m.stopCh)
	}
}

func (m *MemoryLimiter) getOrCreate(key string) *limiterEntry {
	m.mu.RLock()
	entry, ok := m.entries[key]
	m.mu.RUnlock()

	if ok {
		return entry
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 双检确保不重复创建
	if entry, ok = m.entries[key]; ok {
		return entry
	}

	// 懒启动清理协程
	if !m.started.Load() {
		m.startCleanupLoop()
	}

	entry = &limiterEntry{
		limiter:    rate.NewLimiter(m.rl, m.burst),
		lastAccess: time.Now(),
	}
	m.entries[key] = entry
	return entry
}

func (m *MemoryLimiter) startCleanupLoop() {
	m.stopCh = make(chan struct{})
	m.started.Store(true)
	go m.cleanupLoop()
}

func (m *MemoryLimiter) cleanupLoop() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopCh:
			return
		}
	}
}

func (m *MemoryLimiter) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, entry := range m.entries {
		if now.Sub(entry.lastAccess) > m.maxIdleDuration {
			slog.Debug("removed idle rate limiter",
				"key", key,
				"idle", now.Sub(entry.lastAccess).Round(time.Second).String(),
			)
			delete(m.entries, key)
		}
	}
}
