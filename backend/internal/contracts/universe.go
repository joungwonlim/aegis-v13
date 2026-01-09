package contracts

import "time"

// Universe represents investable stocks passed from S1 to S2
// ⭐ SSOT: S1 → S2 투자 가능 종목 전달
type Universe struct {
	Date       time.Time         `json:"date"`
	Stocks     []string          `json:"stocks"`                // 투자 가능 종목 코드
	Excluded   map[string]string `json:"excluded"`              // 제외 종목: 사유
	TotalCount int               `json:"total_count,omitempty"` // 전체 종목 수
}

// Contains checks if a stock code is in the universe
func (u *Universe) Contains(code string) bool {
	for _, stock := range u.Stocks {
		if stock == code {
			return true
		}
	}
	return false
}

// IsExcluded checks if a stock code is excluded with reason
func (u *Universe) IsExcluded(code string) (bool, string) {
	reason, exists := u.Excluded[code]
	return exists, reason
}

// Count returns the number of investable stocks
func (u *Universe) Count() int {
	return len(u.Stocks)
}
