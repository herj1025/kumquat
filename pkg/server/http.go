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
	"github.com/herj1025/kumquat/pkg/app"
	"github.com/herj1025/kumquat/pkg/logger"
	"github.com/herj1025/kumquat/pkg/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ServerOption 服务器配置选项
type ServerOption func(*Server)

// WithRoutes 注册业务路由
func WithRoutes(registerFn func(engine *gin.Engine)) ServerOption {
	return func(s *Server) {
		registerFn(s.engine)
	}
}

// Server HTTP 服务器
type Server struct {
	engine  *gin.Engine
	cfg     *config.Config
	httpSrv *http.Server
	app     *app.App
}

// New 创建并初始化服务器
func New(cfg *config.Config, options ...ServerOption) (*Server, error) {
	if cfg.Server.Mode != "" {
		gin.SetMode(cfg.Server.Mode)
	}

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.RequestTrace())
	r.Use(middleware.I18n())

	application, err := app.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize app: %w", err)
	}

	srv := &Server{
		engine: r,
		cfg:    cfg,
		app:    application,
	}

	// 应用可选配置
	for _, opt := range options {
		opt(srv)
	}

	return srv, nil
}

func (s *Server) getServerAddr() string {
	port := s.cfg.Server.Port
	if port == 0 {
		logger.Warn("Server port is not set, using default port 8080")
		port = 8080
	}
	return fmt.Sprintf(":%d", port)
}

func (s *Server) initHTTPServer(addr string) {
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

	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
}

func (s *Server) startServer(addr string) <-chan error {
	errChan := make(chan error, 1)
	go func() {
		logger.Info(fmt.Sprintf("Server starting on %s...", addr))
		if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("server listen error: %w", err)
		}
	}()
	return errChan
}

func (s *Server) shutdownServer() error {
	timeout := time.Duration(s.cfg.Server.ShutdownTimeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}
	return nil
}

// Run 启动服务器并阻塞等待退出信号
func (s *Server) Run() error {
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

	if err := s.app.Close(); err != nil {
		logger.Error("failed to close app", zap.Error(err))
	}

	logger.Info("Server exiting")
	return nil
}

// Engine 返回 gin.Engine 实例，用于外部注册路由或自定义配置
func (s *Server) Engine() *gin.Engine {
	return s.engine
}
