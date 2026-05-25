package app

import (
	"github.com/herj1025/kumquat/config"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// App 包含应用顶层模块的依赖（DB、Redis、分布式锁）
type App struct {
	gormDB      *gorm.DB
	redisClient redis.UniversalClient
}

// New 初始化所有的依赖关系
func New(cfg *config.Config) (*App, error) {
	gormDB, err := initDB(&cfg.Database)
	if err != nil {
		return nil, err
	}

	redisClient, err := initRedis(&cfg.Redis)
	if err != nil {
		return nil, err
	}

	return &App{
		gormDB:      gormDB,
		redisClient: redisClient,
	}, nil
}

// DB 返回 GORM 数据库实例
func (a *App) DB() *gorm.DB {
	return a.gormDB
}

// Redis 返回 Redis 客户端
func (a *App) Redis() redis.UniversalClient {
	return a.redisClient
}

// Close 关闭所有资源
func (a *App) Close() error {
	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			return err
		}
	}

	if a.gormDB != nil {
		sqlDB, err := a.gormDB.DB()
		if err != nil {
			return err
		}
		if err := sqlDB.Close(); err != nil {
			return err
		}
	}
	return nil
}
