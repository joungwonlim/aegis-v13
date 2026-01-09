package repos

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// SignalRepository implements contracts.SignalRepository
// ⭐ SSOT: Signal 데이터 저장/조회는 여기서만
type SignalRepository struct {
	pool *pgxpool.Pool
}

// NewSignalRepository creates a new signal repository
func NewSignalRepository(pool *pgxpool.Pool) *SignalRepository {
	return &SignalRepository{pool: pool}
}

// GetByDate retrieves signal set for a specific date
func (r *SignalRepository) GetByDate(ctx context.Context, date time.Time) (*contracts.SignalSet, error) {
	// Query factor_scores for the date
	query := `
		SELECT
			stock_code,
			momentum, technical, value, quality, flow, event
		FROM signals.factor_scores
		WHERE calc_date = $1
		ORDER BY stock_code
	`

	rows, err := r.pool.Query(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query factor scores: %w", err)
	}
	defer rows.Close()

	signalSet := &contracts.SignalSet{
		Date:    date,
		Signals: make(map[string]*contracts.StockSignals),
	}

	for rows.Next() {
		var code string
		var momentum, technical, value, quality, flow, event float64

		err := rows.Scan(&code, &momentum, &technical, &value, &quality, &flow, &event)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		signals := &contracts.StockSignals{
			Code:      code,
			Momentum:  momentum,
			Technical: technical,
			Value:     value,
			Quality:   quality,
			Flow:      flow,
			Event:     event,
		}

		// Load details
		if err := r.loadSignalDetails(ctx, code, date, signals); err != nil {
			// Log but don't fail
			continue
		}

		signalSet.Signals[code] = signals
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return signalSet, nil
}

// GetByCodeAndDate retrieves signals for a specific stock and date
func (r *SignalRepository) GetByCodeAndDate(ctx context.Context, code string, date time.Time) (*contracts.StockSignals, error) {
	query := `
		SELECT
			stock_code,
			momentum, technical, value, quality, flow, event
		FROM signals.factor_scores
		WHERE stock_code = $1 AND calc_date = $2
	`

	var signals contracts.StockSignals
	err := r.pool.QueryRow(ctx, query, code, date).Scan(
		&signals.Code,
		&signals.Momentum,
		&signals.Technical,
		&signals.Value,
		&signals.Quality,
		&signals.Flow,
		&signals.Event,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get signals: %w", err)
	}

	// Load details
	if err := r.loadSignalDetails(ctx, code, date, &signals); err != nil {
		return nil, err
	}

	return &signals, nil
}

// Save saves the entire signal set
func (r *SignalRepository) Save(ctx context.Context, signalSet *contracts.SignalSet) error {
	// Begin transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert factor scores
	for code, signals := range signalSet.Signals {
		if err := r.saveFactorScores(ctx, tx, code, signalSet.Date, signals); err != nil {
			return fmt.Errorf("failed to save factor scores for %s: %w", code, err)
		}

		// Save details if available
		if err := r.saveSignalDetails(ctx, tx, code, signalSet.Date, signals); err != nil {
			return fmt.Errorf("failed to save signal details for %s: %w", code, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// saveFactorScores saves factor scores to database
func (r *SignalRepository) saveFactorScores(ctx context.Context, tx pgx.Tx, code string, date time.Time, signals *contracts.StockSignals) error {
	query := `
		INSERT INTO signals.factor_scores (
			stock_code, calc_date,
			momentum, technical, value, quality, flow, event,
			total_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (stock_code, calc_date) DO UPDATE SET
			momentum = EXCLUDED.momentum,
			technical = EXCLUDED.technical,
			value = EXCLUDED.value,
			quality = EXCLUDED.quality,
			flow = EXCLUDED.flow,
			event = EXCLUDED.event,
			total_score = EXCLUDED.total_score,
			updated_at = NOW()
	`

	// Calculate total score (weighted average)
	// Weights: Flow 25%, Momentum 20%, Technical 20%, Value 15%, Quality 15%, Event 5%
	totalScore := signals.Flow*0.25 +
		signals.Momentum*0.20 +
		signals.Technical*0.20 +
		signals.Value*0.15 +
		signals.Quality*0.15 +
		signals.Event*0.05

	_, err := tx.Exec(ctx, query,
		code, date,
		signals.Momentum, signals.Technical, signals.Value,
		signals.Quality, signals.Flow, signals.Event,
		totalScore,
	)

	return err
}

// saveSignalDetails saves signal details to appropriate tables
func (r *SignalRepository) saveSignalDetails(ctx context.Context, tx pgx.Tx, code string, date time.Time, signals *contracts.StockSignals) error {
	// Save flow details
	if signals.Details.ForeignNet5D != 0 || signals.Details.InstNet5D != 0 {
		flowQuery := `
			INSERT INTO signals.flow_details (
				stock_code, calc_date,
				foreign_net_5d, foreign_net_20d,
				inst_net_5d, inst_net_20d
			) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (stock_code, calc_date) DO UPDATE SET
				foreign_net_5d = EXCLUDED.foreign_net_5d,
				foreign_net_20d = EXCLUDED.foreign_net_20d,
				inst_net_5d = EXCLUDED.inst_net_5d,
				inst_net_20d = EXCLUDED.inst_net_20d,
				updated_at = NOW()
		`

		_, err := tx.Exec(ctx, flowQuery,
			code, date,
			signals.Details.ForeignNet5D, signals.Details.ForeignNet20D,
			signals.Details.InstNet5D, signals.Details.InstNet20D,
		)
		if err != nil {
			return fmt.Errorf("failed to save flow details: %w", err)
		}
	}

	// Save technical details
	if signals.Details.RSI != 0 || signals.Details.MACD != 0 {
		techQuery := `
			INSERT INTO signals.technical_details (
				stock_code, calc_date,
				rsi14, macd, macd_signal
			) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (stock_code, calc_date) DO UPDATE SET
				rsi14 = EXCLUDED.rsi14,
				macd = EXCLUDED.macd,
				macd_signal = EXCLUDED.macd_signal,
				updated_at = NOW()
		`

		_, err := tx.Exec(ctx, techQuery,
			code, date,
			signals.Details.RSI, signals.Details.MACD, 0.0, // macd_signal placeholder
		)
		if err != nil {
			return fmt.Errorf("failed to save technical details: %w", err)
		}
	}

	return nil
}

// loadSignalDetails loads signal details from database
func (r *SignalRepository) loadSignalDetails(ctx context.Context, code string, date time.Time, signals *contracts.StockSignals) error {
	// Load flow details
	flowQuery := `
		SELECT
			foreign_net_5d, foreign_net_20d,
			inst_net_5d, inst_net_20d
		FROM signals.flow_details
		WHERE stock_code = $1 AND calc_date = $2
	`

	err := r.pool.QueryRow(ctx, flowQuery, code, date).Scan(
		&signals.Details.ForeignNet5D, &signals.Details.ForeignNet20D,
		&signals.Details.InstNet5D, &signals.Details.InstNet20D,
	)
	if err != nil && err.Error() != "no rows in result set" {
		return fmt.Errorf("failed to load flow details: %w", err)
	}

	// Load technical details
	techQuery := `
		SELECT
			rsi14, macd
		FROM signals.technical_details
		WHERE stock_code = $1 AND calc_date = $2
	`

	err = r.pool.QueryRow(ctx, techQuery, code, date).Scan(
		&signals.Details.RSI, &signals.Details.MACD,
	)
	if err != nil && err.Error() != "no rows in result set" {
		return fmt.Errorf("failed to load technical details: %w", err)
	}

	return nil
}
