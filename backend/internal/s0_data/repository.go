package s0_data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// Repository handles data persistence for S0
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new Repository instance
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// SaveQualitySnapshot saves a quality snapshot to the database
func (r *Repository) SaveQualitySnapshot(ctx context.Context, snapshot *contracts.DataQualitySnapshot) error {
	coverageJSON, err := json.Marshal(snapshot.Coverage)
	if err != nil {
		return fmt.Errorf("marshal coverage: %w", err)
	}

	query := `
		INSERT INTO data.quality_snapshots (
			snapshot_date,
			total_stocks,
			valid_stocks,
			coverage,
			quality_score,
			created_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (snapshot_date) DO UPDATE SET
			total_stocks = EXCLUDED.total_stocks,
			valid_stocks = EXCLUDED.valid_stocks,
			coverage = EXCLUDED.coverage,
			quality_score = EXCLUDED.quality_score,
			created_at = NOW()
	`

	_, err = r.db.Exec(ctx, query,
		snapshot.Date,
		snapshot.TotalStocks,
		snapshot.ValidStocks,
		coverageJSON,
		snapshot.QualityScore,
	)
	if err != nil {
		return fmt.Errorf("insert quality snapshot: %w", err)
	}

	return nil
}

// GetLatestQualitySnapshot retrieves the most recent quality snapshot
func (r *Repository) GetLatestQualitySnapshot(ctx context.Context) (*contracts.DataQualitySnapshot, error) {
	query := `
		SELECT
			snapshot_date,
			total_stocks,
			valid_stocks,
			coverage,
			quality_score
		FROM data.quality_snapshots
		ORDER BY snapshot_date DESC
		LIMIT 1
	`

	snapshot := &contracts.DataQualitySnapshot{
		Coverage: make(map[string]float64),
	}

	var coverageJSON []byte
	err := r.db.QueryRow(ctx, query).Scan(
		&snapshot.Date,
		&snapshot.TotalStocks,
		&snapshot.ValidStocks,
		&coverageJSON,
		&snapshot.QualityScore,
	)
	if err != nil {
		return nil, fmt.Errorf("query latest quality snapshot: %w", err)
	}

	if err := json.Unmarshal(coverageJSON, &snapshot.Coverage); err != nil {
		return nil, fmt.Errorf("unmarshal coverage: %w", err)
	}

	return snapshot, nil
}

// GetQualitySnapshotByDate retrieves a quality snapshot for a specific date
func (r *Repository) GetQualitySnapshotByDate(ctx context.Context, date time.Time) (*contracts.DataQualitySnapshot, error) {
	query := `
		SELECT
			snapshot_date,
			total_stocks,
			valid_stocks,
			coverage,
			quality_score
		FROM data.quality_snapshots
		WHERE snapshot_date = $1
	`

	snapshot := &contracts.DataQualitySnapshot{
		Coverage: make(map[string]float64),
	}

	var coverageJSON []byte
	err := r.db.QueryRow(ctx, query, date).Scan(
		&snapshot.Date,
		&snapshot.TotalStocks,
		&snapshot.ValidStocks,
		&coverageJSON,
		&snapshot.QualityScore,
	)
	if err != nil {
		return nil, fmt.Errorf("query quality snapshot by date: %w", err)
	}

	if err := json.Unmarshal(coverageJSON, &snapshot.Coverage); err != nil {
		return nil, fmt.Errorf("unmarshal coverage: %w", err)
	}

	return snapshot, nil
}
