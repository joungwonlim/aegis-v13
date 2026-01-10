package s0_data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// DisclosureRepository implements contracts.DisclosureRepository
// ⭐ SSOT: 공시 데이터 저장소는 여기서만
type DisclosureRepository struct {
	pool *pgxpool.Pool
}

// NewDisclosureRepository creates a new disclosure repository
func NewDisclosureRepository(pool *pgxpool.Pool) *DisclosureRepository {
	return &DisclosureRepository{pool: pool}
}

// GetByCodeAndDateRange retrieves disclosures for a code within date range
func (r *DisclosureRepository) GetByCodeAndDateRange(ctx context.Context, code string, from, to time.Time) ([]*contracts.Disclosure, error) {
	query := `
		SELECT stock_code, disclosed_at, category, title, content
		FROM data.disclosures
		WHERE stock_code = $1 AND disclosed_at BETWEEN $2 AND $3
		ORDER BY disclosed_at DESC
	`

	rows, err := r.pool.Query(ctx, query, code, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var disclosures []*contracts.Disclosure
	for rows.Next() {
		var d contracts.Disclosure
		if err := rows.Scan(&d.Code, &d.Date, &d.Type, &d.Title, &d.Content); err != nil {
			return nil, err
		}
		disclosures = append(disclosures, &d)
	}
	return disclosures, rows.Err()
}

// GetLatestByCode retrieves the most recent disclosures for a code
func (r *DisclosureRepository) GetLatestByCode(ctx context.Context, code string, limit int) ([]*contracts.Disclosure, error) {
	query := `
		SELECT stock_code, disclosed_at, category, title, content
		FROM data.disclosures
		WHERE stock_code = $1
		ORDER BY disclosed_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, code, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var disclosures []*contracts.Disclosure
	for rows.Next() {
		var d contracts.Disclosure
		if err := rows.Scan(&d.Code, &d.Date, &d.Type, &d.Title, &d.Content); err != nil {
			return nil, err
		}
		disclosures = append(disclosures, &d)
	}
	return disclosures, rows.Err()
}

// Save saves a single disclosure record
func (r *DisclosureRepository) Save(ctx context.Context, disclosure *contracts.Disclosure) error {
	query := `
		INSERT INTO data.disclosures (stock_code, disclosed_at, category, title, content)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (stock_code, disclosed_at, title) DO UPDATE SET
			category = EXCLUDED.category,
			content = EXCLUDED.content
	`

	_, err := r.pool.Exec(ctx, query,
		disclosure.Code, disclosure.Date, disclosure.Type, disclosure.Title, disclosure.Content,
	)
	return err
}

// SaveBatch saves multiple disclosure records
func (r *DisclosureRepository) SaveBatch(ctx context.Context, disclosures []*contracts.Disclosure) error {
	if len(disclosures) == 0 {
		return nil
	}

	for _, disclosure := range disclosures {
		if err := r.Save(ctx, disclosure); err != nil {
			return err
		}
	}
	return nil
}
