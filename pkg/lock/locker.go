package lock

import "context"

// Locker 定义锁的通用接口
// 分布式锁 (distlock) 和单机分段锁 (segmentlock) 均实现此接口
// 业务代码可依赖此接口，在测试和本地开发环境中灵活切换实现
type Locker interface {
	Lock(ctx context.Context) error
	Unlock(ctx context.Context) error
}
