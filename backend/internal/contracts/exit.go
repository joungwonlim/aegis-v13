package contracts

import "time"

// =============================================================================
// Exit Rules Configuration (ATR 기반 동적 청산 v1.2.0)
// ⭐ SSOT: 청산규칙 설정은 여기서만
// =============================================================================

// ExitRulesConfig 청산 규칙 설정
type ExitRulesConfig struct {
	// ===== ATR 기반 익절 설정 =====
	UseATRBased bool `json:"use_atr_based" yaml:"use_atr_based"` // ATR 기반 트리거 사용

	// TP1: ATR * 1.5, clamp [6%, 8%], 25% 매도
	TP1ATRMultiplier float64 `json:"tp1_atr_multiplier" yaml:"tp1_atr_multiplier"`
	TP1MinPercent    float64 `json:"tp1_min_percent" yaml:"tp1_min_percent"`
	TP1MaxPercent    float64 `json:"tp1_max_percent" yaml:"tp1_max_percent"`
	TP1SellPercent   float64 `json:"tp1_sell_percent" yaml:"tp1_sell_percent"`

	// TP2: ATR * 2.5, clamp [10%, 12%], 25% 매도
	TP2ATRMultiplier float64 `json:"tp2_atr_multiplier" yaml:"tp2_atr_multiplier"`
	TP2MinPercent    float64 `json:"tp2_min_percent" yaml:"tp2_min_percent"`
	TP2MaxPercent    float64 `json:"tp2_max_percent" yaml:"tp2_max_percent"`
	TP2SellPercent   float64 `json:"tp2_sell_percent" yaml:"tp2_sell_percent"`

	// TP3: ATR * 3.5, clamp [15%, 18%], 20% 매도
	TP3ATRMultiplier float64 `json:"tp3_atr_multiplier" yaml:"tp3_atr_multiplier"`
	TP3MinPercent    float64 `json:"tp3_min_percent" yaml:"tp3_min_percent"`
	TP3MaxPercent    float64 `json:"tp3_max_percent" yaml:"tp3_max_percent"`
	TP3SellPercent   float64 `json:"tp3_sell_percent" yaml:"tp3_sell_percent"`

	// ===== 손절 설정 =====
	FirstStopPercent     float64 `json:"first_stop_percent" yaml:"first_stop_percent"`         // 1차 손절 (-3%)
	FirstStopSellPercent float64 `json:"first_stop_sell_percent" yaml:"first_stop_sell_percent"` // 1차 손절 매도 비율 (50%)
	SecondStopPercent    float64 `json:"second_stop_percent" yaml:"second_stop_percent"`       // 2차 손절 (-5%, 전량)
	HardStopPercent      float64 `json:"hard_stop_percent" yaml:"hard_stop_percent"`           // 하드 스탑 (-7%)

	// Stop Floor: TP1 이후 손익분기점 + buffer
	StopFloorBuffer float64 `json:"stop_floor_buffer" yaml:"stop_floor_buffer"` // 손익분기점 버퍼 (0.6%)

	// HWM Trailing Stop: TP3 이후 트레일링
	TrailATRMultiplier float64 `json:"trail_atr_multiplier" yaml:"trail_atr_multiplier"` // 트레일 ATR 배수 (2.0)
	TrailMinPercent    float64 `json:"trail_min_percent" yaml:"trail_min_percent"`       // 최소 트레일 거리 (3%)
	TrailMaxPercent    float64 `json:"trail_max_percent" yaml:"trail_max_percent"`       // 최대 트레일 거리 (5%)

	// 모니터링 주기
	CheckIntervalSeconds int `json:"check_interval_seconds" yaml:"check_interval_seconds"`
}

// DefaultExitRulesConfig 기본 청산 규칙 설정 반환
func DefaultExitRulesConfig() *ExitRulesConfig {
	return &ExitRulesConfig{
		// ATR 기반 활성화
		UseATRBased: true,

		// TP1: ATR * 1.5, clamp [6%, 8%], 25% 매도
		TP1ATRMultiplier: 1.5,
		TP1MinPercent:    6.0,
		TP1MaxPercent:    8.0,
		TP1SellPercent:   25.0,

		// TP2: ATR * 2.5, clamp [10%, 12%], 25% 매도
		TP2ATRMultiplier: 2.5,
		TP2MinPercent:    10.0,
		TP2MaxPercent:    12.0,
		TP2SellPercent:   25.0,

		// TP3: ATR * 3.5, clamp [15%, 18%], 20% 매도
		TP3ATRMultiplier: 3.5,
		TP3MinPercent:    15.0,
		TP3MaxPercent:    18.0,
		TP3SellPercent:   20.0,

		// 손절
		FirstStopPercent:     -3.0,
		FirstStopSellPercent: 50.0,
		SecondStopPercent:    -5.0,
		HardStopPercent:      -7.0,

		// Stop Floor: 손익분기점 + 0.6% 버퍼
		StopFloorBuffer: 0.6,

		// HWM Trailing: ATR * 2.0, clamp [3%, 5%]
		TrailATRMultiplier: 2.0,
		TrailMinPercent:    3.0,
		TrailMaxPercent:    5.0,

		// 모니터링
		CheckIntervalSeconds: 30,
	}
}

