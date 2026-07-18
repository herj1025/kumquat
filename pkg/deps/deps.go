package deps

import (
	"errors"
	"fmt"
	"strings"

	"github.com/herj1025/kumquat/config"
	"github.com/herj1025/kumquat/pkg/lock/distlock"
	"github.com/herj1025/kumquat/pkg/lock/segmentlock"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrDBNotInitialized       = errors.New("db is not initialized")
	ErrRedisNotInitialized    = errors.New("redis is not initialized")
	ErrDistLockNotInitialized = errors.New("redis distlock is not initialized")
)

// Deps 包含应用顶层模块的依赖（DB、Redis、分布式锁、分段锁）
type Deps struct {
	cfg         *config.Config
	gormDB      *gorm.DB
	redisClient redis.UniversalClient
	distLock    distlock.Client
	segLock     *segmentlock.SegmentLock
}

// New 初始化所有的依赖关系
func New(cfg *config.Config) (*Deps, error) {
	var (
		gormDB      *gorm.DB
		redisClient redis.UniversalClient
	)

	if shouldInitDatabase(cfg.Database) {
		var err error
		gormDB, err = initDB(cfg.Database)
		if err != nil {
			return nil, err
		}
	}

	if shouldInitRedis(cfg.Redis) {
		var err error
		redisClient, err = initRedis(cfg.Redis)
		if err != nil {
			return nil, err
		}
	}

	var lockClient distlock.Client
	if redisClient != nil {
		lockClient = distlock.NewRedisClient(redisClient)
	}

	segLock := segmentlock.New(
		segmentlock.WithSegmentCount(cfg.SegmentLock.SegmentCount),
	)

	return &Deps{
		cfg:         cfg,
		gormDB:      gormDB,
		redisClient: redisClient,
		distLock:    lockClient,
		segLock:     segLock,
	}, nil
}

func shouldInitDatabase(cfg *config.DatabaseConfig) bool {
	if cfg == nil {
		return false
	}
	return !(cfg.Host == "" &&
		cfg.Port == 0 &&
		cfg.Username == "" &&
		cfg.Password == "" &&
		cfg.Database == "")
}

func shouldInitRedis(cfg *config.RedisConfig) bool {
	if cfg == nil {
		return false
	}
	for _, addr := range cfg.Addrs {
		if strings.TrimSpace(addr) != "" {
			return true
		}
	}
	return false
}

// Config 返回应用配置。
func (c *Deps) Config() *config.Config {
	if c == nil {
		return nil
	}
	return c.cfg
}

// DB 返回 GORM 数据库实例；未初始化时返回包装错误。
func (c *Deps) DB() (*gorm.DB, error) {
	if c == nil || c.gormDB == nil {
		return nil, fmt.Errorf("get db: %w", ErrDBNotInitialized)
	}
	return c.gormDB, nil
}

// Redis 返回 Redis 客户端；未初始化时返回包装错误。
func (c *Deps) Redis() (redis.UniversalClient, error) {
	if c == nil || c.redisClient == nil {
		return nil, fmt.Errorf("get redis: %w", ErrRedisNotInitialized)
	}
	return c.redisClient, nil
}

// DistLock 返回分布式锁客户端；未初始化时返回包装错误。
func (c *Deps) DistLock() (distlock.Client, error) {
	if c == nil || c.distLock == nil {
		return nil, fmt.Errorf("get redis distlock: %w", ErrDistLockNotInitialized)
	}
	return c.distLock, nil
}

// SegmentLock 返回分段锁，用于单机高并发场景下减少锁竞争。
// 相同 key 映射到同一 segment，不同 key 大概率分散到不同 segment。
func (c *Deps) SegmentLock() *segmentlock.SegmentLock {
	if c == nil {
		return nil
	}
	return c.segLock
}

// Close 关闭所有资源，确保每个资源都被尝试关闭
func (c *Deps) Close() error {
	var errs []error

	if c.redisClient != nil {
		if err := c.redisClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.gormDB != nil {
		sqlDB, err := c.gormDB.DB()
		if err != nil {
			errs = append(errs, err)
		} else {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close resources: %w", errors.Join(errs...))
	}
	return nil
}
