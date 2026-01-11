package s0_data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// FinancialRepository implements contracts.FinancialRepository
// ⭐ SSOT: 재무 데이터 저장소는 여기서만
type FinancialRepository struct {
	pool *pgxpool.Pool
}

// NewFinancialRepository creates a new financial repository
func NewFinancialRepository(pool *pgxpool.Pool) *FinancialRepository {
	return &FinancialRepository{pool: pool}
}

// GetLatestByCode retrieves the most recent financial data for a code before given date
func (r *FinancialRepository) GetLatestByCode(ctx context.Context, code string, date time.Time) (*contracts.Financial, error) {
	query := `
		SELECT stock_code,
		       EXTRACT(YEAR FROM report_date)::int as year,
		       EXTRACT(QUARTER FROM report_date)::int as quarter,
		       COALESCE(revenue, 0), COALESCE(operating_profit, 0), COALESCE(net_profit, 0),
		       COALESCE(roe, 0), COALESCE(debt_ratio, 0),
		       COALESCE(per, 0), COALESCE(pbr, 0)
		FROM data.fundamentals
		WHERE stock_code = $1 AND report_date <= $2
		ORDER BY report_date DESC
		LIMIT 1
	`

	var f contracts.Financial
	err := r.pool.QueryRow(ctx, query, code, date).Scan(
		&f.Code, &f.Year, &f.Quarter, &f.Revenue, &f.OpProfit, &f.NetProfit,
		&f.ROE, &f.DebtRatio, &f.PER, &f.PBR,
	)
	if err != nil {
		return nil, err
	}
	// Assets, Equity, Debt, PSR are not available in current schema
	f.Assets = 0
	f.Equity = 0
	f.Debt = 0
	f.PSR = 0
	return &f, nil
}

// GetByCodeAndQuarter retrieves financial data for specific year and quarter
func (r *FinancialRepository) GetByCodeAndQuarter(ctx context.Context, code string, year int, quarter int) (*contracts.Financial, error) {
	// Calculate date range for the quarter
	startMonth := (quarter-1)*3 + 1
	endMonth := quarter * 3
	startDate := time.Date(year, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, time.Month(endMonth+1), 0, 0, 0, 0, 0, time.UTC) // Last day of end month

	query := `
		SELECT stock_code,
		       EXTRACT(YEAR FROM report_date)::int as year,
		       EXTRACT(QUARTER FROM report_date)::int as quarter,
		       COALESCE(revenue, 0), COALESCE(operating_profit, 0), COALESCE(net_profit, 0),
		       COALESCE(roe, 0), COALESCE(debt_ratio, 0),
		       COALESCE(per, 0), COALESCE(pbr, 0)
		FROM data.fundamentals
		WHERE stock_code = $1 AND report_date BETWEEN $2 AND $3
		ORDER BY report_date DESC
		LIMIT 1
	`

	var f contracts.Financial
	err := r.pool.QueryRow(ctx, query, code, startDate, endDate).Scan(
		&f.Code, &f.Year, &f.Quarter, &f.Revenue, &f.OpProfit, &f.NetProfit,
		&f.ROE, &f.DebtRatio, &f.PER, &f.PBR,
	)
	if err != nil {
		return nil, err
	}
	// Assets, Equity, Debt, PSR are not available in current schema
	f.Assets = 0
	f.Equity = 0
	f.Debt = 0
	f.PSR = 0
	return &f, nil
}

// Save saves a single financial record
func (r *FinancialRepository) Save(ctx context.Context, financial *contracts.Financial) error {
	// Calculate report_date from year and quarter (last day of the quarter)
	endMonth := financial.Quarter * 3
	reportDate := time.Date(financial.Year, time.Month(endMonth+1), 0, 0, 0, 0, 0, time.UTC) // Last day of quarter

	query := `
		INSERT INTO data.fundamentals (
			stock_code, report_date, revenue, operating_profit, net_profit,
			roe, debt_ratio, per, pbr
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (stock_code, report_date) DO UPDATE SET
			revenue = EXCLUDED.revenue,
			operating_profit = EXCLUDED.operating_profit,
			net_profit = EXCLUDED.net_profit,
			roe = EXCLUDED.roe,
			debt_ratio = EXCLUDED.debt_ratio,
			per = EXCLUDED.per,
			pbr = EXCLUDED.pbr
	`

	_, err := r.pool.Exec(ctx, query,
		financial.Code, reportDate, financial.Revenue, financial.OpProfit, financial.NetProfit,
		financial.ROE, financial.DebtRatio, financial.PER, financial.PBR,
	)
	return err
}

// SaveBatch saves multiple financial records
func (r *FinancialRepository) SaveBatch(ctx context.Context, financials []*contracts.Financial) error {
	if len(financials) == 0 {
		return nil
	}

	for _, financial := range financials {
		if err := r.Save(ctx, financial); err != nil {
			return err
		}
	}
	return nil
}
