package brain

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/audit"
	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/internal/execution"
	"github.com/wonny/aegis/v13/backend/internal/portfolio"
	"github.com/wonny/aegis/v13/backend/internal/s0_data/quality"
	"github.com/wonny/aegis/v13/backend/internal/s1_universe"
	"github.com/wonny/aegis/v13/backend/internal/s2_signals"
	"github.com/wonny/aegis/v13/backend/internal/selection"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Orchestrator coordinates the entire 7-stage pipeline
// ⭐ SSOT: 파이프라인 조율은 여기서만
type Orchestrator struct {
	// Stage components
	qualityGate       *quality.QualityGate
	universeBuilder   *s1_universe.Builder
	signalBuilder     *s2_signals.Builder
	screener          *selection.Screener
	ranker            *selection.Ranker
	portfolioBuilder  *portfolio.Constructor
	executionPlanner    *execution.Planner
	performanceAnalyzer *audit.Analyzer

	// Repositories for saving intermediate results
	qualityRepo  *quality.Repository
	universeRepo *s1_universe.Repository
	signalRepo   *s2_signals.SignalRepository
	selectionRepo  *selection.Repository
	portfolioRepo  *portfolio.Repository
	executionRepo  *execution.Repository
	auditRepo      *audit.Repository

	logger *logger.Logger
}

// RunConfig holds configuration for a pipeline run
type RunConfig struct {
	Date           time.Time
	RunID          string
	GitSHA         string
	FeatureVersion string
	Capital        int64 // Available capital
	DryRun         bool  // If true, skip execution stage
}

