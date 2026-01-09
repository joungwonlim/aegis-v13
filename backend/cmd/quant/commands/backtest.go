package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wonny/aegis/v13/backend/internal/backtest"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// backtestCmd represents the backtest command
var backtestCmd = &cobra.Command{
	Use:   "backtest",
	Short: "백테스팅 프레임워크",
	Long: `과거 데이터를 사용하여 파이프라인을 시뮬레이션합니다.

백테스팅은 다음을 검증합니다:
- 전략 수익률
- 리스크 지표 (Sharpe, Sortino, MDD)
- 승률 및 회전율
- 시그널 기여도

Example:
  go run ./cmd/quant backtest run --from 2023-01-01 --to 2023-12-31
  go run ./cmd/quant backtest run --capital 100000000 --rebalance 7`,
}

var (
	backtestRunCmd = &cobra.Command{
		Use:   "run",
		Short: "백테스트 실행",
		Long: `지정된 기간 동안 백테스트를 실행합니다.

Flags:
  --from        시작 날짜 (YYYY-MM-DD)
  --to          종료 날짜 (YYYY-MM-DD, 기본: 오늘)
  --capital     초기 자본 (기본: 1억원)
  --rebalance   리밸런싱 주기 (일, 기본: 7일)
  --commission  수수료율 (기본: 0.0015 = 0.15%)
  --slippage    슬리피지율 (기본: 0.001 = 0.1%)

Example:
  go run ./cmd/quant backtest run --from 2023-01-01 --to 2023-12-31
  go run ./cmd/quant backtest run --capital 100000000 --rebalance 7
  go run ./cmd/quant backtest run --from 2023-01-01 --commission 0.002`,
		RunE: runBacktest,
	}

	// Flags
	backtestFrom       string
	backtestTo         string
	backtestCapital    int64
	backtestRebalance  int
	backtestCommission float64
	backtestSlippage   float64
)

func init() {
	rootCmd.AddCommand(backtestCmd)
	backtestCmd.AddCommand(backtestRunCmd)

	// Flags
	backtestRunCmd.Flags().StringVar(&backtestFrom, "from", "", "시작 날짜 (YYYY-MM-DD, 필수)")
	backtestRunCmd.Flags().StringVar(&backtestTo, "to", "", "종료 날짜 (YYYY-MM-DD, 기본: 오늘)")
	backtestRunCmd.Flags().Int64Var(&backtestCapital, "capital", 100_000_000, "초기 자본 (원)")
	backtestRunCmd.Flags().IntVar(&backtestRebalance, "rebalance", 7, "리밸런싱 주기 (일)")
	backtestRunCmd.Flags().Float64Var(&backtestCommission, "commission", 0.0015, "수수료율")
	backtestRunCmd.Flags().Float64Var(&backtestSlippage, "slippage", 0.001, "슬리피지율")

	backtestRunCmd.MarkFlagRequired("from")
}

func runBacktest(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Aegis v13 Backtest Engine ===")

	// Parse dates
	startDate, err := time.Parse("2006-01-02", backtestFrom)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}

	var endDate time.Time
	if backtestTo != "" {
		endDate, err = time.Parse("2006-01-02", backtestTo)
		if err != nil {
			return fmt.Errorf("invalid end date: %w", err)
		}
	} else {
		endDate = time.Now()
	}

	fmt.Printf("\n📅 Period: %s ~ %s\n", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	fmt.Printf("💰 Initial Capital: %s원\n", formatNumber(backtestCapital))
	fmt.Printf("🔄 Rebalance: %d days\n", backtestRebalance)
	fmt.Printf("💸 Commission: %.2f%%\n", backtestCommission*100)
	fmt.Printf("📉 Slippage: %.2f%%\n\n", backtestSlippage*100)

	// Initialize dependencies
	engine, err := initBacktestEngine()
	if err != nil {
		return fmt.Errorf("init backtest engine: %w", err)
	}

	// Create backtest config
	backtestConfig := backtest.Config{
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: backtestCapital,
		RebalanceDays:  backtestRebalance,
		Commission:     backtestCommission,
		Slippage:       backtestSlippage,
	}

	fmt.Println("🚀 Starting backtest...")
	fmt.Println()

	// Run backtest
	result, err := engine.Run(cmd.Context(), backtestConfig)
	if err != nil {
		return fmt.Errorf("backtest failed: %w", err)
	}

	// Print results
	printBacktestResult(result)

	return nil
}

