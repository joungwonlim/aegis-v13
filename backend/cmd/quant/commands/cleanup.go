package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "데이터 정리 도구",
	Long: `데이터베이스 정리 작업을 수행합니다.

Example:
  quant cleanup investor-flow`,
}

var cleanupInvestorFlowCmd = &cobra.Command{
	Use:   "investor-flow",
	Short: "투자자 매매동향 데이터 정리",
	Long: `단위가 잘못된 투자자 매매동향 데이터를 삭제합니다.

2025-12-24 이전 데이터는 금액(원) 단위로 저장되어 있어,
주식수 단위 데이터와 혼재되어 차트 표시에 문제가 있습니다.
이 명령어는 2025-12-24 이전 데이터를 삭제합니다.

Example:
  quant cleanup investor-flow`,
	RunE: runCleanupInvestorFlow,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.AddCommand(cleanupInvestorFlowCmd)
}

func runCleanupInvestorFlow(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Investor Flow Data Cleanup ===")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("❌ Failed to load config: %w", err)
	}

	// Create database connection
	db, err := database.New(cfg)
	if err != nil {
		return fmt.Errorf("❌ Failed to connect to database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Cutoff date: 2025-12-24
	cutoffDate := time.Date(2025, 12, 24, 0, 0, 0, 0, time.UTC)

	// Count records before cleanup
	var beforeCount int64
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM data.investor_flow WHERE trade_date < $1
	`, cutoffDate).Scan(&beforeCount)
	if err != nil {
		return fmt.Errorf("❌ Failed to count records: %w", err)
	}

	fmt.Printf("📊 Found %d records with wrong units (before %s)\n", beforeCount, cutoffDate.Format("2006-01-02"))

	if beforeCount == 0 {
		fmt.Println("✅ No data to clean up")
		return nil
	}

	// Delete old data
	fmt.Println("🗑️ Deleting old data...")
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM data.investor_flow WHERE trade_date < $1
	`, cutoffDate)
	if err != nil {
		return fmt.Errorf("❌ Failed to delete records: %w", err)
	}

	rowsAffected := result.RowsAffected()
	fmt.Printf("✅ Deleted %d records\n", rowsAffected)

	// Count remaining records
	var afterCount int64
	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM data.investor_flow`).Scan(&afterCount)
	if err != nil {
		return fmt.Errorf("❌ Failed to count remaining records: %w", err)
	}

	fmt.Printf("📊 Remaining records: %d\n", afterCount)
	fmt.Println("\n✅ Cleanup complete!")

	return nil
}
