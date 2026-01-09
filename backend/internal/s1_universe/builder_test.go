package s1_universe

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

func TestBuilder_Build(t *testing.T) {
	// Skip if running in CI without database
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup database connection
	connString := "postgres://aegis_v13:aegis_v13_won@localhost:5432/aegis_v13?sslmode=disable"
	db, err := pgxpool.New(context.Background(), connString)
	require.NoError(t, err, "database connection failed")
	defer db.Close()

	// Create Builder with test config (relaxed for testing without market_cap data)
	config := Config{
		MinMarketCap:   0,     // 0억 (시가총액 데이터 없음)
		MinVolume:      0,     // 0백만 (거래대금 제한 없음)
		MinListingDays: 30,    // 30일
		ExcludeAdmin:   false, // TODO 데이터 없음
		ExcludeHalt:    false, // TODO 데이터 없음
		ExcludeSPAC:    false, // TODO 데이터 없음
		ExcludeSectors: []string{},
	}

	builder := NewBuilder(db, config)

	// Create a quality snapshot
	snapshot := &contracts.DataQualitySnapshot{
		Date:         time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC),
		TotalStocks:  922,
		ValidStocks:  850,
		QualityScore: 0.92,
		Coverage: map[string]float64{
			"price":  0.95,
			"volume": 0.95,
		},
	}

	// Build universe
	ctx := context.Background()
	universe, err := builder.Build(ctx, snapshot)
	require.NoError(t, err, "universe build failed")

	// Assertions
	assert.NotNil(t, universe)
	assert.Equal(t, snapshot.Date, universe.Date)
	assert.Greater(t, len(universe.Stocks), 0, "should have investable stocks")
	assert.NotNil(t, universe.Excluded)
	assert.Equal(t, len(universe.Stocks), universe.TotalCount)

	t.Logf("Universe: date=%s, total=%d, excluded=%d",
		universe.Date.Format("2006-01-02"),
		universe.TotalCount,
		len(universe.Excluded),
	)

	// Sample excluded stocks
	excludedCount := 0
	for code, reason := range universe.Excluded {
		if excludedCount < 5 {
			t.Logf("Excluded: %s -> %s", code, reason)
			excludedCount++
		}
	}

	// Sample included stocks
	if len(universe.Stocks) > 0 {
		t.Logf("Sample stocks: %v", universe.Stocks[:min(5, len(universe.Stocks))])
	}
}

func TestBuilder_checkExclusion(t *testing.T) {
	builder := &Builder{
		config: Config{
			MinMarketCap:   1000, // 1000억
			MinVolume:      500,  // 5억
			MinListingDays: 90,
			ExcludeAdmin:   true,
			ExcludeHalt:    true,
			ExcludeSPAC:    true,
			ExcludeSectors: []string{"금융"},
		},
	}

	tests := []struct {
		name   string
		stock  Stock
		want   string
	}{
		{
			name: "valid stock",
			stock: Stock{
				Code:         "005930",
				Name:         "삼성전자",
				Market:       "KOSPI",
				Sector:       "전기전자",
				MarketCap:    500_000_000_000_000, // 500조
				AvgVolume:    1_000_000_000_000,   // 1조
				ListingDays:  10000,
				IsAdmin:      false,
				IsHalted:     false,
				IsSPAC:       false,
			},
			want: "",
		},
		{
			name: "halted stock",
			stock: Stock{
				Code:     "999999",
				IsHalted: true,
			},
			want: "거래정지",
		},
		{
			name: "admin stock",
			stock: Stock{
				Code:    "999998",
				IsAdmin: true,
			},
			want: "관리종목",
		},
		{
			name: "SPAC",
			stock: Stock{
				Code:   "999997",
				IsSPAC: true,
			},
			want: "SPAC",
		},
		{
			name: "low market cap",
			stock: Stock{
				Code:        "999996",
				MarketCap:   50_000_000_000, // 500억 (< 1000억)
				AvgVolume:   1_000_000_000,
				ListingDays: 100,
			},
			want: "시가총액 미달 (500억)",
		},
		{
			name: "low volume",
			stock: Stock{
				Code:        "999995",
				MarketCap:   200_000_000_000, // 2000억
				AvgVolume:   100_000_000,     // 1억 (< 5억)
				ListingDays: 100,
			},
			want: "거래대금 미달 (100백만)",
		},
		{
			name: "newly listed",
			stock: Stock{
				Code:        "999994",
				MarketCap:   200_000_000_000,
				AvgVolume:   1_000_000_000,
				ListingDays: 30, // < 90일
			},
			want: "상장일수 미달 (30일)",
		},
		{
			name: "excluded sector",
			stock: Stock{
				Code:        "999993",
				Sector:      "금융",
				MarketCap:   200_000_000_000,
				AvgVolume:   1_000_000_000,
				ListingDays: 100,
			},
			want: "제외 섹터 (금융)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.checkExclusion(tt.stock)
			assert.Equal(t, tt.want, got)
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
