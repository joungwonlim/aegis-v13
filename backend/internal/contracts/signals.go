package contracts

import "time"

// SignalSet represents all stock signals passed from S2 to S3/S4
// ⭐ SSOT: S2 → S3/S4 시그널 데이터 전달
type SignalSet struct {
	Date    time.Time                `json:"date"`
	Signals map[string]*StockSignals `json:"signals"` // key: stock code
}

// StockSignals represents all signals for a single stock
type StockSignals struct {
	Code string `json:"code"`

	// 시그널 점수 (-1.0 ~ 1.0)
	Momentum  float64 `json:"momentum"`
	Technical float64 `json:"technical"`
	Value     float64 `json:"value"`
	Quality   float64 `json:"quality"`
	Flow      float64 `json:"flow"`  // 수급 시그널
	Event     float64 `json:"event"` // 이벤트 시그널

	// 원본 데이터
	Details   SignalDetails `json:"details"`
	Events    []EventSignal `json:"events"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// SignalDetails contains raw data behind signals
type SignalDetails struct {
	// Momentum
	Return1M   float64 `json:"return_1m"`
	Return3M   float64 `json:"return_3m"`
	VolumeRate float64 `json:"volume_rate"`

	// Technical
	RSI       float64 `json:"rsi"`
	MACD      float64 `json:"macd"`
	MA20Cross int     `json:"ma20_cross"` // -1: 하락, 0: 중립, 1: 상승

	// Value
	PER float64 `json:"per"`
	PBR float64 `json:"pbr"`
	PSR float64 `json:"psr"`

	// Quality
	ROE       float64 `json:"roe"`
	DebtRatio float64 `json:"debt_ratio"`

	// Flow (수급)
	ForeignNet5D  int64 `json:"foreign_net_5d"`
	ForeignNet20D int64 `json:"foreign_net_20d"`
	InstNet5D     int64 `json:"inst_net_5d"`
	InstNet20D    int64 `json:"inst_net_20d"`
}

// EventSignal represents an event-driven signal
type EventSignal struct {
	Type      string    `json:"type"`      // "earnings", "buyback", "dividend", etc.
	Score     float64   `json:"score"`     // -1.0 ~ 1.0
	Source    string    `json:"source"`    // "DART", "KIS", "Naver", etc.
	Timestamp time.Time `json:"timestamp"` // When the event occurred
}

// Get returns signals for a specific stock code
func (s *SignalSet) Get(code string) (*StockSignals, bool) {
	signals, exists := s.Signals[code]
	return signals, exists
}

// Count returns the number of stocks with signals
func (s *SignalSet) Count() int {
	return len(s.Signals)
}

// TotalScore calculates the total signal score for a stock
// Weights: Momentum(0.25), Technical(0.20), Value(0.20), Quality(0.20), Flow(0.10), Event(0.05)
func (ss *StockSignals) TotalScore() float64 {
	return ss.Momentum*0.25 +
		ss.Technical*0.20 +
		ss.Value*0.20 +
		ss.Quality*0.20 +
		ss.Flow*0.10 +
		ss.Event*0.05
}

// IsPositive checks if the overall signal is positive
func (ss *StockSignals) IsPositive() bool {
	return ss.TotalScore() > 0
}
