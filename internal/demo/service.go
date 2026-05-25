package demo

import (
	"context"
	"time"

	"github.com/herj1025/kumquat/pkg/distlock"
	"github.com/herj1025/kumquat/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Service 定义业务逻辑接口
type Service interface {
	FindById(context.Context) (*Demo, error)
	RunDistributedTask(context.Context) (int64, error)
}

type service struct {
	repo        Repository
	redisClient redis.UniversalClient
	lockClient  distlock.Client
}

// NewService 创建服务实例
func NewService(repo Repository, redisClient redis.UniversalClient, lockClient distlock.Client) Service {
	return &service{
		repo:        repo,
		redisClient: redisClient,
		lockClient:  lockClient,
	}
}

func (s *service) FindById(ctx context.Context) (*Demo, error) {
	return s.repo.FindById(ctx)
}

var data int64 = 0

// RunDistributedTask 演示分布式锁的使用
func (s *service) RunDistributedTask(ctx context.Context) (int64, error) {
	// 1. 创建互斥锁 (使用注入的客户端)
	// 锁名为 "task:lock"，带重试机制
	mutex := s.lockClient.NewMutex("demo:task:lock",
		distlock.WithExpiration(10*time.Second),
		distlock.WithRetryCount(3),
		distlock.WithRetryDelay(100*time.Millisecond),
	)

	// 3. 获取锁
	logger.C(ctx).Info("Attempting to acquire lock...")
	if err := mutex.Lock(ctx); err != nil {
		logger.C(ctx).Error("Failed to acquire lock", zap.Error(err))
		return 0, err
	}
	defer func() {
		// 5. 释放锁
		logger.C(ctx).Info("Releasing lock...")
		if err := mutex.Unlock(ctx); err != nil {
			logger.C(ctx).Error("Failed to release lock", zap.Error(err))
		}
	}()

	// 4. 执行临界区业务逻辑
	logger.C(ctx).Info("Lock acquired, processing critical task...")

	// 模拟耗时操作
	// select {
	// case <-ctx.Done():
	// 	return ctx.Err()
	// case <-time.After(2 * time.Second):
	// 	logger.C(ctx).Info("Task completed successfully")
	// }
	data = data + 1
	logger.C(ctx).Info("Task completed successfully", zap.Int64("data", data))
	return data, nil
}
