package strategyconfig

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"
)

// ValidationError 검증 실패 (프로그램 중단)
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Warning 권장 위반 (경고만)
type Warning struct {
	Code    string
	Message string
}

// Validate checks all required constraints
// 실패 시 error 반환 (프로그램 중단)
func Validate(cfg *Config) error {
	// === Meta ===
	if cfg.Meta.StrategyID == "" {
		return ValidationError{"meta.strategy_id", "required"}
	}
	if err := validateHHMM(cfg.Meta.DecisionTimeLocal); err != nil {
		return ValidationError{"meta.decision_time_local", err.Error()}
	}
	if err := validateHHMM(cfg.Meta.ExecutionWindow.Start); err != nil {
		return ValidationError{"meta.execution_window.start", err.Error()}
	}
	if err := validateHHMM(cfg.Meta.ExecutionWindow.End); err != nil {
		return ValidationError{"meta.execution_window.end", err.Error()}
	}

	// execution_window: start < end
	startTime, _ := time.Parse("15:04", cfg.Meta.ExecutionWindow.Start)
	endTime, _ := time.Parse("15:04", cfg.Meta.ExecutionWindow.End)
	if !startTime.Before(endTime) {
		return ValidationError{"meta.execution_window", "start must be before end"}
	}

	// === Universe ===
	if cfg.Universe.Filters.MarketcapMinKRW <= 0 {
		return ValidationError{"universe.filters.marketcap_min_krw", "must be > 0"}
	}
	if cfg.Universe.Filters.ADTV20MinKRW <= 0 {
		return ValidationError{"universe.filters.adtv20_min_krw", "must be > 0"}
	}
	if cfg.Universe.Filters.Spread.MaxPct <= 0 || cfg.Universe.Filters.Spread.MaxPct > 0.05 {
		return ValidationError{"universe.filters.spread.max_pct", "must be in (0, 0.05]"}
	}
	expectedFormula := "((ask1-bid1)/((ask1+bid1)/2))"
	if cfg.Universe.Filters.Spread.Formula != expectedFormula {
		return ValidationError{"universe.filters.spread.formula", fmt.Sprintf("must be '%s'", expectedFormula)}
	}

	// === Signals ===
	// score_range: min < max
	if cfg.Signals.Normalization.ScoreRangeMin >= cfg.Signals.Normalization.ScoreRangeMax {
		return ValidationError{"signals.normalization", "score_range_min must be < score_range_max"}
	}

	// lookbacks_days와 weights 배열 길이 일치 확인
	if len(cfg.Signals.Momentum.LookbacksDays) != len(cfg.Signals.Momentum.Weights) {
		return ValidationError{"signals.momentum", "lookbacks_days length must match weights length"}
	}
	if len(cfg.Signals.Flow.LookbacksDays) != len(cfg.Signals.Flow.Weights) {
		return ValidationError{"signals.flow", "lookbacks_days length must match weights length"}
	}

	if err := validateWeightsSum(cfg.Signals.Momentum.Weights, 1.0, 1e-6); err != nil {
		return ValidationError{"signals.momentum.weights", err.Error()}
	}
	if err := validateWeightsSum(cfg.Signals.Flow.Weights, 1.0, 1e-6); err != nil {
		return ValidationError{"signals.flow.weights", err.Error()}
	}

	// === Ranking ===
	if cfg.Ranking.WeightsPct.Sum() != 100 {
		return ValidationError{"ranking.weights_pct", fmt.Sprintf("must sum to 100, got %d", cfg.Ranking.WeightsPct.Sum())}
	}
	momTech := cfg.Ranking.WeightsPct.Momentum + cfg.Ranking.WeightsPct.Technical
	if momTech > cfg.Ranking.Constraints.MomentumPlusTechnicalMaxPct {
		return ValidationError{"ranking.constraints", fmt.Sprintf("momentum+technical=%d exceeds max=%d", momTech, cfg.Ranking.Constraints.MomentumPlusTechnicalMaxPct)}
	}

	// === Portfolio ===
	h := cfg.Portfolio.Holdings
	if h.Min > h.Target || h.Target > h.Max {
		return ValidationError{"portfolio.holdings", "must satisfy min <= target <= max"}
	}

	a := cfg.Portfolio.Allocation
	// 퍼센트 범위 검증 (0~1)
	if err := validatePctRange(a.CashTargetPct, "portfolio.allocation.cash_target_pct"); err != nil {
		return err
	}
	if err := validatePctRange(a.PositionMinPct, "portfolio.allocation.position_min_pct"); err != nil {
		return err
	}
	if err := validatePctRange(a.PositionMaxPct, "portfolio.allocation.position_max_pct"); err != nil {
		return err
	}
	if err := validatePctRange(a.SectorMaxPct, "portfolio.allocation.sector_max_pct"); err != nil {
		return err
	}
	if err := validatePctRange(a.TurnoverDailyMaxPct, "portfolio.allocation.turnover_daily_max_pct"); err != nil {
		return err
	}

	if a.PositionMinPct > a.PositionMaxPct {
		return ValidationError{"portfolio.allocation", "position_min_pct must be <= position_max_pct"}
	}
	if a.SectorMaxPct < a.PositionMaxPct {
		return ValidationError{"portfolio.allocation", "sector_max_pct must be >= position_max_pct"}
	}

	// Tier count == holdings.target
	w := cfg.Portfolio.Weighting
	if w.TotalCount() != h.Target {
		return ValidationError{"portfolio.weighting.tiers", fmt.Sprintf("count sum must equal holdings.target=%d, got %d", h.Target, w.TotalCount())}
	}

	// 각 tier가 position_min/max 범위 내인지 검증
	for i, tier := range w.Tiers {
		if tier.WeightEachPct < a.PositionMinPct {
			return ValidationError{
				Field:   fmt.Sprintf("portfolio.weighting.tiers[%d]", i),
				Message: fmt.Sprintf("weight_each_pct=%.4f < position_min_pct=%.4f", tier.WeightEachPct, a.PositionMinPct),
			}
		}
		if tier.WeightEachPct > a.PositionMaxPct {
			return ValidationError{
				Field:   fmt.Sprintf("portfolio.weighting.tiers[%d]", i),
				Message: fmt.Sprintf("weight_each_pct=%.4f > position_max_pct=%.4f", tier.WeightEachPct, a.PositionMaxPct),
			}
		}
	}

	// Tier weights + cash ≈ 1.0 (±0.5%)
	totalAlloc := w.TotalWeightPct() + a.CashTargetPct
	if math.Abs(totalAlloc-1.0) > 0.005 {
		return ValidationError{"portfolio", fmt.Sprintf("tiers + cash must equal 1.0±0.005, got %.4f", totalAlloc)}
	}

	// === Execution ===
	if len(cfg.Execution.SlippageModel.Segments) == 0 {
		return ValidationError{"execution.slippage_model.segments", "required"}
	}

	// slippage_pct >= 0 검증
	for i, seg := range cfg.Execution.SlippageModel.Segments {
		if seg.SlippagePct < 0 {
			return ValidationError{
				Field:   fmt.Sprintf("execution.slippage_model.segments[%d].slippage_pct", i),
				Message: "must be >= 0",
			}
		}
	}

	// splitting 제약 조건
	if cfg.Execution.Splitting.Enable {
		if cfg.Execution.Splitting.MinSlices < 1 {
			return ValidationError{"execution.splitting.min_slices", "must be >= 1"}
		}
		if cfg.Execution.Splitting.MinSlices > cfg.Execution.Splitting.MaxSlices {
			return ValidationError{"execution.splitting", "min_slices must be <= max_slices"}
		}
		if cfg.Execution.Splitting.IntervalSeconds <= 0 {
			return ValidationError{"execution.splitting.interval_seconds", "must be > 0"}
		}
	}

	// === Exit ===
	if cfg.Exit.Mode != "FIXED" && cfg.Exit.Mode != "ATR" {
		return ValidationError{"exit.mode", "must be FIXED or ATR"}
	}

	// === RiskOverlay ===
	if cfg.RiskOverlay.NasdaqAdjust.Enable {
		// clamp: min <= max
		clamp := cfg.RiskOverlay.NasdaqAdjust.Clamp
		if clamp.MinEquityExposurePct > clamp.MaxEquityExposurePct {
			return ValidationError{"risk_overlay.nasdaq_adjust.clamp", "min must be <= max"}
		}

		// trigger: 각 트리거에 ret_le 또는 ret_ge 중 하나는 반드시 존재
		for i, trigger := range cfg.RiskOverlay.NasdaqAdjust.Triggers {
			if trigger.NasdaqRetLe == nil && trigger.NasdaqRetGe == nil {
				return ValidationError{
					Field:   fmt.Sprintf("risk_overlay.nasdaq_adjust.triggers[%d]", i),
					Message: "must have nasdaq_ret_le or nasdaq_ret_ge",
				}
			}
		}
	}

	// === BacktestCosts ===
	if cfg.BacktestCost.CommissionBps < 0 {
		return ValidationError{"backtest_costs.commission_bps", "must be >= 0"}
	}
	if cfg.BacktestCost.TaxBps < 0 {
		return ValidationError{"backtest_costs.tax_bps", "must be >= 0"}
	}

	return nil
}

