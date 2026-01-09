package httputil_test

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Example_basic demonstrates basic HTTP client usage
func Example_basic() {
	// Create config and logger
	cfg := &config.Config{
		Env:      "production",
		LogLevel: "info",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	// Create HTTP client (SSOT)
	client := httputil.New(cfg, log)

	// Make GET request
	ctx := context.Background()
	resp, err := client.Get(ctx, "https://api.example.com/data")
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	// Output:
	// (Status code from real request)
}

// Example_withRetry demonstrates retry configuration
func Example_withRetry() {
	cfg := &config.Config{
		Env:      "production",
		LogLevel: "info",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	// Create client with custom retry settings
	client := httputil.New(cfg, log).
		WithRetry(5, 2*time.Second) // 5 retries, 2s initial delay

	ctx := context.Background()
	resp, err := client.Get(ctx, "https://api.example.com/data")
	if err != nil {
		fmt.Printf("Request failed after retries: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Request succeeded")
	// Output:
	// (Success or failure after retries)
}

// Example_postJSON demonstrates JSON POST requests
func Example_postJSON() {
	cfg := &config.Config{
		Env:      "production",
		LogLevel: "info",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	client := httputil.New(cfg, log)

	// Prepare JSON data
	data := map[string]interface{}{
		"symbol": "005930",
		"action": "buy",
		"qty":    100,
	}

	// Send POST request with JSON body
	ctx := context.Background()
	resp, err := client.PostJSON(ctx, "https://api.example.com/orders", data)
	if err != nil {
		fmt.Printf("POST request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Order created: %d\n", resp.StatusCode)
	// Output:
	// (Status code from real request)
}

// Example_timeout demonstrates custom timeout
func Example_timeout() {
	cfg := &config.Config{
		Env:      "production",
		LogLevel: "info",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	// Create client with 5 second timeout
	client := httputil.NewWithTimeout(cfg, log, 5*time.Second)

	ctx := context.Background()
	resp, err := client.Get(ctx, "https://api.example.com/slow-endpoint")
	if err != nil {
		fmt.Printf("Request timeout: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Request completed within timeout")
	// Output:
	// (Success or timeout error)
}

// Example_disableRetry demonstrates disabling retry
func Example_disableRetry() {
	cfg := &config.Config{
		Env:      "production",
		LogLevel: "info",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	// Create client without retry
	client := httputil.New(cfg, log).DisableRetry()

	ctx := context.Background()
	resp, err := client.Get(ctx, "https://api.example.com/data")
	if err != nil {
		fmt.Printf("Request failed (no retry): %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Request succeeded on first attempt")
	// Output:
	// (Success or immediate failure)
}
