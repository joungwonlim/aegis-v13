package contracts

// RankedStock represents a stock with ranking information passed from S4 to S5
// ⭐ SSOT: S4 → S5 랭킹 결과 전달
type RankedStock struct {
	Code       string      `json:"code"`
	Name       string      `json:"name"`
	Rank       int         `json:"rank"`        // 1-based ranking
	TotalScore float64     `json:"total_score"` // Composite score
	Scores     ScoreDetail `json:"scores"`      // Individual scores
}

// ScoreDetail contains breakdown of individual signal scores
type ScoreDetail struct {
	Momentum  float64 `json:"momentum"`
	Technical float64 `json:"technical"`
	Value     float64 `json:"value"`
	Quality   float64 `json:"quality"`
	Flow      float64 `json:"flow"`  // 수급
	Event     float64 `json:"event"` // 이벤트
}

// IsTopRanked checks if the stock is in top N ranks
func (r *RankedStock) IsTopRanked(n int) bool {
	return r.Rank <= n && r.Rank > 0
}
