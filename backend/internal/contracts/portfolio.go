package contracts

import "time"

// TargetPortfolio represents the target portfolio passed from S5 to S6
// ⭐ SSOT: S5 → S6 목표 포트폴리오 전달
type TargetPortfolio struct {
	Date      time.Time        `json:"date"`
	Positions []TargetPosition `json:"positions"`
	Cash      float64          `json:"cash"` // 목표 현금 비중 (0.0 ~ 1.0)
}

// TargetPosition represents a target position in the portfolio
type TargetPosition struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Weight    float64 `json:"weight"`     // 목표 비중 (0.0 ~ 1.0)
	TargetQty int     `json:"target_qty"` // 목표 수량
	Action    Action  `json:"action"`     // BUY, SELL, HOLD
	Reason    string  `json:"reason"`     // 매수/매도 사유
}

// Action represents the action to take for a position
type Action string

const (
	ActionBuy  Action = "BUY"
	ActionSell Action = "SELL"
	ActionHold Action = "HOLD"
)

// TotalWeight returns the sum of all position weights
func (tp *TargetPortfolio) TotalWeight() float64 {
	total := 0.0
	for _, pos := range tp.Positions {
		total += pos.Weight
	}
	return total
}

// Count returns the number of positions
func (tp *TargetPortfolio) Count() int {
	return len(tp.Positions)
}

// GetPosition finds a position by stock code
func (tp *TargetPortfolio) GetPosition(code string) (*TargetPosition, bool) {
	for i := range tp.Positions {
		if tp.Positions[i].Code == code {
			return &tp.Positions[i], true
		}
	}
	return nil, false
}
