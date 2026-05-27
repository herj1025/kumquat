// Package kumquat 是一个基于 Gin 的 Web 框架，提供了开箱即用的服务端基础能力。
package kumquat

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/herj1025/kumquat/config"
	"github.com/herj1025/kumquat/pkg/i18n"
	"github.com/herj1025/kumquat/pkg/logger"
	"github.com/herj1025/kumquat/pkg/server"

	"github.com/spf13/cobra"

	"github.com/herj1025/kumquat/pkg/generator"
)

// Application 是 server.Application 的类型别名，方便外部项目引用。
// type Application = server.Application

// loadConfigWithPath 尝试从路径加载配置，支持文件路径或目录路径
func loadConfigWithPath(path string) (*config.Config, error) {
	if path == "" {
		return config.Load()
	}
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// ---------------------------------------------------------------------------
// App – 命令行方式（支持自定义命令参数）
// ---------------------------------------------------------------------------

// App 是基于 cobra 的命令行应用，内置 server 子命令及 -c / -p / --server-mode 参数，
// 同时支持二次开发添加自定义命令和参数。
type App struct {
	root      *cobra.Command
	serverCmd *cobra.Command
	routes    []func(*server.Application)
}

// AppOption 定义创建 App 时的可选配置
type AppOption func(*appOptions)

type appOptions struct {
	use        string
	short      string
	configPath string
}

// WithUse 自定义根命令名称（默认 "kumquat"）
func WithUse(use string) AppOption {
	return func(o *appOptions) {
		o.use = use
	}
}

// WithConfigPath 自定义配置文件路径（默认 "./config/config.yaml"）
func WithConfigPath(path string) AppOption {
	return func(o *appOptions) {
		o.configPath = path
	}
}

// WithShort 自定义根命令描述（默认 "kumquat application"）
func WithShort(short string) AppOption {
	return func(o *appOptions) {
		o.short = short
	}
}

// NewApp 创建一个命令行应用。
//
// 内置参数（全部子命令可用）：
//
//	-c, --config         配置文件路径（默认 "./config/config.yaml"）
//	-p, --port           服务端口号
//	    --server-mode    运行模式（debug / release / test）
//
// 内置子命令：
//
//	server               启动 Web 服务器
//
// 使用示例：
//
//	app := kumquat.NewApp(
//	    kumquat.WithUse("myapp"),
//	    kumquat.WithShort("my application"),
//	)
//	app.RegisterRoutes(registerRoutes)
//	app.Run()
func NewApp(opts ...AppOption) *App {
	o := &appOptions{
		use:   "kumquat",
		short: "kumquat application",
	}
	for _, opt := range opts {
		opt(o)
	}

	root := &cobra.Command{
		Use:   o.use,
		Short: o.short,
	}

	// 内置全局参数：所有子命令均可使用
	defaultConfig := o.configPath
	if defaultConfig == "" {
		defaultConfig = "./config/config.yaml"
	}
	root.PersistentFlags().StringP("config", "c", defaultConfig, "config file path (file or directory)")
	root.PersistentFlags().IntP("port", "p", 0, "server port (overrides config file)")
	root.PersistentFlags().String("server-mode", "debug", "server mode (debug, release, test)")

	a := &App{root: root}

	// 内置 server 子命令
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start the web server",
		RunE:  a.runServer,
	}
	root.AddCommand(serverCmd)
	a.serverCmd = serverCmd

	return a
}

// RegisterRoutes 注册业务路由。回调函数会收到已初始化的 *server.Application，
// 通过 srv.Engine() 注册路由，通过 srv.Container() 获取基础设施依赖。
func (a *App) RegisterRoutes(fn func(*server.Application)) {
	a.routes = append(a.routes, fn)
}

// AddCommand 添加自定义子命令。
func (a *App) AddCommand(cmd *cobra.Command) {
	a.root.AddCommand(cmd)
}

// Flags 返回根命令的持久化参数集，用于添加自定义参数。
// 必须在 Run() 之前调用。
func (a *App) Flags() *cobra.Command {
	return a.root
}

// ServerCommand 返回内置的 server 子命令，用于添加 server 专属参数。
// 必须在 Run() 之前调用。
func (a *App) ServerCommand() *cobra.Command {
	return a.serverCmd
}

// Run 执行命令行应用，解析参数并运行匹配的子命令。
func (a *App) Run() error {
	return a.root.Execute()
}

// runServer 是内置 server 子命令的执行逻辑
func (a *App) runServer(cmd *cobra.Command, args []string) error {
	// 1. 加载配置文件
	configPath, _ := cmd.Root().PersistentFlags().GetString("config")
	cfg, err := loadConfigWithPath(configPath)
	if err != nil {
		if configPath == "" {
			// 未指定配置文件时，不阻塞，使用框架默认值 + 环境变量
			fmt.Fprintf(os.Stderr, "Warning: no config file found, using defaults: %v\n", err)
			cfg = config.Default()
		} else {
			return fmt.Errorf("failed to load config from %s: %w", configPath, err)
		}
	}

	// 2. 端口覆盖（-p 优先级最高）
	if port, _ := cmd.Root().PersistentFlags().GetInt("port"); port > 0 {
		cfg.Server.Port = port
	}

	// 3. 运行模式覆盖
	if mode, _ := cmd.Root().PersistentFlags().GetString("server-mode"); mode != "" {
		cfg.Server.Mode = mode
	}

	// 4. 初始化日志
	if err := logger.InitLogger(&cfg.Log); err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	defer logger.Sync()

	// 5. 加载国际化文件（相对于配置文件所在目录）
	localesDir := "config/locales"
	if absPath, err := filepath.Abs(configPath); err == nil && absPath != "" {
		localesDir = filepath.Join(filepath.Dir(absPath), "locales")
	}
	if err := i18n.Load(localesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load locales from %s: %v\n", localesDir, err)
	}

	// 6. 创建服务器并注册路由
	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	for _, fn := range a.routes {
		fn(srv)
	}

	// 7. 启动服务器（阻塞）
	return srv.Run()
}

// AddGenerateCommand 添加生成命令
func (a *App) AddGenerateCommand() {
	generateCmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate code templates",
	}

	// gen config 子命令
	generateCmd.AddCommand(&cobra.Command{
		Use:   "config [output]",
		Short: "Generate default config file",
		Long:  `Generate a default configuration file. If output is not specified, uses ./config/config.yaml`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output := "./config/config.yaml"
			if len(args) > 0 {
				output = args[0]
			}
			return generator.GenerateDefaultConfig(output)
		},
	})

	// gen handler 子命令
	generateCmd.AddCommand(&cobra.Command{
		Use:   "handler [name] [output]",
		Short: "Generate handler template",
		Long:  `Generate a handler template. If output is not specified, uses ./internal/handler/[name].go`,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			output := fmt.Sprintf("./internal/handler/%s.go", name)
			if len(args) > 1 {
				output = args[1]
			}
			return generator.GenerateHandlerTemplate(name, output)
		},
	})

	// gen service 子命令
	generateCmd.AddCommand(&cobra.Command{
		Use:   "service [name] [output]",
		Short: "Generate service template",
		Long:  `Generate a service template. If output is not specified, uses ./internal/service/%s.go`,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			output := fmt.Sprintf("./internal/service/%s.go", name)
			if len(args) > 1 {
				output = args[1]
			}
			return generator.GenerateServiceTemplate(name, output)
		},
	})

	a.root.AddCommand(generateCmd)
}
