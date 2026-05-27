package deps

import (
	"context"
	"fmt"
	"time"

	"github.com/herj1025/kumquat/config"

	"github.com/redis/go-redis/v9"
)

type RedisMode string

const (
	singleMode  RedisMode = "single"
	clusterMode RedisMode = "cluster"
)

func initRedis(cfg *config.RedisConfig) (redis.UniversalClient, error) {
	addrs := cfg.Addrs
	if len(addrs) == 0 {
		return nil, fmt.Errorf("redis addrs is empty")
	}
	if cfg.Mode != string(singleMode) {
		if len(addrs) == 1 {
			return nil, fmt.Errorf("Configure multiple Redis node addresses for non-single-node deployments.")
		}
	}

	options := &redis.UniversalOptions{
		Addrs:    addrs,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	}

	if cfg.Mode == "cluster" {
		options.MasterName = ""
	}

	client := redis.NewUniversalClient(options)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping error: %w", err)
	}

	return client, nil
}
