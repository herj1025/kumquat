package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// 确保 MemoryLimiter 实现了 Limiter 接口
var _ Limiter = (*MemoryLimiter)(nil)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------- MemoryLimiter ----------

func TestNewMemoryLimiter(t *testing.T) {
	l := NewMemoryLimiter(10, 20)
	if l == nil {
		t.Fatal("NewMemoryLimiter returned nil")
	}
	if l.burst != 20 {
		t.Errorf("burst = %d, want 20", l.burst)
	}
}

func TestMemoryLimiter_Allow_Basic(t *testing.T) {
	l := NewMemoryLimiter(10, 5)

	// burst 个请求都应该允许
	for i := 0; i < 5; i++ {
		result, err := l.Allow(context.Background(), "key1")
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !result.Allowed {
			t.Errorf("iter %d: expected allowed=true", i)
		}
	}

	// 第 6 个应该被限流
	result, err := l.Allow(context.Background(), "key1")
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if result.Allowed {
		t.Error("expected allowed=false after burst exhausted")
	}
	if result.Remaining != 0 {
		t.Errorf("expected Remaining=0, got %d", result.Remaining)
	}
	if result.Limit != 5 {
		t.Errorf("expected Limit=5, got %d", result.Limit)
	}
}

func TestMemoryLimiter_Allow_IndependentKeys(t *testing.T) {
	l := NewMemoryLimiter(10, 3)

	// 两个独立 key 互不影响
	for i := 0; i < 3; i++ {
		r1, _ := l.Allow(context.Background(), "a")
		r2, _ := l.Allow(context.Background(), "b")
		if !r1.Allowed || !r2.Allowed {
			t.Fatalf("iter %d: both keys should be allowed", i)
		}
	}
}

func TestMemoryLimiter_Allow_WaitRefill(t *testing.T) {
	l := NewMemoryLimiter(100, 1) // 1 token per 10ms

	result, err := l.Allow(context.Background(), "key")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed {
		t.Fatal("first request should be allowed")
	}
	if result.Remaining != 0 {
		t.Errorf("after consuming burst=1, remaining should be 0, got %d", result.Remaining)
	}

	// 等待超过一个 token 产生时间
	time.Sleep(15 * time.Millisecond)

	result, err = l.Allow(context.Background(), "key")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed {
		t.Error("request after refill should be allowed")
	}
}

func TestMemoryLimiter_Concurrency(t *testing.T) {
	l := NewMemoryLimiter(1000, 100)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				l.Allow(context.Background(), "concurrent-key")
			}
		}()
	}
	wg.Wait()

	// 验证没有 panic 且统计正常
	total := l.Allowed() + l.Denied()
	if total == 0 {
		t.Error("expected some requests to be processed")
	}
}

func TestMemoryLimiter_Stats(t *testing.T) {
	l := NewMemoryLimiter(1, 1)

	l.Allow(context.Background(), "k")
	l.Allow(context.Background(), "k") // 应该被限流

	if l.Allowed() != 1 {
		t.Errorf("Allowed = %d, want 1", l.Allowed())
	}
	if l.Denied() != 1 {
		t.Errorf("Denied = %d, want 1", l.Denied())
	}
}

func TestMemoryLimiter_Wait(t *testing.T) {
	l := NewMemoryLimiter(100, 1)

	// 等待不需要阻塞（令牌可用）
	ctx := context.Background()
	if err := l.Wait(ctx, "key"); err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if l.Allowed() != 1 {
		t.Errorf("Allowed = %d, want 1", l.Allowed())
	}
}

func TestMemoryLimiter_WaitContextCancel(t *testing.T) {
	l := NewMemoryLimiter(0.001, 1)      // ~1 request per 1000s
	l.Allow(context.Background(), "key") // consume the only token

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := l.Wait(ctx, "key")
	if err == nil {
		t.Error("expected context deadline exceeded error")
	}
}

