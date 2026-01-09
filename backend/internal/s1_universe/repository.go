package s1_universe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// Repository handles data persistence for S1
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new Repository instance
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// SaveUniverse saves a universe snapshot to the database
func (r *Repository) SaveUniverse(ctx context.Context, universe *contracts.Universe) error {
	excludedJSON, err := json.Marshal(universe.Excluded)
	if err != nil {
		return fmt.Errorf("marshal excluded: %w", err)
	}

	query := `
		INSERT INTO data.universe_snapshots (
			snapshot_date,
			eligible_stocks,
			total_count,
			criteria,
			created_at
		) VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (snapshot_date) DO UPDATE SET
			eligible_stocks = EXCLUDED.eligible_stocks,
			total_count = EXCLUDED.total_count,
			criteria = EXCLUDED.criteria,
			created_at = NOW()
	`

	_, err = r.db.Exec(ctx, query,
		universe.Date,
		universe.Stocks,
		universe.TotalCount,
		excludedJSON,
	)
	if err != nil {
		return fmt.Errorf("insert universe: %w", err)
	}

	return nil
}

// GetLatestUniverse retrieves the most recent universe snapshot
func (r *Repository) GetLatestUniverse(ctx context.Context) (*contracts.Universe, error) {
	query := `
		SELECT
			snapshot_date,
			eligible_stocks,
			total_count,
			criteria
		FROM data.universe_snapshots
		ORDER BY snapshot_date DESC
		LIMIT 1
	`

	universe := &contracts.Universe{
		Excluded: make(map[string]string),
	}

	var excludedJSON []byte
	err := r.db.QueryRow(ctx, query).Scan(
		&universe.Date,
		&universe.Stocks,
		&universe.TotalCount,
		&excludedJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("query latest universe: %w", err)
	}

	if len(excludedJSON) > 0 {
		if err := json.Unmarshal(excludedJSON, &universe.Excluded); err != nil {
			return nil, fmt.Errorf("unmarshal excluded: %w", err)
		}
	}

	return universe, nil
}
