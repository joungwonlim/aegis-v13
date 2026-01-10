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
		SELECT code, year, quarter, revenue, op_profit, net_profit,
		       assets, equity, debt, roe, debt_ratio, per, pbr, psr
		FROM fundamental.financials
		WHERE code = $1
		  AND (year < EXTRACT(YEAR FROM $2)::int
		       OR (year = EXTRACT(YEAR FROM $2)::int
		           AND quarter <= EXTRACT(QUARTER FROM $2)::int))
		ORDER BY year DESC, quarter DESC
		LIMIT 1
	`

	var f contracts.Financial
	err := r.pool.QueryRow(ctx, query, code, date).Scan(
		&f.Code, &f.Year, &f.Quarter, &f.Revenue, &f.OpProfit, &f.NetProfit,
		&f.Assets, &f.Equity, &f.Debt, &f.ROE, &f.DebtRatio, &f.PER, &f.PBR, &f.PSR,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// GetByCodeAndQuarter retrieves financial data for specific year and quarter
func (r *FinancialRepository) GetByCodeAndQuarter(ctx context.Context, code string, year int, quarter int) (*contracts.Financial, error) {
	query := `
		SELECT code, year, quarter, revenue, op_profit, net_profit,
		       assets, equity, debt, roe, debt_ratio, per, pbr, psr
		FROM fundamental.financials
		WHERE code = $1 AND year = $2 AND quarter = $3
	`

	var f contracts.Financial
	err := r.pool.QueryRow(ctx, query, code, year, quarter).Scan(
		&f.Code, &f.Year, &f.Quarter, &f.Revenue, &f.OpProfit, &f.NetProfit,
		&f.Assets, &f.Equity, &f.Debt, &f.ROE, &f.DebtRatio, &f.PER, &f.PBR, &f.PSR,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// Save saves a single financial record
func (r *FinancialRepository) Save(ctx context.Context, financial *contracts.Financial) error {
	query := `
		INSERT INTO fundamental.financials (
			code, year, quarter, revenue, op_profit, net_profit,
			assets, equity, debt, roe, debt_ratio, per, pbr, psr
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (code, year, quarter) DO UPDATE SET
			revenue = EXCLUDED.revenue,
			op_profit = EXCLUDED.op_profit,
			net_profit = EXCLUDED.net_profit,
			assets = EXCLUDED.assets,
			equity = EXCLUDED.equity,
			debt = EXCLUDED.debt,
			roe = EXCLUDED.roe,
			debt_ratio = EXCLUDED.debt_ratio,
			per = EXCLUDED.per,
			pbr = EXCLUDED.pbr,
			psr = EXCLUDED.psr
	`

	_, err := r.pool.Exec(ctx, query,
		financial.Code, financial.Year, financial.Quarter, financial.Revenue,
		financial.OpProfit, financial.NetProfit, financial.Assets, financial.Equity,
		financial.Debt, financial.ROE, financial.DebtRatio, financial.PER, financial.PBR, financial.PSR,
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
