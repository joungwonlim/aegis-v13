package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/wonny/aegis/v13/backend/internal/external/dart"
	"github.com/wonny/aegis/v13/backend/internal/external/kis"
	"github.com/wonny/aegis/v13/backend/internal/external/krx"
	"github.com/wonny/aegis/v13/backend/internal/external/naver"
	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/internal/s0_data/collector"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// fetcherCmd represents the fetcher command
var fetcherCmd = &cobra.Command{
	Use:   "fetcher",
	Short: "데이터 수집 도구",
	Long: `외부 API (KIS, DART, Naver)에서 데이터를 수집합니다.

이 명령어는:
- KIS API에서 시세/체결 데이터 수집
- DART에서 공시 데이터 수집
- Naver Finance에서 보조 데이터 수집

Example:
  go run ./cmd/quant fetcher collect --source kis
  go run ./cmd/quant fetcher collect --source dart
  go run ./cmd/quant fetcher collect all`,
}

// fetcherCollectCmd represents the collect subcommand
var fetcherCollectCmd = &cobra.Command{
	Use:   "collect [source]",
	Short: "데이터 수집 실행",
	Long: `지정된 소스에서 데이터를 수집합니다.

소스:
  kis    - 한국투자증권 API (시세, 체결)
  dart   - DART 공시 데이터
  naver  - Naver Finance
  all    - 모든 소스

Example:
  go run ./cmd/quant fetcher collect kis
  go run ./cmd/quant fetcher collect dart
  go run ./cmd/quant fetcher collect all`,
	Args: cobra.ExactArgs(1),
	RunE: runFetcherCollect,
}

// fetcherMarketCapCmd represents the marketcap subcommand
var fetcherMarketCapCmd = &cobra.Command{
	Use:   "marketcap",
	Short: "시가총액/상장주식수 데이터 수집",
	Long: `모든 종목의 시가총액 및 상장주식수 데이터를 수집합니다.

KRX API에서 KOSPI/KOSDAQ 전 종목의 시가총액과 상장주식수를 수집하여
data.market_cap 테이블에 저장합니다.

Example:
  go run ./cmd/quant fetcher marketcap`,
	RunE: runFetcherMarketCap,
}

var (
	// Fetcher flags
	fetcherAsync bool
)

func init() {
	rootCmd.AddCommand(fetcherCmd)
	fetcherCmd.AddCommand(fetcherCollectCmd)
	fetcherCmd.AddCommand(fetcherMarketCapCmd)

	// Flags
	fetcherCollectCmd.Flags().BoolVar(&fetcherAsync, "async", false, "비동기 수집 (큐에 작업 추가)")
}

// runFetcherMarketCap collects market cap data from Naver Finance
func runFetcherMarketCap(cmd *cobra.Command, args []string) error {
	fmt.Printf("=== Aegis v13 Market Cap Fetcher ===\n\n")
	fmt.Println("💰 시가총액/상장주식수 데이터 수집 시작... (Naver Finance)")

	// Initialize collector
	col, ctx, err := initCollector()
	if err != nil {
		return fmt.Errorf("init collector: %w", err)
	}

	// Fetch market caps from Naver (includes shares outstanding)
	if err := col.FetchMarketCaps(ctx); err != nil {
		return fmt.Errorf("fetch market caps: %w", err)
	}

	fmt.Println("\n✅ 시가총액/상장주식수 데이터 수집 완료!")
	return nil
}

func runFetcherCollect(cmd *cobra.Command, args []string) error {
	source := args[0]

	fmt.Printf("=== Aegis v13 Data Fetcher ===\n\n")
	fmt.Printf("Source: %s\n", source)
	fmt.Printf("Mode: %s\n\n", getMode(fetcherAsync))

	switch source {
	case "kis":
		return collectKIS()
	case "dart":
		return collectDART()
	case "naver":
		return collectNaver()
	case "all":
		return collectAll()
	default:
		return fmt.Errorf("unknown source: %s (valid: kis, dart, naver, all)", source)
	}
}

// initCollector initializes all dependencies and returns a collector
func initCollector() (*collector.Collector, context.Context, error) {
	ctx := context.Background()

	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, ctx, fmt.Errorf("load config: %w", err)
	}

	// 2. Initialize logger
	log := logger.New(cfg)

	// 3. Connect to database
	db, err := database.New(cfg)
	if err != nil {
		return nil, ctx, fmt.Errorf("connect to database: %w", err)
	}

	// 4. Create HTTP client
	httpClient := httputil.New(cfg, log)

	// 5. Create external API clients
	naverClient := naver.NewClient(httpClient, log)
	dartClient := dart.NewClient(cfg.DART.APIKey, log)
	krxClient := krx.NewClient(httpClient, log)

	// 6. Create repository
	repo := s0_data.NewRepository(db.Pool)

	// 7. Create collector
	col := collector.NewCollector(naverClient, dartClient, krxClient, repo, log)

	return col, ctx, nil
}

