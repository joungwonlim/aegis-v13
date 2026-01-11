package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/internal/forecast"
	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// ForecastHandler handles forecast API endpoints
// ⭐ SSOT: Forecast API 핸들러는 이 구조체에서만
type ForecastHandler struct {
	forecastRepo *forecast.Repository
	priceRepo    *s0_data.PriceRepository
	detector     *forecast.Detector
	predictor    *forecast.Predictor
	aggregator   *forecast.Aggregator
	logger       *logger.Logger
}

// NewForecastHandler creates a new forecast handler
func NewForecastHandler(
	forecastRepo *forecast.Repository,
	priceRepo *s0_data.PriceRepository,
	detector *forecast.Detector,
	predictor *forecast.Predictor,
	aggregator *forecast.Aggregator,
	log *logger.Logger,
) *ForecastHandler {
	return &ForecastHandler{
		forecastRepo: forecastRepo,
		priceRepo:    priceRepo,
		detector:     detector,
		predictor:    predictor,
		aggregator:   aggregator,
		logger:       log,
	}
}

// AnalyzeForecast analyzes forecast for a stock
// POST /api/forecast/analyze/:symbol
func (h *ForecastHandler) AnalyzeForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	if symbol == "" {
		respondError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	// 최근 30일 가격 데이터 조회
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	prices, err := h.priceRepo.GetByCodeAndDateRange(ctx, symbol, startDate, endDate)
	if err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"symbol": symbol,
		}).Error("Failed to get prices")
		respondError(w, http.StatusInternalServerError, "failed to get price data")
		return
	}

	if len(prices) == 0 {
		respondError(w, http.StatusNotFound, "no price data found")
		return
	}

	// 최근 거래일의 이벤트 감지
	latestPrice := prices[len(prices)-1]
	var prevClose float64
	if len(prices) >= 2 {
		prevClose = float64(prices[len(prices)-2].Close)
	}

	// TODO: 거래량 통계 계산 (20일 평균)
	// 현재는 nil로 전달
	priceData := forecast.PriceData{
		Code:      symbol,
		Date:      latestPrice.Date,
		Open:      float64(latestPrice.Open),
		High:      float64(latestPrice.High),
		Low:       float64(latestPrice.Low),
		Close:     float64(latestPrice.Close),
		Volume:    latestPrice.Volume,
		PrevClose: prevClose,
		// TODO: Sector, MarketCap 조회
	}

	event := h.detector.DetectEvent(ctx, priceData, nil)

	result := &contracts.ForecastResult{
		Symbol:         symbol,
		AnalyzedAt:     time.Now().Format(time.RFC3339),
		EventDetected:  event != nil,
		FallbackLevel:  string(contracts.StatsLevelMarket),
		SampleSize:     0,
		PredictionType: contracts.PredictionNone,
		Quality:        contracts.QualityUnknown,
		Warnings:       []string{},
	}

	// 이벤트가 감지되면 예측 생성
	if event != nil {
		result.EventDetected = true
		result.EventType = string(event.EventType)
		result.CurrentEvent = &contracts.EventCharacteristics{
			TradeDate:   event.Date.Format("2006-01-02"),
			Ret:         event.DayReturn,
			Gap:         event.GapRatio,
			CloseToHigh: event.CloseToHigh,
			VolZ:        event.VolumeZScore,
		}

		// 예측 생성
		prediction, err := h.predictor.Predict(ctx, *event)
		if err != nil {
			h.logger.WithError(err).WithFields(map[string]interface{}{
				"symbol": symbol,
			}).Error("Failed to predict")
			result.Warnings = append(result.Warnings, "예측 생성 실패")
		} else if prediction != nil {
			// 통계 조회하여 P90 runup 추가
			stats, err := h.forecastRepo.GetStats(ctx, contracts.StatsLevelSymbol, symbol, event.EventType)
			if err == nil && stats != nil {
				result.SampleSize = stats.SampleCount
				result.FallbackLevel = prediction.FallbackLvl

				// 신뢰도 계산
				if stats.SampleCount >= 30 {
					result.Quality = contracts.QualityHigh
				} else if stats.SampleCount >= 10 {
					result.Quality = contracts.QualityMedium
				} else {
					result.Quality = contracts.QualityLow
				}

				result.PredictionType = contracts.PredictionEventBased
				result.Prediction = &contracts.Prediction{
					ExpectedRet5D: prediction.ExpRet5D,
					WinRate5D:     stats.WinRate5D,
					P10MDD5D:      stats.P10MDD,
					P90Runup5D:    0, // TODO: P90 runup 계산
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, result)
}

// GetEvents returns historical events for a stock
// GET /api/forecast/events/:symbol?limit=10
func (h *ForecastHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	if symbol == "" {
		respondError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	// limit 파라미터 (기본값: 10)
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// 이벤트와 전방 성과 함께 조회
	events, err := h.forecastRepo.GetEventsByCodeWithPerformance(ctx, symbol)
	if err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"symbol": symbol,
		}).Error("Failed to get events")
		respondError(w, http.StatusInternalServerError, "failed to get events")
		return
	}

	// limit 적용 (최근 N개)
	// Repository에서 이미 ORDER BY event_date DESC로 정렬되어 최신이 앞에 있음
	if len(events) > limit {
		events = events[:limit]
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
	})
}
