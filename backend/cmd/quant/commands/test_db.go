package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
)

// testDBCmd represents the test-db command
var testDBCmd = &cobra.Command{
	Use:   "test-db",
	Short: "PostgreSQL 연결 테스트",
	Long: `데이터베이스 연결을 테스트하고 풀 통계를 표시합니다.

이 명령어는:
- config에서 DATABASE_URL 로드
- 데이터베이스 연결 생성
- Ping 테스트
- Health Check 실행
- Connection Pool 통계 표시

Example:
  go run ./cmd/quant test-db
  go run ./cmd/quant test-db --env production`,
	RunE: runTestDB,
}

func init() {
	rootCmd.AddCommand(testDBCmd)
}

func runTestDB(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Aegis v13 Database Connection Test ===")

	// Load configuration
	fmt.Println("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("❌ Failed to load config: %w", err)
	}
	fmt.Printf("✅ Config loaded (ENV: %s)\n", cfg.Env)
	fmt.Printf("   Database URL: %s\n\n", maskPassword(cfg.Database.URL))

	// Create database connection
	fmt.Println("Connecting to database...")
	db, err := database.New(cfg)
	if err != nil {
		return fmt.Errorf("❌ Failed to connect to database: %w", err)
	}
	defer db.Close()
	fmt.Println("✅ Database connection established")

	// Check connection
	fmt.Println("Testing connection (Ping)...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("❌ Failed to ping database: %w", err)
	}
	fmt.Println("✅ Ping successful")

	// Get health status
	fmt.Println("Getting health status...")
	status, err := db.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("❌ Health check failed: %w", err)
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
	return nil
}

// maskPassword masks the password in the database URL for display
func maskPassword(url string) string {
	// Simple masking: postgresql://user:password@host:port/dbname
	// → postgresql://user:***@host:port/dbname
	if len(url) < 55 {
		return url[:30] + "***"
	}
	return url[:30] + "***" + url[len(url)-25:]
}
