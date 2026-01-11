package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
)

// dataCheckCmd represents the data check command
var dataCheckCmd = &cobra.Command{
	Use:   "data-check",
	Short: "DB 데이터 상태 확인",
	Long: `데이터베이스의 각 테이블 데이터 상태를 확인합니다.

확인 항목:
- 종목 수
- 가격 데이터 (Momentum, Technical 시그널용)
- 투자자 수급 데이터 (Flow 시그널용)
- 재무 데이터 (Value, Quality 시그널용)
- 공시 데이터 (Event 시그널용)
- 시그널 데이터

Example:
  go run ./cmd/quant data-check`,
	RunE: runDataCheck,
}

func init() {
	rootCmd.AddCommand(dataCheckCmd)
}

func runDataCheck(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Aegis v13 Data Check ===\n")

	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Connect to database
	db, err := database.New(cfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	fmt.Println("📊 데이터베이스 상태 확인\n")

	// Check each data source
	checkStocks(ctx, db.Pool)
	checkPrices(ctx, db.Pool)
	checkInvestorFlow(ctx, db.Pool)
	checkFinancials(ctx, db.Pool)
	checkDisclosures(ctx, db.Pool)
	checkSignals(ctx, db.Pool)
	checkRanking(ctx, db.Pool)

	// Summary for S2 Signals
	fmt.Println("\n" + "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📈 S2 시그널 데이터 요구사항 충족 여부")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	checkSignalDataRequirements(ctx, db.Pool)

	return nil
}

func checkStocks(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("📋 종목 데이터 (data.stocks)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var total, kospi, kosdaq, active int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM data.stocks`).Scan(&total)
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM data.stocks WHERE market = 'KOSPI'`).Scan(&kospi)
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM data.stocks WHERE market = 'KOSDAQ'`).Scan(&kosdaq)
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM data.stocks WHERE status = 'active'`).Scan(&active)

	fmt.Printf("  전체: %d종목 (KOSPI: %d, KOSDAQ: %d)\n", total, kospi, kosdaq)
	fmt.Printf("  활성: %d종목\n", active)
	fmt.Println()
}

func checkPrices(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("💹 가격 데이터 (data.daily_prices) - Momentum, Technical 시그널")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var total int64
	var stockCount int
	var minDate, maxDate time.Time

	pool.QueryRow(ctx, `SELECT COUNT(*) FROM data.daily_prices`).Scan(&total)
	pool.QueryRow(ctx, `SELECT COUNT(DISTINCT stock_code) FROM data.daily_prices`).Scan(&stockCount)
	pool.QueryRow(ctx, `SELECT MIN(trade_date), MAX(trade_date) FROM data.daily_prices`).Scan(&minDate, &maxDate)

	fmt.Printf("  총 레코드: %d\n", total)
	fmt.Printf("  종목 수: %d\n", stockCount)
	if total > 0 {
		fmt.Printf("  기간: %s ~ %s\n", minDate.Format("2006-01-02"), maxDate.Format("2006-01-02"))

		// Check stocks with 60+ and 120+ records
		var count60, count120 int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM (
				SELECT stock_code, COUNT(*) as cnt
				FROM data.daily_prices
				GROUP BY stock_code
				HAVING COUNT(*) >= 60
			) t
		`).Scan(&count60)
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM (
				SELECT stock_code, COUNT(*) as cnt
				FROM data.daily_prices
				GROUP BY stock_code
				HAVING COUNT(*) >= 120
			) t
		`).Scan(&count120)

		fmt.Printf("  60일+ 데이터: %d종목 (Momentum 가능)\n", count60)
		fmt.Printf("  120일+ 데이터: %d종목 (Technical 가능)\n", count120)
	}
	fmt.Println()
}

func checkInvestorFlow(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("🏦 투자자 수급 데이터 (data.investor_flow) - Flow 시그널")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var total int64
	var stockCount int
	var minDate, maxDate time.Time

	pool.QueryRow(ctx, `SELECT COUNT(*) FROM data.investor_flow`).Scan(&total)
	pool.QueryRow(ctx, `SELECT COUNT(DISTINCT stock_code) FROM data.investor_flow`).Scan(&stockCount)

	fmt.Printf("  총 레코드: %d\n", total)
	fmt.Printf("  종목 수: %d\n", stockCount)

	if total > 0 {
		pool.QueryRow(ctx, `SELECT MIN(trade_date), MAX(trade_date) FROM data.investor_flow`).Scan(&minDate, &maxDate)
		fmt.Printf("  기간: %s ~ %s\n", minDate.Format("2006-01-02"), maxDate.Format("2006-01-02"))

		// Check stocks with 20+ records
		var count20 int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM (
				SELECT stock_code, COUNT(*) as cnt
				FROM data.investor_flow
				GROUP BY stock_code
				HAVING COUNT(*) >= 20
			) t
		`).Scan(&count20)
		fmt.Printf("  20일+ 데이터: %d종목 (Flow 시그널 가능)\n", count20)
	} else {
		fmt.Println("  ⚠️  데이터 없음 - Flow 시그널 계산 불가")
	}
	fmt.Println()
}

