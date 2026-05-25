// Package generator 提供代码和配置文件生成工具
package generator

import (
	"fmt"
	"os"
	"path/filepath"

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