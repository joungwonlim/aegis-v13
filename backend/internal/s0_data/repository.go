package s0_data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/internal/external/dart"
	"github.com/wonny/aegis/v13/backend/internal/external/krx"
	"github.com/wonny/aegis/v13/backend/internal/external/naver"
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

// Stock represents a stock entity
type Stock struct {
	Code   string
	Name   string
	Market string
	Status string
}

// GetActiveStocks retrieves all active stocks
func (r *Repository) GetActiveStocks(ctx context.Context) ([]Stock, error) {
	query := `
		SELECT code, name, market, status
		FROM data.stocks
		WHERE status = 'active'
		ORDER BY code
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query active stocks: %w", err)
	}
	defer rows.Close()

	var stocks []Stock
	for rows.Next() {
		var s Stock
		if err := rows.Scan(&s.Code, &s.Name, &s.Market, &s.Status); err != nil {
			return nil, fmt.Errorf("scan stock: %w", err)
		}
		stocks = append(stocks, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return stocks, nil
}

// SavePrices saves price data to the database (bulk upsert)
func (r *Repository) SavePrices(ctx context.Context, prices []naver.PriceData) error {
	if len(prices) == 0 {
		return nil
	}

	query := `
		INSERT INTO data.daily_prices (
			stock_code, trade_date, open_price, high_price, low_price,
			close_price, volume, trading_value, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (stock_code, trade_date) DO UPDATE SET
			open_price = EXCLUDED.open_price,
			high_price = EXCLUDED.high_price,
			low_price = EXCLUDED.low_price,
			close_price = EXCLUDED.close_price,
			volume = EXCLUDED.volume,
			trading_value = EXCLUDED.trading_value,
			updated_at = NOW()
	`

	// Batch insert using transactions
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, p := range prices {
		_, err := tx.Exec(ctx, query,
			p.StockCode, p.TradeDate, p.OpenPrice, p.HighPrice, p.LowPrice,
			p.ClosePrice, p.Volume, p.TradingValue,
		)
		if err != nil {
			return fmt.Errorf("insert price for %s: %w", p.StockCode, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// SaveInvestorFlow saves investor flow data to the database (bulk upsert)
func (r *Repository) SaveInvestorFlow(ctx context.Context, flows []naver.InvestorFlowData) error {
	if len(flows) == 0 {
		return nil
	}

	query := `
		INSERT INTO data.investor_flow (
			stock_code, trade_date, foreign_net, institution_net, individual_net,
			financial_net, insurance_net, trust_net, pension_net,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (stock_code, trade_date) DO UPDATE SET
			foreign_net = EXCLUDED.foreign_net,
			institution_net = EXCLUDED.institution_net,
			individual_net = EXCLUDED.individual_net,
			financial_net = EXCLUDED.financial_net,
			insurance_net = EXCLUDED.insurance_net,
			trust_net = EXCLUDED.trust_net,
			pension_net = EXCLUDED.pension_net,
			updated_at = NOW()
	`

	// Batch insert using transactions
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, f := range flows {
		_, err := tx.Exec(ctx, query,
			f.StockCode, f.TradeDate, f.ForeignNet, f.InstitutionNet, f.IndividualNet,
			f.FinancialNet, f.InsuranceNet, f.TrustNet, f.PensionNet,
		)
		if err != nil {
			return fmt.Errorf("insert investor flow for %s: %w", f.StockCode, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// SaveDisclosures saves disclosure data to the database (bulk insert)
// ⭐ SSOT: DART 공시 데이터 저장은 이 함수에서만
func (r *Repository) SaveDisclosures(ctx context.Context, disclosures []dart.Disclosure) error {
	if len(disclosures) == 0 {
		return nil
	}

	query := `
		INSERT INTO data.disclosures (
			stock_code, disclosed_at, title, category, content, url, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (stock_code, disclosed_at, title) DO NOTHING
	`

	// Batch insert using transactions
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, d := range disclosures {
		// Parse report date (YYYYMMDD -> time.Time)
		reportDate, err := time.Parse("20060102", d.RceptDt)
		if err != nil {
			return fmt.Errorf("parse report date %s: %w", d.RceptDt, err)
		}

		// Generate URL from receipt number
		url := dart.GetDARTURL(d.RceptNo)

		// Get category from corp_cls
		category := string(dart.GetCategory(d.CorpCls))

		_, err = tx.Exec(ctx, query,
			d.StockCode, reportDate, d.ReportNm, category, "", url,
		)
		if err != nil {
			return fmt.Errorf("insert disclosure for %s: %w", d.StockCode, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// SaveMarketTrend saves market trend data (KRX) to the database
// ⭐ SSOT: KRX 시장 지표 저장은 이 함수에서만
func (r *Repository) SaveMarketTrend(ctx context.Context, indexName string, data *krx.MarketTrendData) error {
	if data == nil {
		return nil
	}

	query := `
		INSERT INTO data.market_indicators (
			trade_date, indicator_type, indicator_value, source, created_at
		) VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (trade_date, indicator_type) DO UPDATE SET
			indicator_value = EXCLUDED.indicator_value,
			source = EXCLUDED.source,
			updated_at = NOW()
	`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Save foreign net
	indicatorType := fmt.Sprintf("%s_foreign_net", strings.ToLower(indexName))
	_, err = tx.Exec(ctx, query, data.TradeDate, indicatorType, data.ForeignNet, "naver")
	if err != nil {
		return fmt.Errorf("insert foreign net: %w", err)
	}

	// Save institution net
	indicatorType = fmt.Sprintf("%s_institution_net", strings.ToLower(indexName))
	_, err = tx.Exec(ctx, query, data.TradeDate, indicatorType, data.InstitutionNet, "naver")
	if err != nil {
		return fmt.Errorf("insert institution net: %w", err)
	}

	// Save individual net
	indicatorType = fmt.Sprintf("%s_individual_net", strings.ToLower(indexName))
	_, err = tx.Exec(ctx, query, data.TradeDate, indicatorType, data.IndividualNet, "naver")
	if err != nil {
		return fmt.Errorf("insert individual net: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
