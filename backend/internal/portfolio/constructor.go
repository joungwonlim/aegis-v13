package portfolio

import (
	"context"
	"math"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Constructor implements S5: Portfolio construction
// ⭐ SSOT: S5 포트폴리오 구성 로직은 여기서만
type Constructor struct {
	config      PortfolioConfig
	constraints Constraints
	logger      *logger.Logger
}

// PortfolioConfig defines portfolio construction parameters
type PortfolioConfig struct {
	MaxPositions  int     // 최대 종목 수
	MaxWeight     float64 // 종목당 최대 비중 (0.0 ~ 1.0)
	MinWeight     float64 // 종목당 최소 비중 (0.0 ~ 1.0)
	CashReserve   float64 // 현금 보유 비중 (0.0 ~ 1.0)
	TurnoverLimit float64 // 회전율 제한 (0.0 ~ 1.0)
	WeightingMode string  // "equal", "score_based", "risk_parity"
}

// NewConstructor creates a new portfolio constructor
func NewConstructor(config PortfolioConfig, constraints Constraints, logger *logger.Logger) *Constructor {
	return &Constructor{
		config:      config,
		constraints: constraints,
		logger:      logger,
	}
}

// Construct constructs target portfolio from ranked stocks
func (c *Constructor) Construct(ctx context.Context, ranked []contracts.RankedStock) (*contracts.TargetPortfolio, error) {
	target := &contracts.TargetPortfolio{
		Date:      time.Now(),
		Positions: make([]contracts.TargetPosition, 0),
		Cash:      c.config.CashReserve,
	}

	// 1. Select top N stocks
	topN := c.selectTopN(ranked)
	if len(topN) == 0 {
		c.logger.Warn("No stocks selected for portfolio")
		return target, nil
	}

	// 2. Calculate weights
	weights := c.calculateWeights(topN)

	// 3. Apply constraints
	weights = c.applyConstraints(weights)

	// 4. Create target positions
	for code, weight := range weights {
		// Find stock info from ranked
		var stock *contracts.RankedStock
		for i := range topN {
			if topN[i].Code == code {
				stock = &topN[i]
				break
			}
		}

		if stock == nil {
			continue
		}

		target.Positions = append(target.Positions, contracts.TargetPosition{
			Code:      code,
			Name:      stock.Name,
			Weight:    weight,
			TargetQty: 0, // 계산 필요 (가격 정보 필요)
			Action:    contracts.ActionBuy,
			Reason:    c.getActionReason(stock),
		})
	}

	c.logger.WithFields(map[string]interface{}{
		"positions":    len(target.Positions),
		"total_weight": target.TotalWeight(),
		"cash":         target.Cash,
	}).Info("Portfolio constructed")

	return target, nil
}

// selectTopN selects top N stocks from ranked list
func (c *Constructor) selectTopN(ranked []contracts.RankedStock) []contracts.RankedStock {
	n := c.config.MaxPositions
	if len(ranked) < n {
		n = len(ranked)
	}

	return ranked[:n]
}

// calculateWeights calculates position weights based on weighting mode
func (c *Constructor) calculateWeights(stocks []contracts.RankedStock) map[string]float64 {
	switch c.config.WeightingMode {
	case "equal":
		return c.equalWeight(stocks)
	case "score_based":
		return c.scoreBasedWeight(stocks)
	case "risk_parity":
		// TODO: Implement risk parity (변동성 필요)
		c.logger.Warn("Risk parity not implemented, using equal weight")
		return c.equalWeight(stocks)
	default:
		return c.equalWeight(stocks)
	}
}

// equalWeight calculates equal weights for all stocks
func (c *Constructor) equalWeight(stocks []contracts.RankedStock) map[string]float64 {
	available := 1.0 - c.config.CashReserve
	weight := available / float64(len(stocks))

	weights := make(map[string]float64)
	for _, stock := range stocks {
		weights[stock.Code] = weight
	}

	return weights
}

// scoreBasedWeight calculates weights proportional to total score
func (c *Constructor) scoreBasedWeight(stocks []contracts.RankedStock) map[string]float64 {
	available := 1.0 - c.config.CashReserve

	// Calculate total score (only positive scores)
	var totalScore float64
	for _, stock := range stocks {
		// Normalize score to 0 ~ 1 (from -1 ~ 1)
		normalizedScore := (stock.TotalScore + 1.0) / 2.0
		if normalizedScore > 0 {
			totalScore += normalizedScore
		}
	}

	if totalScore == 0 {
		// Fallback to equal weight
		return c.equalWeight(stocks)
	}

	weights := make(map[string]float64)
	for _, stock := range stocks {
		normalizedScore := (stock.TotalScore + 1.0) / 2.0
		if normalizedScore > 0 {
			weights[stock.Code] = (normalizedScore / totalScore) * available
		}
	}

	return weights
}

// applyConstraints applies portfolio constraints to weights
func (c *Constructor) applyConstraints(weights map[string]float64) map[string]float64 {
	result := make(map[string]float64)

	for code, weight := range weights {
		// Apply max weight constraint
		if weight > c.constraints.MaxWeight {
			weight = c.constraints.MaxWeight
		}

		// Apply min weight constraint
		if weight < c.constraints.MinWeight {
			continue // 제외
		}

		// Check blacklist
		if c.constraints.IsBlackListed(code) {
			continue
		}

		result[code] = weight
	}

	// Normalize weights to sum to (1.0 - CashReserve)
	return c.normalizeWeights(result)
}

// normalizeWeights normalizes weights to sum to target total
func (c *Constructor) normalizeWeights(weights map[string]float64) map[string]float64 {
	targetTotal := 1.0 - c.config.CashReserve

	// Calculate current total
	var currentTotal float64
	for _, weight := range weights {
		currentTotal += weight
	}

	if currentTotal == 0 {
		return weights
	}

	// Scale weights
	factor := targetTotal / currentTotal
	normalized := make(map[string]float64)
	for code, weight := range weights {
		normalized[code] = weight * factor
	}

	return normalized
}

// getActionReason generates action reason for a stock
func (c *Constructor) getActionReason(stock *contracts.RankedStock) string {
	// Generate reason based on top signals
	topSignal := ""
	maxScore := 0.0

	signals := map[string]float64{
		"Momentum":  stock.Scores.Momentum,
		"Technical": stock.Scores.Technical,
		"Value":     stock.Scores.Value,
		"Quality":   stock.Scores.Quality,
		"Flow":      stock.Scores.Flow,
		"Event":     stock.Scores.Event,
	}

	for name, score := range signals {
		if math.Abs(score) > math.Abs(maxScore) {
			maxScore = score
			topSignal = name
		}
	}

	if maxScore > 0 {
		return "Strong " + topSignal + " signal"
	} else if maxScore < 0 {
		return "Weak " + topSignal + " signal"
	}

	return "Selected by ranking"
}

// DefaultPortfolioConfig returns default configuration
func DefaultPortfolioConfig() PortfolioConfig {
	return PortfolioConfig{
		MaxPositions:  20,    // 최대 20 종목
		MaxWeight:     0.15,  // 최대 15%
		MinWeight:     0.03,  // 최소 3%
		CashReserve:   0.05,  // 5% 현금 보유
		TurnoverLimit: 0.30,  // 30% 회전율 제한
		WeightingMode: "equal", // 기본: 동일 비중
	}
}
