package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wonny/aegis/v13/backend/internal/execution"
	"github.com/wonny/aegis/v13/backend/internal/risk"
	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// gateCmd represents the gate command
var gateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Risk Gate - S6 리스크 게이트 관리",
	Long: `S6 리스크 게이트를 관리합니다.

리스크 게이트 모드:
- shadow:  로깅만 (실제 차단 안함)
- enforce: 실제 차단
- off:     비활성화

Example:
  go run ./cmd/quant gate status
  go run ./cmd/quant gate test --demo
  go run ./cmd/quant gate stats --from 2024-01-01`,
}

var (
	gateStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "현재 게이트 상태 확인",
		RunE:  runGateStatus,
	}

	gateTestCmd = &cobra.Command{
		Use:   "test",
		Short: "게이트 테스트 실행",
		Long: `샘플 포트폴리오로 게이트 테스트를 실행합니다.

Example:
  go run ./cmd/quant gate test --demo
  go run ./cmd/quant gate test --demo --mode enforce`,
		RunE: runGateTest,
	}

	gateStatsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Shadow 모드 통계 조회",
		Long: `Shadow 모드에서의 차단 통계를 조회합니다.

Example:
  go run ./cmd/quant gate stats
  go run ./cmd/quant gate stats --from 2024-01-01 --to 2024-01-31`,
		RunE: runGateStats,
	}

	// Flags
	gateMode     string
	gateDemo     bool
	gateFrom     string
	gateTo       string
)

func init() {
	rootCmd.AddCommand(gateCmd)
	gateCmd.AddCommand(gateStatusCmd)
	gateCmd.AddCommand(gateTestCmd)
	gateCmd.AddCommand(gateStatsCmd)

	// test flags
	gateTestCmd.Flags().BoolVar(&gateDemo, "demo", false, "Use sample portfolio for testing")
	gateTestCmd.Flags().StringVar(&gateMode, "mode", "shadow", "Gate mode: shadow, enforce, off")

	// stats flags
	gateStatsCmd.Flags().StringVar(&gateFrom, "from", "", "Start date (YYYY-MM-DD)")
	gateStatsCmd.Flags().StringVar(&gateTo, "to", "", "End date (YYYY-MM-DD)")
}

func runGateStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Risk Gate Status ===")
	fmt.Println()
	fmt.Println("📊 Gate Configuration")
	fmt.Println("  Default Mode: shadow (log only, no blocking)")
	fmt.Println()
	fmt.Println("🎯 Risk Limits (Default)")
	limits := risk.DefaultRiskLimits()
	fmt.Printf("  Max VaR 95%%:          %.1f%%\n", limits.MaxVaR95*100)
	fmt.Printf("  Max VaR 99%%:          %.1f%%\n", limits.MaxVaR99*100)
	fmt.Printf("  Max Single Exposure:  %.1f%%\n", limits.MaxSingleExposure*100)
	fmt.Printf("  Max Sector Exposure:  %.1f%%\n", limits.MaxSectorExposure*100)
	fmt.Printf("  Max Concentration:    %.1f%%\n", limits.MaxConcentration*100)
	fmt.Printf("  Min Liquidity Score:  %.1f%%\n", limits.MinLiquidityScore*100)
	fmt.Println()
	fmt.Println("💡 Use 'gate test --demo' to run a test check")
	fmt.Println("   Use 'gate stats' to view shadow mode statistics")

	return nil
}

