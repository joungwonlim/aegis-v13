package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/wonny/aegis/v13/backend/internal/audit"
	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/internal/risk"
	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "S7 Audit - 리스크 분석 및 성과 감사",
	Long: `S7 Audit 모듈은 리스크 분석과 성과 감사를 담당합니다.

명령어:
  montecarlo   Monte Carlo 시뮬레이션 실행
  risk-report  리스크 리포트 생성`,
}

var (
	// montecarlo 플래그
	mcSimulations  int
	mcHoldingDays  int
	mcLookbackDays int
	mcMethod       string
	mcSeed         int64
	mcOutput       string
	mcDemo         bool // 데모 모드 (샘플 포트폴리오 사용)

	// risk-report 플래그
	reportDate   string
	reportOutput string
	reportDemo   bool // 데모 모드
)

var auditMonteCarloCmd = &cobra.Command{
	Use:   "montecarlo",
	Short: "Monte Carlo 시뮬레이션 실행",
	Long: `과거 포트폴리오 수익률을 기반으로 Monte Carlo 시뮬레이션을 실행합니다.

시뮬레이션 방법:
- historical: Historical Bootstrap (기본)
- normal: 정규분포 가정 (Parametric)
- t: t-분포 가정 (Fat Tail)

출력:
- VaR (Value at Risk): 지정 신뢰수준에서 최대 손실
- CVaR (Expected Shortfall): VaR 이상 손실의 평균
- 수익률 분포 백분위수

Example:
  go run ./cmd/quant audit montecarlo
  go run ./cmd/quant audit montecarlo --simulations 50000 --holding 5
  go run ./cmd/quant audit montecarlo --method t --seed 42
  go run ./cmd/quant audit montecarlo --output json`,
	RunE: runAuditMonteCarlo,
}

var auditRiskReportCmd = &cobra.Command{
	Use:   "risk-report",
	Short: "리스크 리포트 생성",
	Long: `포트폴리오 리스크 리포트를 생성합니다.

리포트 내용:
- 포트폴리오 VaR/CVaR
- Monte Carlo 시뮬레이션 결과
- 스트레스 테스트 결과

Example:
  go run ./cmd/quant audit risk-report
  go run ./cmd/quant audit risk-report --date 2024-01-15
  go run ./cmd/quant audit risk-report --output json`,
	RunE: runAuditRiskReport,
}

func init() {
	rootCmd.AddCommand(auditCmd)
	auditCmd.AddCommand(auditMonteCarloCmd)
	auditCmd.AddCommand(auditRiskReportCmd)

	// montecarlo 플래그
	auditMonteCarloCmd.Flags().IntVar(&mcSimulations, "simulations", 10000, "시뮬레이션 횟수")
	auditMonteCarloCmd.Flags().IntVar(&mcHoldingDays, "holding", 5, "보유 기간 (일)")
	auditMonteCarloCmd.Flags().IntVar(&mcLookbackDays, "lookback", 200, "과거 데이터 조회 기간 (일)")
	auditMonteCarloCmd.Flags().StringVar(&mcMethod, "method", "historical", "시뮬레이션 방법 (historical, normal, t)")
	auditMonteCarloCmd.Flags().Int64Var(&mcSeed, "seed", 0, "재현성용 시드 (0=랜덤)")
	auditMonteCarloCmd.Flags().StringVar(&mcOutput, "output", "text", "출력 형식 (text, json)")
	auditMonteCarloCmd.Flags().BoolVar(&mcDemo, "demo", false, "데모 모드 (샘플 포트폴리오 사용)")

	// risk-report 플래그
	auditRiskReportCmd.Flags().StringVar(&reportDate, "date", "", "리포트 날짜 (YYYY-MM-DD, 기본: 오늘)")
	auditRiskReportCmd.Flags().StringVar(&reportOutput, "output", "text", "출력 형식 (text, json)")
	auditRiskReportCmd.Flags().BoolVar(&reportDemo, "demo", false, "데모 모드 (샘플 포트폴리오 사용)")
}

