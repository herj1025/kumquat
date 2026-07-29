// Package generator 提供代码和配置文件生成工具
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/herj1025/kumquat/config"
	"gopkg.in/yaml.v3"
)

// GenerateDefaultConfig 生成默认配置文件
func GenerateDefaultConfig(outputPath string) error {
	defaultCfg := config.Default()

	data, err := yaml.Marshal(defaultCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 确保输出目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Configuration file generated successfully: %s\n", outputPath)
	return nil
}

// GenerateHandlerTemplate 生成处理器模板
func GenerateHandlerTemplate(name, outputPath string) error {
	template := fmt.Sprintf(`package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/herj1025/kumquat/pkg/response"
)

// %sHandler %s处理器
func %sHandler(c *gin.Context) {
	// TODO: 实现业务逻辑
	response.GinSuccess(c, gin.H{"message": "%s handler called"})
}
`, name, name, name, name)

	// 确保输出目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write handler file: %w", err)
	}

	fmt.Printf("Handler template generated successfully: %s\n", outputPath)
	return nil
}

// GenerateServiceTemplate 生成服务层模板
func GenerateServiceTemplate(name, outputPath string) error {
	template := fmt.Sprintf(`package service

import (
	"context"
)

// %sService %s服务
type %sService struct{}

// New%sService 创建新的%s服务
func New%sService() *%sService {
	return &%sService{}
}

// Get%s 获取%s信息
func (s *%sService) Get%s(ctx context.Context) error {
	// TODO: 实现业务逻辑
	return nil
}
`, name, name, name, name, name, name, name, name, name, name, name, name)

	// 确保输出目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	fmt.Printf("Service template generated successfully: %s\n", outputPath)
	return nil
}

// GenerateModuleTemplate 生成 Go 风格业务模块模板。
func GenerateModuleTemplate(name, outputDir string) error {
	packageName := filepath.Base(outputDir)
	moduleType := toCamel(name)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	files := map[string]string{
		"model.go": fmt.Sprintf(`package %s

// %s 是模块领域模型示例，可按业务需要扩展字段。
type %s struct {
	ID int64 `+"`json:\"id\"`"+`
}
`, packageName, moduleType, moduleType),
		"repository.go": fmt.Sprintf(`package %s

import "context"

type repository struct{}

func newRepository() *repository {
	return &repository{}
}

func (r *repository) ping(context.Context) error {
	return nil
}
`, packageName),
		"service.go": fmt.Sprintf(`package %s

import "context"

type service struct {
	repo *repository
}

func newService(repo *repository) *service {
	return &service{repo: repo}
}

func (s *service) get(ctx context.Context) error {
	return s.repo.ping(ctx)
}
`, packageName),
		"handler.go": fmt.Sprintf(`package %s

import (
	"github.com/gin-gonic/gin"
	"github.com/herj1025/kumquat/pkg/response"
)

type handler struct {
	svc *service
}

func newHandler(svc *service) *handler {
	return &handler{svc: svc}
}

func (h *handler) Get(c *gin.Context) {
	if err := h.svc.get(c); err != nil {
		response.GinError(c, err)
		return
	}
	response.GinSuccess(c, gin.H{"module": "%s"})
}
`, packageName, packageName),
		"routes.go": fmt.Sprintf(`package %s

import "github.com/herj1025/kumquat/pkg/server"

// RegisterRoutes 注册 %s 模块路由，并在模块内部完成依赖组装。
func RegisterRoutes(app *server.Application) error {
	repo := newRepository()
	svc := newService(repo)
	handler := newHandler(svc)

	routes := app.Engine().Group("/%s")
	routes.GET("", handler.Get)

	return nil
}
`, packageName, packageName, packageName),
	}

	for fileName, content := range files {
		outputPath := filepath.Join(outputDir, fileName)
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outputPath, err)
		}
	}

	fmt.Printf("Module template generated successfully: %s\n", outputDir)
	return nil
}

func toCamel(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}
