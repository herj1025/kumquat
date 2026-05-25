package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// NewRootCommand 创建根命令，它是所有子命令的入口
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kumquat",
		Short: "A brief description of your application",
	}

	// 全局 PersistentFlags 定义，将 --config 作为全局配置
	// 这样所有子命令（如 server）都可以共享配置读取逻辑
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "./config/config.yaml", "config file (default is ./config/config.yaml)")

	// 在 Cobra 初始化时加载配置
	cobra.OnInitialize(initConfig)

	// 添加子命令
	serverCmd := NewServerCommand()
	cmd.AddCommand(serverCmd)

	return cmd
}

// initConfig 读取配置文件和环境变量
func initConfig() {
	if cfgFile != "" {
		// 使用 flag 指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 默认查找路径
		viper.AddConfigPath("./config")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// 读取环境变量（前缀 KUMQUAT，点替换为下划线）
	viper.SetEnvPrefix("KUMQUAT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		// 配置文件不存在时，仅打印错误，不退出程序
		// 因为可能有些配置是通过环境变量传入的
		fmt.Fprintf(os.Stderr, "Config file not found or unreadable: %v\n", err)
	}
}
