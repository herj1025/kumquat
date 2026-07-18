package deps

import (
	"errors"
	"testing"

	"github.com/herj1025/kumquat/config"
)

func TestNewSkipsEmptyDatabaseAndRedisConfig(t *testing.T) {
	cfg := &config.Config{
		Database: &config.DatabaseConfig{
			Driver:   "postgres",
			Host:     "",
			Port:     0,
			Username: "",
			Password: "",
			Database: "",
		},
		Redis: &config.RedisConfig{
			Mode:  "standalone",
			Addrs: []string{""},
		},
		SegmentLock: config.SegmentLockConfig{
			SegmentCount: 64,
		},
	}

	deps, err := New(cfg)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := deps.DB(); !errors.Is(err, ErrDBNotInitialized) {
		t.Fatalf("DB should wrap ErrDBNotInitialized, got: %v", err)
	}
	if _, err := deps.Redis(); !errors.Is(err, ErrRedisNotInitialized) {
		t.Fatalf("Redis should wrap ErrRedisNotInitialized, got: %v", err)
	}
	if _, err := deps.DistLock(); !errors.Is(err, ErrDistLockNotInitialized) {
		t.Fatalf("DistLock should wrap ErrDistLockNotInitialized, got: %v", err)
	}
}

func TestMethodsReturnWrappedUninitializedErrors(t *testing.T) {
	deps := &Deps{}

	if _, err := deps.DB(); !errors.Is(err, ErrDBNotInitialized) {
		t.Fatalf("DB should wrap ErrDBNotInitialized, got: %v", err)
	}
	if _, err := deps.Redis(); !errors.Is(err, ErrRedisNotInitialized) {
		t.Fatalf("Redis should wrap ErrRedisNotInitialized, got: %v", err)
	}
	if _, err := deps.DistLock(); !errors.Is(err, ErrDistLockNotInitialized) {
		t.Fatalf("DistLock should wrap ErrDistLockNotInitialized, got: %v", err)
	}
}
