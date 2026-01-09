package s2_signals

import (
	"context"
	"math"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// ValueCalculator calculates value signals
// ⭐ SSOT: 가치 지표 계산은 여기서만
type ValueCalculator struct {
	logger *logger.Logger
}

// NewValueCalculator creates a new value calculator
func NewValueCalculator(log *logger.Logger) *ValueCalculator {
	return &ValueCalculator{
		logger: log,
	}
}

// ValueMetrics represents valuation metrics for a stock
type ValueMetrics struct {
	PER float64 // Price to Earnings Ratio
	PBR float64 // Price to Book Ratio
	PSR float64 // Price to Sales Ratio
}

// Calculate calculates value signal for a stock
func (c *ValueCalculator) Calculate(ctx context.Context, code string, metrics ValueMetrics) (float64, contracts.SignalDetails, error) {
	details := contracts.SignalDetails{
		PER: metrics.PER,
		PBR: metrics.PBR,
		PSR: metrics.PSR,
	}

	// Calculate value score
	score := c.calculateScore(metrics)

	c.logger.WithFields(map[string]interface{}{
		"code":  code,
		"per":   metrics.PER,
		"pbr":   metrics.PBR,
		"psr":   metrics.PSR,
		"score": score,
	}).Debug("Calculated value signal")

	return score, details, nil
}

// calculateScore calculates value score (-1.0 ~ 1.0)
// Lower multiples = higher score (value stocks)
func (c *ValueCalculator) calculateScore(metrics ValueMetrics) float64 {
	// PER score: Lower is better
	// Typical range: 5 to 30
	// PER < 10: undervalued (positive)
	// PER > 20: overvalued (negative)
	perScore := 0.0
	if metrics.PER > 0 {
		// Normalize: 10 = 0.5, 5 = 1.0, 20 = -0.5, 30 = -1.0
		perScore = (15 - metrics.PER) / 15
		if perScore > 1.0 {
			perScore = 1.0
		} else if perScore < -1.0 {
			perScore = -1.0
		}
	}

	// PBR score: Lower is better
	// Typical range: 0.5 to 3.0
	// PBR < 1.0: undervalued (positive)
	// PBR > 2.0: overvalued (negative)
	pbrScore := 0.0
	if metrics.PBR > 0 {
		// Normalize: 1.0 = 0.5, 0.5 = 1.0, 2.0 = -0.5, 3.0 = -1.0
		pbrScore = (1.5 - metrics.PBR) / 1.5
		if pbrScore > 1.0 {
			pbrScore = 1.0
		} else if pbrScore < -1.0 {
			pbrScore = -1.0
		}
	}

	// PSR score: Lower is better
	// Typical range: 0.5 to 5.0
	// PSR < 1.0: undervalued (positive)
	// PSR > 3.0: overvalued (negative)
	psrScore := 0.0
	if metrics.PSR > 0 {
		// Normalize: 1.0 = 0.5, 0.5 = 1.0, 3.0 = -0.5, 5.0 = -1.0
		psrScore = (2.0 - metrics.PSR) / 2.0
		if psrScore > 1.0 {
			psrScore = 1.0
		} else if psrScore < -1.0 {
			psrScore = -1.0
		}
	}

	// Weight the factors
	// PER: 50%, PBR: 30%, PSR: 20%
	score := perScore*0.5 + pbrScore*0.3 + psrScore*0.2

	// Apply sigmoid to smooth the score
	score = math.Tanh(score * 1.5)

	return score
}
