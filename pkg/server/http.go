package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/herj1025/kumquat/config"
	"github.com/herj1025/kumquat/pkg/deps"
	"github.com/herj1025/kumquat/pkg/logger"
	"github.com/herj1025/kumquat/pkg/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Application 是应用运行时宿主，负责 HTTP 服务生命周期和基础设施访问。
type Application struct {
	engine     *gin.Engine
	cfg        *config.Config
	httpServer *http.Server
	deps       *deps.Deps
}

// New 创建并初始化应用运行时
func New(cfg *config.Config) (*Application, error) {
	if cfg.Server.Mode != "" {
		gin.SetMode(cfg.Server.Mode)
	}

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.RequestTrace())
	r.Use(middleware.I18n())

	deps, err := deps.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize app: %w", err)
	}

	srv := &Application{
		engine: r,
		cfg:    cfg,
		deps:   deps,
	}

	return srv, nil
}

func (s *Application) getServerAddr() string {
	port := s.cfg.Server.Port
	if port == 0 {
		logger.Warn("Server port is not set, using default port 8080")
		port = 8080
	}
	return fmt.Sprintf(":%d", port)
}

func (s *Application) initHTTPServer(addr string) {
	cfg := s.cfg.Server

	readTimeout := time.Duration(cfg.ReadTimeout) * time.Second
	if readTimeout == 0 {
		readTimeout = 10 * time.Second
	}
	writeTimeout := time.Duration(cfg.WriteTimeout) * time.Second
	if writeTimeout == 0 {
		writeTimeout = 10 * time.Second
	}
	idleTimeout := time.Duration(cfg.IdleTimeout) * time.Second
	if idleTimeout == 0 {
		idleTimeout = 120 * time.Second
	}

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
}

func (s *Application) startServer(addr string) <-chan error {
	errChan := make(chan error, 1)
	go func() {
		logger.Info(fmt.Sprintf("Server starting on %s...", addr))
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("server listen error: %w", err)
		}
	}()
	return errChan
}

func (s *Application) shutdownServer() error {
	timeout := time.Duration(s.cfg.Server.ShutdownTimeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}
	return nil
}

// Run 启动服务器并阻塞等待退出信号
func (s *Application) Run() error {
	addr := s.getServerAddr()
	s.initHTTPServer(addr)
	errChan := s.startServer(addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return fmt.Errorf("server failed to start: %w", err)
	case <-quit:
		logger.Info("Shutting down server...")
	}

	if err := s.shutdownServer(); err != nil {
		return err
	}

	if err := s.deps.Close(); err != nil {
		logger.Error("failed to close deps", zap.Error(err))
	}

	logger.Info("Server exiting")
	return nil
}

// Engine 返回 gin.Engine 实例，用于外部注册路由或自定义配置
func (s *Application) Engine() *gin.Engine {
	return s.engine
}

// deps 返回应用依赖容器，便于外部复用框架初始化好的基础设施。
func (s *Application) Deps() *deps.Deps {
	if s == nil {
		return nil
	}
	return s.deps
}
