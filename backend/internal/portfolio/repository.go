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
	// ⭐ P0 수정: target_qty → target_value (금액 기반)
	query := `
		INSERT INTO portfolio.target_positions (
			target_date, stock_code, stock_name, weight, target_value, action, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	for _, pos := range target.Positions {
		_, err := tx.Exec(ctx, query,
			target.Date, pos.Code, pos.Name, pos.Weight, pos.TargetValue, pos.Action, pos.Reason,
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
// ⭐ P0 수정: target_qty → target_value
func (r *Repository) GetTargetPortfolio(ctx context.Context, date time.Time) (*contracts.TargetPortfolio, error) {
	query := `
		SELECT stock_code, stock_name, weight, target_value, action, reason
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
		err := rows.Scan(&pos.Code, &pos.Name, &pos.Weight, &pos.TargetValue, &pos.Action, &pos.Reason)
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

// ==============================================
// Watchlist CRUD Operations
// ==============================================

// WatchlistItem represents a watchlist entry
type WatchlistItem struct {
	ID              int       `json:"id"`
	StockCode       string    `json:"stock_code"`
	StockName       string    `json:"stock_name,omitempty"`
	Market          string    `json:"market,omitempty"`
	Category        string    `json:"category"`
	AlertEnabled    bool      `json:"alert_enabled"`
	GrokAnalysis    *string   `json:"grok_analysis,omitempty"`
	GeminiAnalysis  *string   `json:"gemini_analysis,omitempty"`
	ChatGPTAnalysis *string   `json:"chatgpt_analysis,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GetWatchlistAll retrieves all watchlist items with stock info from data.stocks
func (r *Repository) GetWatchlistAll(ctx context.Context) ([]WatchlistItem, error) {
	query := `
		SELECT
			w.id, w.stock_code,
			COALESCE(s.name, '') as stock_name,
			COALESCE(s.market, '') as market,
			w.category, w.alert_enabled,
			w.grok_analysis, w.gemini_analysis, w.chatgpt_analysis,
			w.created_at, w.updated_at
		FROM portfolio.watchlist w
		LEFT JOIN data.stocks s ON w.stock_code = s.code
		ORDER BY w.category, w.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query watchlist: %w", err)
	}
	defer rows.Close()

	items := make([]WatchlistItem, 0)
	for rows.Next() {
		var item WatchlistItem
		err := rows.Scan(
			&item.ID, &item.StockCode, &item.StockName, &item.Market,
			&item.Category, &item.AlertEnabled,
			&item.GrokAnalysis, &item.GeminiAnalysis, &item.ChatGPTAnalysis,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watchlist item: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// GetWatchlistByCategory retrieves watchlist items by category
func (r *Repository) GetWatchlistByCategory(ctx context.Context, category string) ([]WatchlistItem, error) {
	query := `
		SELECT
			w.id, w.stock_code,
			COALESCE(s.name, '') as stock_name,
			COALESCE(s.market, '') as market,
			w.category, w.alert_enabled,
			w.grok_analysis, w.gemini_analysis, w.chatgpt_analysis,
			w.created_at, w.updated_at
		FROM portfolio.watchlist w
		LEFT JOIN data.stocks s ON w.stock_code = s.code
		WHERE w.category = $1
		ORDER BY w.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query watchlist by category: %w", err)
	}
	defer rows.Close()

	items := make([]WatchlistItem, 0)
	for rows.Next() {
		var item WatchlistItem
		err := rows.Scan(
			&item.ID, &item.StockCode, &item.StockName, &item.Market,
			&item.Category, &item.AlertEnabled,
			&item.GrokAnalysis, &item.GeminiAnalysis, &item.ChatGPTAnalysis,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watchlist item: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// CreateWatchlistItem creates a new watchlist entry
func (r *Repository) CreateWatchlistItem(ctx context.Context, stockCode, category string, alertEnabled bool) (*WatchlistItem, error) {
	query := `
		INSERT INTO portfolio.watchlist (stock_code, category, alert_enabled)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	var item WatchlistItem
	item.StockCode = stockCode
	item.Category = category
	item.AlertEnabled = alertEnabled

	err := r.pool.QueryRow(ctx, query, stockCode, category, alertEnabled).Scan(
		&item.ID, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create watchlist item: %w", err)
	}

	// Get stock name and market from data.stocks
	_ = r.pool.QueryRow(ctx,
		"SELECT name, market FROM data.stocks WHERE code = $1",
		stockCode,
	).Scan(&item.StockName, &item.Market)

	return &item, nil
}

// UpdateWatchlistItem updates a watchlist entry
func (r *Repository) UpdateWatchlistItem(ctx context.Context, id int, category *string, alertEnabled *bool) (*WatchlistItem, error) {
	// Build dynamic update query
	setParts := make([]string, 0)
	args := make([]interface{}, 0)
	argIndex := 1

	if category != nil {
		setParts = append(setParts, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, *category)
		argIndex++
	}
	if alertEnabled != nil {
		setParts = append(setParts, fmt.Sprintf("alert_enabled = $%d", argIndex))
		args = append(args, *alertEnabled)
		argIndex++
	}

	if len(setParts) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE portfolio.watchlist
		SET %s
		WHERE id = $%d
		RETURNING id, stock_code, category, alert_enabled,
			grok_analysis, gemini_analysis, chatgpt_analysis,
			created_at, updated_at
	`, join(setParts, ", "), argIndex)

	var item WatchlistItem
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&item.ID, &item.StockCode, &item.Category, &item.AlertEnabled,
		&item.GrokAnalysis, &item.GeminiAnalysis, &item.ChatGPTAnalysis,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("watchlist item not found: %d", id)
		}
		return nil, fmt.Errorf("failed to update watchlist item: %w", err)
	}

	// Get stock name and market from data.stocks
	_ = r.pool.QueryRow(ctx,
		"SELECT name, market FROM data.stocks WHERE code = $1",
		item.StockCode,
	).Scan(&item.StockName, &item.Market)

	return &item, nil
}

// DeleteWatchlistItem deletes a watchlist entry
func (r *Repository) DeleteWatchlistItem(ctx context.Context, id int) error {
	result, err := r.pool.Exec(ctx,
		"DELETE FROM portfolio.watchlist WHERE id = $1",
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete watchlist item: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("watchlist item not found: %d", id)
	}

	return nil
}

// Helper function for joining strings
func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// ==============================================
// Exit Monitoring Operations
// ==============================================

// ExitMonitoringStatus represents exit monitoring status for a stock
type ExitMonitoringStatus struct {
	StockCode string `json:"stock_code"`
	Enabled   bool   `json:"enabled"`
}

// GetExitMonitoringAll retrieves all exit monitoring statuses
func (r *Repository) GetExitMonitoringAll(ctx context.Context) ([]ExitMonitoringStatus, error) {
	query := `
		SELECT stock_code, enabled
		FROM portfolio.exit_monitoring
		ORDER BY stock_code
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query exit monitoring: %w", err)
	}
	defer rows.Close()

	statuses := make([]ExitMonitoringStatus, 0)
	for rows.Next() {
		var s ExitMonitoringStatus
		if err := rows.Scan(&s.StockCode, &s.Enabled); err != nil {
			return nil, fmt.Errorf("failed to scan exit monitoring: %w", err)
		}
		statuses = append(statuses, s)
	}

	return statuses, rows.Err()
}

// GetExitMonitoringByCode retrieves exit monitoring status for a specific stock
func (r *Repository) GetExitMonitoringByCode(ctx context.Context, stockCode string) (*ExitMonitoringStatus, error) {
	query := `SELECT stock_code, enabled FROM portfolio.exit_monitoring WHERE stock_code = $1`

	var s ExitMonitoringStatus
	err := r.pool.QueryRow(ctx, query, stockCode).Scan(&s.StockCode, &s.Enabled)
	if err == pgx.ErrNoRows {
		return nil, nil // Not found, not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exit monitoring: %w", err)
	}

	return &s, nil
}

// SetExitMonitoring sets exit monitoring status for a stock (upsert)
func (r *Repository) SetExitMonitoring(ctx context.Context, stockCode string, enabled bool) error {
	query := `
		INSERT INTO portfolio.exit_monitoring (stock_code, enabled, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (stock_code) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			updated_at = NOW()
	`

	_, err := r.pool.Exec(ctx, query, stockCode, enabled)
	if err != nil {
		return fmt.Errorf("failed to set exit monitoring: %w", err)
	}

	return nil
}

// DeleteExitMonitoring removes exit monitoring status for a stock
func (r *Repository) DeleteExitMonitoring(ctx context.Context, stockCode string) error {
	_, err := r.pool.Exec(ctx,
		"DELETE FROM portfolio.exit_monitoring WHERE stock_code = $1",
		stockCode,
	)
	if err != nil {
		return fmt.Errorf("failed to delete exit monitoring: %w", err)
	}

	return nil
}

// ==============================================
// Stock Info Operations (for market lookup)
// ==============================================

// StockInfo represents basic stock information
type StockInfo struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Market string `json:"market"`
}

// GetStockMarkets retrieves market info for multiple stock codes in a single query
// 효율적인 IN 쿼리 사용 - N+1 문제 방지
func (r *Repository) GetStockMarkets(ctx context.Context, codes []string) (map[string]string, error) {
	if len(codes) == 0 {
		return make(map[string]string), nil
	}

	query := `SELECT code, market FROM data.stocks WHERE code = ANY($1)`

	rows, err := r.pool.Query(ctx, query, codes)
	if err != nil {
		return nil, fmt.Errorf("failed to query stock markets: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var code, market string
		if err := rows.Scan(&code, &market); err != nil {
			return nil, fmt.Errorf("failed to scan stock market: %w", err)
		}
		result[code] = market
	}

	return result, rows.Err()
}
