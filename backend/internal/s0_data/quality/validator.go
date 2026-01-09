package quality

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// QualityGate validates data quality and generates snapshots
type QualityGate struct {
	db     *pgxpool.Pool
	config Config
}

// Config holds quality gate thresholds
type Config struct {
	MinPriceCoverage     float64 `yaml:"min_price_coverage"`      // 1.0 (100%)
	MinVolumeCoverage    float64 `yaml:"min_volume_coverage"`     // 1.0 (100%)
	MinMarketCapCoverage float64 `yaml:"min_market_cap_coverage"` // 0.95
	MinFinancialCoverage float64 `yaml:"min_financial_coverage"`  // 0.80
	MinInvestorCoverage  float64 `yaml:"min_investor_coverage"`   // 0.80
	MinDisclosureCoverage float64 `yaml:"min_disclosure_coverage"` // 0.70
}

// NewQualityGate creates a new QualityGate instance
func NewQualityGate(db *pgxpool.Pool, config Config) *QualityGate {
	return &QualityGate{
		db:     db,
		config: config,
	}
}

// Check validates data quality for a given date
// ⭐ SSOT: S0 → S1 품질 검증
func (g *QualityGate) Check(ctx context.Context, date time.Time) (*contracts.DataQualitySnapshot, error) {
	snapshot := &contracts.DataQualitySnapshot{
		Date:     date,
		Coverage: make(map[string]float64),
	}

	// 1. 전체 종목 수
	totalStocks, err := g.countTotalStocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("count total stocks: %w", err)
	}
	snapshot.TotalStocks = totalStocks

	// 2. 커버리지 체크
	coverage, err := g.checkCoverage(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("check coverage: %w", err)
	}
	snapshot.Coverage = coverage

	// 3. 품질 점수 계산
	snapshot.QualityScore = g.calculateScore(coverage)
	snapshot.ValidStocks = int(float64(totalStocks) * snapshot.QualityScore)

	return snapshot, nil
}

