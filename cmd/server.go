package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/herj1025/kumquat/config"
	"github.com/herj1025/kumquat/pkg/i18n"
	"github.com/herj1025/kumquat/pkg/logger"
	"github.com/herj1025/kumquat/pkg/server"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ServerOptions 包含启动服务器所需的所有配置
type ServerOptions struct {
	Config *config.Config
}

// NewServerOptions 创建选项
func NewServerOptions() *ServerOptions {
	return &ServerOptions{
		Config: &config.Config{},
	}
}

// NewServerCommand 创建 server 子命令
func NewServerCommand() *cobra.Command {
	o := NewServerOptions()

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the web server",
		RunE: func(c *cobra.Command, args []string) error {
			// 1. 先加载配置
			if err := o.Complete(c); err != nil {
				return err
			}
			// 2. 配置加载完成后，初始化日志（o.Config.Log 可能来自配置文件、环境变量或默认值）
			if err := logger.InitLogger(&o.Config.Log); err != nil {
				return fmt.Errorf("failed to init logger: %w", err)
			}
			// 3. 验证配置
			// if err := o.Validate(); err != nil {
			// 	return err
			// }
			// 4. 运行服务器
			return o.Run()
		},
	}

	cmd.Flags().IntP("port", "p", 8080, "Port to run the server on (default: 8080)")
	cmd.Flags().String("server-mode", "debug", "server mode (debug, release, test) (default: debug)")

	return cmd
}

// Complete 补全配置并加载到 o.Config
func (o *ServerOptions) Complete(cmd *cobra.Command) error {

	// 绑定所有 Flag
	if err := viper.BindPFlag("server.port", cmd.Flags().Lookup("port")); err != nil {
		return fmt.Errorf("failed to bind port flag: %w", err)
	}
	if err := viper.BindPFlag("server.mode", cmd.Flags().Lookup("server-mode")); err != nil {
		return fmt.Errorf("failed to bind server-mode flag: %w", err)
	}

	// 汇总并写入结构体（遵循 Flag > Env > Config > Default）
	// 注意：配置文件的读取已经在 Root 命令的 initConfig 中完成了
	if err := viper.Unmarshal(o.Config); err != nil {
		return fmt.Errorf("unable to decode into struct, %v", err)
	}

	return nil
}

// Validate 校验参数合法性
func (o *ServerOptions) Validate() error {
	if o.Config.Server.Port < 1 || o.Config.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", o.Config.Server.Port)
	}
	return nil
}

// Run 执行真正的业务逻辑
func (o *ServerOptions) Run() error {
	defer func() {
		//同步日志
		_ = logger.Sync()
	}()

	// 获取当前使用的配置文件路径，用于加载本地化文件
	configFile := viper.ConfigFileUsed()
	absConfigFile, err := filepath.Abs(configFile)
	if err != nil {
		absConfigFile = configFile
	}

	// 如果没有配置文件，可能使用默认路径或者完全依赖环境变量
	// 这里做一个简单的容错处理
	localesDir := "config/locales"
	if absConfigFile != "" {
		localesDir = filepath.Join(filepath.Dir(absConfigFile), "locales")
	}

	if err := i18n.Load(localesDir); err != nil {
		// 国际化文件加载失败不应阻塞服务器启动，除非它是必须的
		// 这里改为打印日志而不是返回错误，或者根据业务需求决定
		// return fmt.Errorf("failed to load locales: %w", err)
		fmt.Printf("Warning: failed to load locales from %s: %v\n", localesDir, err)
	}

	srv, err := server.New(o.Config)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	return srv.Run()
}
