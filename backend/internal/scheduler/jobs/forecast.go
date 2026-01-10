package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/internal/forecast"
	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// ForecastJob runs the forecast pipeline daily
// Schedule: 6:30 PM (after universe generation at 6 PM)
type ForecastJob struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

// NewForecastJob creates a new forecast job
func NewForecastJob(pool *pgxpool.Pool, log *logger.Logger) *ForecastJob {
	return &ForecastJob{
		pool:   pool,
		logger: log,
	}
}

// Name returns the job name
func (j *ForecastJob) Name() string {
	return "forecast_pipeline"
}

// Schedule returns the cron schedule (6:30 PM daily, after universe generation)
func (j *ForecastJob) Schedule() string {
	return "0 30 18 * * *" // 6:30 PM daily (with seconds)
}

// Run executes the forecast pipeline
func (j *ForecastJob) Run(ctx context.Context) error {
	j.logger.Info("Starting scheduled forecast pipeline")

	// 저장소 초기화
	forecastRepo := forecast.NewRepository(j.pool)
	priceRepo := s0_data.NewPriceRepository(j.pool)

	// 오늘 날짜만 처리
	today := time.Now()

	// ===== 1. Event Detection =====
	j.logger.Info("Step 1: Event Detection")

	detector := forecast.NewDetector(j.logger.Zerolog())

	// 오늘 가격 데이터 조회
	prices, err := priceRepo.GetDailyPrices(ctx, today)
	if err != nil {
		return fmt.Errorf("get daily prices: %w", err)
	}

	if len(prices) == 0 {
		j.logger.Info("No price data for today, skipping")
		return nil
	}

	// 가격 데이터 변환
	var priceDataList []forecast.PriceData
	for _, p := range prices {
		if p.PrevClose == 0 {
			continue
		}
		priceDataList = append(priceDataList, forecast.PriceData{
			Code:      p.Code,
			Date:      p.Date,
			Open:      p.Open,
			High:      p.High,
			Low:       p.Low,
			Close:     p.Close,
			Volume:    p.Volume,
			PrevClose: p.PrevClose,
			Sector:    p.Sector,
			MarketCap: p.MarketCap,
		})
	}

	// 이벤트 감지
	events := detector.DetectEvents(ctx, priceDataList, nil)
	if len(events) > 0 {
		if err := forecastRepo.SaveEvents(ctx, events); err != nil {
			return fmt.Errorf("save events: %w", err)
		}
		j.logger.WithFields(map[string]interface{}{
			"date":   today.Format("2006-01-02"),
			"events": len(events),
		}).Info("Events detected and saved")
	}

	// ===== 2. Fill Forward Performance =====
	j.logger.Info("Step 2: Fill Forward Performance")

	tracker := forecast.NewTracker(j.logger.Zerolog())

	// 전방 성과가 없는 이벤트 조회
	eventsWithoutFwd, err := forecastRepo.GetEventsWithoutForward(ctx)
	if err != nil {
		return fmt.Errorf("get events without forward: %w", err)
	}

	var filled int
	for _, event := range eventsWithoutFwd {
		// 이벤트 이후 5거래일 가격 조회
		forwardPrices, err := priceRepo.GetForwardPrices(ctx, event.Code, event.Date, 5)
		if err != nil || len(forwardPrices) < 5 {
			continue
		}

		// 가격 데이터 변환
		var fwdPriceData []forecast.ForwardPriceData
		for _, p := range forwardPrices {
			fwdPriceData = append(fwdPriceData, forecast.ForwardPriceData{
				Date:  p.Date,
				Open:  p.Open,
				High:  p.High,
				Low:   p.Low,
				Close: p.Close,
			})
		}

		// 이벤트일 종가 조회
		basePrice, err := priceRepo.GetPrice(ctx, event.Code, event.Date)
		if err != nil {
			continue
		}

		// 전방 성과 계산
		perf := tracker.CalculateForwardPerformance(ctx, event.ID, basePrice.Close, fwdPriceData)
		if perf == nil {
			continue
		}

		// 저장
		if err := forecastRepo.SaveForwardPerformance(ctx, *perf); err != nil {
			continue
		}
		filled++
	}

	j.logger.WithFields(map[string]interface{}{
		"total":  len(eventsWithoutFwd),
		"filled": filled,
	}).Info("Forward performance filled")

	// ===== 3. Aggregate Statistics =====
	j.logger.Info("Step 3: Aggregate Statistics")

	aggregator := forecast.NewAggregator(j.logger.Zerolog())

	// 이벤트와 전방 성과 조회
	eventsWithPerf, err := forecastRepo.GetEventsWithPerformance(ctx)
	if err != nil {
		return fmt.Errorf("get events with performance: %w", err)
	}

	if len(eventsWithPerf) == 0 {
		j.logger.Warn("No events with performance to aggregate")
		return nil
	}

	// 통계 집계
	stats := aggregator.AggregateAll(ctx, eventsWithPerf)

	// 저장
	if err := forecastRepo.SaveAllStats(ctx, stats); err != nil {
		return fmt.Errorf("save stats: %w", err)
	}

	// 마켓 레벨 통계 로그
	for _, s := range stats {
		if s.Level == contracts.StatsLevelMarket {
			j.logger.WithFields(map[string]interface{}{
				"event_type":  s.EventType,
				"samples":     s.SampleCount,
				"avg_ret_5d":  fmt.Sprintf("%.2f%%", s.AvgRet5D*100),
				"win_rate_5d": fmt.Sprintf("%.1f%%", s.WinRate5D*100),
			}).Info("Market level stats")
		}
	}

	j.logger.WithFields(map[string]interface{}{
		"total_stats": len(stats),
	}).Info("Forecast pipeline completed successfully")

	return nil
}
