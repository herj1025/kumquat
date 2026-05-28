package config

// Config 聚合了所有配置
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Log         LogConfig         `mapstructure:"log"`
	Redis       RedisConfig       `mapstructure:"redis"`
	JWT         JWTConfig         `mapstructure:"jwt"`
	SegmentLock SegmentLockConfig `mapstructure:"segmentlock"`
}

type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	Name            string `mapstructure:"name"`
	Mode            string `mapstructure:"mode"`             // debug, release, test
	ReadTimeout     int    `mapstructure:"read-timeout"`     // 秒，默认 10
	WriteTimeout    int    `mapstructure:"write-timeout"`    // 秒，默认 10
	IdleTimeout     int    `mapstructure:"idle-timeout"`     // 秒，默认 120
	ShutdownTimeout int    `mapstructure:"shutdown-timeout"` // 秒，默认 5
}

type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"` // mysql, sqlite
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max-idle-conns"`
	MaxOpenConns int    `mapstructure:"max-open-conns"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max-size"`
	MaxBackups int    `mapstructure:"max-backups"`
	MaxAge     int    `mapstructure:"max-age"`
	Compress   bool   `mapstructure:"compress"`
}

type RedisConfig struct {
	Mode     string   `mapstructure:"mode"`
	Addrs    []string `mapstructure:"addrs"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
	DB       int      `mapstructure:"db"`
	PoolSize int      `mapstructure:"pool-size"`
}

type JWTConfig struct {
	AccessSecret  string `mapstructure:"access-secret"`
	AccessExpire  int    `mapstructure:"access-expire"` // minutes
	RefreshSecret string `mapstructure:"refresh-secret"`
	RefreshExpire int    `mapstructure:"refresh-expire"` // hours
}

// SegmentLockConfig 分段锁配置
type SegmentLockConfig struct {
	SegmentCount int `mapstructure:"segment-count"` // 分段数量，默认 64，建议 2 的幂
}
