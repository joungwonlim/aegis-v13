package forecast

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// ForwardPriceData 전방 성과 계산에 필요한 가격 데이터
type ForwardPriceData struct {
	Date  time.Time
	Open  float64
	High  float64
	Low   float64
	Close float64
}

// Tracker 전방 성과 추적기
type Tracker struct {
	log zerolog.Logger
}

// NewTracker 새 추적기 생성
func NewTracker(log zerolog.Logger) *Tracker {
	return &Tracker{
		log: log.With().Str("component", "forecast.tracker").Logger(),
	}
}

// CalculateForwardPerformance 이벤트 이후 전방 성과 계산
// forwardPrices: 이벤트 발생일 이후 5거래일 가격 데이터 (t+1, t+2, t+3, t+4, t+5)
// baseClose: 이벤트 발생일 종가
func (t *Tracker) CalculateForwardPerformance(
	ctx context.Context,
	eventID int64,
	baseClose float64,
	forwardPrices []ForwardPriceData,
) *contracts.ForwardPerformance {
	if len(forwardPrices) < 5 {
		t.log.Warn().
			Int64("event_id", eventID).
			Int("forward_days", len(forwardPrices)).
			Msg("insufficient forward data")
		return nil
	}

	if baseClose <= 0 {
		t.log.Warn().
			Int64("event_id", eventID).
			Float64("base_close", baseClose).
			Msg("invalid base close price")
		return nil
	}

	perf := &contracts.ForwardPerformance{
		EventID:  eventID,
		FilledAt: time.Now(),
	}

	// 수익률 계산 (t+1, t+2, t+3, t+5)
	if len(forwardPrices) >= 1 {
		perf.FwdRet1D = (forwardPrices[0].Close - baseClose) / baseClose
	}
	if len(forwardPrices) >= 2 {
		perf.FwdRet2D = (forwardPrices[1].Close - baseClose) / baseClose
	}
	if len(forwardPrices) >= 3 {
		perf.FwdRet3D = (forwardPrices[2].Close - baseClose) / baseClose
	}
	if len(forwardPrices) >= 5 {
		perf.FwdRet5D = (forwardPrices[4].Close - baseClose) / baseClose
	}

	// 5일간 최대 상승/하락 계산
	var maxHigh, minLow float64 = forwardPrices[0].High, forwardPrices[0].Low
	for i := 0; i < min(5, len(forwardPrices)); i++ {
		if forwardPrices[i].High > maxHigh {
			maxHigh = forwardPrices[i].High
		}
		if forwardPrices[i].Low < minLow {
			minLow = forwardPrices[i].Low
		}
	}
	perf.MaxRunup5D = (maxHigh - baseClose) / baseClose
	perf.MaxDrawdown5D = (minLow - baseClose) / baseClose

	// 3일간 갭 유지 여부 (시가가 이벤트일 종가 위 유지)
	perf.GapHold3D = true
	for i := 0; i < min(3, len(forwardPrices)); i++ {
		if forwardPrices[i].Open < baseClose {
			perf.GapHold3D = false
			break
		}
	}

	t.log.Debug().
		Int64("event_id", eventID).
		Float64("fwd_ret_1d", perf.FwdRet1D).
		Float64("fwd_ret_5d", perf.FwdRet5D).
		Float64("max_runup_5d", perf.MaxRunup5D).
		Float64("max_drawdown_5d", perf.MaxDrawdown5D).
		Bool("gap_hold_3d", perf.GapHold3D).
		Msg("forward performance calculated")

	return perf
}

// BatchCalculateForwardPerformance 일괄 전방 성과 계산
type EventWithPrices struct {
	EventID       int64
	BaseClose     float64
	ForwardPrices []ForwardPriceData
}

func (t *Tracker) BatchCalculateForwardPerformance(
	ctx context.Context,
	eventsWithPrices []EventWithPrices,
) []contracts.ForwardPerformance {
	var performances []contracts.ForwardPerformance

	for _, ewp := range eventsWithPrices {
		select {
		case <-ctx.Done():
			t.log.Warn().Msg("context cancelled during forward performance calculation")
			return performances
		default:
		}

		perf := t.CalculateForwardPerformance(ctx, ewp.EventID, ewp.BaseClose, ewp.ForwardPrices)
		if perf != nil {
			performances = append(performances, *perf)
		}
	}

	t.log.Info().
		Int("total_events", len(eventsWithPrices)).
		Int("calculated", len(performances)).
		Msg("batch forward performance calculation completed")

	return performances
}