func runAuditMonteCarlo(cmd *cobra.Command, args []string) error {
	fmt.Println("=== S7 Audit: Monte Carlo Simulation ===")

	ctx := cmd.Context()

	// 의존성 초기화
	cfg, log, db, err := initAuditDeps()
	if err != nil {
		return err
	}
	defer db.Close()
	_ = cfg

	// Monte Carlo 설정
	mcConfig := risk.MonteCarloConfig{
		NumSimulations:   mcSimulations,
		HoldingPeriod:    mcHoldingDays,
		ConfidenceLevels: []float64{0.95, 0.99},
		Method:           mcMethod, // "historical" or "parametric"
		LookbackDays:     mcLookbackDays,
		Seed:             mcSeed,
	}

	fmt.Printf("\n📊 Configuration:\n")
	fmt.Printf("  Simulations: %d\n", mcConfig.NumSimulations)
	fmt.Printf("  Holding Period: %d days\n", mcConfig.HoldingPeriod)
	fmt.Printf("  Lookback: %d days\n", mcConfig.LookbackDays)
	fmt.Printf("  Method: %s\n", mcConfig.Method)
	if mcConfig.Seed != 0 {
		fmt.Printf("  Seed: %d\n", mcConfig.Seed)
	}
	fmt.Println()

	// 포트폴리오 수익률 조회 (S7에서 데이터 조립)
	auditRepo := audit.NewRepository(db.Pool)
	priceRepo := s0_data.NewPriceRepository(db.Pool)

	toDate := time.Now()
	var weights map[string]float64
	minSamples := 30

	if mcDemo {
		// 데모 모드: 샘플 포트폴리오 사용 (대형주 동일가중)
		fmt.Println("🧪 Demo Mode: Using sample portfolio")
		weights = map[string]float64{
			"005930": 0.10, // 삼성전자
			"000660": 0.10, // SK하이닉스
			"035420": 0.10, // NAVER
			"035720": 0.10, // 카카오
			"051910": 0.10, // LG화학
			"006400": 0.10, // 삼성SDI
			"005380": 0.10, // 현대차
			"000270": 0.10, // 기아
			"068270": 0.10, // 셀트리온
			"105560": 0.10, // KB금융
		}
		fmt.Printf("📈 Sample Portfolio: %d stocks (equal weight)\n", len(weights))
	} else {
		// 실제 모드: 최근 스냅샷에서 포지션 조회
		fromDate := toDate.AddDate(0, 0, -7) // 최근 7일 스냅샷
		snapshots, err := auditRepo.GetSnapshotHistory(ctx, fromDate, toDate)
		if err != nil || len(snapshots) == 0 {
			fmt.Println("\n💡 Tip: 포트폴리오 스냅샷이 없습니다. --demo 플래그로 샘플 포트폴리오를 사용하세요.")
			fmt.Println("   예: go run ./cmd/quant audit montecarlo --demo")
			return fmt.Errorf("no portfolio snapshots found (use --demo for sample portfolio)")
		}

		snapshot := snapshots[len(snapshots)-1] // 가장 최근 스냅샷
		fmt.Printf("📅 Using portfolio from: %s\n", snapshot.Date.Format("2006-01-02"))
		fmt.Printf("💰 Portfolio Value: %s원\n\n", formatNumber(int64(snapshot.TotalValue)))

		// 종목별 비중 계산
		weights = make(map[string]float64)
		for _, pos := range snapshot.Positions {
			weights[pos.Code] = pos.Weight
		}

		if len(weights) == 0 {
			fmt.Println("⚠️ No positions in portfolio")
			return nil
		}

		fmt.Printf("📈 Positions: %d stocks\n", len(weights))
	}

	// 종목별 과거 수익률 조회
	lookbackFrom := toDate.AddDate(0, 0, -mcLookbackDays)

	assetReturns := make(map[string][]float64)
	for code := range weights {
		prices, err := priceRepo.GetByCodeAndDateRange(ctx, code, lookbackFrom, toDate)
		if err != nil || len(prices) < minSamples {
			continue
		}
		// 가격에서 수익률 계산
		returns := calculateReturnsFromPrices(prices)
		if len(returns) >= minSamples {
			assetReturns[code] = returns
		}
	}

	if len(assetReturns) == 0 {
		return fmt.Errorf("insufficient return data for simulation")
	}

	// 포트폴리오 수익률 계산
	portfolioReturns := risk.CalculatePortfolioReturns(weights, assetReturns)
	if len(portfolioReturns) < minSamples {
		return fmt.Errorf("insufficient portfolio returns: got %d, need %d",
			len(portfolioReturns), minSamples)
	}

	fmt.Printf("📊 Portfolio Returns: %d days\n\n", len(portfolioReturns))

	// Monte Carlo 실행 (새 API 사용)
	fmt.Println("🎲 Running Monte Carlo simulation...")
	engine := risk.NewEngine(risk.DefaultRiskLimits(), mcConfig, log)

	result, err := engine.SimulateSimple(ctx, portfolioReturns)
	if err != nil {
		return fmt.Errorf("monte carlo failed: %w", err)
	}

	// 결과 출력
	if mcOutput == "json" {
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		printMonteCarloResult(result, len(portfolioReturns))
	}

	return nil
}

