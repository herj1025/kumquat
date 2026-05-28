package distlock

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// setupRedisClient 连接到测试用的 Redis
// 注意：这里硬编码了配置中的地址，实际项目中应该从环境变量或配置文件读取
func setupRedisClient(t *testing.T) redis.UniversalClient {
	opts := &redis.UniversalOptions{
		Addrs:    []string{"172.18.79.100:6379"},
		Password: "Redis$951025",
	}
	client := redis.NewUniversalClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("skipping test: failed to connect to redis: %v", err)
	}
	return client
}

func TestMutex_LockUnlock(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	distClient := NewRedisClient(client)
	lockKey := "test:lock:basic"

	// 1. 获取锁
	mu := distClient.NewMutex(lockKey)
	ctx := context.Background()

	if err := mu.Lock(ctx); err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// 2. 验证锁是否存在
	if exists := client.Exists(ctx, lockKey).Val(); exists != 1 {
		t.Errorf("lock key should exist in redis")
	}

	// 3. 释放锁
	if err := mu.Unlock(ctx); err != nil {
		t.Errorf("failed to unlock: %v", err)
	}

	// 4. 验证锁已释放
	// 注意：由于 Pub/Sub 可能会有一点点延迟，或者 Unlock 即使 key 没了也可能返回 nil (取决于实现)
	// 但我们的 Unlock 实现会显式 Release
	// 稍等一下让 Release 完成（通常是同步的）
	if exists := client.Exists(ctx, lockKey).Val(); exists != 0 {
		t.Errorf("lock key should not exist in redis after unlock")
	}
}

func TestMutex_Watchdog(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	distClient := NewRedisClient(client)
	lockKey := "test:lock:watchdog"

	// 设置较短的过期时间，以便快速验证看门狗
	expiration := 2 * time.Second
	mu := distClient.NewMutex(lockKey, WithExpiration(expiration))
	ctx := context.Background()

	if err := mu.Lock(ctx); err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	defer mu.Unlock(ctx)

	// 等待超过过期时间，验证锁是否还被持有
	time.Sleep(3 * time.Second)

	if exists := client.Exists(ctx, lockKey).Val(); exists != 1 {
		t.Errorf("lock should be held by watchdog")
	}
}

func TestMutex_Wait(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	distClient := NewRedisClient(client)
	lockKey := "test:lock:wait"

	// 协程 A 先获取锁
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		muA := distClient.NewMutex(lockKey, WithExpiration(5*time.Second))
		if err := muA.Lock(context.Background()); err != nil {
			t.Errorf("A failed to lock: %v", err)
			return
		}
		// 持有锁 2 秒
		time.Sleep(2 * time.Second)
		muA.Unlock(context.Background())
	}()

	// 确保 A 先获取锁
	time.Sleep(500 * time.Millisecond)

	// 协程 B 尝试获取锁，应该会被阻塞直到 A 释放
	start := time.Now()
	muB := distClient.NewMutex(lockKey)

	// 使用带超时的 Context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := muB.Lock(ctx); err != nil {
		t.Fatalf("B failed to lock: %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < 1500*time.Millisecond {
		t.Errorf("B should have waited for A, but only waited %v", elapsed)
	}

	muB.Unlock(context.Background())
	wg.Wait()
}

// TestMutex_UnlockSafety 验证非锁持有者无法误删锁
func TestMutex_UnlockSafety(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	distClient := NewRedisClient(client)
	lockKey := "test:lock:safety"
	ctx := context.Background()

	// 1. Client A 获取锁
	muA := distClient.NewMutex(lockKey)
	if err := muA.Lock(ctx); err != nil {
		t.Fatalf("A failed to lock: %v", err)
	}

	// 2. Client B (模拟另一个进程或线程) 尝试解锁同一个 key
	muB := distClient.NewMutex(lockKey)
	// 注意：muB 没有 Lock 成功，所以它的 token 肯定和 Redis 里的不一样
	// 或者即便是 muB 也 Lock 失败了，我们直接调用 Unlock 看看会不会误删

	if err := muB.Unlock(ctx); err != nil {
		// Unlock 可能会返回 nil (如果脚本执行成功但返回0)，或者返回 error
		// 我们的实现中，Lua 返回 0 也是 nil error，但不会删除 key
		// 这里只要不 panic 就行
	}

	// 3. 验证锁依然存在 (A 还持有)
	if exists := client.Exists(ctx, lockKey).Val(); exists != 1 {
		t.Errorf("lock should still exist after unauthorized unlock attempt")
	}

	// 4. Client A 释放锁
	muA.Unlock(ctx)

	// 5. 验证锁已释放
	if exists := client.Exists(ctx, lockKey).Val(); exists != 0 {
		t.Errorf("lock should be released after owner unlocks")
	}
}
