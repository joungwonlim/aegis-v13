package s1_universe

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// SPAC 판별을 위한 정규식 패턴
var spacPattern = regexp.MustCompile(`(?i)(스팩|SPAC|스펙|\d+호$|제\d+호)`)

// Builder constructs the investable universe
type Builder struct {
	db     *pgxpool.Pool
	config Config
}

// Config holds universe filter criteria
type Config struct {
	MinMarketCap   int64    `yaml:"min_market_cap"`    // 최소 시가총액 (억원)
	MinVolume      int64    `yaml:"min_volume"`        // 최소 거래대금 (백만원)
	MinListingDays int      `yaml:"min_listing_days"`  // 최소 상장일수
	ExcludeAdmin   bool     `yaml:"exclude_admin"`     // 관리종목 제외
	ExcludeHalt    bool     `yaml:"exclude_halt"`      // 거래정지 제외
	ExcludeSPAC    bool     `yaml:"exclude_spac"`      // SPAC 제외
	ExcludeSectors []string `yaml:"exclude_sectors"`   // 제외 섹터
}

// Stock represents a stock with filter criteria
type Stock struct {
	Code         string
	Name         string
	Market       string
	Sector       string
	ListingDate  time.Time
	MarketCap    int64  // 시가총액 (원)
	AvgVolume    int64  // 평균 거래대금 (원)
	ListingDays  int    // 상장일수
	IsAdmin      bool   // 관리종목 여부
	IsHalted     bool   // 거래정지 여부
	IsSPAC       bool   // SPAC 여부
}

// NewBuilder creates a new Universe Builder
func NewBuilder(db *pgxpool.Pool, config Config) *Builder {
	return &Builder{
		db:     db,
		config: config,
	}
}

// Build constructs the investable universe based on quality snapshot
// ⭐ SSOT: S1 → S2 유니버스 생성
func (b *Builder) Build(ctx context.Context, snapshot *contracts.DataQualitySnapshot) (*contracts.Universe, error) {
	// Quality gate 통과 확인
	if !snapshot.IsValid() {
		return nil, fmt.Errorf("data quality gate not passed: score=%.2f", snapshot.QualityScore)
	}

	universe := &contracts.Universe{
		Date:     snapshot.Date,
		Stocks:   make([]string, 0),
		Excluded: make(map[string]string),
	}

	// 전체 종목 조회
	stocks, err := b.getAllStocks(ctx, snapshot.Date)
	if err != nil {
		return nil, fmt.Errorf("get all stocks: %w", err)
	}

	// 필터링
	for _, stock := range stocks {
		reason := b.checkExclusion(stock)
		if reason != "" {
			universe.Excluded[stock.Code] = reason
			continue
		}
		universe.Stocks = append(universe.Stocks, stock.Code)
	}

	universe.TotalCount = len(universe.Stocks)
	return universe, nil
}

// getAllStocks retrieves all active stocks with necessary data
func (b *Builder) getAllStocks(ctx context.Context, date time.Time) ([]Stock, error) {
	// Note: market_cap 데이터는 가장 최근 것을 사용 (시가총액은 매일 크게 변하지 않음)
	query := `
		SELECT
			s.code,
			s.name,
			s.market,
			COALESCE(s.sector, ''),
			s.listing_date,
			COALESCE(mc.market_cap, 0),
			COALESCE(avg_vol.avg_volume, 0),
			($1::date - s.listing_date) as listing_days
		FROM data.stocks s
		LEFT JOIN LATERAL (
			SELECT market_cap FROM data.market_cap
			WHERE stock_code = s.code
			ORDER BY trade_date DESC LIMIT 1
		) mc ON TRUE
		LEFT JOIN (
			SELECT
				stock_code,
				AVG(trading_value)::BIGINT as avg_volume
			FROM data.daily_prices
			WHERE trade_date BETWEEN ($1::date - INTERVAL '20 days') AND $1
			GROUP BY stock_code
		) avg_vol ON s.code = avg_vol.stock_code
		WHERE s.status = 'active'
		ORDER BY s.code
	`

	rows, err := b.db.Query(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("query stocks: %w", err)
	}
	defer rows.Close()

	stocks := make([]Stock, 0)
	for rows.Next() {
		var stock Stock
		err := rows.Scan(
			&stock.Code,
			&stock.Name,
			&stock.Market,
			&stock.Sector,
			&stock.ListingDate,
			&stock.MarketCap,
			&stock.AvgVolume,
			&stock.ListingDays,
		)
		if err != nil {
			return nil, fmt.Errorf("scan stock: %w", err)
		}

		// 상태 플래그 설정
		stock.IsSPAC = isSPAC(stock.Name)
		stock.IsAdmin = isAdminStock(stock.Name)
		stock.IsHalted = false // DB에서 별도 조회 필요 시 확장

		stocks = append(stocks, stock)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate stocks: %w", rows.Err())
	}

	return stocks, nil
}

// checkExclusion checks if a stock should be excluded and returns the reason
func (b *Builder) checkExclusion(stock Stock) string {
	// 우선순위 순서로 체크

	// 1. 거래정지
	if b.config.ExcludeHalt && stock.IsHalted {
		return "거래정지"
	}

	// 2. 관리종목
	if b.config.ExcludeAdmin && stock.IsAdmin {
		return "관리종목"
	}

	// 3. SPAC
	if b.config.ExcludeSPAC && stock.IsSPAC {
		return "SPAC"
	}

	// 4. 시가총액 미달
	minMarketCap := b.config.MinMarketCap * 100_000_000 // 억 → 원
	if stock.MarketCap < minMarketCap {
		return fmt.Sprintf("시가총액 미달 (%d억)", stock.MarketCap/100_000_000)
	}

	// 5. 거래대금 미달
	minVolume := b.config.MinVolume * 1_000_000 // 백만 → 원
	if stock.AvgVolume < minVolume {
		return fmt.Sprintf("거래대금 미달 (%d백만)", stock.AvgVolume/1_000_000)
	}

	// 6. 상장일수 미달
	if stock.ListingDays < b.config.MinListingDays {
		return fmt.Sprintf("상장일수 미달 (%d일)", stock.ListingDays)
	}

	// 7. 제외 섹터
	for _, sector := range b.config.ExcludeSectors {
		if stock.Sector == sector {
			return fmt.Sprintf("제외 섹터 (%s)", sector)
		}
	}

	return "" // 통과
}

// isSPAC checks if a stock is a SPAC based on name pattern
func isSPAC(name string) bool {
	return spacPattern.MatchString(name)
}

// isAdminStock checks if a stock is under administrative supervision
// 관리종목은 종목명에 특정 표식이 붙거나 DB에서 별도 관리
func isAdminStock(name string) bool {
	// 관리종목 패턴: "관리" 또는 "*" 표시 등
	adminPatterns := []string{"관리", "*"}
	for _, pattern := range adminPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}
	return false
}
