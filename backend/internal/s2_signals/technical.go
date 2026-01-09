package s2_signals

import (
	"context"
	"math"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// TechnicalCalculator calculates technical signals
// ⭐ SSOT: 기술적 지표 계산은 여기서만
type TechnicalCalculator struct {
	logger *logger.Logger
}

// NewTechnicalCalculator creates a new technical calculator
func NewTechnicalCalculator(log *logger.Logger) *TechnicalCalculator {
	return &TechnicalCalculator{
		logger: log,
	}
}

// Calculate calculates technical signal for a stock
func (c *TechnicalCalculator) Calculate(ctx context.Context, code string, prices []PricePoint) (float64, contracts.SignalDetails, error) {
	details := contracts.SignalDetails{}

	if len(prices) < 120 {
		// Need at least 120 days for MA120
		return 0.0, details, nil
	}

	// Calculate RSI
	rsi := c.calculateRSI(prices, 14)
	details.RSI = rsi

	// Calculate MACD
	macd, _ := c.calculateMACD(prices)
	details.MACD = macd

	// Calculate MA20 cross
	ma20Cross := c.calculateMA20Cross(prices)
	details.MA20Cross = ma20Cross

	// Calculate technical score
	score := c.calculateScore(rsi, macd, ma20Cross)

	c.logger.WithFields(map[string]interface{}{
		"code":       code,
		"rsi":        rsi,
		"macd":       macd,
		"ma20_cross": ma20Cross,
		"score":      score,
	}).Debug("Calculated technical signal")

	return score, details, nil
}

// calculateRSI calculates Relative Strength Index
func (c *TechnicalCalculator) calculateRSI(prices []PricePoint, period int) float64 {
	if len(prices) < period+1 {
		return 50.0 // Neutral
	}

	var gains, losses float64

	for i := 0; i < period; i++ {
		change := float64(prices[i].Price - prices[i+1].Price)
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	if losses == 0 {
		return 100.0
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateMACD calculates Moving Average Convergence Divergence
func (c *TechnicalCalculator) calculateMACD(prices []PricePoint) (float64, float64) {
	if len(prices) < 26 {
		return 0.0, 0.0
	}

	// Calculate EMA12 and EMA26
	ema12 := c.calculateEMA(prices, 12)
	ema26 := c.calculateEMA(prices, 26)

	// MACD = EMA12 - EMA26
	macd := ema12 - ema26

	// Signal line = EMA9 of MACD (simplified: use current value)
	signal := macd

	return macd, signal
}

// calculateEMA calculates Exponential Moving Average
func (c *TechnicalCalculator) calculateEMA(prices []PricePoint, period int) float64 {
	if len(prices) < period {
		return 0.0
	}

	// Calculate initial SMA
	var sum float64
	for i := 0; i < period; i++ {
		sum += float64(prices[len(prices)-period+i].Price)
	}
	sma := sum / float64(period)

	// Calculate EMA
	multiplier := 2.0 / (float64(period) + 1.0)
	ema := sma

	for i := len(prices) - period - 1; i >= 0; i-- {
		ema = (float64(prices[i].Price) * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

// calculateMA20Cross calculates MA20 cross signal
// Returns: -1 (death cross), 0 (neutral), 1 (golden cross)
func (c *TechnicalCalculator) calculateMA20Cross(prices []PricePoint) int {
	if len(prices) < 20 {
		return 0
	}

	// Calculate MA20
	var sum int64
	for i := 0; i < 20; i++ {
		sum += prices[i].Price
	}
	ma20 := float64(sum) / 20.0

	currentPrice := float64(prices[0].Price)

	// Check if price crossed MA20
	priceDiff := (currentPrice - ma20) / ma20

	if priceDiff > 0.02 { // Price > MA20 by 2%
		return 1 // Golden cross
	} else if priceDiff < -0.02 { // Price < MA20 by 2%
		return -1 // Death cross
	}

	return 0 // Neutral
}

// calculateScore calculates technical score (-1.0 ~ 1.0)
func (c *TechnicalCalculator) calculateScore(rsi, macd float64, ma20Cross int) float64 {
	// RSI component: normalize 0-100 to -1 to 1
	// RSI < 30: oversold (positive)
	// RSI > 70: overbought (negative)
	// RSI = 50: neutral
	rsiScore := 0.0
	if rsi < 30 {
		rsiScore = (30 - rsi) / 30 // 0 to 1
	} else if rsi > 70 {
		rsiScore = (70 - rsi) / 30 // 0 to -1
	} else {
		rsiScore = (50 - rsi) / 20 // -1 to 1
	}

	// MACD component: normalize to -1 to 1
	// Typical MACD values range from -1000 to 1000
	macdScore := math.Tanh(macd / 500)

	// MA20 cross component: already -1, 0, or 1
	ma20Score := float64(ma20Cross)

	// Weight the factors
	// RSI: 40%, MACD: 40%, MA20: 20%
	score := rsiScore*0.4 + macdScore*0.4 + ma20Score*0.2

	// Clamp to -1.0 ~ 1.0
	if score > 1.0 {
		score = 1.0
	} else if score < -1.0 {
		score = -1.0
	}

	return score
}
