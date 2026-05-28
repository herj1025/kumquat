package segmentlock

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMutex_LockUnlock(t *testing.T) {
	sl := New()
	mu := sl.NewMutex("test:key:1")
	ctx := context.Background()

	if err := mu.Lock(ctx); err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	if err := mu.Unlock(ctx); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}
}

func TestMutex_MutualExclusion(t *testing.T) {
	sl := New()
	mu := sl.NewMutex("test:key:exclusion")
	ctx := context.Background()

	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := mu.Lock(ctx); err != nil {
				t.Errorf("failed to lock: %v", err)
				return
			}
			counter++
			time.Sleep(10 * time.Millisecond)
			mu.Unlock(ctx)
		}()
	}

	wg.Wait()

	if counter != 10 {
		t.Errorf("expected counter=10, got %d", counter)
	}
}

func TestMutex_DifferentKeysParallelism(t *testing.T) {
	sl := New()
	ctx := context.Background()

	mu1 := sl.NewMutex("test:key:a")
	mu2 := sl.NewMutex("test:key:b")

	if err := mu1.Lock(ctx); err != nil {
		t.Fatalf("failed to lock mu1: %v", err)
	}
	defer mu1.Unlock(ctx)

	locked := make(chan struct{}, 1)
	go func() {
		mu2.Lock(ctx)
		close(locked)
		mu2.Unlock(ctx)
	}()

	select {
	case <-locked:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("mu2 should be able to lock while mu1 is held (different keys)")
	}
}

func TestMutex_TryLock(t *testing.T) {
	sl := New()
	mu := sl.NewMutex("test:key:trylock")
	ctx := context.Background()

	if err := mu.TryLock(ctx); err != nil {
		t.Fatalf("TryLock on free lock should succeed: %v", err)
	}

	if err := mu.TryLock(ctx); err != ErrLockNotObtained {
		t.Fatalf("TryLock on held lock should return ErrLockNotObtained, got %v", err)
	}

	mu.Unlock(ctx)

	if err := mu.TryLock(ctx); err != nil {
		t.Fatalf("TryLock after unlock should succeed: %v", err)
	}
	mu.Unlock(ctx)
}

func TestMutex_ContextCancel(t *testing.T) {
	sl := New()
	mu := sl.NewMutex("test:key:cancel")

	ctx, cancel := context.WithCancel(context.Background())

	mu.Lock(ctx)
	cancel()

	err := mu.Lock(ctx)
	if err == nil {
		t.Fatal("Lock with cancelled context should return error")
	}
	mu.Unlock(context.Background())

	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()

	mu.Lock(context.Background())
	defer mu.Unlock(context.Background())

	start := time.Now()
	err = mu.Lock(ctx2)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Lock with timeout should return error when lock held")
	}

	if elapsed < 40*time.Millisecond {
		t.Errorf("Lock with timeout should have waited, got %v", elapsed)
	}
}

func TestSegmentLock_HighConcurrency(t *testing.T) {
	sl := New(WithSegmentCount(128))
	var counter int64
	var wg sync.WaitGroup
	ctx := context.Background()

	numGoroutines := 50
	iterations := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("test:key:concurrent:%d:%d", id, j)
				mu := sl.NewMutex(key)
				if err := mu.Lock(ctx); err != nil {
					t.Errorf("goroutine %d failed to lock: %v", id, err)
					return
				}
				atomic.AddInt64(&counter, 1)
				mu.Unlock(ctx)
			}
		}(i)
	}

	wg.Wait()

	expected := int64(numGoroutines * iterations)
	if counter != expected {
		t.Errorf("expected counter=%d, got %d", expected, counter)
	}
}

func TestSegmentLock_Distribution(t *testing.T) {
	segmentCount := 64

	segmentCounts := make(map[uint32]int)
	numKeys := 10000

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("test:dist:key:%d", i)
		h := fnv.New32a()
		h.Write([]byte(key))
		idx := h.Sum32() % uint32(segmentCount)
		segmentCounts[idx]++
	}

	if len(segmentCounts) < segmentCount*8/10 {
		t.Errorf("keys should be distributed across most segments, got %d/%d segments used", len(segmentCounts), segmentCount)
	}

	expected := numKeys / segmentCount
	for idx, count := range segmentCounts {
		lower := expected / 3
		upper := expected * 3
		if count < lower || count > upper {
			t.Errorf("segment %d has %d keys, expected around %d (too unbalanced)", idx, count, expected)
		}
	}
}

func BenchmarkMutex_LockUnlock(b *testing.B) {
	sl := New()
	mu := sl.NewMutex("bench:key")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock(ctx)
		mu.Unlock(ctx)
	}
}

func BenchmarkMutex_Contended(b *testing.B) {
	sl := New()
	mu := sl.NewMutex("bench:contended")
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock(ctx)
			mu.Unlock(ctx)
		}
	})
}

func BenchmarkSegmentLock_DifferentKeys(b *testing.B) {
	sl := New(WithSegmentCount(128))
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench:parallel:%d", i)
			mu := sl.NewMutex(key)
			mu.Lock(ctx)
			mu.Unlock(ctx)
			i++
		}
	})
}