func checkFinancials(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("📊 재무 데이터 (data.fundamentals) - Value, Quality 시그널")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var total int64
	var stockCount int

	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM data.fundamentals`).Scan(&total)
	if err != nil {
		fmt.Println("  ❌ 테이블 없음 또는 오류")
		fmt.Println("  ⚠️  Value, Quality 시그널 계산 불가")
		fmt.Println()
		return
	}

	pool.QueryRow(ctx, `SELECT COUNT(DISTINCT stock_code) FROM data.fundamentals`).Scan(&stockCount)

	fmt.Printf("  총 레코드: %d\n", total)
	fmt.Printf("  종목 수: %d\n", stockCount)

	if total == 0 {
		fmt.Println("  ⚠️  데이터 없음 - Value, Quality 시그널 계산 불가")
	}
	fmt.Println()
}

func checkDisclosures(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("📝 공시 데이터 (data.disclosures) - Event 시그널")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var total int64
	var stockCount int
	var minDate, maxDate time.Time

	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM data.disclosures`).Scan(&total)
	if err != nil {
		fmt.Println("  ❌ 테이블 없음 또는 오류")
		fmt.Println("  ⚠️  Event 시그널 계산 불가")
		fmt.Println()
		return
	}

	pool.QueryRow(ctx, `SELECT COUNT(DISTINCT stock_code) FROM data.disclosures`).Scan(&stockCount)

	fmt.Printf("  총 레코드: %d\n", total)
	fmt.Printf("  종목 수: %d\n", stockCount)

	if total > 0 {
		pool.QueryRow(ctx, `SELECT MIN(disclosure_date), MAX(disclosure_date) FROM data.disclosures`).Scan(&minDate, &maxDate)
		fmt.Printf("  기간: %s ~ %s\n", minDate.Format("2006-01-02"), maxDate.Format("2006-01-02"))
	} else {
		fmt.Println("  ⚠️  데이터 없음 - Event 시그널 계산 불가")
	}
	fmt.Println()
}

