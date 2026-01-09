package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/s0_data/quality"
	"github.com/wonny/aegis/v13/backend/internal/s1_universe"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// UniverseJob generates the universe daily
// ⭐ SSOT: Universe 생성 스케줄은 이 Job에서만
type UniverseJob struct {
	builder     *s1_universe.Builder
	qualityGate *quality.QualityGate
	logger      *logger.Logger
}

// NewUniverseJob creates a new universe job
func NewUniverseJob(builder *s1_universe.Builder, qg *quality.QualityGate, log *logger.Logger) *UniverseJob {
	return &UniverseJob{
		builder:     builder,
		qualityGate: qg,
		logger:      log,
	}
}

// Name returns the job name
func (j *UniverseJob) Name() string {
	return "universe_generation"
}

// Schedule returns the cron schedule (every day at 6 PM KST, after data collection)
func (j *UniverseJob) Schedule() string {
	return "0 0 18 * * *" // 6 PM daily (with seconds)
}

// Run executes the universe generation
func (j *UniverseJob) Run(ctx context.Context) error {
	j.logger.Info("Starting scheduled universe generation")

	// 1. Validate data quality
	j.logger.Info("Validating data quality")
	snapshot, err := j.qualityGate.Check(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("quality validation failed: %w", err)
	}

	if !snapshot.IsValid() {
		j.logger.WithFields(map[string]interface{}{
			"quality_score":  snapshot.QualityScore,
			"total_stocks":   snapshot.TotalStocks,
			"valid_stocks":   snapshot.ValidStocks,
		}).Warn("Data quality below threshold, but continuing with universe generation")
	}

	// 2. Build universe
	j.logger.Info("Building universe")
	universe, err := j.builder.Build(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("build universe: %w", err)
	}

	j.logger.WithFields(map[string]interface{}{
		"total_count":    universe.TotalCount,
		"included_count": len(universe.Stocks),
		"excluded_count": len(universe.Excluded),
	}).Info("Universe generated successfully")

	return nil
}
