package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/s0_data/collector"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// DataCollectionJob collects all data daily
// ⭐ SSOT: 데이터 수집 스케줄은 이 Job에서만
type DataCollectionJob struct {
	collector *collector.Collector
	config    *config.Config
	logger    *logger.Logger
}

// NewDataCollectionJob creates a new data collection job
func NewDataCollectionJob(col *collector.Collector, cfg *config.Config, log *logger.Logger) *DataCollectionJob {
	return &DataCollectionJob{
		collector: col,
		config:    cfg,
		logger:    log,
	}
}

// Name returns the job name
func (j *DataCollectionJob) Name() string {
	return "data_collection"
}

// Schedule returns the cron schedule (every day at 4 PM KST)
func (j *DataCollectionJob) Schedule() string {
	return "0 0 16 * * *" // 4 PM daily (with seconds)
}

// Run executes the data collection
func (j *DataCollectionJob) Run(ctx context.Context) error {
	j.logger.Info("Starting scheduled data collection")

	// Calculate date range (last 5 days)
	to := time.Now()
	from := to.AddDate(0, 0, -5)

	// 1. Collect prices
	j.logger.Info("Collecting prices")
	collectorCfg := collector.Config{Workers: 5}
	if _, err := j.collector.FetchAllPrices(ctx, from, to, collectorCfg); err != nil {
		return fmt.Errorf("fetch prices: %w", err)
	}

	// 2. Collect investor flow
	j.logger.Info("Collecting investor flow")
	if _, err := j.collector.FetchAllInvestorFlow(ctx, from, to, collectorCfg); err != nil {
		return fmt.Errorf("fetch investor flow: %w", err)
	}

	// 3. Collect market caps
	j.logger.Info("Collecting market caps")
	if err := j.collector.FetchMarketCaps(ctx); err != nil {
		return fmt.Errorf("fetch market caps: %w", err)
	}

	// 4. Collect KRX trends
	j.logger.Info("Collecting KRX trends")
	if err := j.collector.FetchMarketTrends(ctx); err != nil {
		return fmt.Errorf("fetch market trends: %w", err)
	}

	// 5. Collect DART disclosures (last 7 days)
	j.logger.Info("Collecting DART disclosures")
	dartFrom := to.AddDate(0, 0, -7)
	if err := j.collector.FetchDisclosures(ctx, dartFrom, to); err != nil {
		return fmt.Errorf("fetch disclosures: %w", err)
	}

	j.logger.Info("Scheduled data collection completed successfully")
	return nil
}

// PriceCollectionJob collects only price data
type PriceCollectionJob struct {
	collector *collector.Collector
	config    *config.Config
	logger    *logger.Logger
}

// NewPriceCollectionJob creates a new price collection job
func NewPriceCollectionJob(col *collector.Collector, cfg *config.Config, log *logger.Logger) *PriceCollectionJob {
	return &PriceCollectionJob{
		collector: col,
		config:    cfg,
		logger:    log,
	}
}

// Name returns the job name
func (j *PriceCollectionJob) Name() string {
	return "price_collection"
}

// Schedule returns the cron schedule (every hour during trading hours)
func (j *PriceCollectionJob) Schedule() string {
	return "0 0 9-15 * * MON-FRI" // Every hour from 9 AM to 3 PM on weekdays
}

// Run executes the price collection
func (j *PriceCollectionJob) Run(ctx context.Context) error {
	j.logger.Info("Starting scheduled price collection")

	// Collect today's prices
	to := time.Now()
	from := to.AddDate(0, 0, -1)

	collectorCfg := collector.Config{Workers: 5}
	if _, err := j.collector.FetchAllPrices(ctx, from, to, collectorCfg); err != nil {
		return fmt.Errorf("fetch prices: %w", err)
	}

	j.logger.Info("Scheduled price collection completed successfully")
	return nil
}

// InvestorFlowJob collects investor flow data
type InvestorFlowJob struct {
	collector *collector.Collector
	config    *config.Config
	logger    *logger.Logger
}

// NewInvestorFlowJob creates a new investor flow job
func NewInvestorFlowJob(col *collector.Collector, cfg *config.Config, log *logger.Logger) *InvestorFlowJob {
	return &InvestorFlowJob{
		collector: col,
		config:    cfg,
		logger:    log,
	}
}

// Name returns the job name
func (j *InvestorFlowJob) Name() string {
	return "investor_flow"
}

// Schedule returns the cron schedule (every day at 5 PM KST)
func (j *InvestorFlowJob) Schedule() string {
	return "0 0 17 * * *" // 5 PM daily (with seconds)
}

// Run executes the investor flow collection
func (j *InvestorFlowJob) Run(ctx context.Context) error {
	j.logger.Info("Starting scheduled investor flow collection")

	// Collect last 3 days
	to := time.Now()
	from := to.AddDate(0, 0, -3)

	collectorCfg := collector.Config{Workers: 5}
	if _, err := j.collector.FetchAllInvestorFlow(ctx, from, to, collectorCfg); err != nil {
		return fmt.Errorf("fetch investor flow: %w", err)
	}

	j.logger.Info("Scheduled investor flow collection completed successfully")
	return nil
}

// DisclosureJob collects DART disclosures
type DisclosureJob struct {
	collector *collector.Collector
	logger    *logger.Logger
}

// NewDisclosureJob creates a new disclosure job
func NewDisclosureJob(col *collector.Collector, log *logger.Logger) *DisclosureJob {
	return &DisclosureJob{
		collector: col,
		logger:    log,
	}
}

// Name returns the job name
func (j *DisclosureJob) Name() string {
	return "disclosure_collection"
}

// Schedule returns the cron schedule (every 6 hours)
func (j *DisclosureJob) Schedule() string {
	return "0 0 */6 * * *" // Every 6 hours
}

// Run executes the disclosure collection
func (j *DisclosureJob) Run(ctx context.Context) error {
	j.logger.Info("Starting scheduled disclosure collection")

	// Collect last 1 day
	to := time.Now()
	from := to.AddDate(0, 0, -1)

	if err := j.collector.FetchDisclosures(ctx, from, to); err != nil {
		return fmt.Errorf("fetch disclosures: %w", err)
	}

	j.logger.Info("Scheduled disclosure collection completed successfully")
	return nil
}
