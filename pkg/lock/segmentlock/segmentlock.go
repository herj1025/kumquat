// Package segmentlock 提供基于分段互斥锁的高并发锁机制。
//
// 适用场景
//
// 当大量并发请求操作不同的 key（如用户 ID、订单号）时，
// 传统全局互斥锁会成为瓶颈。SegmentLock 将锁资源分散到 N 个 segment 上，
// 通过 FNV-1a 哈希将 key 均匀映射到对应 segment，不同 key 的锁操作互不干扰。
//
// 快速开始
//
//	sl := segmentlock.New()
//	mu := sl.NewMutex("my:lock:key")
//
//	if err := mu.Lock(ctx); err != nil {
//		return err
//	}
//	defer mu.Unlock(ctx)
//
//	// 非阻塞尝试:
//	if err := mu.TryLock(ctx); err == nil {
//		defer mu.Unlock(ctx)
//	}
//
// SegmentLock 与 distlock 均实现 lock.Locker 接口，业务代码可依赖该接口
// 在本地开发和分布式部署间灵活切换。
package segmentlock

import (
	"context"
	"errors"
	"hash/fnv"
	"sync"

	"github.com/herj1025/kumquat/pkg/lock"
)

var (
	// ErrLockNotObtained TryLock 在锁已被占有时返回此错误
	ErrLockNotObtained = errors.New("segmentlock: lock not obtained")
)

// Option 定义分段锁的配置选项
type Option func(*Options)

// Options 分段锁配置
type Options struct {
	// SegmentCount 分段数量
	// 默认 64，建议范围 16-256，推荐使用 2 的幂以利用编译器优化
	SegmentCount int
}

// WithSegmentCount 设置分段数量
// 建议设为 2 的幂（16, 32, 64, 128, 256），使 Go 编译器将取模优化为位运算
// segment 越多锁冲突概率越低，但内存开销线性增长
func WithSegmentCount(n int) Option {
	return func(o *Options) {
		o.SegmentCount = n
	}
}

// isPowerOf2 判断 n 是否为 2 的幂
func isPowerOf2(n int) bool {
	return n > 0 && n&(n-1) == 0
}

// SegmentLock 将锁资源分散到多个 segment 上以减少锁竞争。
//
// 通过 FNV-1a 哈希将 key 映射到 [0, SegmentCount) 范围内的 segment，
// 保证均匀分布。不同 key 的锁操作互不干扰，同一 key 总是映射到同一 segment。
//
// SegmentLock 创建后不可变，所有字段均为只读，方法可安全并发调用。
type SegmentLock struct {
	segments []sync.Mutex
	count    int // segment 数量，用于取模
}

// New 创建分段锁，默认 64 个 segment，可通过 WithSegmentCount 调整
func New(options ...Option) *SegmentLock {
	opts := &Options{
		SegmentCount: 64,
	}
	for _, o := range options {
		o(opts)
	}

	if opts.SegmentCount <= 0 || !isPowerOf2(opts.SegmentCount) {
		opts.SegmentCount = 64
	}

	segments := make([]sync.Mutex, opts.SegmentCount)
	return &SegmentLock{
		segments: segments,
		count:    opts.SegmentCount,
	}
}

// SegmentCount 返回当前分段锁的 segment 数量
func (sl *SegmentLock) SegmentCount() int {
	return sl.count
}

// getSegment 根据 key 的哈希值选取对应的 segment
// 使用 FNV-1a 32 位算法，速度快且分布均匀
func (sl *SegmentLock) getSegment(key string) *sync.Mutex {
	h := fnv.New32a()
	h.Write([]byte(key))
	idx := h.Sum32() % uint32(sl.count)
	return &sl.segments[idx]
}

// Mutex 是 SegmentLock 为特定 key 创建的互斥锁。
// 实现 lock.Locker 接口，可与 distlock 替换使用。
var _ lock.Locker = (*Mutex)(nil)

type Mutex struct {
	segment *sync.Mutex
	name    string // key 名称，用于日志和调试
}

// NewMutex 根据 key 创建分段互斥锁。
// 相同 key 总是映射到同一 segment（互斥），不同 key 大概率映射到不同 segment（并行）。
func (sl *SegmentLock) NewMutex(name string) *Mutex {
	return &Mutex{
		segment: sl.getSegment(name),
		name:    name,
	}
}

// Name 返回该 Mutex 对应的 key 名称，用于日志和调试
func (m *Mutex) Name() string {
	return m.name
}

// Lock 获取锁，支持 context 取消。
//
// 先尝试 TryLock 快速路径（无 goroutine 开销），
// 失败后通过 goroutine 等待，支持 context 超时/取消。
//
// 当 context 被取消时，内部 goroutine 会在获取锁后自动释放，
// 不留下脏锁状态。用完后仍须调用 Unlock 确保清理。
func (m *Mutex) Lock(ctx context.Context) error {
	if m.segment.TryLock() {
		return nil
	}

	locked := make(chan struct{}, 1)
	go func() {
		m.segment.Lock()
		locked <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		go func() {
			<-locked
			m.segment.Unlock()
		}()
		return ctx.Err()
	case <-locked:
		return nil
	}
}

// TryLock 尝试获取锁，非阻塞。获取成功返回 nil，失败返回 ErrLockNotObtained。
// 参数 ctx 保留以匹配外部接口签名，本次调用不受 context 影响。
func (m *Mutex) TryLock(_ context.Context) error {
	if m.segment.TryLock() {
		return nil
	}
	return ErrLockNotObtained
}

// Unlock 释放锁
func (m *Mutex) Unlock(_ context.Context) error {
	m.segment.Unlock()
	return nil
}
