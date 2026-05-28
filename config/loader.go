package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Default 返回框架默认配置
func Default() *Config {
	return defaultConfig()
}

// defaultConfig 是核心框架的默认配置，二次开发未配置时使用这些值
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            8080,
			Name:            "kumquat",
			Mode:            "release",
			ReadTimeout:     10,
			WriteTimeout:    10,
			IdleTimeout:     120,
			ShutdownTimeout: 5,
		},
		Log: LogConfig{
			Level:      "info",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     30,
		},
		Database: DatabaseConfig{
			Driver:       "mysql",
			Host:         "127.0.0.1",
			Port:         3306,
			MaxIdleConns: 10,
			MaxOpenConns: 100,
		},
		Redis: RedisConfig{
			Mode:     "standalone",
			DB:       0,
			PoolSize: 100,
		},
		JWT: JWTConfig{
			AccessExpire:  30,
			RefreshExpire: 720,
		},
		SegmentLock: SegmentLockConfig{
			SegmentCount: 64,
		},
	}
}

// setDefaults 将默认配置注册到 viper，配置文件和环境变量会覆盖这些值
func setDefaults(v *viper.Viper) {
	cfg := defaultConfig()

	v.SetDefault("server.port", cfg.Server.Port)
	v.SetDefault("server.name", cfg.Server.Name)
	v.SetDefault("server.mode", cfg.Server.Mode)
	v.SetDefault("server.read-timeout", cfg.Server.ReadTimeout)
	v.SetDefault("server.write-timeout", cfg.Server.WriteTimeout)
	v.SetDefault("server.idle-timeout", cfg.Server.IdleTimeout)
	v.SetDefault("server.shutdown-timeout", cfg.Server.ShutdownTimeout)

	v.SetDefault("database.driver", cfg.Database.Driver)
	v.SetDefault("database.host", cfg.Database.Host)
	v.SetDefault("database.port", cfg.Database.Port)
	v.SetDefault("database.max-idle-conns", cfg.Database.MaxIdleConns)
	v.SetDefault("database.max-open-conns", cfg.Database.MaxOpenConns)

	v.SetDefault("log.level", cfg.Log.Level)
	v.SetDefault("log.max-size", cfg.Log.MaxSize)
	v.SetDefault("log.max-backups", cfg.Log.MaxBackups)
	v.SetDefault("log.max-age", cfg.Log.MaxAge)

	v.SetDefault("redis.mode", cfg.Redis.Mode)
	v.SetDefault("redis.db", cfg.Redis.DB)
	v.SetDefault("redis.pool-size", cfg.Redis.PoolSize)

	v.SetDefault("jwt.access-secret", cfg.JWT.AccessSecret)
	v.SetDefault("jwt.access-expire", cfg.JWT.AccessExpire)
	v.SetDefault("jwt.refresh-secret", cfg.JWT.RefreshSecret)
	v.SetDefault("jwt.refresh-expire", cfg.JWT.RefreshExpire)

	v.SetDefault("segmentlock.segment-count", cfg.SegmentLock.SegmentCount)
}

// Load 加载默认路径下的配置文件，默认路径为 ./config/config.yaml
func Load() (*Config, error) {
	return LoadFromFile("./config/config.yaml")
}

// LoadFromFile 从指定文件路径加载配置文件
func LoadFromFile(path string) (*Config, error) {
	v := viper.New()

	// 1. 框架默认值（二次开发未配置时使用）
	setDefaults(v)

	// 2. 读取配置文件（覆盖框架默认值）
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// 3. 环境变量覆盖（优先级最高）
	v.SetEnvPrefix("KUMQUAT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &cfg, nil
}

// LoadInto 从指定文件加载配置到自定义结构体，包含框架默认值 + 环境变量覆盖。
// 利用泛型，调用方无需 interface{} 转换。
//
//	cfg, err := config.LoadInto[DemoConfig]("config.yaml")
func LoadInto[T any](path string) (*T, error) {
	v := viper.New()

	// 1. 框架默认值
	setDefaults(v)

	// 2. 读取配置文件
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// 3. 环境变量覆盖
	v.SetEnvPrefix("KUMQUAT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 4. 映射到目标结构体
	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &cfg, nil
}

// LoadFromDir 从指定目录加载配置文件，目录下需包含 config.yaml
// func LoadFromDir(dir string) (*Config, error) {
// 	v := viper.New()

// 	// 1. 框架默认值（二次开发未配置时使用）
// 	setDefaults(v)

// 	// 2. 读取配置文件（覆盖框架默认值）
// 	v.SetConfigName("config")
// 	v.SetConfigType("yaml")
// 	v.AddConfigPath(dir)
// 	if err := v.ReadInConfig(); err != nil {
// 		return nil, fmt.Errorf("failed to read config from dir %s: %w", dir, err)
// 	}

// 	// 3. 环境变量覆盖（优先级最高）
// 	v.SetEnvPrefix("KUMQUAT")
// 	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
// 	v.AutomaticEnv()

// 	var cfg Config
// 	if err := v.Unmarshal(&cfg); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
// 	}
// 	return &cfg, nil
// }
