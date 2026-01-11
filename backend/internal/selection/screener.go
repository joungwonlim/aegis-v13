package selection

import (
	"context"
	"sort"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Screener implements S3: Hard Cut filtering
// ⭐ SSOT: S3 스크리닝 로직은 여기서만
type Screener struct {
	config ScreenerConfig
	logger *logger.Logger
}

// ScreenerConfig defines hard cut conditions
// SSOT: config/strategy/korea_equity_v13.yaml screening
type ScreenerConfig struct {
	// Momentum filter
	MinMomentum float64 // -1.0 이상 (예: 0.0)

	// Technical filter
	MinTechnical float64 // 기술적 지표 최소값

	// Value filter (fundamentals)
	MaxPER float64 // PER 최대값 (예: 50)
	MinPBR float64 // PBR 최소값 (예: 0.2)

	// Quality filter (fundamentals)
	MinROE       float64 // ROE 최소값 (예: 5%)
	MaxDebtRatio float64 // 부채비율 최대값 (예: 200%)

	// Flow filter (수급)
	MinFlow float64 // 수급 점수 최소값

	// Drawdown filter (급락 제외)
	MinReturn1D float64 // 1일 수익률 최소값 (예: -0.09 = -9%)
	MinReturn5D float64 // 5일 수익률 최소값 (예: -0.18 = -18%)

	// Overheat filter (과열 제외)
	EnableOverheat bool    // 과열 필터 활성화
	MaxReturn5D    float64 // 5일 수익률 최대값 (예: 0.35 = 35%)

	// Volatility filter (고변동성 제외)
	EnableVolatility     bool    // 변동성 필터 활성화
	MaxVolatilityPercent float64 // 변동성 상위 N% 제외 (예: 0.10 = 상위 10%)

	// Exclusions
	ExcludeNegativeEarnings bool // 적자 기업 제외
}

// NewScreener creates a new screener
func NewScreener(config ScreenerConfig, logger *logger.Logger) *Screener {
	return &Screener{
		config: config,
		logger: logger,
	}
}

// Screen applies hard cut filters to signal set
func (s *Screener) Screen(ctx context.Context, signals *contracts.SignalSet) ([]string, error) {
	passed := make([]string, 0)
	filtered := make(map[string]int) // Filter name -> count

	// Phase 1: Apply absolute filters (checkConditions)
	for code, signal := range signals.Signals {
		reason := s.checkConditions(signal)
		if reason == "" {
			passed = append(passed, code)
		} else {
			filtered[reason]++
		}
	}

	// Phase 2: Apply relative filter (volatility - top N% exclusion)
	if s.config.EnableVolatility && s.config.MaxVolatilityPercent > 0 && len(passed) > 0 {
		passed, filtered = s.applyVolatilityFilter(signals, passed, filtered)
	}

	s.logger.WithFields(map[string]interface{}{
		"total_input":  len(signals.Signals),
		"passed":       len(passed),
		"filtered_out": len(signals.Signals) - len(passed),
		"filters":      filtered,
	}).Info("Screening completed")

	return passed, nil
}

// applyVolatilityFilter excludes top N% high volatility stocks
func (s *Screener) applyVolatilityFilter(
	signals *contracts.SignalSet,
	passed []string,
	filtered map[string]int,
) ([]string, map[string]int) {
	// Collect volatility data for passed stocks
	type volStock struct {
		code       string
		volatility float64
	}
	volStocks := make([]volStock, 0, len(passed))

	for _, code := range passed {
		if signal, exists := signals.Signals[code]; exists {
			volStocks = append(volStocks, volStock{
				code:       code,
				volatility: signal.Details.Volatility20D,
			})
		}
	}

	// Sort by volatility descending (highest first)
	sort.Slice(volStocks, func(i, j int) bool {
		return volStocks[i].volatility > volStocks[j].volatility
	})

	// Calculate how many to exclude (top N%)
	excludeCount := int(float64(len(volStocks)) * s.config.MaxVolatilityPercent)
	if excludeCount <= 0 {
		return passed, filtered
	}

	// Build new passed list, excluding top N%
	excludeSet := make(map[string]bool)
	for i := 0; i < excludeCount && i < len(volStocks); i++ {
		excludeSet[volStocks[i].code] = true
	}

	newPassed := make([]string, 0, len(passed)-excludeCount)
	for _, code := range passed {
		if !excludeSet[code] {
			newPassed = append(newPassed, code)
		}
	}

	filtered["volatility"] = excludeCount
	return newPassed, filtered
}

// checkConditions checks if stock passes all conditions
// Returns empty string if passed, otherwise returns filter name
func (s *Screener) checkConditions(signal *contracts.StockSignals) string {
	// Momentum filter
	if signal.Momentum < s.config.MinMomentum {
		return "momentum"
	}

	// Technical filter
	if signal.Technical < s.config.MinTechnical {
		return "technical"
	}

	// Flow filter (수급)
	if signal.Flow < s.config.MinFlow {
		return "flow"
	}

	// Value filters (fundamentals)
	if s.config.MaxPER > 0 {
		per := signal.Details.PER
		if per > s.config.MaxPER || per <= 0 {
			return "per"
		}
	}

	if s.config.MinPBR > 0 {
		pbr := signal.Details.PBR
		if pbr < s.config.MinPBR || pbr <= 0 {
			return "pbr"
		}
	}

	// Quality filters (fundamentals)
	if s.config.MinROE > 0 {
		roe := signal.Details.ROE
		if roe < s.config.MinROE {
			return "roe"
		}

		// Exclude negative earnings if configured
		if s.config.ExcludeNegativeEarnings && roe < 0 {
			return "negative_earnings"
		}
	}

	if s.config.MaxDebtRatio > 0 {
		debtRatio := signal.Details.DebtRatio
		if debtRatio > s.config.MaxDebtRatio {
			return "debt_ratio"
		}
	}

	// Drawdown filter (급락 제외)
	// MinReturn1D/5D가 음수로 설정됨 (예: -0.09)
	if s.config.MinReturn1D != 0 {
		if signal.Details.Return1D < s.config.MinReturn1D {
			return "drawdown_1d"
		}
	}
	if s.config.MinReturn5D != 0 {
		if signal.Details.Return5D < s.config.MinReturn5D {
			return "drawdown_5d"
		}
	}

	// Overheat filter (과열 제외)
	if s.config.EnableOverheat && s.config.MaxReturn5D > 0 {
		if signal.Details.Return5D > s.config.MaxReturn5D {
			return "overheat"
		}
	}

	// Note: Volatility filter는 상대적 필터 (상위 N% 제외)이므로
	// Screen() 레벨에서 별도 처리 필요 (checkConditions에서 불가)
	// 현재는 EnableVolatility 플래그만 제공

	// Passed all filters
	return ""
}

// DefaultScreenerConfig returns default configuration
// SSOT: config/strategy/korea_equity_v13.yaml screening
func DefaultScreenerConfig() ScreenerConfig {
	return ScreenerConfig{
		// Signal filters
		MinMomentum:  0.0,  // 중립 이상
		MinTechnical: -0.5, // 약한 부정까지 허용
		MinFlow:      -0.3, // 약한 매도세까지 허용

		// Fundamentals (YAML: screening.fundamentals)
		MaxPER:                  50.0,  // per_max: 50
		MinPBR:                  0.2,   // pbr_min: 0.2
		MinROE:                  5.0,   // roe_min: 5
		MaxDebtRatio:            200.0, // 부채비율 200% 이하
		ExcludeNegativeEarnings: true,  // per_min: 0 (적자 제외)

		// Drawdown (YAML: screening.drawdown)
		MinReturn1D: -0.09, // day1_return_min: -0.09 (-9%)
		MinReturn5D: -0.18, // day5_return_min: -0.18 (-18%)

		// Overheat (YAML: screening.overheat)
		EnableOverheat: true, // enable: true
		MaxReturn5D:    0.35, // day5_return_max: 0.35 (35%)

		// Volatility (YAML: screening.volatility)
		EnableVolatility:     true, // enable: true
		MaxVolatilityPercent: 0.10, // vol20_exclude_top_pct: 0.10 (상위 10%)
	}
}
