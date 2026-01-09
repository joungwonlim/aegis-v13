package contracts

import "time"

// PerformanceReport represents performance analysis results from S7
// ⭐ SSOT: S7 성과 분석 결과
type PerformanceReport struct {
	Period    string    `json:"period"` // "2024-01-01~2024-12-31"
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`

	// 수익률
	TotalReturn   float64 `json:"total_return"`   // 누적 수익률
	AnnualReturn  float64 `json:"annual_return"`  // 연환산 수익률
	BenchmarkComp float64 `json:"benchmark_comp"` // 벤치마크 대비 초과 수익률

	// 리스크
	Volatility   float64 `json:"volatility"`    // 변동성
	Sharpe       float64 `json:"sharpe"`        // 샤프 비율
	MaxDrawdown  float64 `json:"max_drawdown"`  // 최대 낙폭
	WinRate      float64 `json:"win_rate"`      // 승률
	ProfitFactor float64 `json:"profit_factor"` // 손익비

	// 거래 통계
	TotalTrades int     `json:"total_trades"` // 총 거래 수
	AvgHolding  float64 `json:"avg_holding"`  // 평균 보유 일수
	Turnover    float64 `json:"turnover"`     // 회전율

	// Attribution (기여도 분석)
	Attribution map[string]float64 `json:"attribution"` // 팩터별 기여도
}

// IsOutperforming checks if the strategy outperformed benchmark
func (pr *PerformanceReport) IsOutperforming() bool {
	return pr.BenchmarkComp > 0
}

// IsHealthy checks if the strategy has healthy risk metrics
func (pr *PerformanceReport) IsHealthy() bool {
	return pr.Sharpe > 1.0 && pr.MaxDrawdown > -0.30 && pr.WinRate > 0.50
}
