package forecast

import (
	"context"
	"math"
	"time"

	"github.com/rs/zerolog"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// PriceData 이벤트 감지에 필요한 가격 데이터
type PriceData struct {
	Code      string
	Date      time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int64
	PrevClose float64 // 전일 종가
	Sector    string
	MarketCap int64
}

// VolumeStats 거래량 통계 (20일 기준)
type VolumeStats struct {
	Mean   float64
	StdDev float64
}

// Detector 이벤트 감지기
type Detector struct {
	thresholds contracts.EventThresholds
	log        zerolog.Logger
}

// NewDetector 새 감지기 생성
func NewDetector(log zerolog.Logger) *Detector {
	return &Detector{
		thresholds: contracts.DefaultEventThresholds(),
		log:        log.With().Str("component", "forecast.detector").Logger(),
	}
}

// NewDetectorWithThresholds 커스텀 임계값으로 감지기 생성
func NewDetectorWithThresholds(thresholds contracts.EventThresholds, log zerolog.Logger) *Detector {
	return &Detector{
		thresholds: thresholds,
		log:        log.With().Str("component", "forecast.detector").Logger(),
	}
}

// DetectEvent 가격 데이터에서 이벤트 감지
func (d *Detector) DetectEvent(ctx context.Context, price PriceData, volumeStats *VolumeStats) *contracts.ForecastEvent {
	// 기본 유효성 검사
	if price.PrevClose <= 0 || price.High <= 0 || price.Low <= 0 {
		return nil
	}

	// 당일 수익률 계산
	dayReturn := (price.Close - price.PrevClose) / price.PrevClose

	// 고가 대비 종가 위치 (0~1)
	var closeToHigh float64
	if price.High > price.Low {
		closeToHigh = (price.Close - price.Low) / (price.High - price.Low)
	}

	// 갭 비율 계산
	gapRatio := (price.Open - price.PrevClose) / price.PrevClose

	// 거래량 z-score 계산
	var volumeZScore float64
	if volumeStats != nil && volumeStats.StdDev > 0 {
		volumeZScore = (float64(price.Volume) - volumeStats.Mean) / volumeStats.StdDev
	}

	// E1 조건 체크: 급등
	isE1 := dayReturn >= d.thresholds.MinDayReturn && closeToHigh >= d.thresholds.MinCloseToHigh

	if !isE1 {
		return nil
	}

	// E2 조건 체크: 갭 + 급등
	isE2 := isE1 && gapRatio >= d.thresholds.MinGapRatio

	// 거래량 z-score 필터 (설정된 경우)
	if d.thresholds.MinVolumeZScore > 0 && volumeZScore < d.thresholds.MinVolumeZScore {
		return nil
	}

	// 이벤트 타입 결정
	eventType := contracts.EventE1Surge
	if isE2 {
		eventType = contracts.EventE2GapSurge
	}

	event := &contracts.ForecastEvent{
		Code:            price.Code,
		Date:            price.Date,
		EventType:       eventType,
		DayReturn:       dayReturn,
		CloseToHigh:     closeToHigh,
		GapRatio:        gapRatio,
		VolumeZScore:    volumeZScore,
		Sector:          price.Sector,
		MarketCapBucket: contracts.MarketCapBucket(price.MarketCap),
		CreatedAt:       time.Now(),
	}

	d.log.Debug().
		Str("code", price.Code).
		Str("date", price.Date.Format("2006-01-02")).
		Str("event_type", string(eventType)).
		Float64("day_return", dayReturn).
		Float64("close_to_high", closeToHigh).
		Float64("gap_ratio", gapRatio).
		Float64("volume_z_score", volumeZScore).
		Msg("event detected")

	return event
}

// DetectEvents 여러 가격 데이터에서 이벤트 일괄 감지
func (d *Detector) DetectEvents(ctx context.Context, prices []PriceData, volumeStatsMap map[string]*VolumeStats) []contracts.ForecastEvent {
	var events []contracts.ForecastEvent

	for _, price := range prices {
		select {
		case <-ctx.Done():
			d.log.Warn().Msg("context cancelled during event detection")
			return events
		default:
		}

		stats := volumeStatsMap[price.Code]
		event := d.DetectEvent(ctx, price, stats)
		if event != nil {
			events = append(events, *event)
		}
	}

	d.log.Info().
		Int("total_prices", len(prices)).
		Int("detected_events", len(events)).
		Msg("batch event detection completed")

	return events
}

// CalculateVolumeStats 20일 거래량 통계 계산
func CalculateVolumeStats(volumes []int64) *VolumeStats {
	if len(volumes) == 0 {
		return nil
	}

	// 평균 계산
	var sum float64
	for _, v := range volumes {
		sum += float64(v)
	}
	mean := sum / float64(len(volumes))

	// 표준편차 계산
	var variance float64
	for _, v := range volumes {
		diff := float64(v) - mean
		variance += diff * diff
	}
	variance /= float64(len(volumes))
	stdDev := math.Sqrt(variance)

	return &VolumeStats{
		Mean:   mean,
		StdDev: stdDev,
	}
}
