package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog"

	"github.com/wonny/aegis/v13/backend/internal/risk"
)

// =============================================================================
// Risk Reporter (S7용)
// =============================================================================

// RiskReporter S7 리스크 리포트 생성기
// ⭐ SSOT: 리스크 리포팅은 여기서만
type RiskReporter struct {
	engine *risk.Engine
	repo   *Repository
	log    zerolog.Logger
}

// NewRiskReporter 새 리스크 리포터 생성
func NewRiskReporter(engine *risk.Engine, repo *Repository, log zerolog.Logger) *RiskReporter {
	return &RiskReporter{
		engine: engine,
		repo:   repo,
		log:    log.With().Str("component", "audit.risk_reporter").Logger(),
	}
}

// =============================================================================
// Report Types
// =============================================================================

// RiskReport S7 리스크 리포트
type RiskReport struct {
	ReportDate time.Time               `json:"report_date"`
	RunID      string                  `json:"run_id"`
	Portfolio  *PortfolioRiskSummary   `json:"portfolio"`
	MonteCarlo *risk.MonteCarloResult  `json:"monte_carlo,omitempty"`
	Accuracy   *risk.AccuracyReport    `json:"forecast_accuracy,omitempty"`
	Stress     map[string]float64      `json:"stress_test,omitempty"`
	Metadata   ReportMetadata          `json:"metadata"`
}

// PortfolioRiskSummary 포트폴리오 리스크 요약
type PortfolioRiskSummary struct {
	SampleCount int     `json:"sample_count"`
	VaR95       float64 `json:"var_95"`
	VaR99       float64 `json:"var_99"`
	CVaR95      float64 `json:"cvar_95"`
	CVaR99      float64 `json:"cvar_99"`
	MeanReturn  float64 `json:"mean_return"`
	StdDev      float64 `json:"std_dev"`
	MaxDrawdown float64 `json:"max_drawdown"`
}

// ReportMetadata 리포트 메타데이터
type ReportMetadata struct {
	GeneratedAt    time.Time              `json:"generated_at"`
	DataFrom       time.Time              `json:"data_from"`
	DataTo         time.Time              `json:"data_to"`
	SampleCount    int                    `json:"sample_count"`
	MonteCarloConf *risk.MonteCarloConfig `json:"monte_carlo_config,omitempty"`
}

// =============================================================================
// Report Generation
// =============================================================================

// RiskReportInput 리스크 리포트 생성 입력
// ⭐ 데이터 조립은 호출자(S7)에서, 계산은 RiskEngine에서
type RiskReportInput struct {
	RunID                  string
	PortfolioReturns       []float64           // 포트폴리오 일별 수익률
	Weights                map[string]float64  // 종목별 비중
	ForecastValidations    []risk.ValidationResult
	MonteCarloConfig       *risk.MonteCarloConfig
	DecisionSnapshotID     *int64
}

// GenerateReport 전체 리스크 리포트 생성
func (r *RiskReporter) GenerateReport(ctx context.Context, input RiskReportInput) (*RiskReport, error) {
	report := &RiskReport{
		ReportDate: time.Now(),
		RunID:      input.RunID,
		Metadata: ReportMetadata{
			GeneratedAt: time.Now(),
		},
	}

	// 1. 포트폴리오 기본 리스크 계산
	if len(input.PortfolioReturns) > 0 {
		portfolioRisk := r.calculatePortfolioRisk(input.PortfolioReturns)
		report.Portfolio = portfolioRisk
		report.Metadata.SampleCount = len(input.PortfolioReturns)
	}

	// 2. Monte Carlo 시뮬레이션 (선택적)
	if input.MonteCarloConfig != nil && len(input.PortfolioReturns) >= 30 {
		mcResult, err := r.engine.SimulateSimple(ctx, input.PortfolioReturns)
		if err != nil {
			r.log.Warn().Err(err).Msg("Monte Carlo simulation failed")
		} else {
			report.MonteCarlo = mcResult
			report.Metadata.MonteCarloConf = input.MonteCarloConfig
		}
	}

	// 3. Forecast 정확도 계산 (선택적)
	if len(input.ForecastValidations) > 0 {
		accuracy := r.calculateAccuracy(input.ForecastValidations)
		report.Accuracy = accuracy
	}

	// 4. 스트레스 테스트 (선택적) - 향후 구현
	// TODO: Stress test scenarios

	r.log.Info().
		Str("run_id", input.RunID).
		Int("sample_count", len(input.PortfolioReturns)).
		Msg("risk report generated")

	return report, nil
}

