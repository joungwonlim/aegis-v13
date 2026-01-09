package logger_test

import (
	"errors"

	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Example_basic demonstrates basic logger usage
func Example_basic() {
	// Load config
	cfg := &config.Config{
		Env:       "development",
		LogLevel:  "info",
		LogFormat: "console",
	}

	// Create logger (SSOT)
	log := logger.New(cfg)

	// Basic logging
	log.Debug("This won't appear (level is info)")
	log.Info("Application started")
	log.Warn("Low disk space")
	log.Error("Failed to connect")

	// Formatted logging
	log.Infof("User %s logged in", "alice")
	log.Warnf("Retry attempt %d of %d", 3, 5)

	// Output:
	// (console output with timestamps)
}

// Example_withFields demonstrates structured logging with fields
func Example_withFields() {
	cfg := &config.Config{
		Env:       "production",
		LogLevel:  "info",
		LogFormat: "json",
	}

	log := logger.New(cfg)

	// Add single field
	userLog := log.WithField("user_id", "12345")
	userLog.Info("User action performed")

	// Add multiple fields
	tradeLog := log.WithFields(map[string]interface{}{
		"stock_code": "005930",
		"price":      72300,
		"quantity":   100,
		"action":     "buy",
	})
	tradeLog.Info("Trade executed")

	// Output:
	// {"level":"info","user_id":"12345","message":"User action performed",...}
	// {"level":"info","stock_code":"005930","price":72300,"quantity":100,"action":"buy","message":"Trade executed",...}
}

// Example_withError demonstrates error logging
func Example_withError() {
	cfg := &config.Config{
		Env:       "production",
		LogLevel:  "error",
		LogFormat: "json",
	}

	log := logger.New(cfg)

	// Log with error
	err := errors.New("database connection timeout")
	log.WithError(err).Error("Failed to fetch stock data")

	// Combine error with fields
	log.WithError(err).
		WithFields(map[string]interface{}{
			"retry_count": 3,
			"timeout_ms":  5000,
		}).
		Error("Connection failed after retries")

	// Output:
	// {"level":"error","error":"database connection timeout","message":"Failed to fetch stock data",...}
	// {"level":"error","error":"database connection timeout","retry_count":3,"timeout_ms":5000,"message":"Connection failed after retries",...}
}

// Example_environments demonstrates different log formats
func Example_environments() {
	// Development: Pretty console logs
	devCfg := &config.Config{
		Env:       "development",
		LogLevel:  "debug",
		LogFormat: "console",
	}
	devLog := logger.New(devCfg)
	devLog.Debug("Debugging application flow")
	devLog.Info("Request received")

	// Production: JSON logs
	prodCfg := &config.Config{
		Env:       "production",
		LogLevel:  "info",
		LogFormat: "json",
	}
	prodLog := logger.New(prodCfg)
	prodLog.Info("Service started")
	prodLog.Warn("High memory usage detected")

	// Output:
	// (human-readable console output for development)
	// (machine-parseable JSON for production)
}