// =============================================================================
// Position State Machine
// =============================================================================

// PositionState 포지션 상태 (상태 머신)
type PositionState string

const (
	PositionStateOpen    PositionState = "S0_OPEN"     // 진입 완료, 아직 익절 없음
	PositionStateTP1Done PositionState = "S1_TP1_DONE" // TP1 완료, StopFloor 활성화
	PositionStateTP2Done PositionState = "S2_TP2_DONE" // TP2 완료
	PositionStateTP3Done PositionState = "S3_TP3_DONE" // TP3 완료, HWM Trailing 활성화
	PositionStateExiting PositionState = "S4_EXITING"  // 트레일링 스탑 발동, 청산 진행 중
	PositionStateClosed  PositionState = "S5_CLOSED"   // 완전 청산
)

// =============================================================================
// Exit Signal Types
// =============================================================================

// ExitReason 청산 사유
type ExitReason string

const (
	ExitReasonTP1        ExitReason = "TP1"         // 1차 익절 (+6~8%)
	ExitReasonTP2        ExitReason = "TP2"         // 2차 익절 (+10~12%)
	ExitReasonTP3        ExitReason = "TP3"         // 3차 익절 (+15~18%)
	ExitReasonHardStop   ExitReason = "HARD_STOP"   // 하드 스탑 (-7% 전량)
	ExitReasonFirstStop  ExitReason = "FIRST_STOP"  // 1차 손절 (-3% 50%)
	ExitReasonSecondStop ExitReason = "SECOND_STOP" // 2차 손절 (-5% 전량)
	ExitReasonStopFloor  ExitReason = "STOP_FLOOR"  // 스탑 플로어 (손익분기점+0.6%)
	ExitReasonHWMTrail   ExitReason = "HWM_TRAIL"   // HWM 트레일링 스탑
	ExitReasonManual     ExitReason = "MANUAL"      // 수동 청산
)

// ExitSignal 청산 신호
// ⭐ SSOT: 청산 신호 데이터는 여기서만
type ExitSignal struct {
	PositionID   string     `json:"position_id"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Reason       ExitReason `json:"reason"`
	CurrentPrice int64      `json:"current_price"`
	EntryPrice   int64      `json:"entry_price"`
	PnLPercent   float64    `json:"pnl_percent"`
	SellQuantity int        `json:"sell_quantity"`
	IsPartial    bool       `json:"is_partial"` // 분할 청산 여부
	Message      string     `json:"message"`
	TriggeredAt  time.Time  `json:"triggered_at"`
}

// MonitoredPosition 모니터링 중인 포지션
// ⭐ SSOT: 포지션 모니터링 상태는 여기서만
type MonitoredPosition struct {
	ID                string        `json:"id"`
	Code              string        `json:"code"`
	Name              string        `json:"name"`
	EntryPrice        int64         `json:"entry_price"`
	ReferencePrice    int64         `json:"reference_price"`    // 기준점 (익절 시 갱신)
	InitialQuantity   int           `json:"initial_quantity"`   // 최초 매수 수량
	RemainingQuantity int           `json:"remaining_quantity"` // 잔여 수량
	CurrentPrice      int64         `json:"current_price"`
	HighestPrice      int64         `json:"highest_price"`
	HighestPriceTime  time.Time     `json:"highest_price_time"`
	UnrealizedPnL     float64       `json:"unrealized_pnl"` // 진입가 대비 손익률
	RefPnL            float64       `json:"ref_pnl"`        // 기준점 대비 손익률
	State             PositionState `json:"state"`
	ATRPercent        float64       `json:"atr_percent"` // ATR% (진입 시 계산)

	// 트리거 가격 (ATR 기반)
	TP1TriggerPrice int64 `json:"tp1_trigger_price"`
	TP2TriggerPrice int64 `json:"tp2_trigger_price"`
	TP3TriggerPrice int64 `json:"tp3_trigger_price"`
	StopFloorPrice  int64 `json:"stop_floor_price"` // Stop Floor 가격
	TrailStopPrice  int64 `json:"trail_stop_price"` // HWM 트레일링 스탑 가격

	// 상태 플래그
	FirstStopTriggered bool `json:"first_stop_triggered"`
	TP1Done            bool `json:"tp1_done"`
	TP2Done            bool `json:"tp2_done"`
	TP3Done            bool `json:"tp3_done"`
	TakeProfitCount    int  `json:"take_profit_count"`

	// 시간
	EntryTime   time.Time `json:"entry_time"`
	LastUpdated time.Time `json:"last_updated"`
}

// ScaleOutRecord 분할 청산 기록
type ScaleOutRecord struct {
	Level        int       `json:"level"`
	TriggerPrice int64     `json:"trigger_price"`
	SoldQuantity int       `json:"sold_quantity"`
	SoldPercent  float64   `json:"sold_percent"`
	PnLPercent   float64   `json:"pnl_percent"`
	ExecutedAt   time.Time `json:"executed_at"`
}
