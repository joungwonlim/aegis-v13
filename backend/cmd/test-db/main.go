package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
)

func main() {
	fmt.Println("=== Aegis v13 Database Connection Test ===")

	// Load configuration
	fmt.Println("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}
	fmt.Printf("✅ Config loaded (ENV: %s)\n", cfg.Env)
	fmt.Printf("   Database URL: %s\n\n", maskPassword(cfg.Database.URL))

	// Create database connection
	fmt.Println("Connecting to database...")
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()
	fmt.Println("✅ Database connection established")

	// Check connection
	fmt.Println("Testing connection (Ping)...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}
	fmt.Println("✅ Ping successful")

	// Get health status
	fmt.Println("Getting health status...")
	status, err := db.HealthCheck(ctx)
	if err != nil {
		log.Fatalf("❌ Health check failed: %v", err)
	}

	fmt.Println("✅ Health Check Results:")
	fmt.Printf("   Healthy: %v\n", status.Healthy)
	fmt.Printf("   Response Time: %v\n", status.ResponseTime)
	fmt.Printf("   Timestamp: %v\n\n", status.Timestamp.Format(time.RFC3339))

	// Pool statistics
	fmt.Println("📊 Connection Pool Statistics:")
	fmt.Printf("   Max Connections: %d\n", status.Stats.MaxConns)
	fmt.Printf("   Total Connections: %d\n", status.Stats.TotalConns)
	fmt.Printf("   Acquired Connections: %d\n", status.Stats.AcquiredConns)
	fmt.Printf("   Idle Connections: %d\n", status.Stats.IdleConns)
	fmt.Printf("   Constructing Connections: %d\n", status.Stats.ConstructingConns)
	fmt.Printf("   Acquire Count: %d\n", status.Stats.AcquireCount)
	fmt.Printf("   Acquire Duration: %v\n", status.Stats.AcquireDuration)

	fmt.Println("\n✅ All tests passed!")
}

// maskPassword masks the password in the database URL
func maskPassword(url string) string {
	// Simple masking for display purposes
	// postgresql://user:password@host:port/dbname
	// → postgresql://user:***@host:port/dbname
	return url[:30] + "***" + url[len(url)-25:]
}
