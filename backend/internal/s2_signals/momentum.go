package s2_signals

import (
	"context"
	"math"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// MomentumCalculator calculates momentum signals
// ⭐ SSOT: 모멘텀 시그널 계산은 여기서만
type MomentumCalculator struct {
	logger *logger.Logger
}

// NewMomentumCalculator creates a new momentum calculator
func NewMomentumCalculator(log *logger.Logger) *MomentumCalculator {
	return &MomentumCalculator{
		logger: log,
	}
}

// Calculate calculates momentum signal for a stock
func (c *MomentumCalculator) Calculate(ctx context.Context, code string, prices []PricePoint) (float64, contracts.SignalDetails, error) {
	details := contracts.SignalDetails{}

	if len(prices) == 0 {
		return 0.0, details, nil
	}

	// Calculate returns for different periods
	return1M := c.calculateReturn(prices, 20)   // ~1 month (20 trading days)
	return3M := c.calculateReturn(prices, 60)   // ~3 months
	volumeRate := c.calculateVolumeGrowth(prices, 20)

	details.Return1M = return1M
	details.Return3M = return3M
	details.VolumeRate = volumeRate

	// Calculate momentum score
	score := c.calculateScore(return1M, return3M, volumeRate)

	c.logger.WithFields(map[string]interface{}{
		"code":        code,
		"return_1m":   return1M,
		"return_3m":   return3M,
		"volume_rate": volumeRate,
		"score":       score,
	}).Debug("Calculated momentum signal")

	return score, details, nil
}

// calculateReturn calculates price return over a period
func (c *MomentumCalculator) calculateReturn(prices []PricePoint, days int) float64 {
	if len(prices) < days+1 {
		return 0.0
	}

	// Get current price (most recent)
	currentPrice := prices[0].Price
	if currentPrice == 0 {
		return 0.0
	}

	// Get price from N days ago
	pastPrice := prices[days].Price
	if pastPrice == 0 {
		return 0.0
	}

	// Calculate return
	ret := (float64(currentPrice) - float64(pastPrice)) / float64(pastPrice)
	return ret
}

// calculateVolumeGrowth calculates volume growth rate
func (c *MomentumCalculator) calculateVolumeGrowth(prices []PricePoint, days int) float64 {
	if len(prices) < days*2 {
		return 0.0
	}

	// Average volume for recent period
	recentVolume := c.averageVolume(prices[:days])

	// Average volume for past period
	pastVolume := c.averageVolume(prices[days : days*2])

	if pastVolume == 0 {
		return 0.0
	}

	// Calculate growth rate
	growth := (recentVolume - pastVolume) / pastVolume
	return growth
}

// averageVolume calculates average volume
func (c *MomentumCalculator) averageVolume(prices []PricePoint) float64 {
	if len(prices) == 0 {
		return 0.0
	}

	var sum int64
	for _, p := range prices {
		sum += p.Volume
	}

	return float64(sum) / float64(len(prices))
}

// calculateScore calculates momentum score (-1.0 ~ 1.0)
func (c *MomentumCalculator) calculateScore(return1M, return3M, volumeRate float64) float64 {
	// Weight the factors
	// Return1M: 40%, Return3M: 40%, VolumeRate: 20%
	score := return1M*0.4 + return3M*0.4 + volumeRate*0.2

	// Normalize to -1.0 ~ 1.0 using tanh
	// tanh maps (-inf, inf) to (-1, 1)
	// Scale input: typical returns are -50% to +50% (-0.5 to 0.5)
	normalizedScore := math.Tanh(score * 2)

	// Clamp to -1.0 ~ 1.0 (tanh should already do this, but ensure)
	if normalizedScore > 1.0 {
		normalizedScore = 1.0
	} else if normalizedScore < -1.0 {
		normalizedScore = -1.0
	}

	return normalizedScore
}

// PricePoint represents a price data point
type PricePoint struct {
	Date   time.Time
	Price  int64
	Volume int64
}
