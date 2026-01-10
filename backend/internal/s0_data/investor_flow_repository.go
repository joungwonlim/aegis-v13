package s0_data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// InvestorFlowRepository implements contracts.InvestorFlowRepository
// ⭐ SSOT: 수급 데이터 저장소는 여기서만
type InvestorFlowRepository struct {
	pool *pgxpool.Pool
}

// NewInvestorFlowRepository creates a new investor flow repository
func NewInvestorFlowRepository(pool *pgxpool.Pool) *InvestorFlowRepository {
	return &InvestorFlowRepository{pool: pool}
}

// GetByCodeAndDate retrieves investor flow for a specific code and date
func (r *InvestorFlowRepository) GetByCodeAndDate(ctx context.Context, code string, date time.Time) (*contracts.InvestorFlow, error) {
	query := `
		SELECT stock_code, trade_date, foreign_net_value, inst_net_value, indiv_net_value
		FROM data.investor_flow
		WHERE stock_code = $1 AND trade_date = $2
	`

	var f contracts.InvestorFlow
	err := r.pool.QueryRow(ctx, query, code, date).Scan(
		&f.Code, &f.Date, &f.ForeignNet, &f.InstitutionNet, &f.IndividualNet,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// GetByCodeAndDateRange retrieves investor flows for a code within date range
func (r *InvestorFlowRepository) GetByCodeAndDateRange(ctx context.Context, code string, from, to time.Time) ([]*contracts.InvestorFlow, error) {
	query := `
		SELECT stock_code, trade_date, foreign_net_value, inst_net_value, indiv_net_value
		FROM data.investor_flow
		WHERE stock_code = $1 AND trade_date BETWEEN $2 AND $3
		ORDER BY trade_date ASC
	`

	rows, err := r.pool.Query(ctx, query, code, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flows []*contracts.InvestorFlow
	for rows.Next() {
		var f contracts.InvestorFlow
		if err := rows.Scan(&f.Code, &f.Date, &f.ForeignNet, &f.InstitutionNet, &f.IndividualNet); err != nil {
			return nil, err
		}
		flows = append(flows, &f)
	}
	return flows, rows.Err()
}

// Save saves a single investor flow record
func (r *InvestorFlowRepository) Save(ctx context.Context, flow *contracts.InvestorFlow) error {
	query := `
		INSERT INTO data.investor_flow (stock_code, trade_date, foreign_net_value, inst_net_value, indiv_net_value)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (stock_code, trade_date) DO UPDATE SET
			foreign_net_value = EXCLUDED.foreign_net_value,
			inst_net_value = EXCLUDED.inst_net_value,
			indiv_net_value = EXCLUDED.indiv_net_value
	`

	_, err := r.pool.Exec(ctx, query,
		flow.Code, flow.Date, flow.ForeignNet, flow.InstitutionNet, flow.IndividualNet,
	)
	return err
}

// SaveBatch saves multiple investor flow records
func (r *InvestorFlowRepository) SaveBatch(ctx context.Context, flows []*contracts.InvestorFlow) error {
	if len(flows) == 0 {
		return nil
	}

	for _, flow := range flows {
		if err := r.Save(ctx, flow); err != nil {
			return err
		}
	}
	return nil
}
