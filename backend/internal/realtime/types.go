package realtime

import "time"

// PriceTick represents a real-time price update
// ⭐ SSOT: 실시간 가격 데이터 구조
type PriceTick struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Price       int64     `json:"price"`        // 현재가
	Change      int64     `json:"change"`       // 전일대비
	ChangeRate  float64   `json:"change_rate"`  // 등락율
	Volume      int64     `json:"volume"`       // 거래량
	Value       int64     `json:"value"`        // 거래대금
	High        int64     `json:"high"`         // 고가
	Low         int64     `json:"low"`          // 저가
	Open        int64     `json:"open"`         // 시가
	PrevClose   int64     `json:"prev_close"`   // 전일종가
	Timestamp   time.Time `json:"timestamp"`    // 시간
	Source      string    `json:"source"`       // 소스: "KIS_WS", "KIS_REST", "NAVER"
	IsStale     bool      `json:"is_stale"`     // 오래된 데이터 여부
}

// PriceSource represents the source of price data
type PriceSource string

const (
	SourceKISWebSocket PriceSource = "KIS_WS"
	SourceKISREST      PriceSource = "KIS_REST"
	SourceNaver        PriceSource = "NAVER"
)

// SourcePriority returns priority for source (higher = better)
func (s PriceSource) Priority() int {
	switch s {
	case SourceKISWebSocket:
		return 3
	case SourceKISREST:
		return 2
	case SourceNaver:
		return 1
	default:
		return 0
	}
}

// SymbolPriority represents priority for symbol subscription
type SymbolPriority struct {
	Code           string    `json:"code"`
	Score          float64   `json:"score"`           // Higher = more important
	LastTradeTime  time.Time `json:"last_trade_time"`
	Volatility     float64   `json:"volatility"`
	UserWatching   bool      `json:"user_watching"`
	InPortfolio    bool      `json:"in_portfolio"`
	HasActiveOrder bool      `json:"has_active_order"`
}

// CalculateScore calculates priority score for symbol
func (sp *SymbolPriority) CalculateScore() float64 {
	score := 0.0

	// Portfolio holdings: highest priority
	if sp.InPortfolio {
		score += 100.0
	}

	// Active orders: very high priority
	if sp.HasActiveOrder {
		score += 80.0
	}

	// User watching: high priority
	if sp.UserWatching {
		score += 50.0
	}

	// Volatility: moderate priority
	score += sp.Volatility * 10.0

	// Recent activity
	if time.Since(sp.LastTradeTime) < time.Hour {
		score += 20.0
	}

	sp.Score = score
	return score
}

// Tier represents polling tier
type Tier int

const (
	Tier1 Tier = 1 // Highest priority (2-5sec)
	Tier2 Tier = 2 // Medium priority (10-15sec)
	Tier3 Tier = 3 // Lowest priority (30-60sec)
)

// GetTierFromScore determines tier from priority score
func GetTierFromScore(score float64) Tier {
	if score >= 80.0 {
		return Tier1
	} else if score >= 30.0 {
		return Tier2
	}
	return Tier3
}