func TestMemoryLimiter_Cleanup(t *testing.T) {
	l := NewMemoryLimiterWithCleanup(10, 5, 50*time.Millisecond, 50*time.Millisecond)
	l.StartCleanup()
	defer l.Stop()

	// 创建一些 limiter
	l.Allow(context.Background(), "clean-me")
	l.Allow(context.Background(), "clean-me-too")
	if l.entries == nil {
		t.Fatal("entries map is nil")
	}
	if len(l.entries) != 2 {
		t.Errorf("entries = %d, want 2", len(l.entries))
	}

	// 等待超过 maxIdleDuration
	time.Sleep(100 * time.Millisecond)

	if len(l.entries) != 0 {
		t.Errorf("after cleanup, entries = %d, want 0", len(l.entries))
	}
}

func TestMemoryLimiter_CleanupActiveEntry(t *testing.T) {
	l := NewMemoryLimiterWithCleanup(10, 5, 50*time.Millisecond, 80*time.Millisecond)
	l.StartCleanup()
	defer l.Stop()

	l.Allow(context.Background(), "active")

	// 在接近清理阈值时刷新 lastAccess
	time.Sleep(50 * time.Millisecond)
	l.Allow(context.Background(), "active") // 刷新

	// 等待一个清理周期，但 active 应该因为被刷新而保留
	time.Sleep(60 * time.Millisecond)

	if len(l.entries) != 1 {
		t.Errorf("active entry should survive cleanup, entries = %d, want 1", len(l.entries))
	}
}

func TestMemoryLimiter_StopCleanup(t *testing.T) {
	l := NewMemoryLimiterWithCleanup(10, 5, 10*time.Millisecond, 10*time.Millisecond)
	l.StartCleanup()

	l.Allow(context.Background(), "k")
	l.Stop()

	// 停止后不应该再有清理
	time.Sleep(50 * time.Millisecond)
	if len(l.entries) != 1 {
		t.Errorf("after Stop, entries should remain, got %d", len(l.entries))
	}
}

// ---------- Middleware ----------

func TestMiddleware_Allow(t *testing.T) {
	l := NewMemoryLimiter(10, 10)
	mw := Middleware(l)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	var handled bool
	next := func() { handled = true }
	mw(c)
	next()
	if !handled {
		t.Error("next handler should be called when allowed")
	}
}

func TestMiddleware_Deny(t *testing.T) {
	l := NewMemoryLimiter(0.001, 1) // 1 request per 1000s

	mw := Middleware(l)

	// 第一个请求通过
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	mw(c1)

	// 第二个请求被限流
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	mw(c2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
}

func TestMiddleware_Headers(t *testing.T) {
	l := NewMemoryLimiter(10, 5)
	mw := Middleware(l)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	mw(c)

	limit := w.Header().Get("X-RateLimit-Limit")
	if limit != "5" {
		t.Errorf("X-RateLimit-Limit = %q, want \"5\"", limit)
	}

	remaining := w.Header().Get("X-RateLimit-Remaining")
	if remaining != "4" {
		t.Errorf("X-RateLimit-Remaining = %q, want \"4\"", remaining)
	}
}

func TestMiddleware_DisableHeaders(t *testing.T) {
	l := NewMemoryLimiter(10, 5)
	mw := Middleware(l, WithDisableHeaders())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	mw(c)

	if v := w.Header().Get("X-RateLimit-Limit"); v != "" {
		t.Errorf("expected no X-RateLimit-Limit header, got %q", v)
	}
}

func TestMiddleware_WithKeyFunc(t *testing.T) {
	l := NewMemoryLimiter(10, 3)
	mw := Middleware(l, WithKeyFunc(func(c *gin.Context) string {
		return c.Query("api_key")
	}))

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/?api_key=test-user", nil)
		mw(c)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("iter %d: unexpected deny", i)
		}
	}

	// 用不同的 key 应该继续通过
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?api_key=other-user", nil)
	mw(c)
	if w.Code == http.StatusTooManyRequests {
		t.Error("different key should not be rate limited")
	}
}

