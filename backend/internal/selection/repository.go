package selection

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// Repository handles selection data persistence
// ⭐ SSOT: Selection 데이터 저장/조회는 여기서만
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new selection repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SaveScreeningResult saves screening result to database
func (r *Repository) SaveScreeningResult(ctx context.Context, date time.Time, passed []string, filtered map[string]int, totalInput int) error {
	filteredJSON, err := json.Marshal(filtered)
	if err != nil {
		return fmt.Errorf("failed to marshal filtered: %w", err)
	}

	passedArray := make([]string, len(passed))
	copy(passedArray, passed)

	query := `
		INSERT INTO selection.screening_results (
			screen_date, passed_stocks, filtered, total_input, total_passed
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (screen_date) DO UPDATE SET
			passed_stocks = EXCLUDED.passed_stocks,
			filtered = EXCLUDED.filtered,
			total_input = EXCLUDED.total_input,
			total_passed = EXCLUDED.total_passed,
			created_at = NOW()
	`

	_, err = r.pool.Exec(ctx, query, date, passedArray, filteredJSON, totalInput, len(passed))
	if err != nil {
		return fmt.Errorf("failed to save screening result: %w", err)
	}

	return nil
}

// SaveRankingResults saves ranking results to database
func (r *Repository) SaveRankingResults(ctx context.Context, date time.Time, ranked []contracts.RankedStock) error {
	// Begin transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing results for the date
	_, err = tx.Exec(ctx, "DELETE FROM selection.ranking_results WHERE rank_date = $1", date)
	if err != nil {
		return fmt.Errorf("failed to delete old results: %w", err)
	}

	// Insert new results
	query := `
		INSERT INTO selection.ranking_results (
			stock_code, rank_date, rank, total_score,
			momentum, technical, value, quality, flow, event
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	for _, r := range ranked {
		_, err := tx.Exec(ctx, query,
			r.Code, date, r.Rank, r.TotalScore,
			r.Scores.Momentum, r.Scores.Technical, r.Scores.Value,
			r.Scores.Quality, r.Scores.Flow, r.Scores.Event,
		)
		if err != nil {
			return fmt.Errorf("failed to insert ranking result: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetScreeningResult retrieves screening result for a date
func (r *Repository) GetScreeningResult(ctx context.Context, date time.Time) (*ScreeningResult, error) {
	query := `
		SELECT passed_stocks, filtered, total_input, total_passed, created_at
		FROM selection.screening_results
		WHERE screen_date = $1
	`

	var result ScreeningResult
	var passedArray []string
	var filteredJSON []byte

	err := r.pool.QueryRow(ctx, query, date).Scan(
		&passedArray, &filteredJSON, &result.TotalInput, &result.TotalPassed, &result.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("no screening result found for date %s", date.Format("2006-01-02"))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get screening result: %w", err)
	}

	result.Date = date
	result.Passed = passedArray

	if err := json.Unmarshal(filteredJSON, &result.Filtered); err != nil {
		return nil, fmt.Errorf("failed to unmarshal filtered: %w", err)
	}

	return &result, nil
}

// GetRankingResults retrieves ranking results for a date
func (r *Repository) GetRankingResults(ctx context.Context, date time.Time, limit int) ([]contracts.RankedStock, error) {
	query := `
		SELECT
			stock_code, rank, total_score,
			momentum, technical, value, quality, flow, event
		FROM selection.ranking_results
		WHERE rank_date = $1
		ORDER BY rank ASC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, date, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query ranking results: %w", err)
	}
	defer rows.Close()

	results := make([]contracts.RankedStock, 0)

	for rows.Next() {
		var r contracts.RankedStock
		err := rows.Scan(
			&r.Code, &r.Rank, &r.TotalScore,
			&r.Scores.Momentum, &r.Scores.Technical, &r.Scores.Value,
			&r.Scores.Quality, &r.Scores.Flow, &r.Scores.Event,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// ScreeningResult represents screening result
type ScreeningResult struct {
	Date        time.Time
	Passed      []string
	Filtered    map[string]int
	TotalInput  int
	TotalPassed int
	CreatedAt   time.Time
}
