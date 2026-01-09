package selection

import (
	"context"

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
type ScreenerConfig struct {
	// Momentum filter
	MinMomentum float64 // -1.0 이상 (예: 0.0)

	// Technical filter
	MinTechnical float64 // 기술적 지표 최소값

	// Value filter
	MaxPER float64 // PER 최대값 (예: 50)
	MinPBR float64 // PBR 최소값 (예: 0.2)

	// Quality filter
	MinROE       float64 // ROE 최소값 (예: 5%)
	MaxDebtRatio float64 // 부채비율 최대값 (예: 200%)

	// Flow filter (수급)
	MinFlow float64 // 수급 점수 최소값

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

	for code, signal := range signals.Signals {
		reason := s.checkConditions(signal)
		if reason == "" {
			passed = append(passed, code)
		} else {
			filtered[reason]++
		}
	}

	s.logger.WithFields(map[string]interface{}{
		"total_input":  len(signals.Signals),
		"passed":       len(passed),
		"filtered_out": len(signals.Signals) - len(passed),
		"filters":      filtered,
	}).Info("Screening completed")

	return passed, nil
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

	// Value filters
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

	// Quality filters
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

	// Passed all filters
	return ""
}

// DefaultScreenerConfig returns a sensible default configuration
func DefaultScreenerConfig() ScreenerConfig {
	return ScreenerConfig{
		MinMomentum:             0.0,   // 중립 이상
		MinTechnical:            -0.5,  // 약한 부정까지 허용
		MaxPER:                  50.0,  // 고평가 제외
		MinPBR:                  0.2,   // 너무 저평가는 리스크
		MinROE:                  5.0,   // 5% 이상
		MaxDebtRatio:            200.0, // 부채비율 200% 이하
		MinFlow:                 -0.3,  // 약한 매도세까지 허용
		ExcludeNegativeEarnings: true,  // 적자 기업 제외
	}
}
