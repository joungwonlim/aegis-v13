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
// SSOT: config/strategy/korea_equity_v13.yaml portfolio 섹션
type PortfolioConfig struct {
	MaxPositions  int     // 최대 종목 수 (기본: 20)
	MinPositions  int     // 최소 종목 수 (기본: 15)
	MaxWeight     float64 // 종목당 최대 비중 (기본: 0.10)
	MinWeight     float64 // 종목당 최소 비중 (기본: 0.04)
	CashReserve   float64 // 현금 보유 비중 (기본: 0.10)
	SectorMaxPct  float64 // 섹터당 최대 비중 (기본: 0.25)
	TurnoverLimit float64 // 일 회전율 제한 (기본: 0.20)
	WeightingMode string  // "equal", "score_based", "tiered"
	Tiers         []TierConfig // Tiered Weighting 설정
}

// TierConfig defines a weight tier
// SSOT: config/strategy/korea_equity_v13.yaml portfolio.weighting.tiers
type TierConfig struct {
	Count       int     // 이 tier에 포함될 종목 수
	WeightEach  float64 // 종목당 비중 (0.0 ~ 1.0)
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
// totalValue: 전체 포트폴리오 가치 (원화). 0이면 TargetValue 계산 불가
// ⭐ 계약: TargetValue = Weight × totalValue. Execution(S6)이 수량으로 변환
func (c *Constructor) Construct(ctx context.Context, ranked []contracts.RankedStock, totalValue int64) (*contracts.TargetPortfolio, error) {
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

		// TargetValue = Weight × totalValue
		// ⭐ Execution(S6)에서 현재가로 수량 계산
		targetValue := int64(float64(totalValue) * weight)

		target.Positions = append(target.Positions, contracts.TargetPosition{
			Code:        code,
			Name:        stock.Name,
			Weight:      weight,
			TargetValue: targetValue,
			Action:      contracts.ActionBuy,
			Reason:      c.getActionReason(stock),
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
	case "tiered", "TIERED":
		return c.tieredWeight(stocks)
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

// tieredWeight calculates weights based on ranking tiers
// SSOT: config/strategy/korea_equity_v13.yaml portfolio.weighting.tiers
// 기본: 1-5위 5%, 6-15위 4.5%, 16-20위 4% (총 90% 주식 + 10% 현금)
func (c *Constructor) tieredWeight(stocks []contracts.RankedStock) map[string]float64 {
	weights := make(map[string]float64)

	// tier 설정이 없으면 기본값 사용
	tiers := c.config.Tiers
	if len(tiers) == 0 {
		tiers = DefaultTiers()
	}

	// 각 tier별로 비중 할당
	stockIdx := 0
	for _, tier := range tiers {
		for i := 0; i < tier.Count && stockIdx < len(stocks); i++ {
			weights[stocks[stockIdx].Code] = tier.WeightEach
			stockIdx++
		}
	}

	// 남은 종목이 있으면 마지막 tier 비중으로 할당
	if stockIdx < len(stocks) && len(tiers) > 0 {
		lastTierWeight := tiers[len(tiers)-1].WeightEach
		for ; stockIdx < len(stocks); stockIdx++ {
			weights[stocks[stockIdx].Code] = lastTierWeight
		}
	}

	c.logger.WithFields(map[string]interface{}{
		"mode":         "tiered",
		"total_stocks": len(weights),
		"tiers":        len(tiers),
	}).Debug("Tiered weights calculated")

	return weights
}

// DefaultTiers returns default tier configuration
// SSOT: config/strategy/korea_equity_v13.yaml portfolio.weighting.tiers
// 합: 5×5% + 10×4.5% + 5×4% = 25% + 45% + 20% = 90% (+ 현금 10% = 100%)
func DefaultTiers() []TierConfig {
	return []TierConfig{
		{Count: 5, WeightEach: 0.05},   // 1-5위: 5% 각각
		{Count: 10, WeightEach: 0.045}, // 6-15위: 4.5% 각각
		{Count: 5, WeightEach: 0.04},   // 16-20위: 4% 각각
	}
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
// SSOT: config/strategy/korea_equity_v13.yaml portfolio 섹션
func DefaultPortfolioConfig() PortfolioConfig {
	return PortfolioConfig{
		MaxPositions:  20,       // 최대 20 종목
		MinPositions:  15,       // 최소 15 종목
		MaxWeight:     0.10,     // 최대 10%
		MinWeight:     0.04,     // 최소 4%
		CashReserve:   0.10,     // 10% 현금 보유
		SectorMaxPct:  0.25,     // 섹터당 최대 25%
		TurnoverLimit: 0.20,     // 일 회전율 제한 20%
		WeightingMode: "tiered", // 기본: Tiered Weighting
		Tiers:         DefaultTiers(),
	}
}
