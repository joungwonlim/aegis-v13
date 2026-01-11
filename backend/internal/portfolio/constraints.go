package portfolio

import "slices"

// Constraints defines portfolio construction constraints
// ⭐ SSOT: 포트폴리오 제약조건은 여기서만
type Constraints struct {
	MaxSectorWeight float64  // 섹터당 최대 비중 (0.0 ~ 1.0)
	MaxWeight       float64  // 종목당 최대 비중 (0.0 ~ 1.0)
	MinWeight       float64  // 종목당 최소 비중 (0.0 ~ 1.0)
	BlackList       []string // 제외 종목 리스트
}

// IsBlackListed checks if a stock code is in the blacklist
func (c *Constraints) IsBlackListed(code string) bool {
	return slices.Contains(c.BlackList, code)
}

// DefaultConstraints returns default constraint configuration
// SSOT: config/strategy/korea_equity_v13.yaml portfolio.allocation
func DefaultConstraints() Constraints {
	return Constraints{
		MaxSectorWeight: 0.25, // 섹터당 최대 25%
		MaxWeight:       0.10, // 종목당 최대 10%
		MinWeight:       0.04, // 종목당 최소 4%
		BlackList:       []string{},
	}
}
