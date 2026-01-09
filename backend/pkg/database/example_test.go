package database_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
)

// Example demonstrates how to use the database package
func Example() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create database connection
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Check connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Get health status
	status, err := db.HealthCheck(ctx)
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}

	fmt.Printf("Database is healthy: %v\n", status.Healthy)
	fmt.Printf("Response time: %v\n", status.ResponseTime)
	fmt.Printf("Max connections: %d\n", status.Stats.MaxConns)
	fmt.Printf("Active connections: %d\n", status.Stats.AcquiredConns)
	fmt.Printf("Idle connections: %d\n", status.Stats.IdleConns)
}