func checkSignals(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("📈 시그널 데이터 (signals.factor_scores)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var total int64
	var stockCount int
	var latestDate time.Time

	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM signals.factor_scores`).Scan(&total)
	if err != nil {
		fmt.Println("  ❌ 테이블 없음 또는 오류")
		fmt.Println()
		return
	}

	pool.QueryRow(ctx, `SELECT COUNT(DISTINCT stock_code) FROM signals.factor_scores`).Scan(&stockCount)
	pool.QueryRow(ctx, `SELECT MAX(calc_date) FROM signals.factor_scores`).Scan(&latestDate)

	fmt.Printf("  총 레코드: %d\n", total)
	fmt.Printf("  종목 수: %d\n", stockCount)
	if total > 0 {
		fmt.Printf("  최신 날짜: %s\n", latestDate.Format("2006-01-02"))

		// Sample of factor values
		var avgMom, avgTech, avgVal, avgQual, avgFlow, avgEvt float64
		pool.QueryRow(ctx, `
			SELECT
				AVG(momentum), AVG(technical), AVG(value),
				AVG(quality), AVG(flow), AVG(event)
			FROM signals.factor_scores
			WHERE calc_date = (SELECT MAX(calc_date) FROM signals.factor_scores)
		`).Scan(&avgMom, &avgTech, &avgVal, &avgQual, &avgFlow, &avgEvt)

		fmt.Println("\n  평균 시그널 점수 (최신):")
		fmt.Printf("    Momentum:  %.4f\n", avgMom)
		fmt.Printf("    Technical: %.4f\n", avgTech)
		fmt.Printf("    Value:     %.4f\n", avgVal)
		fmt.Printf("    Quality:   %.4f\n", avgQual)
		fmt.Printf("    Flow:      %.4f\n", avgFlow)
		fmt.Printf("    Event:     %.4f\n", avgEvt)
	}
	fmt.Println()
}

func checkRanking(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("🏆 랭킹 데이터 (selection.ranking_results)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var total int64
	var latestDate time.Time

	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM selection.ranking_results`).Scan(&total)
	if err != nil {
		fmt.Println("  ❌ 테이블 없음 또는 오류")
		fmt.Println()
		return
	}

	fmt.Printf("  총 레코드: %d\n", total)

	if total > 0 {
		pool.QueryRow(ctx, `SELECT MAX(rank_date) FROM selection.ranking_results`).Scan(&latestDate)
		fmt.Printf("  최신 날짜: %s\n", latestDate.Format("2006-01-02"))

		// Top 5 ranked stocks
		rows, err := pool.Query(ctx, `
			SELECT r.stock_code, s.name, r.rank, r.total_score
			FROM selection.ranking_results r
			JOIN data.stocks s ON r.stock_code = s.code
			WHERE r.rank_date = (SELECT MAX(rank_date) FROM selection.ranking_results)
			ORDER BY r.rank
			LIMIT 5
		`)
		if err == nil {
			defer rows.Close()
			fmt.Println("\n  Top 5 종목:")
			for rows.Next() {
				var code, name string
				var rank int
				var score float64
				rows.Scan(&code, &name, &rank, &score)
				fmt.Printf("    %d. %s (%s) - %.4f\n", rank, name, code, score)
			}
		}
	}
	fmt.Println()
}

func checkSignalDataRequirements(ctx context.Context, pool *pgxpool.Pool) {
	// Check each signal's data requirement
	signals := []struct {
		name    string
		check   string
		minRows int
	}{
		{"Momentum", "SELECT COUNT(DISTINCT stock_code) FROM data.daily_prices GROUP BY stock_code HAVING COUNT(*) >= 60", 60},
		{"Technical", "SELECT COUNT(DISTINCT stock_code) FROM data.daily_prices GROUP BY stock_code HAVING COUNT(*) >= 120", 120},
		{"Value", "SELECT COUNT(*) FROM data.fundamentals WHERE per IS NOT NULL OR pbr IS NOT NULL", 0},
		{"Quality", "SELECT COUNT(*) FROM data.fundamentals WHERE roe IS NOT NULL OR debt_ratio IS NOT NULL", 0},
		{"Flow", "SELECT COUNT(DISTINCT stock_code) FROM data.investor_flow GROUP BY stock_code HAVING COUNT(*) >= 20", 20},
		{"Event", "SELECT COUNT(*) FROM data.disclosures", 0},
	}

	for _, sig := range signals {
		var count int
		err := pool.QueryRow(ctx, sig.check).Scan(&count)

		status := "❌ 데이터 없음"
		if err == nil && count > 0 {
			status = fmt.Sprintf("✅ %d건", count)
		}

		fmt.Printf("  %-12s: %s\n", sig.name, status)
	}
}
