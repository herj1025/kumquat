package distlock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/herj1025/kumquat/pkg/lock"
	"github.com/herj1025/kumquat/pkg/logger"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	// Lua 脚本：原子释放锁
	// KEYS[1]: 锁 key
	// ARGV[1]: 锁 value (Token)
	// 返回值: 1 释放成功, 0 锁不存在或 Token 不匹配
	unlockScript = redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)

	// Lua 脚本：原子续期锁
	// KEYS[1]: 锁 key
	// ARGV[1]: 锁 value (Token)
	// ARGV[2]: 过期时间 (毫秒)
	// 返回值: 1 续期成功, 0 锁不存在或 Token 不匹配
	refreshScript = redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)
)

// RedisClient 基于 Redis 的分布式锁客户端实现
type RedisClient struct {
	client redis.UniversalClient
}

// NewRedisClient 创建一个新的 Redis 分布式锁客户端
func NewRedisClient(client redis.UniversalClient) *RedisClient {
	return &RedisClient{
		client: client,
	}
}

func (c *RedisClient) NewMutex(name string, options ...Option) Mutex {
	opts := &Options{
		Expiration: 10 * time.Second, // 默认锁过期时间 10 秒
		RetryDelay: 100 * time.Millisecond,
		RetryCount: 3, // 默认重试 3 次
	}
	for _, o := range options {
		o(opts)
	}

	return &RedisMutex{
		name:   name,
		client: c,
		opts:   opts,
		// 生成唯一的 Token，防止误删别人的锁
		token: uuid.New().String(),
	}
}

// RedisMutex 基于 Redis 的互斥锁实现
var _ lock.Locker = (*RedisMutex)(nil)

type RedisMutex struct {
	name   string
	client *RedisClient
	opts   *Options
	token  string // 锁的唯一标识 (Value)

	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex // 保护内部状态
}

// Lock 尝试获取锁，如果获取失败则等待（阻塞模式）
func (m *RedisMutex) Lock(ctx context.Context) error {
	// 尝试获取锁的循环
	for {
		// 1. 尝试直接获取锁 (内部已包含执行错误重试)
		err := m.tryObtain(ctx)
		if err == nil {
			// 获取成功，启动看门狗
			m.startWatchdog()
			return nil
		}

		// 如果是执行错误导致 tryObtain 失败（且重试耗尽），直接返回错误给上层，结束业务逻辑
		if err != ErrLockHeld {
			return err
		}

		// 2. 获取失败（锁被占用），订阅释放通知
		// 通道名约定：distlock:channel:{name}
		channelName := fmt.Sprintf("distlock:channel:%s", m.name)
		pubsub := m.client.client.Subscribe(ctx, channelName)

		// 优化：订阅后再次尝试获取锁，防止在订阅期间锁已被释放
		// 这样可以避免因极小的时间窗口导致的无谓等待
		err = m.tryObtain(ctx)
		if err == nil {
			pubsub.Close()
			m.startWatchdog()
			return nil
		}
		// 如果第二次尝试遇到系统错误，也直接返回
		if err != ErrLockHeld {
			pubsub.Close()
			return err
		}

		// 等待消息或超时
		select {
		case <-ctx.Done():
			pubsub.Close()
			return ctx.Err()
		case <-pubsub.Channel():
			// 收到锁释放通知，立即重试
			pubsub.Close()
			// 增加随机抖动，防止惊群效应 (Thundering Herd)
			// 所有等待者被唤醒后，稍微错开重试时间
			time.Sleep(time.Duration(time.Now().Nanosecond()%10) * time.Millisecond)
			continue
		case <-time.After(m.opts.RetryDelay):
			// 超时（防止丢失通知导致的永久等待），重试
			pubsub.Close()
			continue
		}
	}
}

// TryLock 尝试获取锁，非阻塞模式
func (m *RedisMutex) TryLock(ctx context.Context) error {
	err := m.tryObtain(ctx)
	if err == nil {
		m.startWatchdog()
	}
	return err
}

func (m *RedisMutex) tryObtain(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error
	var ok bool

	// 执行错误重试机制（覆盖所有非锁占用错误）
	for i := 0; i <= m.opts.RetryCount; i++ {
		// 使用 SET key value NX PX expiration 原子指令
		ok, err = m.client.client.SetNX(ctx, m.name, m.token, m.opts.Expiration).Result()
		if err == nil {
			// Redis 调用成功
			if ok {
				return nil
			}
			// 锁被占用，直接返回，不进行重试
			return ErrLockHeld
		}

		// 如果是 context 相关的错误，直接退出不重试
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 遇到执行错误，且还有重试机会
		if i < m.opts.RetryCount {
			// 指数退避策略：10ms, 20ms, 40ms...
			baseBackoff := time.Duration(10*(1<<i)) * time.Millisecond
			// 增加 0-10ms 的随机抖动，防止惊群效应
			jitter := time.Duration(time.Now().Nanosecond()%10) * time.Millisecond

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(baseBackoff + jitter):
				// continue retry
			}
		}
	}

	// 重试耗尽仍失败
	return err
}

func (m *RedisMutex) startWatchdog() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 确保之前的 watchdog 已经停止
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	go func() {
		ticker := time.NewTicker(m.opts.Expiration / 3)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := m.refresh(ctx); err != nil {
					// 续期失败，可能锁已丢失或 Redis 故障
					return
				}
			}
		}
	}()
}

func (m *RedisMutex) refresh(ctx context.Context) error {
	return m.client.client.EvalSha(ctx, refreshScript.Hash(), []string{m.name}, m.token, int(m.opts.Expiration/time.Millisecond)).Err()
}

// Unlock 释放锁
func (m *RedisMutex) Unlock(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil
	}

	res, err := unlockScript.Run(ctx, m.client.client, []string{m.name}, m.token).Result()
	if err != nil {
		logger.C(ctx).Error("Failed to release lock", zap.Error(err))
		return err
	}

	// 3. 发送释放通知 (Pub/Sub)
	channelName := fmt.Sprintf("distlock:channel:%s", m.name)
	// Publish 是非阻塞的，失败也不影响主流程
	m.client.client.Publish(ctx, channelName, "released")

	_ = res
	return nil
}
