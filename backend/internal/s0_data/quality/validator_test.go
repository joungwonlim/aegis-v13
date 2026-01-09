package quality

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQualityGate_Check(t *testing.T) {
	// Skip if running in CI without database
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup database connection
	connString := "postgres://aegis_v13:aegis_v13_won@localhost:5432/aegis_v13?sslmode=disable"
	db, err := pgxpool.New(context.Background(), connString)
	require.NoError(t, err, "database connection failed")
	defer db.Close()

	// Create QualityGate with test config
	config := Config{
		MinPriceCoverage:     1.0,
		MinVolumeCoverage:    1.0,
		MinMarketCapCoverage: 0.95,
		MinFinancialCoverage: 0.80,
		MinInvestorCoverage:  0.80,
		MinDisclosureCoverage: 0.70,
	}

	gate := NewQualityGate(db, config)

	// Test with a recent date (should have data)
	ctx := context.Background()
	date := time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC)

	snapshot, err := gate.Check(ctx, date)
	require.NoError(t, err, "quality check failed")

	// Assertions
	assert.NotNil(t, snapshot)
	assert.Equal(t, date, snapshot.Date)
	assert.Greater(t, snapshot.TotalStocks, 0, "should have stocks")
	assert.GreaterOrEqual(t, snapshot.ValidStocks, 0)
	assert.NotEmpty(t, snapshot.Coverage, "should have coverage data")
	assert.GreaterOrEqual(t, snapshot.QualityScore, 0.0)
	assert.LessOrEqual(t, snapshot.QualityScore, 1.0)

	// Check coverage keys
	assert.Contains(t, snapshot.Coverage, "price")
	assert.Contains(t, snapshot.Coverage, "volume")
	assert.Contains(t, snapshot.Coverage, "market_cap")
	assert.Contains(t, snapshot.Coverage, "fundamentals")
	assert.Contains(t, snapshot.Coverage, "investor")

	t.Logf("Quality Snapshot: date=%s, total=%d, valid=%d, score=%.4f",
		snapshot.Date.Format("2006-01-02"),
		snapshot.TotalStocks,
		snapshot.ValidStocks,
		snapshot.QualityScore,
	)

	t.Logf("Coverage: price=%.2f%%, volume=%.2f%%, market_cap=%.2f%%, fundamentals=%.2f%%, investor=%.2f%%",
		snapshot.Coverage["price"]*100,
		snapshot.Coverage["volume"]*100,
		snapshot.Coverage["market_cap"]*100,
		snapshot.Coverage["fundamentals"]*100,
		snapshot.Coverage["investor"]*100,
	)
}

func TestQualityGate_calculateScore(t *testing.T) {
	gate := &QualityGate{
		config: Config{},
	}

	tests := []struct {
		name     string
		coverage map[string]float64
		wantMin  float64
		wantMax  float64
	}{
		{
			name: "perfect coverage",
			coverage: map[string]float64{
				"price":        1.0,
				"volume":       1.0,
				"market_cap":   1.0,
				"fundamentals": 1.0,
				"investor":     1.0,
			},
			wantMin: 0.99,
			wantMax: 1.01,
		},
		{
			name: "good coverage",
			coverage: map[string]float64{
				"price":        0.95,
				"volume":       0.95,
				"market_cap":   0.90,
				"fundamentals": 0.85,
				"investor":     0.80,
			},
			wantMin: 0.85,
			wantMax: 0.95,
		},
		{
			name: "poor coverage",
			coverage: map[string]float64{
				"price":        0.60,
				"volume":       0.60,
				"market_cap":   0.50,
				"fundamentals": 0.40,
				"investor":     0.30,
			},
			wantMin: 0.45,
			wantMax: 0.55,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := gate.calculateScore(tt.coverage)
			assert.GreaterOrEqual(t, score, tt.wantMin)
			assert.LessOrEqual(t, score, tt.wantMax)
			t.Logf("Score: %.4f", score)
		})
	}
}
