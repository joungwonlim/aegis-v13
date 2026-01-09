package httputil

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Env:      "test",
		LogLevel: "debug",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	client := New(cfg, log)
	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.httpClient == nil {
		t.Error("Expected http.Client to be initialized")
	}

	if client.logger == nil {
		t.Error("Expected logger to be set")
	}

	if client.retryConfig.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", client.retryConfig.MaxRetries)
	}
}

func TestNewWithTimeout(t *testing.T) {
	cfg := &config.Config{
		Env:      "test",
		LogLevel: "error", // Reduce log noise
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	timeout := 5 * time.Second
	client := NewWithTimeout(cfg, log, timeout)

	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected timeout=%v, got %v", timeout, client.httpClient.Timeout)
	}
}

func TestWithRetry(t *testing.T) {
	cfg := &config.Config{
		Env:      "test",
		LogLevel: "error",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	client := New(cfg, log).WithRetry(5, 2*time.Second)

	if client.retryConfig.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries=5, got %d", client.retryConfig.MaxRetries)
	}

	if client.retryConfig.InitialDelay != 2*time.Second {
		t.Errorf("Expected InitialDelay=2s, got %v", client.retryConfig.InitialDelay)
	}
}

func TestDisableRetry(t *testing.T) {
	cfg := &config.Config{
		Env:      "test",
		LogLevel: "error",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	client := New(cfg, log).DisableRetry()

	if client.retryConfig.Enabled {
		t.Error("Expected retry to be disabled")
	}
}

func TestGet(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Env:      "test",
		LogLevel: "error",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	client := New(cfg, log)
	ctx := context.Background()

	resp, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestPostJSON(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type=application/json, got %s", contentType)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"created":true}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Env:      "test",
		LogLevel: "error",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	client := New(cfg, log)
	ctx := context.Background()

	data := map[string]interface{}{
		"name":  "test",
		"value": 123,
	}

	resp, err := client.PostJSON(ctx, server.URL, data)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}

func TestRetryOn5xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Return 503 for first 2 attempts
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Success on 3rd attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Env:      "test",
		LogLevel: "error",
		Database: config.DatabaseConfig{
			URL: "dummy",
		},
	}
	log := logger.New(cfg)

	client := New(cfg, log).WithRetry(3, 100*time.Millisecond)
	ctx := context.Background()

	resp, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("Request failed after retries: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		statusCode int
		want       bool
	}{
		{200, false},
		{201, false},
		{400, false},
		{404, false},
		{429, true},  // Too Many Requests - should retry
		{500, true},  // Internal Server Error
		{502, true},  // Bad Gateway
		{503, true},  // Service Unavailable
		{504, true},  // Gateway Timeout
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			got := IsRetryableError(tt.statusCode)
			if got != tt.want {
				t.Errorf("IsRetryableError(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}
