package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/config"
)

func TestNew(t *testing.T) {
	// Skip if DATABASE_URL is not set
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	// Skip if DATABASE_URL is not set
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := db.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if !status.Healthy {
		t.Error("Expected database to be healthy")
	}

	if status.Stats.MaxConns == 0 {
		t.Error("Expected MaxConns to be greater than 0")
	}

	t.Logf("Health Status: %+v", status)
	t.Logf("Pool Stats: %+v", status.Stats)
}

func TestStats(t *testing.T) {
	// Skip if DATABASE_URL is not set
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	stats := db.Stats()

	if stats.MaxConns == 0 {
		t.Error("Expected MaxConns to be greater than 0")
	}

	t.Logf("Pool Stats: %+v", stats)
}

func TestNewWithInvalidURL(t *testing.T) {
	// Create config with invalid database URL
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL:             "invalid://url",
			MaxConns:        25,
			MinConns:        5,
			MaxConnLifetime: time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("Expected error with invalid database URL, got nil")
	}
}

func TestClose(t *testing.T) {
	// Skip if DATABASE_URL is not set
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Close should not panic
	db.Close()

	// Double close should not panic
	db.Close()
}