func TestMiddleware_EmptyKeyDefault(t *testing.T) {
	l := NewMemoryLimiter(10, 5)
	mw := Middleware(l, WithKeyFunc(func(c *gin.Context) string {
		return "" // empty key
	}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	var handled bool
	mw(c)
	handled = true

	if !handled {
		t.Error("empty key should skip rate limiting by default")
	}
}

func TestMiddleware_EmptyKeyDeny(t *testing.T) {
	l := NewMemoryLimiter(10, 5)
	mw := Middleware(l,
		WithKeyFunc(func(c *gin.Context) string { return "" }),
		WithEmptyKeyDeny(),
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	mw(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d (500 internal for empty key denial)",
			w.Code, http.StatusInternalServerError)
	}
}

func TestMiddleware_ErrorHandler(t *testing.T) {
	l := NewMemoryLimiter(10, 5)
	var captured error
	mw := Middleware(l,
		WithErrorHandler(func(c *gin.Context, err error) {
			captured = err
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code": 503, "message": "custom error",
			})
		}),
		WithKeyFunc(func(c *gin.Context) string { return "" }),
		WithEmptyKeyDeny(),
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	mw(c)

	if captured == nil {
		t.Error("expected error to be captured")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestMiddleware_DenyHandler(t *testing.T) {
	l := NewMemoryLimiter(0.001, 1)
	var captured AllowResult
	mw := Middleware(l,
		WithDenyHandler(func(c *gin.Context, r AllowResult) {
			captured = r
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"code": 429, "message": "custom deny"})
		}),
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	mw(c) // consume

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	mw(c2) // denied

	if captured.Allowed {
		t.Error("expected captured result to have Allowed=false")
	}
	if captured.Limit != 1 {
		t.Errorf("Limit = %d, want 1", captured.Limit)
	}
}

// ---------- Helpers ----------

func TestMiddleware_WithClientIPKey(t *testing.T) {
	mw := Middleware(NewMemoryLimiter(10, 5), WithKeyFunc(func(c *gin.Context) string {
		return c.ClientIP()
	}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "192.168.1.1:12345"
	mw(c)

	c.Header("X-RateLimit-Limit", "") // reset for next check

	// 请求应该通过
	if w.Code == http.StatusTooManyRequests {
		t.Fatal("unexpected deny on first request")
	}
}

func TestMiddleware_WithPathKey(t *testing.T) {
	// 创建一个 Gin 路由来测试按路径限流
	router := gin.New()
	router.Use(Middleware(NewMemoryLimiter(10, 3), WithKeyFunc(func(c *gin.Context) string {
		return c.FullPath()
	})))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestMiddleware_WithCustomKey(t *testing.T) {
	mw := Middleware(NewMemoryLimiter(10, 2), WithKeyFunc(func(c *gin.Context) string {
		return c.GetHeader("X-User-ID")
	}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("X-User-ID", "user-42")
	mw(c)

	if w.Code == http.StatusTooManyRequests {
		t.Error("unexpected deny on first request")
	}
}

func TestMiddleware_WithLimiterInterface(t *testing.T) {
	// 验证单一入口 Middleware 可以直接和 Limiter 接口整合
	l := NewMemoryLimiter(10, 5)
	h := Middleware(l, WithKeyFunc(func(c *gin.Context) string {
		return c.ClientIP()
	}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	h(c)

	if w.Code == http.StatusTooManyRequests {
		t.Error("unexpected deny")
	}
}

// ---------- Benchmark ----------

func BenchmarkMemoryLimiter_Allow(b *testing.B) {
	l := NewMemoryLimiter(1000, 1000)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			l.Allow(ctx, "bench-key")
			i++
		}
	})
}