func runGateTest(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Risk Gate Test ===")

	ctx := cmd.Context()

	// 모드 파싱
	mode := execution.GateMode(gateMode)
	if mode != execution.GateModeShadow && mode != execution.GateModeEnforce && mode != execution.GateModeOff {
		return fmt.Errorf("invalid mode: %s (use: shadow, enforce, off)", gateMode)
	}

	fmt.Printf("📋 Mode: %s\n", mode)

	if !gateDemo {
		fmt.Println("\n💡 Tip: Use --demo flag for sample portfolio test")
		fmt.Println("   Example: go run ./cmd/quant gate test --demo")
		return nil
	}

	// 의존성 초기화
	cfg, log, db, err := initGateDeps()
	if err != nil {
		return err
	}
	defer db.Close()
	_ = cfg

	// 샘플 포트폴리오
	fmt.Println("\n🧪 Using sample portfolio for test")
	targetHoldings := []risk.Holding{
		{Code: "005930", Name: "삼성전자", Weight: 0.15, MarketValue: 15000000},
		{Code: "000660", Name: "SK하이닉스", Weight: 0.12, MarketValue: 12000000},
		{Code: "035420", Name: "NAVER", Weight: 0.10, MarketValue: 10000000},
		{Code: "035720", Name: "카카오", Weight: 0.10, MarketValue: 10000000},
		{Code: "051910", Name: "LG화학", Weight: 0.08, MarketValue: 8000000},
		{Code: "006400", Name: "삼성SDI", Weight: 0.08, MarketValue: 8000000},
		{Code: "005380", Name: "현대차", Weight: 0.07, MarketValue: 7000000},
		{Code: "000270", Name: "기아", Weight: 0.07, MarketValue: 7000000},
		{Code: "068270", Name: "셀트리온", Weight: 0.06, MarketValue: 6000000},
		{Code: "105560", Name: "KB금융", Weight: 0.05, MarketValue: 5000000},
	}

	// 비중 합계 출력
	totalWeight := 0.0
	for _, h := range targetHoldings {
		totalWeight += h.Weight
	}
	fmt.Printf("📈 Portfolio: %d stocks, %.1f%% total weight\n", len(targetHoldings), totalWeight*100)

	// 리스크 엔진 생성
	engine := risk.NewEngine(risk.DefaultRiskLimits(), risk.DefaultMonteCarloConfig(), log)

	// 가격 데이터 어댑터 생성
	priceRepo := s0_data.NewPriceRepository(db.Pool)
	priceAdapter := &priceRepoAdapter{repo: priceRepo}

	// 게이트 생성
	execRepo := execution.NewRepository(db.Pool)
	gateConfig := execution.RiskGateConfig{
		Mode:         mode,
		LookbackDays: 200,
	}
	gate := execution.NewRiskGate(engine, execRepo, priceAdapter, log, gateConfig)

	// 게이트 체크 실행
	fmt.Println("\n🔍 Running gate check...")
	input := execution.GateCheckInput{
		TargetHoldings: targetHoldings,
	}

	result, err := gate.Check(ctx, input)
	if err != nil {
		return fmt.Errorf("gate check failed: %w", err)
	}

	// 결과 출력
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("                  GATE CHECK RESULT")
	fmt.Println(strings.Repeat("=", 50))

	// 결과 상태
	switch result.Action {
	case execution.GateActionPass:
		fmt.Println("✅ PASSED")
	case execution.GateActionReduce:
		fmt.Println("⚠️  REDUCED (positions adjusted)")
	case execution.GateActionBlock:
		fmt.Println("❌ BLOCKED")
	default:
		if result.Passed {
			fmt.Println("✅ PASSED")
		} else {
			fmt.Println("❌ BLOCKED")
		}
	}

	if result.WouldBlock {
		fmt.Println("⚠️  Would Block: YES (in enforce mode)")
	} else {
		fmt.Println("   Would Block: NO")
	}

	fmt.Printf("\n📊 Mode: %s\n", result.Mode)
	fmt.Printf("🎬 Action: %s\n", result.Action)
	fmt.Printf("🆔 Run ID: %s\n", result.RunID)
	fmt.Printf("⏰ Checked At: %s\n", result.CheckedAt.Format("2006-01-02 15:04:05"))

	if result.RiskCheck != nil {
		fmt.Println("\n📈 Risk Metrics")
		fmt.Printf("  VaR 95%%: %.4f (%.2f%%)\n", result.RiskCheck.Metrics.PortfolioVaR95, result.RiskCheck.Metrics.PortfolioVaR95*100)
		fmt.Printf("  VaR 99%%: %.4f (%.2f%%)\n", result.RiskCheck.Metrics.PortfolioVaR99, result.RiskCheck.Metrics.PortfolioVaR99*100)
		fmt.Printf("  Max Single Exposure: %.4f (%.2f%%)\n", result.RiskCheck.Metrics.MaxSingleExposure, result.RiskCheck.Metrics.MaxSingleExposure*100)
		fmt.Printf("  Concentration (Top 5): %.4f (%.2f%%)\n", result.RiskCheck.Metrics.ConcentrationRatio, result.RiskCheck.Metrics.ConcentrationRatio*100)

		if len(result.RiskCheck.Violations) > 0 {
			fmt.Printf("\n⚠️  Violations (%d)\n", len(result.RiskCheck.Violations))
			for i, v := range result.RiskCheck.Violations {
				fmt.Printf("  %d. [%s] %s\n", i+1, v.Severity, v.Type)
				fmt.Printf("     Limit: %.2f%%, Actual: %.2f%%\n", v.Limit*100, v.Actual*100)
				fmt.Printf("     %s\n", v.Message)
			}
		}
	}

	// 축소된 주문 정보 (Enforce 모드)
	if len(result.AdjustedOrders) > 0 {
		fmt.Printf("\n📉 Adjusted Positions (%d)\n", len(result.AdjustedOrders))
		for i, adj := range result.AdjustedOrders {
			fmt.Printf("  %d. %s: %.1f%% → %.1f%%\n", i+1, adj.Code, adj.OriginalWeight*100, adj.AdjustedWeight*100)
			fmt.Printf("     Reason: %s\n", adj.Reason)
		}
	}

	// 차단된 주문 정보
	if len(result.BlockedOrders) > 0 {
		fmt.Printf("\n🚫 Blocked Orders (%d): %s\n", len(result.BlockedOrders), strings.Join(result.BlockedOrders, ", "))
	}

	fmt.Printf("\n💬 Message: %s\n", result.Message)

	return nil
}

