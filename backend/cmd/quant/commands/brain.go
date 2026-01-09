package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wonny/aegis/v13/backend/internal/audit"
	"github.com/wonny/aegis/v13/backend/internal/brain"
	"github.com/wonny/aegis/v13/backend/internal/execution"
	"github.com/wonny/aegis/v13/backend/internal/external/dart"
	"github.com/wonny/aegis/v13/backend/internal/external/krx"
	"github.com/wonny/aegis/v13/backend/internal/external/naver"
	"github.com/wonny/aegis/v13/backend/internal/portfolio"
	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/internal/s0_data/collector"
	"github.com/wonny/aegis/v13/backend/internal/s0_data/quality"
	"github.com/wonny/aegis/v13/backend/internal/s1_universe"
	"github.com/wonny/aegis/v13/backend/internal/s2_signals"
	"github.com/wonny/aegis/v13/backend/internal/selection"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// brainCmd represents the brain command
var brainCmd = &cobra.Command{
	Use:   "brain",
	Short: "Brain Orchestrator - 전체 파이프라인 실행",
	Long: `Brain Orchestrator는 7단계 파이프라인을 조율합니다.

S0 → S1 → S2 → S3 → S4 → S5 → S6 → S7

각 단계:
- S0: Data Quality Gate
- S1: Universe Generation
- S2: Signal Generation
- S3: Screening (Hard Cut)
- S4: Ranking
- S5: Portfolio Construction
- S6: Execution Planning
- S7: Performance Analysis

Example:
  go run ./cmd/quant brain run --date 2024-01-15
  go run ./cmd/quant brain run --dry-run`,
}

var (
	brainRunCmd = &cobra.Command{
		Use:   "run",
		Short: "전체 파이프라인 실행",
		Long: `7단계 파이프라인을 순차적으로 실행합니다.

Flags:
  --date       실행 날짜 (기본: 오늘)
  --capital    사용 가능 자본 (기본: 1억원)
  --dry-run    실행 계획만 생성 (실제 주문 X)

Example:
  go run ./cmd/quant brain run
  go run ./cmd/quant brain run --date 2024-01-15
  go run ./cmd/quant brain run --capital 100000000 --dry-run`,
		RunE: runBrain,
	}

	// Flags
	brainDate    string
	brainCapital int64
	brainDryRun  bool
)

func init() {
	rootCmd.AddCommand(brainCmd)
	brainCmd.AddCommand(brainRunCmd)

	// Flags
	brainRunCmd.Flags().StringVar(&brainDate, "date", "", "실행 날짜 (YYYY-MM-DD, 기본: 오늘)")
	brainRunCmd.Flags().Int64Var(&brainCapital, "capital", 100_000_000, "사용 가능 자본 (원)")
	brainRunCmd.Flags().BoolVar(&brainDryRun, "dry-run", false, "실행 계획만 생성 (실제 주문 X)")
}

func runBrain(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Aegis v13 Brain Orchestrator ===")

	// Parse date
	var runDate time.Time
	if brainDate != "" {
		parsed, err := time.Parse("2006-01-02", brainDate)
		if err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}
		runDate = parsed
	} else {
		runDate = time.Now()
	}

	fmt.Printf("\n📅 Run Date: %s\n", runDate.Format("2006-01-02"))
	fmt.Printf("💰 Capital: %s원\n", formatNumber(brainCapital))
	fmt.Printf("🔧 Dry Run: %v\n\n", brainDryRun)

	// Initialize dependencies
	orchestrator, err := initOrchestrator()
	if err != nil {
		return fmt.Errorf("init orchestrator: %w", err)
	}

	// Get git SHA
	gitSHA := getGitSHA()

	// Create run config
	runConfig := brain.RunConfig{
		Date:           runDate,
		RunID:          brain.GenerateRunID(),
		GitSHA:         gitSHA,
		FeatureVersion: "v1.0.0",
		Capital:        brainCapital,
		DryRun:         brainDryRun,
	}

	fmt.Printf("🚀 Starting pipeline run: %s\n\n", runConfig.RunID)

	// Execute pipeline
	result, err := orchestrator.Run(cmd.Context(), runConfig)
	if err != nil {
		return fmt.Errorf("pipeline run failed: %w", err)
	}

	// Print results
	printRunResult(result)

	return nil
}

