package forecast

import (
	"context"
	"sort"
	"time"

	"github.com/rs/zerolog"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// EventWithPerformance 이벤트와 전방 성과 결합
type EventWithPerformance struct {
	Event       contracts.ForecastEvent
	Performance contracts.ForwardPerformance
}

// Aggregator 통계 집계기
type Aggregator struct {
	minSampleCount int
	log            zerolog.Logger
}

// NewAggregator 새 집계기 생성
func NewAggregator(log zerolog.Logger) *Aggregator {
	return &Aggregator{
		minSampleCount: 5, // 폴백 조건
		log:            log.With().Str("component", "forecast.aggregator").Logger(),
	}
}

// NewAggregatorWithMinSample 최소 샘플 수 지정하여 집계기 생성
func NewAggregatorWithMinSample(minSampleCount int, log zerolog.Logger) *Aggregator {
	return &Aggregator{
		minSampleCount: minSampleCount,
		log:            log.With().Str("component", "forecast.aggregator").Logger(),
	}
}

// AggregateAll 모든 레벨에 대해 통계 집계
func (a *Aggregator) AggregateAll(ctx context.Context, data []EventWithPerformance) []contracts.ForecastStats {
	var allStats []contracts.ForecastStats

	// 이벤트 타입별로 분리
	e1Data := filterByEventType(data, contracts.EventE1Surge)
	e2Data := filterByEventType(data, contracts.EventE2GapSurge)

	// E1 통계
	allStats = append(allStats, a.aggregateByLevel(ctx, e1Data, contracts.EventE1Surge)...)

	// E2 통계
	allStats = append(allStats, a.aggregateByLevel(ctx, e2Data, contracts.EventE2GapSurge)...)

	a.log.Info().
		Int("total_stats", len(allStats)).
		Int("e1_data", len(e1Data)).
		Int("e2_data", len(e2Data)).
		Msg("aggregation completed")

	return allStats
}

func (a *Aggregator) aggregateByLevel(ctx context.Context, data []EventWithPerformance, eventType contracts.ForecastEventType) []contracts.ForecastStats {
	var stats []contracts.ForecastStats

	// 1. SYMBOL 레벨
	symbolGroups := groupByKey(data, func(d EventWithPerformance) string { return d.Event.Code })
	for key, group := range symbolGroups {
		if s := a.calculateStats(contracts.StatsLevelSymbol, key, eventType, group); s != nil {
			stats = append(stats, *s)
		}
	}

	// 2. SECTOR 레벨
	sectorGroups := groupByKey(data, func(d EventWithPerformance) string { return d.Event.Sector })
	for key, group := range sectorGroups {
		if key == "" {
			continue
		}
		if s := a.calculateStats(contracts.StatsLevelSector, key, eventType, group); s != nil {
			stats = append(stats, *s)
		}
	}

	// 3. BUCKET 레벨 (시가총액 구간)
	bucketGroups := groupByKey(data, func(d EventWithPerformance) string { return d.Event.MarketCapBucket })
	for key, group := range bucketGroups {
		if key == "" {
			continue
		}
		if s := a.calculateStats(contracts.StatsLevelBucket, key, eventType, group); s != nil {
			stats = append(stats, *s)
		}
	}

	// 4. MARKET 레벨 (전체)
	if s := a.calculateStats(contracts.StatsLevelMarket, "ALL", eventType, data); s != nil {
		stats = append(stats, *s)
	}

	return stats
}

func (a *Aggregator) calculateStats(level contracts.ForecastStatsLevel, key string, eventType contracts.ForecastEventType, data []EventWithPerformance) *contracts.ForecastStats {
	n := len(data)
	if n == 0 {
		return nil
	}

	// 수익률 합계
	var sumRet1D, sumRet2D, sumRet3D, sumRet5D float64
	var winCount1D, winCount5D int
	var mddValues []float64

	for _, d := range data {
		sumRet1D += d.Performance.FwdRet1D
		sumRet2D += d.Performance.FwdRet2D
		sumRet3D += d.Performance.FwdRet3D
		sumRet5D += d.Performance.FwdRet5D

		if d.Performance.FwdRet1D > 0 {
			winCount1D++
		}
		if d.Performance.FwdRet5D > 0 {
			winCount5D++
		}

		mddValues = append(mddValues, d.Performance.MaxDrawdown5D)
	}

	// P10 MDD 계산 (하위 10%)
	p10MDD := calculatePercentile(mddValues, 10)

	stats := &contracts.ForecastStats{
		Level:       level,
		Key:         key,
		EventType:   eventType,
		SampleCount: n,
		AvgRet1D:    sumRet1D / float64(n),
		AvgRet2D:    sumRet2D / float64(n),
		AvgRet3D:    sumRet3D / float64(n),
		AvgRet5D:    sumRet5D / float64(n),
		WinRate1D:   float64(winCount1D) / float64(n),
		WinRate5D:   float64(winCount5D) / float64(n),
		P10MDD:      p10MDD,
		UpdatedAt:   time.Now(),
	}

	a.log.Debug().
		Str("level", string(level)).
		Str("key", key).
		Str("event_type", string(eventType)).
		Int("sample_count", n).
		Float64("avg_ret_5d", stats.AvgRet5D).
		Float64("win_rate_5d", stats.WinRate5D).
		Float64("p10_mdd", stats.P10MDD).
		Msg("stats calculated")

	return stats
}

// filterByEventType 이벤트 타입으로 필터링
func filterByEventType(data []EventWithPerformance, eventType contracts.ForecastEventType) []EventWithPerformance {
	var filtered []EventWithPerformance
	for _, d := range data {
		if d.Event.EventType == eventType {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// groupByKey 키로 그룹핑
func groupByKey(data []EventWithPerformance, keyFn func(EventWithPerformance) string) map[string][]EventWithPerformance {
	groups := make(map[string][]EventWithPerformance)
	for _, d := range data {
		key := keyFn(d)
		groups[key] = append(groups[key], d)
	}
	return groups
}

// calculatePercentile 백분위수 계산
func calculatePercentile(values []float64, percentile int) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// 백분위 인덱스 계산
	idx := int(float64(len(sorted)-1) * float64(percentile) / 100.0)
	return sorted[idx]
}