func runGateStats(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Shadow Mode Statistics ===")

	ctx := cmd.Context()

	// 날짜 파싱
	var from, to time.Time
	var err error

	if gateFrom != "" {
		from, err = time.Parse("2006-01-02", gateFrom)
		if err != nil {
			return fmt.Errorf("invalid from date: %w", err)
		}
	} else {
		from = time.Now().AddDate(0, 0, -30) // 기본: 최근 30일
	}

	if gateTo != "" {
		to, err = time.Parse("2006-01-02", gateTo)
		if err != nil {
			return fmt.Errorf("invalid to date: %w", err)
		}
	} else {
		to = time.Now()
	}

	fmt.Printf("📅 Period: %s ~ %s\n\n", from.Format("2006-01-02"), to.Format("2006-01-02"))

	// 의존성 초기화
	cfg, _, db, err := initGateDeps()
	if err != nil {
		return err
	}
	defer db.Close()
	_ = cfg

	// 통계 조회
	execRepo := execution.NewRepository(db.Pool)
	stats, err := execRepo.GetShadowBlockStats(ctx, from, to.Add(24*time.Hour))
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Println("📊 Shadow Mode Statistics")
	fmt.Printf("  Total Checks:     %d\n", stats.TotalChecks)
	fmt.Printf("  Would Block:      %d\n", stats.WouldBlockCount)
	fmt.Printf("  Block Rate:       %.2f%%\n", stats.BlockRate*100)
	fmt.Printf("  Avg VaR 95%%:      %.4f\n", stats.AvgVaR95)
	fmt.Printf("  Max VaR 95%%:      %.4f\n", stats.MaxVaR95)

	if stats.TotalChecks == 0 {
		fmt.Println("\n💡 No gate events found in this period.")
		fmt.Println("   Run 'gate test --demo' to generate test events.")
	}

	return nil
}

func initGateDeps() (*config.Config, *logger.Logger, *database.DB, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg)

	db, err := database.New(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("connect to database: %w", err)
	}

	return cfg, log, db, nil
}

// priceRepoAdapter PriceRepository 인터페이스 어댑터
type priceRepoAdapter struct {
	repo *s0_data.PriceRepository
}

func (a *priceRepoAdapter) GetHistoricalReturns(ctx context.Context, codes []string, days int) (map[string][]float64, error) {
	result := make(map[string][]float64)

	toDate := time.Now()
	fromDate := toDate.AddDate(0, 0, -days)

	for _, code := range codes {
		prices, err := a.repo.GetByCodeAndDateRange(ctx, code, fromDate, toDate)
		if err != nil {
			continue // 개별 종목 실패는 무시
		}

		if len(prices) < 2 {
			continue
		}

		// 수익률 계산
		returns := make([]float64, 0, len(prices)-1)
		for i := 1; i < len(prices); i++ {
			if prices[i-1].Close > 0 {
				ret := (float64(prices[i].Close) - float64(prices[i-1].Close)) / float64(prices[i-1].Close)
				returns = append(returns, ret)
			}
		}

		if len(returns) > 0 {
			result[code] = returns
		}
	}

	return result, nil
}