func initOrchestrator() (*brain.Orchestrator, error) {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// 2. Initialize logger
	log := logger.New(cfg)

	// 3. Connect to database
	db, err := database.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	// 4. Create HTTP client
	httpClient := httputil.New(cfg, log)

	// 5. Create external API clients
	naverClient := naver.NewClient(httpClient, log)
	dartClient := dart.NewClient(cfg.DART.APIKey, log)
	krxClient := krx.NewClient(httpClient, log)

	// 6. Create repositories
	dataRepo := s0_data.NewRepository(db.Pool)
	universeRepo := s1_universe.NewRepository(db.Pool)
	signalRepo := s2_signals.NewRepository(db.Pool)
	selectionRepo := selection.NewRepository(db.Pool)
	portfolioRepo := portfolio.NewRepository(db.Pool)
	executionRepo := execution.NewRepository(db.Pool)
	auditRepo := audit.NewRepository(db.Pool)

	// 7. Create S0: Quality Gate
	qualityConfig := quality.Config{
		MinPriceCoverage:      1.0,
		MinVolumeCoverage:     1.0,
		MinMarketCapCoverage:  0.95,
		MinFinancialCoverage:  0.80,
		MinInvestorCoverage:   0.80,
		MinDisclosureCoverage: 0.70,
	}
	qualityGate := quality.NewQualityGate(db.Pool, qualityConfig)
	qualityRepo := quality.NewRepository(db.Pool)

	// 8. Create S1: Universe Builder
	universeConfig := s1_universe.Config{
		MinMarketCap:   10_000_000_000, // 100억 원
		MinVolume:      100_000_000,    // 1억 원
		MinListingDays: 20,
		ExcludeAdmin:   true,
		ExcludeHalt:    true,
		ExcludeSPAC:    true,
	}
	universeBuilder := s1_universe.NewBuilder(db.Pool, universeConfig)

	// 9. Create S2: Signal Builder
	col := collector.NewCollector(naverClient, dartClient, krxClient, dataRepo, log)
	// TODO: Initialize all signal calculators
	signalBuilder := s2_signals.NewBuilder(
		nil, // momentum
		nil, // technical
		nil, // value
		nil, // quality
		nil, // flow
		nil, // event
		dataRepo,
		dataRepo,
		dataRepo,
		dataRepo,
		log,
	)

	// 10. Create S3: Screener
	screenerConfig := selection.ScreenerConfig{
		MinMomentum:  -0.5,
		MinTechnical: -0.5,
		MinValue:     -0.5,
		MinQuality:   -0.5,
		MinFlow:      -0.5,
	}
	screener := selection.NewScreener(screenerConfig, log)

	// 11. Create S4: Ranker
	weights := selection.DefaultWeightConfig()
	ranker := selection.NewRanker(weights, log)

	// 12. Create S5: Portfolio Constructor
	portfolioConfig := portfolio.Config{
		MaxPositions:   20,
		MaxWeight:      0.15,
		MinWeight:      0.03,
		CashReserve:    0.05,
		WeightingMode:  "equal",
		TurnoverLimit:  1.0,
	}
	portfolioConstructor := portfolio.NewConstructor(portfolioConfig, log)

	// 13. Create S6: Execution Planner
	executionConfig := execution.Config{
		DefaultOrderType:  "limit",
		DefaultSlippage:   0.001,
		SplitThreshold:    50_000_000,
		MaxOrderSize:      50_000_000,
	}
	executionPlanner := execution.NewPlanner(executionConfig, log)

	// 14. Create S7: Performance Analyzer
	performanceAnalyzer := audit.NewPerformanceAnalyzer(db.Pool, log)

	// 15. Create Orchestrator
	orchestrator := brain.NewOrchestrator(
		qualityGate,
		universeBuilder,
		signalBuilder,
		screener,
		ranker,
		portfolioConstructor,
		executionPlanner,
		performanceAnalyzer,
		qualityRepo,
		universeRepo,
		signalRepo,
		selectionRepo,
		portfolioRepo,
		executionRepo,
		auditRepo,
		log,
	)

	return orchestrator, nil
}

func printRunResult(result *brain.RunResult) {
	fmt.Println("\n✅ Pipeline Run Completed")
	fmt.Println()

	// Summary
	fmt.Printf("Run ID: %s\n", result.RunID)
	fmt.Printf("Date: %s\n", result.Date.Format("2006-01-02"))
	fmt.Printf("Duration: %.2fs\n", result.Duration.Seconds())
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Println()

	// Stages
	fmt.Println("Completed Stages:")
	for _, stage := range result.CompletedStages {
		fmt.Printf("  ✅ %s\n", stage)
	}
	fmt.Println()

	// Results
	if result.Universe != nil {
		fmt.Printf("Universe: %d stocks\n", result.Universe.TotalCount)
	}
	if result.SignalSet != nil {
		fmt.Printf("Signals: %d stocks\n", len(result.SignalSet.Signals))
	}
	if len(result.ScreenedStocks) > 0 {
		fmt.Printf("Screened: %d stocks\n", len(result.ScreenedStocks))
	}
	if len(result.RankedStocks) > 0 {
		fmt.Printf("Ranked: %d stocks (top: %s, score: %.2f)\n",
			len(result.RankedStocks),
			result.RankedStocks[0].Code,
			result.RankedStocks[0].TotalScore)
	}
	if result.TargetPortfolio != nil {
		fmt.Printf("Portfolio: %d positions, %.0f원\n",
			len(result.TargetPortfolio.Positions),
			float64(result.TargetPortfolio.TotalValue))
	}
	if result.ExecutionPlan != nil {
		fmt.Printf("Execution: %d orders\n", len(result.ExecutionPlan.Orders))
	}
	if result.PerformanceReport != nil {
		fmt.Printf("Performance: Return=%.2f%%, Sharpe=%.2f, MDD=%.2f%%\n",
			result.PerformanceReport.TotalReturn*100,
			result.PerformanceReport.SharpeRatio,
			result.PerformanceReport.MaxDrawdown*100)
	}
}

func getGitSHA() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func formatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	var result []rune
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, c)
	}
	return string(result)
}