// calculateReturnsFromPrices 가격 데이터에서 수익률 계산
func calculateReturnsFromPrices(prices []*contracts.Price) []float64 {
	if len(prices) < 2 {
		return nil
	}

	returns := make([]float64, 0, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1].Close > 0 {
			prevClose := float64(prices[i-1].Close)
			currClose := float64(prices[i].Close)
			ret := (currClose - prevClose) / prevClose
			returns = append(returns, ret)
		}
	}
	return returns
}

func printMonteCarloResult(result *risk.MonteCarloResult, inputSamples int) {
	fmt.Println("\n=== Monte Carlo Results ===")
	fmt.Printf("Run ID: %s\n", result.RunID)
	fmt.Printf("Run Date: %s\n", result.RunDate.Format("2006-01-02 15:04:05"))
	fmt.Printf("Input Samples: %d\n\n", inputSamples)

	fmt.Println("📊 Distribution")
	fmt.Printf("  Mean Return: %+.4f (%+.2f%%)\n", result.MeanReturn, result.MeanReturn*100)
	fmt.Printf("  Std Dev: %.4f (%.2f%%)\n", result.StdDev, result.StdDev*100)

	fmt.Println("\n📉 Risk Metrics (Loss as Positive)")
	fmt.Printf("  VaR 95%%: %.4f (%.2f%%)\n", result.VaR95, result.VaR95*100)
	fmt.Printf("  VaR 99%%: %.4f (%.2f%%)\n", result.VaR99, result.VaR99*100)
	fmt.Printf("  CVaR 95%%: %.4f (%.2f%%)\n", result.CVaR95, result.CVaR95*100)
	fmt.Printf("  CVaR 99%%: %.4f (%.2f%%)\n", result.CVaR99, result.CVaR99*100)

	fmt.Println("\n📊 Percentiles")
	percentiles := []int{1, 5, 10, 25, 50, 75, 90, 95, 99}
	for _, p := range percentiles {
		if val, ok := result.Percentiles[p]; ok {
			fmt.Printf("  P%d: %+.4f\n", p, val)
		}
	}

	fmt.Println("\n💡 Interpretation")
	if result.VaR95 < 0.03 {
		fmt.Println("  ✅ Low risk portfolio (VaR95 < 3%)")
	} else if result.VaR95 < 0.05 {
		fmt.Println("  ⚠️ Moderate risk portfolio (VaR95 3-5%)")
	} else {
		fmt.Println("  ❌ High risk portfolio (VaR95 > 5%)")
	}

	fmt.Printf("\n✅ Simulation completed (seed: %d)\n", result.Config.Seed)
}