// RunResult holds the results of a complete pipeline run
type RunResult struct {
	RunID              string
	Date               time.Time
	GitSHA             string
	FeatureVersion     string
	Success            bool
	Error              error
	CompletedStages    []string
	QualitySnapshot    *contracts.DataQualitySnapshot
	Universe           *contracts.Universe
	SignalSet          *contracts.SignalSet
	ScreenedStocks     []string
	RankedStocks       []contracts.RankedStock
	TargetPortfolio    *contracts.TargetPortfolio
	ExecutionPlan      *contracts.ExecutionPlan
	PerformanceReport  *audit.PerformanceReport
	Duration           time.Duration
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(
	qualityGate *quality.QualityGate,
	universeBuilder *s1_universe.Builder,
	signalBuilder *s2_signals.Builder,
	screener *selection.Screener,
	ranker *selection.Ranker,
	portfolioBuilder *portfolio.Constructor,
	executionPlanner *execution.Planner,
	performanceAnalyzer *audit.Analyzer,
	qualityRepo *quality.Repository,
	universeRepo *s1_universe.Repository,
	signalRepo *s2_signals.SignalRepository,
	selectionRepo *selection.Repository,
	portfolioRepo *portfolio.Repository,
	executionRepo *execution.Repository,
	auditRepo *audit.Repository,
	logger *logger.Logger,
) *Orchestrator {
	return &Orchestrator{
		qualityGate:         qualityGate,
		universeBuilder:     universeBuilder,
		signalBuilder:       signalBuilder,
		screener:            screener,
		ranker:              ranker,
		portfolioBuilder:    portfolioBuilder,
		executionPlanner:    executionPlanner,
		performanceAnalyzer: performanceAnalyzer,
		qualityRepo:         qualityRepo,
		universeRepo:        universeRepo,
		signalRepo:          signalRepo,
		selectionRepo:       selectionRepo,
		portfolioRepo:       portfolioRepo,
		executionRepo:       executionRepo,
		auditRepo:           auditRepo,
		logger:              logger,
	}
}

// Run executes the complete 7-stage pipeline
// S0 → S1 → S2 → S3 → S4 → S5 → S6 → S7
func (o *Orchestrator) Run(ctx context.Context, config RunConfig) (*RunResult, error) {
	startTime := time.Now()

	result := &RunResult{
		RunID:           config.RunID,
		Date:            config.Date,
		GitSHA:          config.GitSHA,
		FeatureVersion:  config.FeatureVersion,
		Success:         false,
		CompletedStages: make([]string, 0),
	}

	o.logger.WithFields(map[string]interface{}{
		"run_id":          config.RunID,
		"date":            config.Date.Format("2006-01-02"),
		"git_sha":         config.GitSHA,
		"feature_version": config.FeatureVersion,
		"capital":         config.Capital,
		"dry_run":         config.DryRun,
	}).Info("Starting pipeline run")

	// S0: Data Quality Gate
	qualitySnapshot, err := o.runS0(ctx, config)
	if err != nil {
		result.Error = fmt.Errorf("S0 failed: %w", err)
		return result, result.Error
	}
	result.QualitySnapshot = qualitySnapshot
	result.CompletedStages = append(result.CompletedStages, "S0:Quality")

	// S1: Universe Generation
	universe, err := o.runS1(ctx, config, qualitySnapshot)
	if err != nil {
		result.Error = fmt.Errorf("S1 failed: %w", err)
		return result, result.Error
	}
	result.Universe = universe
	result.CompletedStages = append(result.CompletedStages, "S1:Universe")

	// S2: Signal Generation
	signalSet, err := o.runS2(ctx, config, universe)
	if err != nil {
		result.Error = fmt.Errorf("S2 failed: %w", err)
		return result, result.Error
	}
	result.SignalSet = signalSet
	result.CompletedStages = append(result.CompletedStages, "S2:Signals")

	// S3: Screening
	screened, err := o.runS3(ctx, config, universe.Stocks, signalSet)
	if err != nil {
		result.Error = fmt.Errorf("S3 failed: %w", err)
		return result, result.Error
	}
	result.ScreenedStocks = screened
	result.CompletedStages = append(result.CompletedStages, "S3:Screener")

	// S4: Ranking
	ranked, err := o.runS4(ctx, config, screened, signalSet)
	if err != nil {
		result.Error = fmt.Errorf("S4 failed: %w", err)
		return result, result.Error
	}
	result.RankedStocks = ranked
	result.CompletedStages = append(result.CompletedStages, "S4:Ranker")

	// S5: Portfolio Construction
	targetPortfolio, err := o.runS5(ctx, config, ranked, config.Capital)
	if err != nil {
		result.Error = fmt.Errorf("S5 failed: %w", err)
		return result, result.Error
	}
	result.TargetPortfolio = targetPortfolio
	result.CompletedStages = append(result.CompletedStages, "S5:Portfolio")

	// S6: Execution Planning (skip if dry run)
	if !config.DryRun {
		executionPlan, err := o.runS6(ctx, config, targetPortfolio)
		if err != nil {
			result.Error = fmt.Errorf("S6 failed: %w", err)
			return result, result.Error
		}
		result.ExecutionPlan = executionPlan
		result.CompletedStages = append(result.CompletedStages, "S6:Execution")
	} else {
		o.logger.Info("Skipping S6:Execution (dry run mode)")
	}

	// S7: Performance Analysis
	performanceReport, err := o.runS7(ctx, config)
	if err != nil {
		result.Error = fmt.Errorf("S7 failed: %w", err)
		return result, result.Error
	}
	result.PerformanceReport = performanceReport
	result.CompletedStages = append(result.CompletedStages, "S7:Audit")

	// Mark success
	result.Success = true
	result.Duration = time.Since(startTime)

	o.logger.WithFields(map[string]interface{}{
		"run_id":   config.RunID,
		"duration": result.Duration.Seconds(),
		"stages":   len(result.CompletedStages),
	}).Info("Pipeline run completed successfully")

	return result, nil
}

// runS0 executes S0: Data Quality Gate
func (o *Orchestrator) runS0(ctx context.Context, config RunConfig) (*contracts.DataQualitySnapshot, error) {
	o.logger.Info("Running S0: Data Quality Gate")

	snapshot, err := o.qualityGate.Check(ctx, config.Date)
	if err != nil {
		return nil, fmt.Errorf("quality gate validation: %w", err)
	}

	if !snapshot.IsValid() {
		return nil, fmt.Errorf("quality gate failed: score=%.2f", snapshot.QualityScore)
	}

	// Save snapshot
	if err := o.qualityRepo.SaveSnapshot(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("save quality snapshot: %w", err)
	}

	o.logger.WithFields(map[string]interface{}{
		"quality_score": snapshot.QualityScore,
		"passed":        snapshot.Passed,
	}).Info("S0 completed")

	return snapshot, nil
}

// runS1 executes S1: Universe Generation
func (o *Orchestrator) runS1(ctx context.Context, config RunConfig, snapshot *contracts.DataQualitySnapshot) (*contracts.Universe, error) {
	o.logger.Info("Running S1: Universe Generation")

	universe, err := o.universeBuilder.Build(ctx, snapshot)
	if err != nil {
		return nil, fmt.Errorf("universe build: %w", err)
	}

	// Save universe
	if err := o.universeRepo.SaveUniverse(ctx, universe); err != nil {
		return nil, fmt.Errorf("save universe: %w", err)
	}

	o.logger.WithFields(map[string]interface{}{
		"total_stocks": universe.TotalCount,
		"excluded":     len(universe.Excluded),
	}).Info("S1 completed")

	return universe, nil
}

// runS2 executes S2: Signal Generation
func (o *Orchestrator) runS2(ctx context.Context, config RunConfig, universe *contracts.Universe) (*contracts.SignalSet, error) {
	o.logger.Info("Running S2: Signal Generation")

	signalSet, err := o.signalBuilder.Build(ctx, universe, config.Date)
	if err != nil {
		return nil, fmt.Errorf("signal build: %w", err)
	}

	// Save signals
	if err := o.signalRepo.Save(ctx, signalSet); err != nil {
		return nil, fmt.Errorf("save signal set: %w", err)
	}

	o.logger.WithFields(map[string]interface{}{
		"signals_generated": len(signalSet.Signals),
	}).Info("S2 completed")

	return signalSet, nil
}

// runS3 executes S3: Screening
func (o *Orchestrator) runS3(ctx context.Context, config RunConfig, stocks []string, signals *contracts.SignalSet) ([]string, error) {
	o.logger.Info("Running S3: Screening")

	screened, err := o.screener.Screen(ctx, signals)
	if err != nil {
		return nil, fmt.Errorf("screening: %w", err)
	}

	o.logger.WithFields(map[string]interface{}{
		"input_stocks":  len(stocks),
		"passed_stocks": len(screened),
	}).Info("S3 completed")

	return screened, nil
}

// runS4 executes S4: Ranking
func (o *Orchestrator) runS4(ctx context.Context, config RunConfig, stocks []string, signals *contracts.SignalSet) ([]contracts.RankedStock, error) {
	o.logger.Info("Running S4: Ranking")

	ranked, err := o.ranker.Rank(ctx, stocks, signals)
	if err != nil {
		return nil, fmt.Errorf("ranking: %w", err)
		}

	// Save ranking results
	if err := o.selectionRepo.SaveRankingResults(ctx, config.Date, ranked); err != nil {
		return nil, fmt.Errorf("save ranking: %w", err)
	}

	o.logger.WithFields(map[string]interface{}{
		"ranked_stocks": len(ranked),
		"top_code":      ranked[0].Code,
		"top_score":     ranked[0].TotalScore,
	}).Info("S4 completed")

	return ranked, nil
}

// runS5 executes S5: Portfolio Construction
func (o *Orchestrator) runS5(ctx context.Context, config RunConfig, ranked []contracts.RankedStock, capital int64) (*contracts.TargetPortfolio, error) {
	o.logger.Info("Running S5: Portfolio Construction")

	// Build target portfolio using Constructor.Construct
	targetPortfolio, err := o.portfolioBuilder.Construct(ctx, ranked)
	if err != nil {
		return nil, fmt.Errorf("portfolio construct: %w", err)
	}

	// Save target portfolio
	if err := o.portfolioRepo.SaveTargetPortfolio(ctx, targetPortfolio); err != nil {
		return nil, fmt.Errorf("save target portfolio: %w", err)
	}

	o.logger.WithFields(map[string]interface{}{
		"stocks":      len(targetPortfolio.Positions),
		"cash_target": targetPortfolio.Cash,
	}).Info("S5 completed")

	return targetPortfolio, nil
}

// runS6 executes S6: Execution Planning
func (o *Orchestrator) runS6(ctx context.Context, config RunConfig, targetPortfolio *contracts.TargetPortfolio) (*contracts.ExecutionPlan, error) {
	o.logger.Info("Running S6: Execution Planning")

	// Create execution plan using Planner.Plan
	orders, err := o.executionPlanner.Plan(ctx, targetPortfolio)
	if err != nil {
		return nil, fmt.Errorf("create execution plan: %w", err)
	}

	executionPlan := &contracts.ExecutionPlan{
		ID:        fmt.Sprintf("plan_%s", config.RunID),
		Date:      config.Date,
		Orders:    orders,
		CreatedAt: time.Now(),
	}

	// Save each order
	for i := range executionPlan.Orders {
		if err := o.executionRepo.SaveOrder(ctx, &executionPlan.Orders[i]); err != nil {
			return nil, fmt.Errorf("save order: %w", err)
		}
	}

	o.logger.WithFields(map[string]interface{}{
		"orders": len(executionPlan.Orders),
	}).Info("S6 completed")

	return executionPlan, nil
}

// runS7 executes S7: Performance Analysis
func (o *Orchestrator) runS7(ctx context.Context, config RunConfig) (*audit.PerformanceReport, error) {
	o.logger.Info("Running S7: Performance Analysis")

	// Analyze performance for the last month (period format: "1M")
	report, err := o.performanceAnalyzer.Analyze(ctx, "1M")
	if err != nil {
		return nil, fmt.Errorf("performance analysis: %w", err)
	}

	// Save performance report
	if err := o.auditRepo.SavePerformanceReport(ctx, report); err != nil {
		return nil, fmt.Errorf("save performance report: %w", err)
	}

	o.logger.WithFields(map[string]interface{}{
		"return": report.TotalReturn,
		"sharpe": report.Sharpe,
		"mdd":    report.MaxDrawdown,
	}).Info("S7 completed")

	return report, nil
}

// GenerateRunID generates a unique run ID
func GenerateRunID() string {
	return fmt.Sprintf("run_%s", time.Now().Format("20060102_150405"))
}