// countTotalStocks returns the number of active stocks
func (g *QualityGate) countTotalStocks(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM data.stocks WHERE status = 'active'`

	err := g.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("query total stocks: %w", err)
	}

	return count, nil
}

// checkCoverage calculates coverage for each data type
func (g *QualityGate) checkCoverage(ctx context.Context, date time.Time) (map[string]float64, error) {
	coverage := make(map[string]float64)

	// Price coverage (필수)
	priceCov, err := g.checkPriceCoverage(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("check price coverage: %w", err)
	}
	coverage["price"] = priceCov

	// Volume coverage (필수)
	volumeCov, err := g.checkVolumeCoverage(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("check volume coverage: %w", err)
	}
	coverage["volume"] = volumeCov

	// Market cap coverage
	marketCapCov, err := g.checkMarketCapCoverage(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("check market cap coverage: %w", err)
	}
	coverage["market_cap"] = marketCapCov

	// Fundamentals coverage
	fundamentalsCov, err := g.checkFundamentalsCoverage(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("check fundamentals coverage: %w", err)
	}
	coverage["fundamentals"] = fundamentalsCov

	// Investor flow coverage
	investorCov, err := g.checkInvestorCoverage(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("check investor coverage: %w", err)
	}
	coverage["investor"] = investorCov

	return coverage, nil
}

// checkPriceCoverage calculates price data coverage
func (g *QualityGate) checkPriceCoverage(ctx context.Context, date time.Time) (float64, error) {
	query := `
		SELECT
			COUNT(DISTINCT dp.stock_code)::FLOAT / COUNT(DISTINCT s.code) as coverage
		FROM data.stocks s
		LEFT JOIN data.daily_prices dp ON s.code = dp.stock_code AND dp.trade_date = $1
		WHERE s.status = 'active'
	`

	var coverage float64
	err := g.db.QueryRow(ctx, query, date).Scan(&coverage)
	if err != nil {
		return 0, fmt.Errorf("query price coverage: %w", err)
	}

	return coverage, nil
}

// checkVolumeCoverage calculates volume data coverage
func (g *QualityGate) checkVolumeCoverage(ctx context.Context, date time.Time) (float64, error) {
	query := `
		SELECT
			COUNT(DISTINCT dp.stock_code)::FLOAT / COUNT(DISTINCT s.code) as coverage
		FROM data.stocks s
		LEFT JOIN data.daily_prices dp ON s.code = dp.stock_code
			AND dp.trade_date = $1
			AND dp.volume IS NOT NULL
			AND dp.volume > 0
		WHERE s.status = 'active'
	`

	var coverage float64
	err := g.db.QueryRow(ctx, query, date).Scan(&coverage)
	if err != nil {
		return 0, fmt.Errorf("query volume coverage: %w", err)
	}

	return coverage, nil
}

// checkMarketCapCoverage calculates market cap data coverage
func (g *QualityGate) checkMarketCapCoverage(ctx context.Context, date time.Time) (float64, error) {
	query := `
		SELECT
			COALESCE(COUNT(DISTINCT mc.stock_code)::FLOAT / NULLIF(COUNT(DISTINCT s.code), 0), 0) as coverage
		FROM data.stocks s
		LEFT JOIN data.market_cap mc ON s.code = mc.stock_code AND mc.trade_date = $1
		WHERE s.status = 'active'
	`

	var coverage float64
	err := g.db.QueryRow(ctx, query, date).Scan(&coverage)
	if err != nil {
		return 0, fmt.Errorf("query market cap coverage: %w", err)
	}

	return coverage, nil
}

// checkFundamentalsCoverage calculates fundamentals data coverage
func (g *QualityGate) checkFundamentalsCoverage(ctx context.Context, date time.Time) (float64, error) {
	// 최근 90일 이내 재무 데이터가 있는 종목 비율
	query := `
		SELECT
			COALESCE(COUNT(DISTINCT f.stock_code)::FLOAT / NULLIF(COUNT(DISTINCT s.code), 0), 0) as coverage
		FROM data.stocks s
		LEFT JOIN data.fundamentals f ON s.code = f.stock_code
			AND f.report_date >= ($1::date - INTERVAL '90 days')
		WHERE s.status = 'active'
	`

	var coverage float64
	err := g.db.QueryRow(ctx, query, date).Scan(&coverage)
	if err != nil {
		return 0, fmt.Errorf("query fundamentals coverage: %w", err)
	}

	return coverage, nil
}

// checkInvestorCoverage calculates investor flow data coverage
func (g *QualityGate) checkInvestorCoverage(ctx context.Context, date time.Time) (float64, error) {
	query := `
		SELECT
			COALESCE(COUNT(DISTINCT ifl.stock_code)::FLOAT / NULLIF(COUNT(DISTINCT s.code), 0), 0) as coverage
		FROM data.stocks s
		LEFT JOIN data.investor_flow ifl ON s.code = ifl.stock_code AND ifl.trade_date = $1
		WHERE s.status = 'active'
	`

	var coverage float64
	err := g.db.QueryRow(ctx, query, date).Scan(&coverage)
	if err != nil {
		return 0, fmt.Errorf("query investor coverage: %w", err)
	}

	return coverage, nil
}

// calculateScore calculates overall quality score using weighted average
func (g *QualityGate) calculateScore(coverage map[string]float64) float64 {
	// 가중치 (합계 = 1.0)
	weights := map[string]float64{
		"price":        0.30, // 가격 데이터 필수
		"volume":       0.30, // 거래량 데이터 필수
		"market_cap":   0.15, // 시가총액
		"fundamentals": 0.15, // 재무제표
		"investor":     0.10, // 수급
	}

	score := 0.0
	for key, weight := range weights {
		if cov, exists := coverage[key]; exists {
			score += cov * weight
		}
	}

	return score
}
