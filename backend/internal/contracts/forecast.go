package contracts

import "time"

// ForecastEventType 이벤트 타입
type ForecastEventType string

const (
	// EventE1Surge 급등 이벤트 (수익률 >= 3.5%, 고가 대비 종가 >= 0.4)
	EventE1Surge ForecastEventType = "E1_SURGE"
	// EventE2GapSurge 갭+급등 이벤트 (E1 조건 + 갭 >= 1.5%)
	EventE2GapSurge ForecastEventType = "E2_GAP_SURGE"
)

// ForecastEvent 감지된 이벤트
type ForecastEvent struct {
	ID              int64             `json:"id"`
	Code            string            `json:"code"`
	Date            time.Time         `json:"date"`
	EventType       ForecastEventType `json:"event_type"`
	DayReturn       float64           `json:"day_return"`       // 당일 수익률
	CloseToHigh     float64           `json:"close_to_high"`    // 고가 대비 종가 위치 (0~1)
	GapRatio        float64           `json:"gap_ratio"`        // 갭 비율 (E2만)
	VolumeZScore    float64           `json:"volume_z_score"`   // 거래량 z-score
	Sector          string            `json:"sector"`           // 섹터
	MarketCapBucket string            `json:"market_cap_bucket"` // small/mid/large
	CreatedAt       time.Time         `json:"created_at"`
}

// ForwardPerformance 전방 성과
type ForwardPerformance struct {
	ID            int64     `json:"id"`
	EventID       int64     `json:"event_id"`
	FwdRet1D      float64   `json:"fwd_ret_1d"`      // t+1 수익률
	FwdRet2D      float64   `json:"fwd_ret_2d"`      // t+2 수익률
	FwdRet3D      float64   `json:"fwd_ret_3d"`      // t+3 수익률
	FwdRet5D      float64   `json:"fwd_ret_5d"`      // t+5 수익률
	MaxRunup5D    float64   `json:"max_runup_5d"`    // 5일간 최대 상승
	MaxDrawdown5D float64   `json:"max_drawdown_5d"` // 5일간 최대 하락
	GapHold3D     bool      `json:"gap_hold_3d"`     // 3일간 갭 유지 여부
	FilledAt      time.Time `json:"filled_at"`
}

// ForecastStatsLevel 통계 집계 레벨
type ForecastStatsLevel string

const (
	StatsLevelSymbol ForecastStatsLevel = "SYMBOL"
	StatsLevelSector ForecastStatsLevel = "SECTOR"
	StatsLevelBucket ForecastStatsLevel = "BUCKET"
	StatsLevelMarket ForecastStatsLevel = "MARKET"
)

// ForecastStats 통계 집계
type ForecastStats struct {
	ID          int64              `json:"id"`
	Level       ForecastStatsLevel `json:"level"`      // SYMBOL/SECTOR/BUCKET/MARKET
	Key         string             `json:"key"`        // 종목코드/섹터명/버킷명/ALL
	EventType   ForecastEventType  `json:"event_type"`
	SampleCount int                `json:"sample_count"`
	AvgRet1D    float64            `json:"avg_ret_1d"`
	AvgRet2D    float64            `json:"avg_ret_2d"`
	AvgRet3D    float64            `json:"avg_ret_3d"`
	AvgRet5D    float64            `json:"avg_ret_5d"`
	WinRate1D   float64            `json:"win_rate_1d"` // 1일 후 양수 비율
	WinRate5D   float64            `json:"win_rate_5d"` // 5일 후 양수 비율
	P10MDD      float64            `json:"p10_mdd"`     // 하위 10% MDD
	UpdatedAt   time.Time          `json:"updated_at"`
}

// ForecastPrediction 예측 결과
type ForecastPrediction struct {
	Code        string            `json:"code"`
	Date        time.Time         `json:"date"`
	EventType   ForecastEventType `json:"event_type"`
	ExpRet1D    float64           `json:"exp_ret_1d"`   // 기대 수익률 (베이지안 조정)
	ExpRet5D    float64           `json:"exp_ret_5d"`
	Confidence  float64           `json:"confidence"`   // 신뢰도 (샘플 수 기반)
	FallbackLvl string            `json:"fallback_lvl"` // 사용된 폴백 레벨
	P10MDD      float64           `json:"p10_mdd"`      // 리스크 지표
}

// MarketCapBucket 시가총액 구간 결정
func MarketCapBucket(marketCap int64) string {
	switch {
	case marketCap >= 10_000_000_000_000: // 10조 이상
		return "large"
	case marketCap >= 1_000_000_000_000: // 1조 이상
		return "mid"
	default:
		return "small"
	}
}

// EventThresholds 이벤트 감지 임계값
type EventThresholds struct {
	MinDayReturn    float64 // 최소 당일 수익률 (기본: 0.035)
	MinCloseToHigh  float64 // 최소 고가 대비 종가 (기본: 0.4)
	MinGapRatio     float64 // 최소 갭 비율 (기본: 0.015)
	MinVolumeZScore float64 // 최소 거래량 z-score (선택적)
}

// DefaultEventThresholds 기본 임계값
func DefaultEventThresholds() EventThresholds {
	return EventThresholds{
		MinDayReturn:    0.035, // 3.5%
		MinCloseToHigh:  0.4,   // 40%
		MinGapRatio:     0.015, // 1.5%
		MinVolumeZScore: 0,     // 필터링 안함
	}
}

// BayesianConfig 베이지안 수축 설정
type BayesianConfig struct {
	K               int     // 수축 강도 (기본: 10)
	MinSampleCount  int     // 최소 샘플 수 (폴백 조건)
	MaxConfidence   float64 // 최대 신뢰도 도달 샘플 수 (기본: 30)
}

// DefaultBayesianConfig 기본 베이지안 설정
func DefaultBayesianConfig() BayesianConfig {
	return BayesianConfig{
		K:              10,
		MinSampleCount: 5,
		MaxConfidence:  30,
	}
}
