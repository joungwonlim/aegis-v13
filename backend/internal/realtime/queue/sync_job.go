package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/realtime"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// SyncJob represents a price synchronization job
type SyncJob struct {
	ID        int64     `json:"id"`
	StockCode string    `json:"stock_code"`
	Price     int64     `json:"price"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"` // pending, processing, done, failed
	Retries   int       `json:"retries"`
	CreatedAt time.Time `json:"created_at"`
}

// SyncQueue manages price synchronization to PostgreSQL
// ⭐ SSOT: 실시간 가격 DB 동기화는 이 구조체에서만
type SyncQueue struct {
	db         *pgxpool.Pool
	batchSize  int
	interval   time.Duration
	maxRetries int
	logger     *logger.Logger
	stopCh     chan struct{}
}

// NewSyncQueue creates a new sync queue
func NewSyncQueue(db *pgxpool.Pool, log *logger.Logger) *SyncQueue {
	return &SyncQueue{
		db:         db,
		batchSize:  100,
		interval:   1 * time.Second,
		maxRetries: 3,
		logger:     log,
		stopCh:     make(chan struct{}),
	}
}

// Enqueue adds a price tick to the sync queue
func (q *SyncQueue) Enqueue(ctx context.Context, tick *realtime.PriceTick) error {
	query := `
		INSERT INTO realtime.sync_jobs (
			stock_code, price, volume, timestamp, status, retries, created_at
		) VALUES ($1, $2, $3, $4, 'pending', 0, NOW())
	`

	_, err := q.db.Exec(ctx, query,
		tick.Code,
		tick.Price,
		tick.Volume,
		tick.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("enqueue sync job: %w", err)
	}

	return nil
}

// EnqueueBatch adds multiple price ticks to the sync queue
func (q *SyncQueue) EnqueueBatch(ctx context.Context, ticks []*realtime.PriceTick) error {
	if len(ticks) == 0 {
		return nil
	}

	tx, err := q.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO realtime.sync_jobs (
			stock_code, price, volume, timestamp, status, retries, created_at
		) VALUES ($1, $2, $3, $4, 'pending', 0, NOW())
	`

	for _, tick := range ticks {
		_, err := tx.Exec(ctx, query,
			tick.Code,
			tick.Price,
			tick.Volume,
			tick.Timestamp,
		)
		if err != nil {
			return fmt.Errorf("enqueue sync job for %s: %w", tick.Code, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	q.logger.WithField("count", len(ticks)).Debug("Enqueued sync jobs")
	return nil
}

// Start starts the sync worker
func (q *SyncQueue) Start(ctx context.Context) {
	q.logger.Info("Starting sync queue worker")

	ticker := time.NewTicker(q.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			q.logger.Info("Sync queue worker stopped (context cancelled)")
			return
		case <-q.stopCh:
			q.logger.Info("Sync queue worker stopped")
			return
		case <-ticker.C:
			if err := q.processBatch(ctx); err != nil {
				q.logger.WithError(err).Error("Failed to process sync batch")
			}
		}
	}
}

// Stop stops the sync worker
func (q *SyncQueue) Stop() {
	close(q.stopCh)
}

// processBatch processes a batch of pending jobs
func (q *SyncQueue) processBatch(ctx context.Context) error {
	// Get pending jobs
	query := `
		SELECT id, stock_code, price, volume, timestamp, retries
		FROM realtime.sync_jobs
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := q.db.Query(ctx, query, q.batchSize)
	if err != nil {
		return fmt.Errorf("query pending jobs: %w", err)
	}
	defer rows.Close()

	var jobs []SyncJob
	for rows.Next() {
		var job SyncJob
		if err := rows.Scan(
			&job.ID,
			&job.StockCode,
			&job.Price,
			&job.Volume,
			&job.Timestamp,
			&job.Retries,
		); err != nil {
			return fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows error: %w", err)
	}

	if len(jobs) == 0 {
		return nil
	}

	// Process jobs
	for _, job := range jobs {
		if err := q.processJob(ctx, job); err != nil {
			q.logger.WithError(err).WithFields(map[string]interface{}{
				"job_id":     job.ID,
				"stock_code": job.StockCode,
				"retries":    job.Retries,
			}).Error("Failed to process sync job")

			// Update retry count or mark as failed
			if job.Retries >= q.maxRetries {
				q.markFailed(ctx, job.ID)
			} else {
				q.incrementRetry(ctx, job.ID)
			}
		} else {
			// Mark as done
			q.markDone(ctx, job.ID)
		}
	}

	q.logger.WithField("count", len(jobs)).Debug("Processed sync batch")
	return nil
}

// processJob processes a single sync job
func (q *SyncQueue) processJob(ctx context.Context, job SyncJob) error {
	// Insert into price_ticks table
	query := `
		INSERT INTO realtime.price_ticks (
			stock_code, price, volume, timestamp, created_at
		) VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (stock_code, timestamp) DO UPDATE SET
			price = EXCLUDED.price,
			volume = EXCLUDED.volume,
			updated_at = NOW()
	`

	_, err := q.db.Exec(ctx, query,
		job.StockCode,
		job.Price,
		job.Volume,
		job.Timestamp,
	)

	return err
}

// markDone marks a job as done
func (q *SyncQueue) markDone(ctx context.Context, jobID int64) error {
	query := `UPDATE realtime.sync_jobs SET status = 'done' WHERE id = $1`
	_, err := q.db.Exec(ctx, query, jobID)
	return err
}

// markFailed marks a job as failed
func (q *SyncQueue) markFailed(ctx context.Context, jobID int64) error {
	query := `UPDATE realtime.sync_jobs SET status = 'failed' WHERE id = $1`
	_, err := q.db.Exec(ctx, query, jobID)
	return err
}

// incrementRetry increments retry count
func (q *SyncQueue) incrementRetry(ctx context.Context, jobID int64) error {
	query := `UPDATE realtime.sync_jobs SET retries = retries + 1 WHERE id = $1`
	_, err := q.db.Exec(ctx, query, jobID)
	return err
}

// GetStats returns queue statistics
func (q *SyncQueue) GetStats(ctx context.Context) (*QueueStats, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'processing') as processing,
			COUNT(*) FILTER (WHERE status = 'done') as done,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) as total
		FROM realtime.sync_jobs
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`

	var stats QueueStats
	err := q.db.QueryRow(ctx, query).Scan(
		&stats.Pending,
		&stats.Processing,
		&stats.Done,
		&stats.Failed,
		&stats.Total,
	)

	if err != nil {
		return nil, fmt.Errorf("get queue stats: %w", err)
	}

	return &stats, nil
}

// QueueStats represents queue statistics
type QueueStats struct {
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Done       int `json:"done"`
	Failed     int `json:"failed"`
	Total      int `json:"total"`
}
