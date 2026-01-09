package main

import (
	"errors"
	"fmt"

	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

func main() {
	fmt.Println("=== Aegis v13 Logger Test ===")

	// Test 1: JSON Format (Production)
	fmt.Println("1. JSON Format (Production)")
	fmt.Println("--------------------------------")
	testJSONFormat()
	fmt.Println()

	// Test 2: Console Format (Development)
	fmt.Println("2. Console Format (Development)")
	fmt.Println("--------------------------------")
	testConsoleFormat()
	fmt.Println()

	// Test 3: Structured Logging
	fmt.Println("3. Structured Logging with Fields")
	fmt.Println("--------------------------------")
	testStructuredLogging()
	fmt.Println()

	// Test 4: Error Logging
	fmt.Println("4. Error Logging")
	fmt.Println("--------------------------------")
	testErrorLogging()
	fmt.Println()

	fmt.Println("✅ All logger tests completed!")
}

func testJSONFormat() {
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
}

func testConsoleFormat() {
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
}

func testStructuredLogging() {
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
}

func testErrorLogging() {
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
}
