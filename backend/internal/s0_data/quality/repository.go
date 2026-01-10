package quality

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// Repository handles data quality snapshot persistence
// ⭐ SSOT: S0 품질 스냅샷 저장/조회
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new quality repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SaveSnapshot saves a data quality snapshot
func (r *Repository) SaveSnapshot(ctx context.Context, snapshot *contracts.DataQualitySnapshot) error {
	query := `
		INSERT INTO audit.data_quality_snapshots (
			snapshot_date, quality_score, total_stocks, valid_stocks,
			price_coverage, volume_coverage, marketcap_coverage,
			fundamentals_coverage, investor_coverage, passed
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (snapshot_date) DO UPDATE SET
			quality_score = EXCLUDED.quality_score,
			total_stocks = EXCLUDED.total_stocks,
			valid_stocks = EXCLUDED.valid_stocks,
			price_coverage = EXCLUDED.price_coverage,
			volume_coverage = EXCLUDED.volume_coverage,
			marketcap_coverage = EXCLUDED.marketcap_coverage,
			fundamentals_coverage = EXCLUDED.fundamentals_coverage,
			investor_coverage = EXCLUDED.investor_coverage,
			passed = EXCLUDED.passed,
			updated_at = NOW()
	`

	_, err := r.pool.Exec(ctx, query,
		snapshot.Date,
		snapshot.QualityScore,
		snapshot.TotalStocks,
		snapshot.ValidStocks,
		snapshot.Coverage["price"],
		snapshot.Coverage["volume"],
		snapshot.Coverage["market_cap"],
		snapshot.Coverage["fundamentals"],
		snapshot.Coverage["investor"],
		snapshot.Passed,
	)

	if err != nil {
		return fmt.Errorf("save quality snapshot: %w", err)
	}

	return nil
}

// GetByDate retrieves a quality snapshot by date
func (r *Repository) GetByDate(ctx context.Context, date time.Time) (*contracts.DataQualitySnapshot, error) {
	query := `
		SELECT
			snapshot_date, quality_score, total_stocks, valid_stocks,
			price_coverage, volume_coverage, marketcap_coverage,
			fundamentals_coverage, investor_coverage, passed
		FROM audit.data_quality_snapshots
		WHERE snapshot_date = $1
	`

	snapshot := &contracts.DataQualitySnapshot{
		Coverage: make(map[string]float64),
	}

	var priceCov, volumeCov, marketCapCov, fundamentalsCov, investorCov float64

	err := r.pool.QueryRow(ctx, query, date).Scan(
		&snapshot.Date,
		&snapshot.QualityScore,
		&snapshot.TotalStocks,
		&snapshot.ValidStocks,
		&priceCov,
		&volumeCov,
		&marketCapCov,
		&fundamentalsCov,
		&investorCov,
		&snapshot.Passed,
	)

	if err != nil {
		return nil, fmt.Errorf("get quality snapshot: %w", err)
	}

	snapshot.Coverage["price"] = priceCov
	snapshot.Coverage["volume"] = volumeCov
	snapshot.Coverage["market_cap"] = marketCapCov
	snapshot.Coverage["fundamentals"] = fundamentalsCov
	snapshot.Coverage["investor"] = investorCov

	return snapshot, nil
}

// GetLatest retrieves the most recent quality snapshot
func (r *Repository) GetLatest(ctx context.Context) (*contracts.DataQualitySnapshot, error) {
	query := `
		SELECT
			snapshot_date, quality_score, total_stocks, valid_stocks,
			price_coverage, volume_coverage, marketcap_coverage,
			fundamentals_coverage, investor_coverage, passed
		FROM audit.data_quality_snapshots
		ORDER BY snapshot_date DESC
		LIMIT 1
	`

	snapshot := &contracts.DataQualitySnapshot{
		Coverage: make(map[string]float64),
	}

	var priceCov, volumeCov, marketCapCov, fundamentalsCov, investorCov float64

	err := r.pool.QueryRow(ctx, query).Scan(
		&snapshot.Date,
		&snapshot.QualityScore,
		&snapshot.TotalStocks,
		&snapshot.ValidStocks,
		&priceCov,
		&volumeCov,
		&marketCapCov,
		&fundamentalsCov,
		&investorCov,
		&snapshot.Passed,
	)

	if err != nil {
		return nil, fmt.Errorf("get latest quality snapshot: %w", err)
	}

	snapshot.Coverage["price"] = priceCov
	snapshot.Coverage["volume"] = volumeCov
	snapshot.Coverage["market_cap"] = marketCapCov
	snapshot.Coverage["fundamentals"] = fundamentalsCov
	snapshot.Coverage["investor"] = investorCov

	return snapshot, nil
}
