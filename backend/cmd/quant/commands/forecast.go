package commands

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/internal/forecast"
	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

var forecastCmd = &cobra.Command{
	Use:   "forecast",
	Short: "Forecast 모듈 - 이벤트 감지 및 예측",
	Long: `Forecast 모듈은 가격 패턴 기반 이벤트를 감지하고 예측합니다.

이벤트 타입:
- E1_SURGE: 급등 (수익률 >= 3.5%, 고가 대비 종가 >= 0.4)
- E2_GAP_SURGE: 갭+급등 (E1 조건 + 갭 >= 1.5%)

명령어:
  detect       과거 데이터에서 이벤트 감지
  fill-forward 전방 성과 채우기
  aggregate    통계 집계
  run          전체 실행 (detect → fill-forward → aggregate)
  predict      특정 종목 예측 조회`,
}

var (
	// detect 플래그
	detectFrom string
	detectTo   string

	// predict 플래그
	predictCode string
)

var forecastDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "과거 데이터에서 이벤트 감지",
	Long: `과거 가격 데이터를 분석하여 E1/E2 이벤트를 감지합니다.

Example:
  go run ./cmd/quant forecast detect --from 2024-01-01 --to 2024-12-31
  go run ./cmd/quant forecast detect --from 2025-01-01`,
	RunE: runForecastDetect,
}

var forecastFillForwardCmd = &cobra.Command{
	Use:   "fill-forward",
	Short: "전방 성과 채우기",
	Long: `감지된 이벤트에 대해 t+1, t+2, t+3, t+5 전방 성과를 채웁니다.

Example:
  go run ./cmd/quant forecast fill-forward`,
	RunE: runForecastFillForward,
}

var forecastAggregateCmd = &cobra.Command{
	Use:   "aggregate",
	Short: "통계 집계",
	Long: `4단계 폴백 계층으로 통계를 집계합니다.
- SYMBOL: 종목별
- SECTOR: 섹터별
- BUCKET: 시가총액 구간별 (small/mid/large)
- MARKET: 전체 시장

Example:
  go run ./cmd/quant forecast aggregate`,
	RunE: runForecastAggregate,
}

var forecastRunCmd = &cobra.Command{
	Use:   "run",
	Short: "전체 실행 (detect → fill-forward → aggregate)",
	Long: `이벤트 감지, 전방 성과 채우기, 통계 집계를 순차 실행합니다.

Example:
  go run ./cmd/quant forecast run --from 2024-01-01
  go run ./cmd/quant forecast run`,
	RunE: runForecastRun,
}

var forecastPredictCmd = &cobra.Command{
	Use:   "predict",
	Short: "특정 종목 예측 조회",
	Long: `특정 종목의 최근 이벤트에 대한 예측을 조회합니다.

Example:
  go run ./cmd/quant forecast predict --code 005930
  go run ./cmd/quant forecast predict --code 000660`,
	RunE: runForecastPredict,
}

func init() {
	rootCmd.AddCommand(forecastCmd)
	forecastCmd.AddCommand(forecastDetectCmd)
	forecastCmd.AddCommand(forecastFillForwardCmd)
	forecastCmd.AddCommand(forecastAggregateCmd)
	forecastCmd.AddCommand(forecastRunCmd)
	forecastCmd.AddCommand(forecastPredictCmd)

	// detect 플래그
	forecastDetectCmd.Flags().StringVar(&detectFrom, "from", "", "시작 날짜 (YYYY-MM-DD)")
	forecastDetectCmd.Flags().StringVar(&detectTo, "to", "", "종료 날짜 (YYYY-MM-DD, 기본: 오늘)")

	// run 플래그 (detect와 동일)
	forecastRunCmd.Flags().StringVar(&detectFrom, "from", "", "시작 날짜 (YYYY-MM-DD)")
	forecastRunCmd.Flags().StringVar(&detectTo, "to", "", "종료 날짜 (YYYY-MM-DD, 기본: 오늘)")

	// predict 플래그
	forecastPredictCmd.Flags().StringVar(&predictCode, "code", "", "종목 코드")
	_ = forecastPredictCmd.MarkFlagRequired("code")
}