func collectKIS() error {
	fmt.Println()
	PrintSeparator()
	fmt.Println("📊 KIS 데이터 수집 시작...")
	PrintSeparator()

	ctx := context.Background()

	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Initialize logger
	log := logger.New(cfg)

	// 3. Create HTTP client
	httpClient := httputil.New(cfg, log)

	// 4. Create KIS client (⭐ SSOT: KIS API는 이 클라이언트만 사용)
	kisClient := kis.NewClient(cfg.KIS, httpClient, log)

	// 5. Test with sample stocks
	testStocks := []string{"005930", "000660", "035720"} // 삼성전자, SK하이닉스, 카카오

	fmt.Printf("\n테스트 종목: %v\n\n", testStocks)

	for _, code := range testStocks {
		price, err := kisClient.GetCurrentPrice(ctx, code)
		if err != nil {
			fmt.Printf("❌ %s: %v\n", code, err)
			continue
		}
		fmt.Printf("✅ %s: 현재가 %.0f, 거래량 %d\n",
			code, price.ClosePrice, price.Volume)
	}

	fmt.Println("\n✅ KIS 데이터 수집 완료!")
	return nil
}

func collectDART() error {
	fmt.Println()
	PrintSeparator()
	fmt.Println("📄 DART 공시 데이터 수집 시작...")
	PrintSeparator()

	// Initialize collector
	col, ctx, err := initCollector()
	if err != nil {
		return fmt.Errorf("init collector: %w", err)
	}

	// Fetch disclosures (last 7 days)
	to := time.Now()
	from := to.AddDate(0, 0, -7)

	fmt.Printf("\n기간: %s ~ %s\n\n", from.Format("2006-01-02"), to.Format("2006-01-02"))

	if err := col.FetchDisclosures(ctx, from, to); err != nil {
		return fmt.Errorf("fetch disclosures: %w", err)
	}

	fmt.Println("\n✅ DART 공시 데이터 수집 완료!")
	return nil
}

func collectNaver() error {
	fmt.Println()
	PrintSeparator()
	fmt.Println("🔍 Naver Finance 데이터 수집 시작...")
	PrintSeparator()

	// Initialize collector
	col, ctx, err := initCollector()
	if err != nil {
		return fmt.Errorf("init collector: %w", err)
	}

	// Date range (last 30 days)
	to := time.Now()
	from := to.AddDate(0, 0, -30)

	fmt.Printf("\n기간: %s ~ %s\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
	fmt.Printf("작업자 수: 5\n\n")

	cfg := collector.Config{Workers: 5}

	// 1. Fetch prices
	fmt.Println("📊 가격 데이터 수집 중...")
	if _, err := col.FetchAllPrices(ctx, from, to, cfg); err != nil {
		return fmt.Errorf("fetch prices: %w", err)
	}

	// 2. Fetch investor flow
	fmt.Println("📈 투자자 수급 데이터 수집 중...")
	if _, err := col.FetchAllInvestorFlow(ctx, from, to, cfg); err != nil {
		return fmt.Errorf("fetch investor flow: %w", err)
	}

	// 3. Fetch market caps
	fmt.Println("💰 시가총액 데이터 수집 중...")
	if err := col.FetchMarketCaps(ctx); err != nil {
		return fmt.Errorf("fetch market caps: %w", err)
	}

	fmt.Println("\n✅ Naver Finance 데이터 수집 완료!")
	return nil
}

func collectAll() error {
	fmt.Println()
	PrintSeparator()
	fmt.Println("🚀 전체 소스 데이터 수집 시작...")
	PrintSeparator()

	// Initialize collector
	col, ctx, err := initCollector()
	if err != nil {
		return fmt.Errorf("init collector: %w", err)
	}

	// Date range (last 30 days)
	to := time.Now()
	from := to.AddDate(0, 0, -30)

	fmt.Printf("\n기간: %s ~ %s\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
	fmt.Printf("작업자 수: 5\n\n")

	cfg := collector.Config{Workers: 5}

	// 1. Naver Finance
	fmt.Println("📊 [1/4] 가격 데이터 수집 중...")
	if _, err := col.FetchAllPrices(ctx, from, to, cfg); err != nil {
		fmt.Printf("⚠️  가격 수집 실패: %v\n", err)
	}

	fmt.Println("📈 [2/4] 투자자 수급 데이터 수집 중...")
	if _, err := col.FetchAllInvestorFlow(ctx, from, to, cfg); err != nil {
		fmt.Printf("⚠️  수급 수집 실패: %v\n", err)
	}

	fmt.Println("💰 [3/4] 시가총액 데이터 수집 중...")
	if err := col.FetchMarketCaps(ctx); err != nil {
		fmt.Printf("⚠️  시가총액 수집 실패: %v\n", err)
	}

	// 2. KRX Market Trends
	fmt.Println("📉 [4/4] KRX 시장 지표 수집 중...")
	if err := col.FetchMarketTrends(ctx); err != nil {
		fmt.Printf("⚠️  시장 지표 수집 실패: %v\n", err)
	}

	// 3. DART Disclosures
	dartFrom := to.AddDate(0, 0, -7)
	fmt.Println("\n📄 [추가] DART 공시 데이터 수집 중...")
	fmt.Printf("   기간: %s ~ %s\n", dartFrom.Format("2006-01-02"), to.Format("2006-01-02"))
	if err := col.FetchDisclosures(ctx, dartFrom, to); err != nil {
		fmt.Printf("⚠️  공시 수집 실패: %v\n", err)
	}

	fmt.Println("\n✅ 전체 데이터 수집 완료!")
	return nil
}

func getMode(async bool) string {
	if async {
		return "Async (Queue)"
	}
	return "Sync (Direct)"
}
