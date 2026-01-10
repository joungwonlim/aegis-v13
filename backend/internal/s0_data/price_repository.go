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
		SELECT stock_code, trade_date, open_price, high_price, low_price, close_price, volume
		FROM data.daily_prices
		WHERE stock_code = $1 AND trade_date = $2
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
		SELECT stock_code, trade_date, open_price, high_price, low_price, close_price, volume
		FROM data.daily_prices
		WHERE stock_code = $1 AND trade_date BETWEEN $2 AND $3
		ORDER BY trade_date ASC
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
		SELECT stock_code, trade_date, open_price, high_price, low_price, close_price, volume
		FROM data.daily_prices
		WHERE stock_code = $1
		ORDER BY trade_date DESC
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
		INSERT INTO data.daily_prices (stock_code, trade_date, open_price, high_price, low_price, close_price, volume)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (stock_code, trade_date) DO UPDATE SET
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

// PriceWithMeta 가격 데이터 + 메타 정보 (forecast용)
type PriceWithMeta struct {
	Code      string
	Date      time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int64
	PrevClose float64
	Sector    string
	MarketCap int64
}

// GetDailyPrices 특정 날짜의 모든 가격 데이터 조회 (전일 종가, 섹터, 시가총액 포함)
func (r *PriceRepository) GetDailyPrices(ctx context.Context, date time.Time) ([]PriceWithMeta, error) {
	query := `
		WITH prev_prices AS (
			SELECT stock_code, close_price as prev_close
			FROM data.daily_prices
			WHERE trade_date = $1::date - INTERVAL '1 day'
		)
		SELECT
			dp.stock_code,
			dp.trade_date,
			dp.open_price,
			dp.high_price,
			dp.low_price,
			dp.close_price,
			dp.volume,
			COALESCE(pp.prev_close, 0) as prev_close,
			COALESCE(s.sector, '') as sector,
			COALESCE(mc.market_cap, 0) as market_cap
		FROM data.daily_prices dp
		LEFT JOIN prev_prices pp ON dp.stock_code = pp.stock_code
		LEFT JOIN data.stocks s ON dp.stock_code = s.code
		LEFT JOIN data.market_cap mc ON dp.stock_code = mc.stock_code AND mc.date = $1
		WHERE dp.trade_date = $1
	`

	rows, err := r.pool.Query(ctx, query, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []PriceWithMeta
	for rows.Next() {
		var p PriceWithMeta
		var openPrice, highPrice, lowPrice, closePrice, prevClose int64
		if err := rows.Scan(
			&p.Code, &p.Date, &openPrice, &highPrice, &lowPrice, &closePrice,
			&p.Volume, &prevClose, &p.Sector, &p.MarketCap,
		); err != nil {
			return nil, err
		}
		// int64 -> float64 변환 (가격은 원 단위)
		p.Open = float64(openPrice)
		p.High = float64(highPrice)
		p.Low = float64(lowPrice)
		p.Close = float64(closePrice)
		p.PrevClose = float64(prevClose)
		prices = append(prices, p)
	}

	return prices, rows.Err()
}

// GetForwardPrices 이벤트 이후 N거래일 가격 조회
func (r *PriceRepository) GetForwardPrices(ctx context.Context, code string, eventDate time.Time, days int) ([]PriceWithMeta, error) {
	query := `
		SELECT stock_code, trade_date, open_price, high_price, low_price, close_price, volume
		FROM data.daily_prices
		WHERE stock_code = $1 AND trade_date > $2
		ORDER BY trade_date ASC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, code, eventDate, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []PriceWithMeta
	for rows.Next() {
		var p PriceWithMeta
		var openPrice, highPrice, lowPrice, closePrice int64
		if err := rows.Scan(&p.Code, &p.Date, &openPrice, &highPrice, &lowPrice, &closePrice, &p.Volume); err != nil {
			return nil, err
		}
		p.Open = float64(openPrice)
		p.High = float64(highPrice)
		p.Low = float64(lowPrice)
		p.Close = float64(closePrice)
		prices = append(prices, p)
	}

	return prices, rows.Err()
}

// GetPrice 특정 종목/날짜의 가격 조회
func (r *PriceRepository) GetPrice(ctx context.Context, code string, date time.Time) (*PriceWithMeta, error) {
	query := `
		SELECT stock_code, trade_date, open_price, high_price, low_price, close_price, volume
		FROM data.daily_prices
		WHERE stock_code = $1 AND trade_date = $2
	`

	var p PriceWithMeta
	var openPrice, highPrice, lowPrice, closePrice int64
	err := r.pool.QueryRow(ctx, query, code, date).Scan(
		&p.Code, &p.Date, &openPrice, &highPrice, &lowPrice, &closePrice, &p.Volume,
	)
	if err != nil {
		return nil, err
	}
	p.Open = float64(openPrice)
	p.High = float64(highPrice)
	p.Low = float64(lowPrice)
	p.Close = float64(closePrice)

	return &p, nil
}
