package deps

import (
	"github.com/herj1025/kumquat/config"
	"github.com/herj1025/kumquat/pkg/distlock"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Deps 包含应用顶层模块的依赖（DB、Redis、分布式锁）
type Deps struct {
	cfg         *config.Config
	gormDB      *gorm.DB
	redisClient redis.UniversalClient
	distLock    distlock.Client
}

// New 初始化所有的依赖关系
func New(cfg *config.Config) (*Deps, error) {
	gormDB, err := initDB(&cfg.Database)
	if err != nil {
		return nil, err
	}

	redisClient, err := initRedis(&cfg.Redis)
	if err != nil {
		return nil, err
	}

	lockClient := distlock.NewRedisClient(redisClient)

	return &Deps{
		cfg:         cfg,
		gormDB:      gormDB,
		redisClient: redisClient,
		distLock:    lockClient,
	}, nil
}

// Config 返回应用配置。
func (c *Deps) Config() *config.Config {
	if c == nil {
		return nil
	}
	return c.cfg
}

// DB 返回 GORM 数据库实例
func (c *Deps) DB() *gorm.DB {
	if c == nil {
		return nil
	}
	return c.gormDB
}

// Redis 返回 Redis 客户端
func (c *Deps) Redis() redis.UniversalClient {
	if c == nil {
		return nil
	}
	return c.redisClient
}

// DistLock 返回分布式锁客户端。
func (c *Deps) DistLock() distlock.Client {
	if c == nil {
		return nil
	}
	return c.distLock
}

// Close 关闭所有资源
func (c *Deps) Close() error {
	if c.redisClient != nil {
		if err := c.redisClient.Close(); err != nil {
			return err
		}
	}

	if c.gormDB != nil {
		sqlDB, err := c.gormDB.DB()
		if err != nil {
			return err
		}
		if err := sqlDB.Close(); err != nil {
			return err
		}
	}
	return nil
}
