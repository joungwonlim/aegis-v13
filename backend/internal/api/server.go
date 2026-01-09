package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Server represents the HTTP API server
// ⭐ SSOT: API 서버 설정은 이 파일에서만
type Server struct {
	httpServer *http.Server
	logger     *logger.Logger
	config     *config.Config
}

// New creates a new API server
func New(cfg *config.Config, log *logger.Logger, router http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         ":" + cfg.Port,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: log,
		config: cfg,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.WithFields(map[string]interface{}{
		"port": s.config.Port,
		"env":  s.config.Env,
	}).Info("Starting API server")

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down API server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}
