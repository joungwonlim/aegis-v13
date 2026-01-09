package s2_signals

import (
	"context"
	"math"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// FlowCalculator calculates investor flow (수급) signals
// ⭐ SSOT: 수급 시그널 계산은 여기서만
type FlowCalculator struct {
	logger *logger.Logger
}

// NewFlowCalculator creates a new flow calculator
func NewFlowCalculator(log *logger.Logger) *FlowCalculator {
	return &FlowCalculator{
		logger: log,
	}
}

// FlowData represents investor flow data for a stock
type FlowData struct {
	Date        string // YYYY-MM-DD
	ForeignNet  int64  // 외국인 순매수
	InstNet     int64  // 기관 순매수
	IndividualNet int64 // 개인 순매수
}

// Calculate calculates flow signal for a stock
func (c *FlowCalculator) Calculate(ctx context.Context, code string, flowData []FlowData) (float64, contracts.SignalDetails, error) {
	details := contracts.SignalDetails{}

	if len(flowData) < 20 {
		// Need at least 20 days of data
		return 0.0, details, nil
	}

	// Calculate net buying for different periods
	foreignNet5D := c.sumNetBuying(flowData[:5], "foreign")
	foreignNet20D := c.sumNetBuying(flowData[:20], "foreign")
	instNet5D := c.sumNetBuying(flowData[:5], "inst")
	instNet20D := c.sumNetBuying(flowData[:20], "inst")

	details.ForeignNet5D = foreignNet5D
	details.ForeignNet20D = foreignNet20D
	details.InstNet5D = instNet5D
	details.InstNet20D = instNet20D

	// Calculate score
	score := c.calculateScore(foreignNet5D, foreignNet20D, instNet5D, instNet20D)

	c.logger.WithFields(map[string]interface{}{
		"code":           code,
		"foreign_net_5d": foreignNet5D,
		"foreign_net_20d": foreignNet20D,
		"inst_net_5d":    instNet5D,
		"inst_net_20d":   instNet20D,
		"score":          score,
	}).Debug("Calculated flow signal")

	return score, details, nil
}

// sumNetBuying sums net buying for a period
func (c *FlowCalculator) sumNetBuying(data []FlowData, investorType string) int64 {
	var sum int64
	for _, d := range data {
		switch investorType {
		case "foreign":
			sum += d.ForeignNet
		case "inst":
			sum += d.InstNet
		case "individual":
			sum += d.IndividualNet
		}
	}
	return sum
}

// calculateStreak calculates consecutive buying/selling streak
// Returns positive for buying streak, negative for selling streak
func (c *FlowCalculator) calculateStreak(data []FlowData, investorType string) int {
	if len(data) == 0 {
		return 0
	}

	streak := 0

	for _, d := range data {
		var currentNet int64
		switch investorType {
		case "foreign":
			currentNet = d.ForeignNet
		case "inst":
			currentNet = d.InstNet
		}

		if currentNet > 0 {
			// Buying
			if streak >= 0 {
				streak++
			} else {
				break
			}
		} else if currentNet < 0 {
			// Selling
			if streak <= 0 {
				streak--
			} else {
				break
			}
		} else {
			// No change
			break
		}
	}

	return streak
}

// calculateScore calculates flow score (-1.0 ~ 1.0)
func (c *FlowCalculator) calculateScore(foreignNet5D, foreignNet20D, instNet5D, instNet20D int64) float64 {
	// Normalize net buying amounts
	// Typical range: -10 billion to +10 billion won
	// Scale: 10 billion = 1.0

	foreignScore5D := math.Tanh(float64(foreignNet5D) / 10_000_000_000)
	foreignScore20D := math.Tanh(float64(foreignNet20D) / 50_000_000_000) // Larger scale for 20D
	instScore5D := math.Tanh(float64(instNet5D) / 10_000_000_000)
	instScore20D := math.Tanh(float64(instNet20D) / 50_000_000_000)

	// Weight the factors
	// Foreign investors are generally considered smart money
	// Recent activity (5D) weighted more than long-term (20D)
	// Foreign: 60%, Institutional: 40%
	// 5D: 70%, 20D: 30%
	foreignScore := foreignScore5D*0.7 + foreignScore20D*0.3
	instScore := instScore5D*0.7 + instScore20D*0.3

	score := foreignScore*0.6 + instScore*0.4

	// Clamp to -1.0 ~ 1.0 (tanh should already do this, but ensure)
	if score > 1.0 {
		score = 1.0
	} else if score < -1.0 {
		score = -1.0
	}

	return score
}