// =============================================================================
// Internal Calculations
// =============================================================================

func (r *RiskReporter) calculatePortfolioRisk(returns []float64) *PortfolioRiskSummary {
	if len(returns) == 0 {
		return nil
	}

	// VaR/CVaR 계산 (RiskEngine 사용)
	var95 := r.engine.CalculateVaR(returns, 0.95)
	var99 := r.engine.CalculateVaR(returns, 0.99)

	// 기본 통계 (risk 패키지 함수 사용)
	mean := risk.CalculateMean(returns)
	stdDev := risk.CalculateVolatility(returns)

	// Maximum Drawdown 계산
	mdd := r.calculateMaxDrawdown(returns)

	return &PortfolioRiskSummary{
		SampleCount: len(returns),
		VaR95:       var95.VaR,
		VaR99:       var99.VaR,
		CVaR95:      var95.CVaR,
		CVaR99:      var99.CVaR,
		MeanReturn:  mean,
		StdDev:      stdDev,
		MaxDrawdown: mdd,
	}
}

func (r *RiskReporter) calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 누적 수익률 계산
	cumReturn := 1.0
	peak := 1.0
	maxDD := 0.0

	for _, ret := range returns {
		cumReturn *= (1 + ret)
		if cumReturn > peak {
			peak = cumReturn
		}
		drawdown := (peak - cumReturn) / peak
		if drawdown > maxDD {
			maxDD = drawdown
		}
	}

	return maxDD
}

func (r *RiskReporter) calculateAccuracy(validations []risk.ValidationResult) *risk.AccuracyReport {
	if len(validations) == 0 {
		return nil
	}

	var sumAbsError, sumSqError, sumError float64
	var hitCount int

	for _, val := range validations {
		sumAbsError += val.AbsError
		sumSqError += val.Error * val.Error
		sumError += val.Error
		if val.DirectionHit {
			hitCount++
		}
	}

	n := float64(len(validations))
	sqrtSumSq := 0.0
	if n > 0 {
		sqrtSumSq = sumSqError / n
	}

	return &risk.AccuracyReport{
		Level:       "PORTFOLIO",
		Key:         "ALL",
		SampleCount: len(validations),
		MAE:         sumAbsError / n,
		RMSE:        math.Sqrt(sqrtSumSq),
		HitRate:     float64(hitCount) / n,
		MeanError:   sumError / n,
		UpdatedAt:   time.Now(),
	}
}

// =============================================================================
// Output Formatting
// =============================================================================

