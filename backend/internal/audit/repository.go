package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles audit data persistence
// ⭐ SSOT: Audit 데이터 저장/조회는 여기서만
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new audit repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SaveSnapshot saves daily snapshot to database
func (r *Repository) SaveSnapshot(ctx context.Context, snapshot *DailySnapshot) error {
	positionsJSON, err := json.Marshal(snapshot.Positions)
	if err != nil {
		return fmt.Errorf("failed to marshal positions: %w", err)
	}

	query := `
		INSERT INTO audit.daily_snapshots (
			date, total_value, cash, positions, daily_return, cum_return
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (date) DO UPDATE SET
			total_value = EXCLUDED.total_value,
			cash = EXCLUDED.cash,
			positions = EXCLUDED.positions,
			daily_return = EXCLUDED.daily_return,
			cum_return = EXCLUDED.cum_return
	`

	_, err = r.pool.Exec(ctx, query,
		snapshot.Date, snapshot.TotalValue, snapshot.Cash, positionsJSON,
		snapshot.DailyReturn, snapshot.CumReturn,
	)

	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	return nil
}

// GetSnapshot retrieves snapshot for a specific date
func (r *Repository) GetSnapshot(ctx context.Context, date time.Time) (*DailySnapshot, error) {
	query := `
		SELECT date, total_value, cash, positions, daily_return, cum_return
		FROM audit.daily_snapshots
		WHERE date = $1
	`

	var snapshot DailySnapshot
	var positionsJSON []byte

	err := r.pool.QueryRow(ctx, query, date).Scan(
		&snapshot.Date, &snapshot.TotalValue, &snapshot.Cash, &positionsJSON,
		&snapshot.DailyReturn, &snapshot.CumReturn,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("snapshot not found for date %s", date.Format("2006-01-02"))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	if err := json.Unmarshal(positionsJSON, &snapshot.Positions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal positions: %w", err)
	}

	return &snapshot, nil
}

// GetPreviousSnapshot retrieves the snapshot before given date
func (r *Repository) GetPreviousSnapshot(ctx context.Context, date time.Time) (*DailySnapshot, error) {
	query := `
		SELECT date, total_value, cash, positions, daily_return, cum_return
		FROM audit.daily_snapshots
		WHERE date < $1
		ORDER BY date DESC
		LIMIT 1
	`

	var snapshot DailySnapshot
	var positionsJSON []byte

	err := r.pool.QueryRow(ctx, query, date).Scan(
		&snapshot.Date, &snapshot.TotalValue, &snapshot.Cash, &positionsJSON,
		&snapshot.DailyReturn, &snapshot.CumReturn,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // No previous snapshot
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get previous snapshot: %w", err)
	}

	if err := json.Unmarshal(positionsJSON, &snapshot.Positions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal positions: %w", err)
	}

	return &snapshot, nil
}

// GetSnapshotHistory retrieves snapshots for a date range
func (r *Repository) GetSnapshotHistory(ctx context.Context, startDate, endDate time.Time) ([]DailySnapshot, error) {
	query := `
		SELECT date, total_value, cash, positions, daily_return, cum_return
		FROM audit.daily_snapshots
		WHERE date BETWEEN $1 AND $2
		ORDER BY date ASC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := make([]DailySnapshot, 0)

	for rows.Next() {
		var snapshot DailySnapshot
		var positionsJSON []byte

		err := rows.Scan(
			&snapshot.Date, &snapshot.TotalValue, &snapshot.Cash, &positionsJSON,
			&snapshot.DailyReturn, &snapshot.CumReturn,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		if err := json.Unmarshal(positionsJSON, &snapshot.Positions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal positions: %w", err)
		}

		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return snapshots, nil
}

// GetDailyReturns retrieves daily returns for a period
func (r *Repository) GetDailyReturns(ctx context.Context, startDate, endDate time.Time) ([]float64, error) {
	query := `
		SELECT daily_return
		FROM audit.daily_snapshots
		WHERE date BETWEEN $1 AND $2
		ORDER BY date ASC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily returns: %w", err)
	}
	defer rows.Close()

	returns := make([]float64, 0)

	for rows.Next() {
		var ret float64
		if err := rows.Scan(&ret); err != nil {
			return nil, fmt.Errorf("failed to scan return: %w", err)
		}
		returns = append(returns, ret)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return returns, nil
}

// GetTrades retrieves closed trades for a period
func (r *Repository) GetTrades(ctx context.Context, startDate, endDate time.Time) ([]Trade, error) {
	// TODO: Implement actual trade retrieval
	// For now, return empty slice
	return []Trade{}, nil
}

// SavePerformanceReport saves performance report to database
func (r *Repository) SavePerformanceReport(ctx context.Context, report *PerformanceReport) error {
	query := `
		INSERT INTO audit.performance_reports (
			report_date, period_start, period_end, total_return, benchmark_return,
			alpha, beta, sharpe_ratio, volatility, max_drawdown,
			win_rate, avg_win, avg_loss, profit_factor
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (report_date) DO UPDATE SET
			period_start = EXCLUDED.period_start,
			period_end = EXCLUDED.period_end,
			total_return = EXCLUDED.total_return,
			benchmark_return = EXCLUDED.benchmark_return,
			alpha = EXCLUDED.alpha,
			beta = EXCLUDED.beta,
			sharpe_ratio = EXCLUDED.sharpe_ratio,
			volatility = EXCLUDED.volatility,
			max_drawdown = EXCLUDED.max_drawdown,
			win_rate = EXCLUDED.win_rate,
			avg_win = EXCLUDED.avg_win,
			avg_loss = EXCLUDED.avg_loss,
			profit_factor = EXCLUDED.profit_factor
	`

	_, err := r.pool.Exec(ctx, query,
		time.Now().Truncate(24*time.Hour), report.StartDate, report.EndDate,
		report.TotalReturn, report.Benchmark, report.Alpha, report.Beta,
		report.Sharpe, report.Volatility, report.MaxDrawdown,
		report.WinRate, report.AvgWin, report.AvgLoss, report.ProfitFactor,
	)

	if err != nil {
		return fmt.Errorf("failed to save performance report: %w", err)
	}

	return nil
}

// GetPerformanceReport retrieves the latest performance report for a period
func (r *Repository) GetPerformanceReport(ctx context.Context, period string) (*PerformanceReport, error) {
	query := `
		SELECT report_data
		FROM audit.performance_reports
		WHERE period = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var reportDataJSON []byte
	err := r.pool.QueryRow(ctx, query, period).Scan(&reportDataJSON)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("no performance report found for period %s", period)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get performance report: %w", err)
	}

	var report PerformanceReport
	if err := json.Unmarshal(reportDataJSON, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report: %w", err)
	}

	return &report, nil
}

// SaveAttribution saves attribution analysis to database
func (r *Repository) SaveAttribution(ctx context.Context, period string, attrs []Attribution) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing attributions for the period
	_, err = tx.Exec(ctx, "DELETE FROM audit.attributions WHERE period = $1", period)
	if err != nil {
		return fmt.Errorf("failed to delete old attributions: %w", err)
	}

	// Insert new attributions
	query := `
		INSERT INTO audit.attributions (
			period, factor, contribution, exposure
		) VALUES ($1, $2, $3, $4)
	`

	for _, attr := range attrs {
		_, err := tx.Exec(ctx, query, period, attr.Factor, attr.Contribution, attr.Exposure)
		if err != nil {
			return fmt.Errorf("failed to insert attribution: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetAttribution retrieves attribution analysis for a period
func (r *Repository) GetAttribution(ctx context.Context, period string) ([]Attribution, error) {
	query := `
		SELECT factor, contribution, exposure
		FROM audit.attributions
		WHERE period = $1
		ORDER BY contribution DESC
	`

	rows, err := r.pool.Query(ctx, query, period)
	if err != nil {
		return nil, fmt.Errorf("failed to query attributions: %w", err)
	}
	defer rows.Close()

	attrs := make([]Attribution, 0)

	for rows.Next() {
		var attr Attribution
		err := rows.Scan(&attr.Factor, &attr.Contribution, &attr.Exposure)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attribution: %w", err)
		}
		attrs = append(attrs, attr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return attrs, nil
}