func initBacktestEngine() (*backtest.Engine, error) {
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

	// 4. Initialize orchestrator
	orchestrator, err := initOrchestrator()
	if err != nil {
		return nil, fmt.Errorf("init orchestrator: %w", err)
	}

	// 5. Create simulator
	simulator := backtest.NewSimulator(db.Pool, log)

	// 6. Create backtest engine
	engine := backtest.NewEngine(orchestrator, simulator, log)

	return engine, nil
}

func printBacktestResult(result *backtest.Result) {
	fmt.Println("\n✅ Backtest Completed")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()

	// Summary
	fmt.Println("📊 Summary")
	fmt.Printf("Period: %s ~ %s (%d days, %d trading days)\n",
		result.StartDate.Format("2006-01-02"),
		result.EndDate.Format("2006-01-02"),
		result.TotalDays,
		result.TradingDays)
	fmt.Printf("Rebalances: %d times\n", result.RebalanceCount)
	fmt.Printf("Duration: %.2f seconds\n", result.Duration.Seconds())
	fmt.Println()

	// Performance
	fmt.Println("💰 Performance")
	fmt.Printf("Initial Capital: %s원\n", formatNumber(result.InitialCapital))
	fmt.Printf("Final Capital:   %s원\n", formatNumber(result.FinalCapital))
	fmt.Printf("P&L:             %s원 (%+.2f%%)\n",
		formatNumber(result.FinalCapital-result.InitialCapital),
		result.TotalReturn*100)
	fmt.Println()

	fmt.Printf("Annual Return:   %+.2f%%\n", result.AnnualizedReturn*100)
	fmt.Printf("CAGR:            %+.2f%%\n", result.CAGR*100)
	fmt.Printf("Volatility:      %.2f%%\n", result.Volatility*100)
	fmt.Println()

	// Risk Metrics
	fmt.Println("📉 Risk Metrics")
	fmt.Printf("Sharpe Ratio:    %.2f", result.SharpeRatio)
	if result.SharpeRatio > 3.0 {
		fmt.Print(" 🌟 (Excellent)")
	} else if result.SharpeRatio > 2.0 {
		fmt.Print(" ✅ (Good)")
	} else if result.SharpeRatio > 1.0 {
		fmt.Print(" ⚠️  (Fair)")
	} else {
		fmt.Print(" ❌ (Poor)")
	}
	fmt.Println()

	fmt.Printf("Sortino Ratio:   %.2f\n", result.SortinoRatio)
	fmt.Printf("Max Drawdown:    %.2f%%", result.MaxDrawdown*100)
	if result.MaxDrawdown < 0.10 {
		fmt.Print(" 🌟 (Excellent)")
	} else if result.MaxDrawdown < 0.20 {
		fmt.Print(" ✅ (Good)")
	} else if result.MaxDrawdown < 0.30 {
		fmt.Print(" ⚠️  (Fair)")
	} else {
		fmt.Print(" ❌ (High)")
	}
	fmt.Println()
	fmt.Println()

	// Trading Metrics
	fmt.Println("💹 Trading Metrics")
	fmt.Printf("Total Trades:    %d\n", result.TotalTrades)
	fmt.Printf("Winning Trades:  %d (%.1f%%)\n", result.WinningTrades, result.WinRate*100)
	fmt.Printf("Losing Trades:   %d\n", result.LosingTrades)
	fmt.Printf("Total Commission: %s원\n", formatNumber(result.TotalCommission))
	fmt.Printf("Total Slippage:   %s원\n", formatNumber(result.TotalSlippage))
	fmt.Println()

	// Equity Curve (last 10 points)
	fmt.Println("📈 Equity Curve (Last 10 Days)")
	startIdx := len(result.EquityCurve) - 10
	if startIdx < 0 {
		startIdx = 0
	}
	for _, point := range result.EquityCurve[startIdx:] {
		fmt.Printf("%s: %s원 (%+.2f%%)\n",
			point.Date.Format("2006-01-02"),
			formatNumber(point.Equity),
			point.Return*100)
	}
	fmt.Println()

	// Recommendation
	fmt.Println("💡 Recommendation")
	if result.SharpeRatio > 2.0 && result.MaxDrawdown < 0.15 {
		fmt.Println("✅ Strong strategy - good risk-adjusted returns")
	} else if result.SharpeRatio > 1.5 && result.MaxDrawdown < 0.25 {
		fmt.Println("⚠️  Acceptable strategy - consider optimization")
	} else {
		fmt.Println("❌ Weak strategy - needs improvement")
	}
	fmt.Println()
}