// ToJSON JSON 형식으로 출력
func (report *RiskReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// ToSummary 요약 문자열 출력
func (report *RiskReport) ToSummary() string {
	var summary string

	summary += fmt.Sprintf("=== Risk Report (%s) ===\n", report.ReportDate.Format("2006-01-02"))
	summary += fmt.Sprintf("Run ID: %s\n\n", report.RunID)

	if report.Portfolio != nil {
		summary += "📊 Portfolio Risk\n"
		summary += fmt.Sprintf("  Samples: %d\n", report.Portfolio.SampleCount)
		summary += fmt.Sprintf("  VaR 95%%: %.4f (%.2f%%)\n", report.Portfolio.VaR95, report.Portfolio.VaR95*100)
		summary += fmt.Sprintf("  VaR 99%%: %.4f (%.2f%%)\n", report.Portfolio.VaR99, report.Portfolio.VaR99*100)
		summary += fmt.Sprintf("  CVaR 95%%: %.4f (%.2f%%)\n", report.Portfolio.CVaR95, report.Portfolio.CVaR95*100)
		summary += fmt.Sprintf("  Max Drawdown: %.4f (%.2f%%)\n", report.Portfolio.MaxDrawdown, report.Portfolio.MaxDrawdown*100)
		summary += "\n"
	}

	if report.MonteCarlo != nil {
		summary += "🎲 Monte Carlo Simulation\n"
		summary += fmt.Sprintf("  Simulations: %d\n", report.MonteCarlo.Config.NumSimulations)
		summary += fmt.Sprintf("  Holding Period: %d days\n", report.MonteCarlo.Config.HoldingPeriod)
		summary += fmt.Sprintf("  Mean Return: %.4f\n", report.MonteCarlo.MeanReturn)
		summary += fmt.Sprintf("  Std Dev: %.4f\n", report.MonteCarlo.StdDev)
		summary += fmt.Sprintf("  MC VaR 95%%: %.4f\n", report.MonteCarlo.VaR95)
		summary += fmt.Sprintf("  MC CVaR 95%%: %.4f\n", report.MonteCarlo.CVaR95)
		summary += "\n"
	}

	if report.Accuracy != nil {
		summary += "🎯 Forecast Accuracy\n"
		summary += fmt.Sprintf("  Samples: %d\n", report.Accuracy.SampleCount)
		summary += fmt.Sprintf("  MAE: %.4f\n", report.Accuracy.MAE)
		summary += fmt.Sprintf("  RMSE: %.4f\n", report.Accuracy.RMSE)
		summary += fmt.Sprintf("  Hit Rate: %.2f%%\n", report.Accuracy.HitRate*100)
		summary += "\n"
	}

	if len(report.Stress) > 0 {
		summary += "⚠️ Stress Test Results\n"
		for scenario, loss := range report.Stress {
			summary += fmt.Sprintf("  %s: %.4f (%.2f%%)\n", scenario, loss, loss*100)
		}
	}

	return summary
}

// =============================================================================
// Report Persistence
// =============================================================================

// SaveMonteCarloResult Monte Carlo 결과 저장
func (r *RiskReporter) SaveMonteCarloResult(ctx context.Context, result *risk.MonteCarloResult) error {
	configJSON, err := json.Marshal(result.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	percentilesJSON, err := json.Marshal(result.Percentiles)
	if err != nil {
		return fmt.Errorf("failed to marshal percentiles: %w", err)
	}

	query := `
		INSERT INTO analytics.montecarlo_results (
			run_id, run_date, config,
			mean_return, std_dev,
			var_95, var_99, cvar_95, cvar_99, percentiles
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (run_id) DO UPDATE SET
			config = EXCLUDED.config,
			mean_return = EXCLUDED.mean_return,
			var_95 = EXCLUDED.var_95
	`

	_, err = r.repo.pool.Exec(ctx, query,
		result.RunID, result.RunDate, configJSON,
		result.MeanReturn, result.StdDev,
		result.VaR95, result.VaR99, result.CVaR95, result.CVaR99, percentilesJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to save monte carlo result: %w", err)
	}

	return nil
}

// SaveForecastValidation Forecast 검증 결과 저장
func (r *RiskReporter) SaveForecastValidation(ctx context.Context, result *risk.ValidationResult) error {
	query := `
		INSERT INTO analytics.forecast_validations (
			event_id, model_version, code, event_type,
			predicted_ret, actual_ret, error, abs_error,
			direction_hit, validated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (event_id, model_version) DO UPDATE SET
			predicted_ret = EXCLUDED.predicted_ret,
			actual_ret = EXCLUDED.actual_ret,
			error = EXCLUDED.error,
			direction_hit = EXCLUDED.direction_hit,
			validated_at = EXCLUDED.validated_at
	`

	_, err := r.repo.pool.Exec(ctx, query,
		result.EventID, result.ModelVersion, result.Code, result.EventType,
		result.PredictedRet, result.ActualRet, result.Error, result.AbsError,
		result.DirectionHit, result.ValidatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save forecast validation: %w", err)
	}

	return nil
}

// SaveAccuracyReport Accuracy 리포트 저장
func (r *RiskReporter) SaveAccuracyReport(ctx context.Context, report *risk.AccuracyReport) error {
	query := `
		INSERT INTO analytics.accuracy_reports (
			model_version, level, key, event_type,
			sample_count, mae, rmse, hit_rate, mean_error, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (model_version, level, key, event_type) DO UPDATE SET
			sample_count = EXCLUDED.sample_count,
			mae = EXCLUDED.mae,
			rmse = EXCLUDED.rmse,
			hit_rate = EXCLUDED.hit_rate,
			mean_error = EXCLUDED.mean_error,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.repo.pool.Exec(ctx, query,
		report.ModelVersion, report.Level, report.Key, report.EventType,
		report.SampleCount, report.MAE, report.RMSE, report.HitRate, report.MeanError, report.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save accuracy report: %w", err)
	}

	return nil
}
