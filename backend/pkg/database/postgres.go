package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v13/backend/pkg/config"
)

// DB wraps the pgxpool.Pool and provides additional functionality
// ⭐ SSOT: DB 연결은 이 패키지에서만 생성
type DB struct {
	Pool *pgxpool.Pool
}

// New creates a new database connection pool
// ⭐ SSOT: 유일하게 pgxpool.New()를 호출하는 함수
func New(cfg *config.Config) (*DB, error) {
	// Build connection pool config
	poolConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(cfg.Database.MaxConns)
	poolConfig.MinConns = int32(cfg.Database.MinConns)
	poolConfig.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Database.MaxConnIdleTime

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// Ping checks if the database is accessible
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// HealthCheck returns detailed health information about the database
func (db *DB) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Healthy:   false,
		Timestamp: time.Now(),
	}

	// Check connection
	start := time.Now()
	if err := db.Pool.Ping(ctx); err != nil {
		status.Error = err.Error()
		return status, err
	}
	status.ResponseTime = time.Since(start)

	// Get pool stats
	stats := db.Pool.Stat()
	status.Stats = PoolStats{
		AcquireCount:         stats.AcquireCount(),
		AcquireDuration:      stats.AcquireDuration(),
		AcquiredConns:        stats.AcquiredConns(),
		CanceledAcquireCount: stats.CanceledAcquireCount(),
		ConstructingConns:    stats.ConstructingConns(),
		EmptyAcquireCount:    stats.EmptyAcquireCount(),
		IdleConns:            stats.IdleConns(),
		MaxConns:             stats.MaxConns(),
		TotalConns:           stats.TotalConns(),
	}

	status.Healthy = true
	return status, nil
}

// HealthStatus represents the health status of the database
type HealthStatus struct {
	Healthy      bool          `json:"healthy"`
	Timestamp    time.Time     `json:"timestamp"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
	Stats        PoolStats     `json:"stats"`
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	AcquireCount         int64         `json:"acquire_count"`
	AcquireDuration      time.Duration `json:"acquire_duration"`
	AcquiredConns        int32         `json:"acquired_conns"`
	CanceledAcquireCount int64         `json:"canceled_acquire_count"`
	ConstructingConns    int32         `json:"constructing_conns"`
	EmptyAcquireCount    int64         `json:"empty_acquire_count"`
	IdleConns            int32         `json:"idle_conns"`
	MaxConns             int32         `json:"max_conns"`
	TotalConns           int32         `json:"total_conns"`
}

// Stats returns the current pool statistics
func (db *DB) Stats() PoolStats {
	stats := db.Pool.Stat()
	return PoolStats{
		AcquireCount:         stats.AcquireCount(),
		AcquireDuration:      stats.AcquireDuration(),
		AcquiredConns:        stats.AcquiredConns(),
		CanceledAcquireCount: stats.CanceledAcquireCount(),
		ConstructingConns:    stats.ConstructingConns(),
		EmptyAcquireCount:    stats.EmptyAcquireCount(),
		IdleConns:            stats.IdleConns(),
		MaxConns:             stats.MaxConns(),
		TotalConns:           stats.TotalConns(),
	}
}