func runForecastDetect(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Forecast: Event Detection ===")

	ctx := cmd.Context()

	// 날짜 파싱
	var from, to time.Time
	var err error
	if detectFrom != "" {
		from, err = time.Parse("2006-01-02", detectFrom)
		if err != nil {
			return fmt.Errorf("invalid from date: %w", err)
		}
	} else {
		from = time.Now().AddDate(0, -3, 0) // 기본: 3개월 전
	}
	if detectTo != "" {
		to, err = time.Parse("2006-01-02", detectTo)
		if err != nil {
			return fmt.Errorf("invalid to date: %w", err)
		}
	} else {
		to = time.Now()
	}

	fmt.Printf("📅 Period: %s ~ %s\n\n", from.Format("2006-01-02"), to.Format("2006-01-02"))

	// 의존성 초기화
	cfg, log, db, err := initForecastDeps()
	if err != nil {
		return err
	}
	defer db.Close()
	_ = cfg

	// 저장소
	forecastRepo := forecast.NewRepository(db.Pool)
	priceRepo := s0_data.NewPriceRepository(db.Pool)

	// 감지기
	detector := forecast.NewDetector(log)

	// 날짜별 가격 데이터 조회 및 이벤트 감지
	var totalEvents int
	currentDate := from
	for !currentDate.After(to) {
		// 해당 날짜의 가격 데이터 조회
		prices, err := priceRepo.GetDailyPrices(ctx, currentDate)
		if err != nil {
			log.Warn().Err(err).Time("date", currentDate).Msg("failed to get prices")
			currentDate = currentDate.AddDate(0, 0, 1)
			continue
		}

		if len(prices) == 0 {
			currentDate = currentDate.AddDate(0, 0, 1)
			continue
		}

		// s0_data.PriceWithMeta를 forecast.PriceData로 변환
		var priceDataList []forecast.PriceData
		for _, p := range prices {
			// PrevClose가 0이면 스킵 (전일 데이터 없음)
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
			// 저장
			if err := forecastRepo.SaveEvents(ctx, events); err != nil {
				log.Error().Err(err).Time("date", currentDate).Msg("failed to save events")
			} else {
				totalEvents += len(events)
				log.Info().
					Time("date", currentDate).
					Int("events", len(events)).
					Msg("events detected and saved")
			}
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	fmt.Printf("\n✅ Detection completed: %d events detected\n", totalEvents)
	return nil
}

func runForecastFillForward(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Forecast: Fill Forward Performance ===")

	ctx := cmd.Context()

	// 의존성 초기화
	cfg, log, db, err := initForecastDeps()
	if err != nil {
		return err
	}
	defer db.Close()
	_ = cfg

	// 저장소
	forecastRepo := forecast.NewRepository(db.Pool)
	priceRepo := s0_data.NewPriceRepository(db.Pool)

	// 추적기
	tracker := forecast.NewTracker(log)

	// 전방 성과가 없는 이벤트 조회
	events, err := forecastRepo.GetEventsWithoutForward(ctx)
	if err != nil {
		return fmt.Errorf("get events without forward: %w", err)
	}

	fmt.Printf("📊 Events to fill: %d\n\n", len(events))

	if len(events) == 0 {
		fmt.Println("✅ All events already have forward performance")
		return nil
	}

	var filled int
	for _, event := range events {
		// 이벤트 이후 5거래일 가격 조회
		forwardPrices, err := priceRepo.GetForwardPrices(ctx, event.Code, event.Date, 5)
		if err != nil || len(forwardPrices) < 5 {
			log.Debug().
				Str("code", event.Code).
				Time("date", event.Date).
				Int("forward_days", len(forwardPrices)).
				Msg("insufficient forward data")
			continue
		}

		// s0_data.PriceWithMeta를 forecast.ForwardPriceData로 변환
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
			log.Warn().Err(err).Str("code", event.Code).Msg("failed to get base price")
			continue
		}

		// 전방 성과 계산
		perf := tracker.CalculateForwardPerformance(ctx, event.ID, basePrice.Close, fwdPriceData)
		if perf == nil {
			continue
		}

		// 저장
		if err := forecastRepo.SaveForwardPerformance(ctx, *perf); err != nil {
			log.Error().Err(err).Int64("event_id", event.ID).Msg("failed to save forward performance")
			continue
		}

		filled++
	}

	fmt.Printf("\n✅ Fill forward completed: %d/%d events filled\n", filled, len(events))
	return nil
}

func runForecastAggregate(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Forecast: Aggregate Statistics ===")

	ctx := cmd.Context()

	// 의존성 초기화
	cfg, log, db, err := initForecastDeps()
	if err != nil {
		return err
	}
	defer db.Close()
	_ = cfg

	// 저장소
	forecastRepo := forecast.NewRepository(db.Pool)

	// 집계기
	aggregator := forecast.NewAggregator(log)

	// 이벤트와 전방 성과 조회
	eventsWithPerf, err := forecastRepo.GetEventsWithPerformance(ctx)
	if err != nil {
		return fmt.Errorf("get events with performance: %w", err)
	}

	fmt.Printf("📊 Events with performance: %d\n", len(eventsWithPerf))

	if len(eventsWithPerf) == 0 {
		fmt.Println("⚠️ No events with performance to aggregate")
		return nil
	}

	// 통계 집계
	stats := aggregator.AggregateAll(ctx, eventsWithPerf)

	fmt.Printf("📈 Statistics calculated: %d entries\n", len(stats))

	// 저장
	if err := forecastRepo.SaveAllStats(ctx, stats); err != nil {
		return fmt.Errorf("save stats: %w", err)
	}

	// 결과 출력
	fmt.Println("\n=== Statistics Summary ===")
	for _, s := range stats {
		if s.Level == contracts.StatsLevelMarket {
			fmt.Printf("\n[%s] %s (%s)\n", s.Level, s.Key, s.EventType)
			fmt.Printf("  Samples: %d\n", s.SampleCount)
			fmt.Printf("  Avg Ret 1D: %.2f%%\n", s.AvgRet1D*100)
			fmt.Printf("  Avg Ret 5D: %.2f%%\n", s.AvgRet5D*100)
			fmt.Printf("  Win Rate 1D: %.1f%%\n", s.WinRate1D*100)
			fmt.Printf("  Win Rate 5D: %.1f%%\n", s.WinRate5D*100)
			fmt.Printf("  P10 MDD: %.2f%%\n", s.P10MDD*100)
		}
	}

	fmt.Printf("\n✅ Aggregation completed: %d statistics saved\n", len(stats))
	return nil
}

func runForecastRun(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Forecast: Full Pipeline ===")

	// 1. Detect
	fmt.Println("\n[1/3] Event Detection")
	if err := runForecastDetect(cmd, args); err != nil {
		return fmt.Errorf("detect: %w", err)
	}

	// 2. Fill Forward
	fmt.Println("\n[2/3] Fill Forward Performance")
	if err := runForecastFillForward(cmd, args); err != nil {
		return fmt.Errorf("fill-forward: %w", err)
	}

	// 3. Aggregate
	fmt.Println("\n[3/3] Aggregate Statistics")
	if err := runForecastAggregate(cmd, args); err != nil {
		return fmt.Errorf("aggregate: %w", err)
	}

	fmt.Println("\n✅ Full pipeline completed!")
	return nil
}

func runForecastPredict(cmd *cobra.Command, args []string) error {
	fmt.Printf("=== Forecast: Predict for %s ===\n\n", predictCode)

	ctx := cmd.Context()

	// 의존성 초기화
	cfg, log, db, err := initForecastDeps()
	if err != nil {
		return err
	}
	defer db.Close()
	_ = cfg

	// 저장소
	forecastRepo := forecast.NewRepository(db.Pool)

	// 예측기
	predictor := forecast.NewPredictor(forecastRepo, log)

	// 해당 종목의 최근 이벤트 조회
	events, err := forecastRepo.GetEventsByCode(ctx, predictCode)
	if err != nil {
		return fmt.Errorf("get events: %w", err)
	}

	if len(events) == 0 {
		fmt.Printf("⚠️ No events found for %s\n", predictCode)
		return nil
	}

	fmt.Printf("📊 Found %d events for %s\n\n", len(events), predictCode)

	// 최근 5개 이벤트에 대해 예측
	limit := 5
	if len(events) < limit {
		limit = len(events)
	}

	fmt.Println("=== Recent Predictions ===")
	for i := 0; i < limit; i++ {
		event := events[i]
		pred, err := predictor.Predict(ctx, event)
		if err != nil {
			log.Error().Err(err).Msg("prediction failed")
			continue
		}
		if pred == nil {
			fmt.Printf("\n[%s] %s - No prediction available\n",
				event.Date.Format("2006-01-02"), event.EventType)
			continue
		}

		fmt.Printf("\n[%s] %s (Fallback: %s)\n",
			event.Date.Format("2006-01-02"), event.EventType, pred.FallbackLvl)
		fmt.Printf("  Expected Return 1D: %+.2f%%\n", pred.ExpRet1D*100)
		fmt.Printf("  Expected Return 5D: %+.2f%%\n", pred.ExpRet5D*100)
		fmt.Printf("  Confidence: %.0f%%\n", pred.Confidence*100)
		fmt.Printf("  P10 MDD Risk: %.2f%%\n", pred.P10MDD*100)
	}

	return nil
}

func initForecastDeps() (*config.Config, zerolog.Logger, *database.DB, error) {
	// 설정 로드
	cfg, err := config.Load()
	if err != nil {
		return nil, zerolog.Logger{}, nil, fmt.Errorf("load config: %w", err)
	}

	// 로거 초기화
	log := logger.New(cfg)

	// DB 연결
	db, err := database.New(cfg)
	if err != nil {
		return nil, zerolog.Logger{}, nil, fmt.Errorf("connect to database: %w", err)
	}

	return cfg, log.Zerolog(), db, nil
}
