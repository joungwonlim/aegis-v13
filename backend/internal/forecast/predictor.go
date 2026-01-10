package forecast

import (
	"context"
	"math"
	"time"

	"github.com/rs/zerolog"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// StatsProvider 통계 데이터 제공 인터페이스
type StatsProvider interface {
	GetStats(ctx context.Context, level contracts.ForecastStatsLevel, key string, eventType contracts.ForecastEventType) (*contracts.ForecastStats, error)
}

// Predictor 예측 생성기
type Predictor struct {
	config   contracts.BayesianConfig
	provider StatsProvider
	log      zerolog.Logger
}

// NewPredictor 새 예측기 생성
func NewPredictor(provider StatsProvider, log zerolog.Logger) *Predictor {
	return &Predictor{
		config:   contracts.DefaultBayesianConfig(),
		provider: provider,
		log:      log.With().Str("component", "forecast.predictor").Logger(),
	}
}

// NewPredictorWithConfig 커스텀 설정으로 예측기 생성
func NewPredictorWithConfig(config contracts.BayesianConfig, provider StatsProvider, log zerolog.Logger) *Predictor {
	return &Predictor{
		config:   config,
		provider: provider,
		log:      log.With().Str("component", "forecast.predictor").Logger(),
	}
}

// Predict 이벤트에 대한 예측 생성
func (p *Predictor) Predict(ctx context.Context, event contracts.ForecastEvent) (*contracts.ForecastPrediction, error) {
	// 4단계 폴백으로 통계 조회
	stats, fallbackLevel := p.getStatsWithFallback(ctx, event)
	if stats == nil {
		p.log.Warn().
			Str("code", event.Code).
			Str("event_type", string(event.EventType)).
			Msg("no stats available for prediction")
		return nil, nil
	}

	// 시장 평균 조회 (베이지안 수축용)
	marketStats, _ := p.provider.GetStats(ctx, contracts.StatsLevelMarket, "ALL", event.EventType)

	// 베이지안 수축 적용
	var expRet1D, expRet5D float64
	if marketStats != nil && stats.SampleCount < 30 {
		expRet1D = p.bayesianShrink(stats.AvgRet1D, marketStats.AvgRet1D, stats.SampleCount)
		expRet5D = p.bayesianShrink(stats.AvgRet5D, marketStats.AvgRet5D, stats.SampleCount)
	} else {
		expRet1D = stats.AvgRet1D
		expRet5D = stats.AvgRet5D
	}

	// 신뢰도 계산
	confidence := p.calculateConfidence(stats.SampleCount)

	prediction := &contracts.ForecastPrediction{
		Code:        event.Code,
		Date:        event.Date,
		EventType:   event.EventType,
		ExpRet1D:    expRet1D,
		ExpRet5D:    expRet5D,
		Confidence:  confidence,
		FallbackLvl: fallbackLevel,
		P10MDD:      stats.P10MDD,
	}

	p.log.Debug().
		Str("code", event.Code).
		Str("event_type", string(event.EventType)).
		Str("fallback_level", fallbackLevel).
		Int("sample_count", stats.SampleCount).
		Float64("exp_ret_5d", expRet5D).
		Float64("confidence", confidence).
		Msg("prediction generated")

	return prediction, nil
}

// getStatsWithFallback 4단계 폴백으로 통계 조회
func (p *Predictor) getStatsWithFallback(ctx context.Context, event contracts.ForecastEvent) (*contracts.ForecastStats, string) {
	// 1. SYMBOL 레벨
	stats, err := p.provider.GetStats(ctx, contracts.StatsLevelSymbol, event.Code, event.EventType)
	if err == nil && stats != nil && stats.SampleCount >= p.config.MinSampleCount {
		return stats, string(contracts.StatsLevelSymbol)
	}

	// 2. SECTOR 레벨
	if event.Sector != "" {
		stats, err = p.provider.GetStats(ctx, contracts.StatsLevelSector, event.Sector, event.EventType)
		if err == nil && stats != nil && stats.SampleCount >= p.config.MinSampleCount {
			return stats, string(contracts.StatsLevelSector)
		}
	}

	// 3. BUCKET 레벨
	if event.MarketCapBucket != "" {
		stats, err = p.provider.GetStats(ctx, contracts.StatsLevelBucket, event.MarketCapBucket, event.EventType)
		if err == nil && stats != nil && stats.SampleCount >= p.config.MinSampleCount {
			return stats, string(contracts.StatsLevelBucket)
		}
	}

	// 4. MARKET 레벨
	stats, err = p.provider.GetStats(ctx, contracts.StatsLevelMarket, "ALL", event.EventType)
	if err == nil && stats != nil {
		return stats, string(contracts.StatsLevelMarket)
	}

	return nil, ""
}

// bayesianShrink 베이지안 수축 적용
// sampleMean: 샘플 평균
// marketMean: 시장 평균 (사전 분포)
// n: 샘플 수
func (p *Predictor) bayesianShrink(sampleMean, marketMean float64, n int) float64 {
	// weight = n / (n + K)
	// K가 클수록 시장 평균 쪽으로 더 많이 수축
	weight := float64(n) / float64(n+p.config.K)
	return weight*sampleMean + (1-weight)*marketMean
}

// calculateConfidence 신뢰도 계산
func (p *Predictor) calculateConfidence(sampleCount int) float64 {
	// 샘플 수 30개 이상이면 신뢰도 1.0
	confidence := float64(sampleCount) / p.config.MaxConfidence
	return math.Min(1.0, confidence)
}

// BatchPredict 일괄 예측 생성
func (p *Predictor) BatchPredict(ctx context.Context, events []contracts.ForecastEvent) ([]contracts.ForecastPrediction, error) {
	var predictions []contracts.ForecastPrediction

	for _, event := range events {
		select {
		case <-ctx.Done():
			p.log.Warn().Msg("context cancelled during batch prediction")
			return predictions, ctx.Err()
		default:
		}

		pred, err := p.Predict(ctx, event)
		if err != nil {
			p.log.Error().Err(err).
				Str("code", event.Code).
				Msg("prediction failed")
			continue
		}
		if pred != nil {
			predictions = append(predictions, *pred)
		}
	}

	p.log.Info().
		Int("total_events", len(events)).
		Int("predictions", len(predictions)).
		Msg("batch prediction completed")

	return predictions, nil
}

// PredictForDate 특정 날짜의 이벤트에 대한 예측 생성
func (p *Predictor) PredictForDate(ctx context.Context, date time.Time, events []contracts.ForecastEvent) ([]contracts.ForecastPrediction, error) {
	// 해당 날짜의 이벤트만 필터링
	var dateEvents []contracts.ForecastEvent
	for _, e := range events {
		if e.Date.Format("2006-01-02") == date.Format("2006-01-02") {
			dateEvents = append(dateEvents, e)
		}
	}

	return p.BatchPredict(ctx, dateEvents)
}
