package s2_signals

import (
	"context"
	"math"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// QualityCalculator calculates quality signals
// ⭐ SSOT: 퀄리티 지표 계산은 여기서만
type QualityCalculator struct {
	logger *logger.Logger
}

// NewQualityCalculator creates a new quality calculator
func NewQualityCalculator(log *logger.Logger) *QualityCalculator {
	return &QualityCalculator{
		logger: log,
	}
}

// QualityMetrics represents quality metrics for a stock
type QualityMetrics struct {
	ROE       float64 // Return on Equity (%)
	DebtRatio float64 // 부채비율 (%)
}

// Calculate calculates quality signal for a stock
func (c *QualityCalculator) Calculate(ctx context.Context, code string, metrics QualityMetrics) (float64, contracts.SignalDetails, error) {
	details := contracts.SignalDetails{
		ROE:       metrics.ROE,
		DebtRatio: metrics.DebtRatio,
	}

	// Calculate quality score
	score := c.calculateScore(metrics)

	c.logger.WithFields(map[string]interface{}{
		"code":       code,
		"roe":        metrics.ROE,
		"debt_ratio": metrics.DebtRatio,
		"score":      score,
	}).Debug("Calculated quality signal")

	return score, details, nil
}

// calculateScore calculates quality score (-1.0 ~ 1.0)
func (c *QualityCalculator) calculateScore(metrics QualityMetrics) float64 {
	// ROE score: Higher is better
	// Typical range: -10% to 30%
	// ROE > 15%: high quality (positive)
	// ROE < 5%: low quality (negative)
	roeScore := 0.0
	// Normalize: 15% = 0.5, 25% = 1.0, 5% = -0.5, -5% = -1.0
	roeScore = (metrics.ROE - 10) / 15
	if roeScore > 1.0 {
		roeScore = 1.0
	} else if roeScore < -1.0 {
		roeScore = -1.0
	}

	// Debt ratio score: Lower is better
	// Typical range: 0% to 300%
	// Debt < 50%: low risk (positive)
	// Debt > 150%: high risk (negative)
	debtScore := 0.0
	if metrics.DebtRatio >= 0 {
		// Normalize: 50% = 0.5, 0% = 1.0, 150% = -0.5, 300% = -1.0
		debtScore = (100 - metrics.DebtRatio) / 100
		if debtScore > 1.0 {
			debtScore = 1.0
		} else if debtScore < -1.0 {
			debtScore = -1.0
		}
	}

	// Weight the factors
	// ROE: 60%, DebtRatio: 40%
	score := roeScore*0.6 + debtScore*0.4

	// Apply sigmoid to smooth the score
	score = math.Tanh(score * 1.5)

	return score
}
