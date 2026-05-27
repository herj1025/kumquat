# kumquat

## 说明

```
简单的go web开发框架
```

## 主要功能

- **HTTP服务器**：基于Gin框架，提供高性能HTTP服务
- **配置管理**：支持YAML配置文件，支持环境变量覆盖
- **日志管理**：集成Zap日志库，支持文件轮转
- **数据库连接**：集成GORM ORM，支持多种数据库
- **Redis连接**：集成Redis客户端，支持集群模式
- **国际化支持**：支持多语言，自动检测Accept-Language头
- **请求追踪**：自动生成Trace ID，便于链路追踪
- **JWT认证**：提供标准JWT认证中间件
- **分布式锁**：基于Redis的分布式锁实现
- **限流器**：提供基于令牌桶算法的限流功能
- **代码生成**：命令行生成配置文件和代码模板
- **简化启动**：命令行方式快速启动应用

## 主要引用框架

- gin
- gorm
- cobra
- viper
- zap
- lumberjack

## 快速开始

### 1. 使用命令行方式

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/herj1025/kumquat"
)

func main() {
    app := kumquat.NewApp()
    
    // 添加生成命令
    app.AddGenerateCommand()
    
    app.RegisterRoutes(func(srv *kumquat.Application) {
        r := srv.Engine()
        r.GET("/ping", func(c *gin.Context) {
            c.JSON(200, gin.H{"message": "pong"})
        })
    })
    
    app.Run()
}
```

### 2. 使用限流器

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/herj1025/kumquat"
    "github.com/herj1025/kumquat/pkg/ratelimit"
    "golang.org/x/time/rate"
)

func main() {
    app := kumquat.NewApp()
    
    app.RegisterRoutes(func(srv *kumquat.Application) {
        r := srv.Engine()
        // 按IP限流：每秒最多10个请求，突发容量20
        ipLimiter := ratelimit.Middleware(
            ratelimit.NewMemoryLimiter(rate.Limit(10), 20),
            ratelimit.WithKeyFunc(func(c *gin.Context) string {
                return c.ClientIP()
            }),
        )
        r.GET("/api/limited", ipLimiter, func(c *gin.Context) {
            c.JSON(200, gin.H{"message": "Limited endpoint"})
        })
        
        r.GET("/api/unlimited", func(c *gin.Context) {
            c.JSON(200, gin.H{"message": "Unlimited endpoint"})
        })
    })
    
    app.Run()
}
```

### 3. 扩展服务器

```go
package main

import (
    "fmt"

    "github.com/gin-gonic/gin"
    "github.com/herj1025/kumquat"
    "github.com/herj1025/kumquat/pkg/middleware"
    "github.com/spf13/cobra"
)

var authEnabled bool

func main() {
    app := kumquat.NewApp()
    app.AddGenerateCommand()

    // 自定义子命令
    demoCmd := &cobra.Command{
        Use:   "demo",
        Short: "A demo command",
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("hello world")
            return nil
        },
    }
    app.AddCommand(demoCmd)

    // server 专属参数
    app.ServerCommand().Flags().BoolVar(
        &authEnabled, "auth", false,
        "enable JWT authentication for all routes",
    )

    app.RegisterRoutes(func(srv *kumquat.Application) {
        r := srv.Engine()
        if authEnabled {
            r.Use(middleware.Authorization("secret-key"))
        }
        r.GET("/ping", func(c *gin.Context) {
            c.JSON(200, gin.H{"message": "pong"})
        })
    })

    app.Run()
}
```

## 命令行功能

### 生成配置文件

```bash
go run cmd/kumquat/main.go gen config [output_path]
```

### 生成处理器模板

```bash
go run cmd/kumquat/main.go gen handler [name] [output_path]
```

### 生成服务层模板

```bash
go run cmd/kumquat/main.go gen service [name] [output_path]
```

## 提交格式规范

| 标识 | 含义 | 具体说明 |
|------|------|----------|
| feat | 新增功能 | 例：feat 添加注册逻辑 |
| fix | 修复缺陷 | 例：fix: 修正验证错误 |
| perf | 性能优化 | 例：perf: 优化查询速度 |
| refactor | 重构代码（非功能变更） | 例：refactor: 重写用户模块 |
| build | 构建变更 | 例：build: 更新webpack配置 |
| ci | CI/CD配置修改 | 例：ci: 配置自动部署 |
| test | 测试相关 | 例：test: 添加单元测试 |
| docs | 文档更新 | 例：docs: 修改使用说明 |
| style | 格式调整（非逻辑） | 例：style: 统一缩进 |
| chore | 杂项维护 | 例：chore: 升级依赖 |
| revert | 回滚提交 | 例：revert: 撤销上一版本 |
| i18n | 国际化支持 | 例：i18n: 添加多语言 |
| security | 安全修复 | 例：security: 修复注入漏洞 |
