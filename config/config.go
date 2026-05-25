package config

// Config 聚合了所有配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	Name            string `mapstructure:"name"`
	Mode            string `mapstructure:"mode"`             // debug, release, test
	ReadTimeout     int    `mapstructure:"read_timeout"`     // 秒，默认 10
	WriteTimeout    int    `mapstructure:"write_timeout"`    // 秒，默认 10
	IdleTimeout     int    `mapstructure:"idle_timeout"`     // 秒，默认 120
	ShutdownTimeout int    `mapstructure:"shutdown_timeout"` // 秒，默认 5
}

type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"` // mysql, sqlite
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type RedisConfig struct {
	Mode     string   `mapstructure:"mode"`
	Addrs    []string `mapstructure:"addrs"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
	DB       int      `mapstructure:"db"`
	PoolSize int      `mapstructure:"pool_size"`
}

type JWTConfig struct {
	AccessSecret  string `mapstructure:"access_secret"`
	AccessExpire  int    `mapstructure:"access_expire"`  // minutes
	RefreshSecret string `mapstructure:"refresh_secret"`
	RefreshExpire int    `mapstructure:"refresh_expire"` // hours
}
