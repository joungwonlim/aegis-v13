package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/wonny/aegis/v13/backend/internal/api"
	"github.com/wonny/aegis/v13/backend/internal/api/handlers"
	"github.com/wonny/aegis/v13/backend/internal/external/dart"
	"github.com/wonny/aegis/v13/backend/internal/external/krx"
	"github.com/wonny/aegis/v13/backend/internal/external/naver"
	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/internal/s0_data/collector"
	"github.com/wonny/aegis/v13/backend/internal/s0_data/quality"
	"github.com/wonny/aegis/v13/backend/internal/s1_universe"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// apiCmd represents the api command
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "API 서버 시작",
	Long: `REST API 서버를 시작합니다.

이 명령어는:
- HTTP API 서버 시작
- 데이터 조회 엔드포인트 제공
- 데이터 수집 트리거 제공

Endpoints:
  GET  /health               - Health check
  GET  /api/data/quality     - 품질 스냅샷 조회
  GET  /api/data/universe    - Universe 조회
  POST /api/data/collect     - 데이터 수집 트리거

Example:
  go run ./cmd/quant api
  go run ./cmd/quant api --port 8080`,
	RunE: runAPIServer,
}

var (
	apiPort string
)

func init() {
	rootCmd.AddCommand(apiCmd)

	// Flags
	apiCmd.Flags().StringVar(&apiPort, "port", "8080", "API 서버 포트")
}

func runAPIServer(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Aegis v13 API Server ===")

	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Override port if flag is set
	if apiPort != "" {
		cfg.Port = apiPort
	}

	// 2. Initialize logger
	log := logger.New(cfg)

	log.WithFields(map[string]interface{}{
		"port": cfg.Port,
		"env":  cfg.Env,
	}).Info("Initializing API server")

	// 3. Connect to database
	db, err := database.New(cfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	log.Info("Connected to database")

	// 4. Create HTTP client
	httpClient := httputil.New(cfg, log)

	// 5. Create external API clients
	naverClient := naver.NewClient(httpClient, log)
	dartClient := dart.NewClient(cfg.DART.APIKey, log)
	krxClient := krx.NewClient(httpClient, log)

	// 6. Create repositories
	dataRepo := s0_data.NewRepository(db.Pool)
	universeRepo := s1_universe.NewRepository(db.Pool)

	// 7. Create quality gate
	qualityConfig := quality.Config{
		MinPriceCoverage:      1.0,
		MinVolumeCoverage:     1.0,
		MinMarketCapCoverage:  0.95,
		MinFinancialCoverage:  0.80,
		MinInvestorCoverage:   0.80,
		MinDisclosureCoverage: 0.70,
	}
	qualityGate := quality.NewQualityGate(db.Pool, qualityConfig)

	// 8. Create collector
	col := collector.NewCollector(naverClient, dartClient, krxClient, dataRepo, log)

	// 9. Create handler
	dataHandler := handlers.NewDataHandler(dataRepo, universeRepo, col, qualityGate, log)

	// 10. Create router
	router := api.NewRouter(dataHandler, log)

	// 11. Create server
	server := api.New(cfg, log, router)

	// 12. Start server with graceful shutdown
	go func() {
		if err := server.Start(); err != nil {
			log.WithError(err).Fatal("Failed to start server")
		}
	}()

	log.Info("API server started successfully")
	fmt.Printf("\n✅ Server running on http://localhost:%s\n", cfg.Port)
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("  GET  /health")
	fmt.Println("  GET  /api/data/quality")
	fmt.Println("  GET  /api/data/universe")
	fmt.Println("  POST /api/data/collect")
	fmt.Println("\nPress Ctrl+C to stop")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Info("Server stopped")
	return nil
}
