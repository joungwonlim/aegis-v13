package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// testLoggerCmd represents the test-logger command
var testLoggerCmd = &cobra.Command{
	Use:   "test-logger",
	Short: "Logger 기능 테스트",
	Long: `구조화된 로깅 기능을 테스트합니다.

이 명령어는:
- JSON/Console 포맷 테스트
- 로그 레벨 테스트
- 구조화된 필드 로깅
- 에러 컨텍스트 로깅

Example:
  go run ./cmd/quant test-logger
  go run ./cmd/quant test-logger --env production`,
	RunE: runTestLogger,
}

func init() {
	rootCmd.AddCommand(testLoggerCmd)
}

func runTestLogger(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Aegis v13 Logger Test ===")

	// Test 1: JSON Format (Production)
	fmt.Println("1. JSON Format (Production)")
	fmt.Println("--------------------------------")
	if err := testJSONFormat(); err != nil {
		return err
	}
	fmt.Println()

	// Test 2: Console Format (Development)
	fmt.Println("2. Console Format (Development)")
	fmt.Println("--------------------------------")
	if err := testConsoleFormat(); err != nil {
		return err
	}
	fmt.Println()

	// Test 3: Structured Logging
	fmt.Println("3. Structured Logging with Fields")
	fmt.Println("--------------------------------")
	if err := testStructuredLogging(); err != nil {
		return err
	}
	fmt.Println()

	// Test 4: Error Logging
	fmt.Println("4. Error Logging")
	fmt.Println("--------------------------------")
	if err := testErrorLogging(); err != nil {
		return err
	}
	fmt.Println()

	fmt.Println("✅ All logger tests completed!")
	return nil
}

func testJSONFormat() error {
	cfg := &config.Config{
		Env:       "production",
		LogLevel:  "info",
		LogFormat: "json",
		Database: config.DatabaseConfig{
			URL: "dummy", // Required by config validation
		},
	}

	log := logger.New(cfg)
	log.Info("Service started")
	log.Warn("High memory usage detected")
	log.Error("Failed to connect to external API")
	return nil
}

func testConsoleFormat() error {
	cfg := &config.Config{
		Env:       "development",
		LogLevel:  "debug",
		LogFormat: "console",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}

	log := logger.New(cfg)
	log.Debug("Debugging application flow")
	log.Info("Request received from client")
	log.Warn("Cache miss, fetching from database")
	return nil
}

func testStructuredLogging() error {
	cfg := &config.Config{
		Env:       "production",
		LogLevel:  "info",
		LogFormat: "json",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}

	log := logger.New(cfg)

	// Single field
	userLog := log.WithField("user_id", "USR-12345")
	userLog.Info("User logged in")

	// Multiple fields
	tradeLog := log.WithFields(map[string]interface{}{
		"stock_code": "005930",
		"price":      72300,
		"quantity":   100,
		"action":     "buy",
	})
	tradeLog.Info("Trade executed successfully")

	// Chained fields
	log.WithField("module", "data-collector").
		WithField("source", "KIS").
		Info("Data collection started")
	return nil
}

func testErrorLogging() error {
	cfg := &config.Config{
		Env:       "production",
		LogLevel:  "error",
		LogFormat: "json",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}

	log := logger.New(cfg)

	// Simple error
	err := errors.New("connection timeout")
	log.WithError(err).Error("Failed to fetch stock prices")

	// Error with context
	log.WithError(err).
		WithFields(map[string]interface{}{
			"retry_count": 3,
			"timeout_ms":  5000,
			"endpoint":    "/api/prices",
		}).
		Error("Connection failed after retries")
	return nil
}
