package portfolio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// Repository handles portfolio data persistence
// ⭐ SSOT: Portfolio 데이터 저장/조회는 여기서만
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new portfolio repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SaveTargetPortfolio saves target portfolio to database
func (r *Repository) SaveTargetPortfolio(ctx context.Context, target *contracts.TargetPortfolio) error {
	// Begin transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing positions for the date
	_, err = tx.Exec(ctx, "DELETE FROM portfolio.target_positions WHERE target_date = $1", target.Date)
	if err != nil {
		return fmt.Errorf("failed to delete old positions: %w", err)
	}

	// Insert new positions
	query := `
		INSERT INTO portfolio.target_positions (
			target_date, stock_code, stock_name, weight, target_qty, action, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	for _, pos := range target.Positions {
		_, err := tx.Exec(ctx, query,
			target.Date, pos.Code, pos.Name, pos.Weight, pos.TargetQty, pos.Action, pos.Reason,
		)
		if err != nil {
			return fmt.Errorf("failed to insert position: %w", err)
		}
	}

	// Save portfolio summary
	summaryQuery := `
		INSERT INTO portfolio.portfolio_snapshots (
			snapshot_date, total_positions, total_weight, cash_reserve, created_at
		) VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (snapshot_date) DO UPDATE SET
			total_positions = EXCLUDED.total_positions,
			total_weight = EXCLUDED.total_weight,
			cash_reserve = EXCLUDED.cash_reserve,
			created_at = NOW()
	`

	_, err = tx.Exec(ctx, summaryQuery,
		target.Date, len(target.Positions), target.TotalWeight(), target.Cash,
	)
	if err != nil {
		return fmt.Errorf("failed to save portfolio snapshot: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetTargetPortfolio retrieves target portfolio for a date
func (r *Repository) GetTargetPortfolio(ctx context.Context, date time.Time) (*contracts.TargetPortfolio, error) {
	query := `
		SELECT stock_code, stock_name, weight, target_qty, action, reason
		FROM portfolio.target_positions
		WHERE target_date = $1
		ORDER BY weight DESC
	`

	rows, err := r.pool.Query(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query target positions: %w", err)
	}
	defer rows.Close()

	portfolio := &contracts.TargetPortfolio{
		Date:      date,
		Positions: make([]contracts.TargetPosition, 0),
	}

	for rows.Next() {
		var pos contracts.TargetPosition
		err := rows.Scan(&pos.Code, &pos.Name, &pos.Weight, &pos.TargetQty, &pos.Action, &pos.Reason)
		if err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		portfolio.Positions = append(portfolio.Positions, pos)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Get cash reserve from snapshot
	var cashReserve float64
	err = r.pool.QueryRow(ctx,
		"SELECT cash_reserve FROM portfolio.portfolio_snapshots WHERE snapshot_date = $1",
		date,
	).Scan(&cashReserve)

	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get cash reserve: %w", err)
	}

	portfolio.Cash = cashReserve

	return portfolio, nil
}

// SaveHoldings saves current holdings to database
func (r *Repository) SaveHoldings(ctx context.Context, date time.Time, holdings []Holding) error {
	// Begin transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing holdings for the date
	_, err = tx.Exec(ctx, "DELETE FROM portfolio.holdings WHERE holding_date = $1", date)
	if err != nil {
		return fmt.Errorf("failed to delete old holdings: %w", err)
	}

	// Insert new holdings
	query := `
		INSERT INTO portfolio.holdings (
			holding_date, stock_code, stock_name, quantity, avg_price, current_price,
			market_value, weight, unrealized_pnl, unrealized_pnl_pct
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	for _, h := range holdings {
		_, err := tx.Exec(ctx, query,
			date, h.Code, h.Name, h.Quantity, h.AvgPrice, h.CurrentPrice,
			h.MarketValue, h.Weight, h.UnrealizedPnL, h.UnrealizedPnLPct,
		)
		if err != nil {
			return fmt.Errorf("failed to insert holding: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetCurrentHoldings retrieves current holdings for a date
func (r *Repository) GetCurrentHoldings(ctx context.Context, date time.Time) ([]Holding, error) {
	query := `
		SELECT
			stock_code, stock_name, quantity, avg_price, current_price,
			market_value, weight, unrealized_pnl, unrealized_pnl_pct
		FROM portfolio.holdings
		WHERE holding_date = $1
		ORDER BY market_value DESC
	`

	rows, err := r.pool.Query(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query holdings: %w", err)
	}
	defer rows.Close()

	holdings := make([]Holding, 0)

	for rows.Next() {
		var h Holding
		err := rows.Scan(
			&h.Code, &h.Name, &h.Quantity, &h.AvgPrice, &h.CurrentPrice,
			&h.MarketValue, &h.Weight, &h.UnrealizedPnL, &h.UnrealizedPnLPct,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan holding: %w", err)
		}
		holdings = append(holdings, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return holdings, nil
}

// Holding represents current portfolio holding
type Holding struct {
	Code             string
	Name             string
	Quantity         int
	AvgPrice         float64
	CurrentPrice     float64
	MarketValue      float64
	Weight           float64
	UnrealizedPnL    float64
	UnrealizedPnLPct float64
}

// GetPortfolioSummary retrieves portfolio summary statistics
func (r *Repository) GetPortfolioSummary(ctx context.Context, date time.Time) (*PortfolioSummary, error) {
	query := `
		SELECT
			snapshot_date, total_positions, total_weight, cash_reserve, created_at
		FROM portfolio.portfolio_snapshots
		WHERE snapshot_date = $1
	`

	var summary PortfolioSummary
	err := r.pool.QueryRow(ctx, query, date).Scan(
		&summary.Date, &summary.TotalPositions, &summary.TotalWeight,
		&summary.CashReserve, &summary.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("no portfolio snapshot found for date %s", date.Format("2006-01-02"))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio summary: %w", err)
	}

	return &summary, nil
}

// PortfolioSummary represents portfolio summary statistics
type PortfolioSummary struct {
	Date           time.Time
	TotalPositions int
	TotalWeight    float64
	CashReserve    float64
	CreatedAt      time.Time
}

// SaveRebalanceLog saves rebalance execution log
func (r *Repository) SaveRebalanceLog(ctx context.Context, log *RebalanceLog) error {
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO portfolio.rebalance_logs (
			rebalance_date, total_orders, executed_orders, failed_orders,
			turnover, execution_time_ms, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.pool.Exec(ctx, query,
		log.Date, log.TotalOrders, log.ExecutedOrders, log.FailedOrders,
		log.Turnover, log.ExecutionTimeMs, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to save rebalance log: %w", err)
	}

	return nil
}

// RebalanceLog represents rebalance execution log
type RebalanceLog struct {
	Date            time.Time
	TotalOrders     int
	ExecutedOrders  int
	FailedOrders    int
	Turnover        float64
	ExecutionTimeMs int64
	Metadata        map[string]interface{}
}