// Warn checks recommended constraints (non-fatal)
func Warn(cfg *Config) []Warning {
	var warnings []Warning

	// ADTV < 10억 경고
	if cfg.Universe.Filters.ADTV20MinKRW < 1_000_000_000 {
		warnings = append(warnings, Warning{
			Code:    "LOW_ADTV",
			Message: "ADTV20 < 10억: 체결/슬리피지 리스크 높음",
		})
	}

	// 슬리피지 낙관적 가정 경고
	for _, seg := range cfg.Execution.SlippageModel.Segments {
		if seg.ADTV20MinKRW == 2_000_000_000 && seg.SlippagePct < 0.0035 {
			warnings = append(warnings, Warning{
				Code:    "OPTIMISTIC_SLIPPAGE",
				Message: "ADTV20 20억 구간 슬리피지 < 0.35%: 낙관적일 수 있음",
			})
		}
	}

	// 과도한 회전율 경고
	if cfg.Portfolio.Allocation.TurnoverDailyMaxPct > 0.25 {
		warnings = append(warnings, Warning{
			Code:    "HIGH_TURNOVER",
			Message: "일 회전율 > 25%: 거래비용 증가 우려",
		})
	}

	return warnings
}

// === Helper Functions ===

func validateHHMM(s string) error {
	re := regexp.MustCompile(`^\d{2}:\d{2}$`)
	if !re.MatchString(s) {
		return errors.New("must be HH:MM format")
	}
	_, err := time.Parse("15:04", s)
	return err
}

func validateWeightsSum(weights []float64, target float64, epsilon float64) error {
	if len(weights) == 0 {
		return errors.New("must not be empty")
	}
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if math.Abs(sum-target) > epsilon {
		return fmt.Errorf("must sum to %.2f, got %.4f", target, sum)
	}
	return nil
}

// validatePctRange는 퍼센트 값이 0~1 범위인지 검증
func validatePctRange(pct float64, field string) error {
	if pct < 0 || pct > 1 {
		return ValidationError{field, "must be in range [0, 1]"}
	}
	return nil
}
