package distlock

import (
	"context"
	"errors"
	"time"
)

var (
	ErrLockNotObtained = errors.New("lock not obtained")
	ErrLockHeld        = errors.New("lock already held")
)

// Client 定义分布式锁客户端接口
type Client interface {
	// NewMutex 创建一个互斥锁对象
	NewMutex(name string, options ...Option) Mutex
}

// Mutex 定义互斥锁接口
type Mutex interface {
	// Lock 获取锁
	Lock(ctx context.Context) error
	// Unlock 释放锁
	Unlock(ctx context.Context) error
}

// Option 定义锁的配置选项
type Option func(*Options)

type Options struct {
	Expiration time.Duration
	RetryDelay time.Duration
	RetryCount int
}

func WithExpiration(d time.Duration) Option {
	return func(o *Options) {
		o.Expiration = d
	}
}

func WithRetryDelay(d time.Duration) Option {
	return func(o *Options) {
		o.RetryDelay = d
	}
}

func WithRetryCount(count int) Option {
	return func(o *Options) {
		o.RetryCount = count
	}
}