func runAuditRiskReport(cmd *cobra.Command, args []string) error {
	fmt.Println("=== S7 Audit: Risk Report ===")

	ctx := cmd.Context()

	// 날짜 파싱
	var targetDate time.Time
	var err error
	if reportDate != "" {
		targetDate, err = time.Parse("2006-01-02", reportDate)
		if err != nil {
			return fmt.Errorf("invalid date: %w", err)
		}
	} else {
		targetDate = time.Now()
	}

	fmt.Printf("📅 Report Date: %s\n\n", targetDate.Format("2006-01-02"))

	// 의존성 초기화
	cfg, log, db, err := initAuditDeps()
	if err != nil {
		return err
	}
	defer db.Close()
	_ = cfg

	// 리포터 초기화
	auditRepo := audit.NewRepository(db.Pool)
	priceRepo := s0_data.NewPriceRepository(db.Pool)
	engine := risk.NewEngine(risk.DefaultRiskLimits(), risk.DefaultMonteCarloConfig(), log)
	reporter := audit.NewRiskReporter(engine, auditRepo, log.Zerolog())

	// 포트폴리오 데이터 조립 (S7 책임)
	var weights map[string]float64

	if reportDemo {
		// 데모 모드: 샘플 포트폴리오 사용
		fmt.Println("🧪 Demo Mode: Using sample portfolio")
		weights = map[string]float64{
			"005930": 0.10, // 삼성전자
			"000660": 0.10, // SK하이닉스
			"035420": 0.10, // NAVER
			"035720": 0.10, // 카카오
			"051910": 0.10, // LG화학
			"006400": 0.10, // 삼성SDI
			"005380": 0.10, // 현대차
			"000270": 0.10, // 기아
			"068270": 0.10, // 셀트리온
			"105560": 0.10, // KB금융
		}
		fmt.Printf("📈 Sample Portfolio: %d stocks (equal weight)\n\n", len(weights))
	} else {
		// 실제 모드: 스냅샷에서 포지션 조회
		lookbackFrom := targetDate.AddDate(0, 0, -7)
		snapshots, err := auditRepo.GetSnapshotHistory(ctx, lookbackFrom, targetDate)
		if err != nil || len(snapshots) == 0 {
			fmt.Println("\n💡 Tip: 포트폴리오 스냅샷이 없습니다. --demo 플래그로 샘플 포트폴리오를 사용하세요.")
			fmt.Println("   예: go run ./cmd/quant audit risk-report --demo")
			return fmt.Errorf("no portfolio snapshots found (use --demo for sample portfolio)")
		}

		snapshot := snapshots[len(snapshots)-1]
		weights = make(map[string]float64)
		for _, pos := range snapshot.Positions {
			weights[pos.Code] = pos.Weight
		}
	}

	// 수익률 조회
	lookbackDays := 200
	toDate := time.Now()
	fromDate := toDate.AddDate(0, 0, -lookbackDays)

	assetReturns := make(map[string][]float64)
	for code := range weights {
		prices, err := priceRepo.GetByCodeAndDateRange(ctx, code, fromDate, toDate)
		if err != nil {
			continue
		}
		returns := calculateReturnsFromPrices(prices)
		assetReturns[code] = returns
	}

	portfolioReturns := risk.CalculatePortfolioReturns(weights, assetReturns)

	// Monte Carlo 설정
	mcConfig := risk.DefaultMonteCarloConfig()

	// 리포트 생성
	input := audit.RiskReportInput{
		RunID:            fmt.Sprintf("report_%s", targetDate.Format("20060102")),
		PortfolioReturns: portfolioReturns,
		Weights:          weights,
		MonteCarloConfig: &mcConfig,
	}

	report, err := reporter.GenerateReport(ctx, input)
	if err != nil {
		return fmt.Errorf("generate report failed: %w", err)
	}

	// 출력
	if reportOutput == "json" {
		jsonData, _ := report.ToJSON()
		fmt.Println(string(jsonData))
	} else {
		fmt.Println(report.ToSummary())
	}

	return nil
}

func initAuditDeps() (*config.Config, *logger.Logger, *database.DB, error) {
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
