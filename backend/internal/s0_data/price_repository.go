package s0_data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// PriceRepository implements contracts.PriceRepository
// ⭐ SSOT: 가격 데이터 저장소는 여기서만
type PriceRepository struct {
	pool *pgxpool.Pool
}

// NewPriceRepository creates a new price repository
func NewPriceRepository(pool *pgxpool.Pool) *PriceRepository {
	return &PriceRepository{pool: pool}
}

// GetByCodeAndDate retrieves price for a specific code and date
func (r *PriceRepository) GetByCodeAndDate(ctx context.Context, code string, date time.Time) (*contracts.Price, error) {
	query := `
		SELECT code, date, open_price, high_price, low_price, close_price, volume
		FROM market.prices
		WHERE code = $1 AND date = $2
	`

	var p contracts.Price
	err := r.pool.QueryRow(ctx, query, code, date).Scan(
		&p.Code, &p.Date, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetByCodeAndDateRange retrieves prices for a code within date range
func (r *PriceRepository) GetByCodeAndDateRange(ctx context.Context, code string, from, to time.Time) ([]*contracts.Price, error) {
	query := `
		SELECT code, date, open_price, high_price, low_price, close_price, volume
		FROM market.prices
		WHERE code = $1 AND date BETWEEN $2 AND $3
		ORDER BY date ASC
	`

	rows, err := r.pool.Query(ctx, query, code, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []*contracts.Price
	for rows.Next() {
		var p contracts.Price
		if err := rows.Scan(&p.Code, &p.Date, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume); err != nil {
			return nil, err
		}
		prices = append(prices, &p)
	}
	return prices, rows.Err()
}

// GetLatestByCode retrieves the most recent price for a code
func (r *PriceRepository) GetLatestByCode(ctx context.Context, code string) (*contracts.Price, error) {
	query := `
		SELECT code, date, open_price, high_price, low_price, close_price, volume
		FROM market.prices
		WHERE code = $1
		ORDER BY date DESC
		LIMIT 1
	`

	var p contracts.Price
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&p.Code, &p.Date, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Save saves a single price record
func (r *PriceRepository) Save(ctx context.Context, price *contracts.Price) error {
	query := `
		INSERT INTO market.prices (code, date, open_price, high_price, low_price, close_price, volume)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (code, date) DO UPDATE SET
			open_price = EXCLUDED.open_price,
			high_price = EXCLUDED.high_price,
			low_price = EXCLUDED.low_price,
			close_price = EXCLUDED.close_price,
			volume = EXCLUDED.volume
	`

	_, err := r.pool.Exec(ctx, query,
		price.Code, price.Date, price.Open, price.High, price.Low, price.Close, price.Volume,
	)
	return err
}

// SaveBatch saves multiple price records
func (r *PriceRepository) SaveBatch(ctx context.Context, prices []*contracts.Price) error {
	if len(prices) == 0 {
		return nil
	}

	// Use simple loop for now (batch optimization can be added later)
	for _, price := range prices {
		if err := r.Save(ctx, price); err != nil {
			return err
		}
	}
	return nil
}
