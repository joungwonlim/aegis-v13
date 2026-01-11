package forecast

import (
	"context"
	"math"
	"time"

	"github.com/rs/zerolog"

	"github.com/wonny/aegis/v13/backend/internal/risk"
)

// =============================================================================
// Forecast Validator
// =============================================================================

// Validator Forecast 검증기
// ⭐ SSOT: 예측 vs 실제 검증 로직
type Validator struct {
	repo         *Repository
	predictor    *Predictor
	modelVersion string // 모델 버전 (다중 모델 비교용)
	log          zerolog.Logger
}

// NewValidator 새 검증기 생성
func NewValidator(repo *Repository, predictor *Predictor, modelVersion string, log zerolog.Logger) *Validator {
	return &Validator{
		repo:         repo,
		predictor:    predictor,
		modelVersion: modelVersion,
		log:          log.With().Str("component", "forecast.validator").Logger(),
	}
}

// =============================================================================
// Validation
// =============================================================================

// ValidateAll 모든 미검증 이벤트 검증
// 전방 성과가 있는 이벤트에 대해 예측과 실제를 비교
func (v *Validator) ValidateAll(ctx context.Context) ([]risk.ValidationResult, error) {
	// 전방 성과가 있는 이벤트 조회
	eventsWithPerf, err := v.repo.GetEventsWithPerformance(ctx)
	if err != nil {
		return nil, err
	}

	var results []risk.ValidationResult

	for _, ewp := range eventsWithPerf {
		// 예측 생성
		pred, err := v.predictor.Predict(ctx, ewp.Event)
		if err != nil {
			v.log.Warn().Err(err).
				Str("code", ewp.Event.Code).
				Msg("prediction failed")
			continue
		}
		if pred == nil {
			continue
		}

		// 실제 수익률
		actualRet := ewp.Performance.FwdRet5D

		// 오차 계산
		error5d := actualRet - pred.ExpRet5D
		absError := math.Abs(error5d)

		// 방향성 적중 (예측과 실제 부호 일치)
		directionHit := (pred.ExpRet5D >= 0 && actualRet >= 0) || (pred.ExpRet5D < 0 && actualRet < 0)

		result := risk.ValidationResult{
			EventID:      ewp.Event.ID,
			ModelVersion: v.modelVersion,
			Code:         ewp.Event.Code,
			EventType:    string(ewp.Event.EventType),
			PredictedRet: pred.ExpRet5D,
			ActualRet:    actualRet,
			Error:        error5d,
			AbsError:     absError,
			DirectionHit: directionHit,
			ValidatedAt:  time.Now(),
		}

		results = append(results, result)
	}

	v.log.Info().
		Int("total_events", len(eventsWithPerf)).
		Int("validated", len(results)).
		Str("model_version", v.modelVersion).
		Msg("validation completed")

	return results, nil
}

// ValidateByDateRange 날짜 범위 내 이벤트 검증
func (v *Validator) ValidateByDateRange(ctx context.Context, from, to time.Time) ([]risk.ValidationResult, error) {
	// 날짜 범위 내 이벤트 조회
	events, err := v.repo.GetEventsByDateRange(ctx, from, to)
	if err != nil {
		return nil, err
	}

	var results []risk.ValidationResult

	for _, event := range events {
		// 해당 이벤트의 전방 성과 확인 (5일 후 데이터 필요)
		eventWithPerf, err := v.repo.GetEventsByCodeWithPerformance(ctx, event.Code)
		if err != nil {
			continue
		}

		// 해당 날짜의 이벤트 찾기
		for _, ewp := range eventWithPerf {
			if ewp.TradeDate != event.Date.Format("2006-01-02") {
				continue
			}
			if ewp.FwdRet5D == nil {
				continue // 아직 전방 성과 없음
			}

			// 예측 생성
			pred, err := v.predictor.Predict(ctx, event)
			if err != nil || pred == nil {
				continue
			}

			actualRet := *ewp.FwdRet5D
			error5d := actualRet - pred.ExpRet5D

			result := risk.ValidationResult{
				EventID:      ewp.ID,
				ModelVersion: v.modelVersion,
				Code:         event.Code,
				EventType:    string(event.EventType),
				PredictedRet: pred.ExpRet5D,
				ActualRet:    actualRet,
				Error:        error5d,
				AbsError:     math.Abs(error5d),
				DirectionHit: (pred.ExpRet5D >= 0 && actualRet >= 0) || (pred.ExpRet5D < 0 && actualRet < 0),
				ValidatedAt:  time.Now(),
			}

			results = append(results, result)
			break
		}
	}

	return results, nil
}

// =============================================================================
// Accuracy Report
// =============================================================================

// CalculateAccuracy 정확도 리포트 생성
func (v *Validator) CalculateAccuracy(ctx context.Context, validations []risk.ValidationResult) *risk.AccuracyReport {
	if len(validations) == 0 {
		return nil
	}

	var sumAbsError, sumSqError float64
	var sumError float64
	var hitCount int

	for _, val := range validations {
		sumAbsError += val.AbsError
		sumSqError += val.Error * val.Error
		sumError += val.Error
		if val.DirectionHit {
			hitCount++
		}
	}

	n := float64(len(validations))

	return &risk.AccuracyReport{
		ModelVersion: v.modelVersion,
		Level:        "ALL",
		Key:          "ALL",
		EventType:    "ALL",
		SampleCount:  len(validations),
		MAE:          sumAbsError / n,
		RMSE:         math.Sqrt(sumSqError / n),
		HitRate:      float64(hitCount) / n,
		MeanError:    sumError / n, // 편향 (bias)
		UpdatedAt:    time.Now(),
	}
}

// CalculateAccuracyByLevel 레벨별 정확도 리포트 생성
func (v *Validator) CalculateAccuracyByLevel(ctx context.Context, validations []risk.ValidationResult, level string) map[string]*risk.AccuracyReport {
	// 레벨별 그룹핑
	groups := make(map[string][]risk.ValidationResult)

	for _, val := range validations {
		var key string
		switch level {
		case "EVENT_TYPE":
			key = val.EventType
		case "CODE":
			key = val.Code
		default:
			key = "ALL"
		}
		groups[key] = append(groups[key], val)
	}

	// 그룹별 정확도 계산
	reports := make(map[string]*risk.AccuracyReport)
	for key, group := range groups {
		report := v.CalculateAccuracy(ctx, group)
		if report != nil {
			report.Level = level
			report.Key = key
			reports[key] = report
		}
	}

	return reports
}

// =============================================================================
// Calibration
// =============================================================================

// CalculateCalibrationBins 캘리브레이션 빈 계산 (신뢰도 다이어그램용)
// validations: 검증 결과
// numBins: 빈 개수 (기본: 10)
func (v *Validator) CalculateCalibrationBins(ctx context.Context, validations []risk.ValidationResult, numBins int) []risk.CalibrationBin {
	if len(validations) == 0 || numBins <= 0 {
		return nil
	}

	// 빈별 데이터 수집
	bins := make([][]risk.ValidationResult, numBins)
	for i := range bins {
		bins[i] = make([]risk.ValidationResult, 0)
	}

	// confidence score 기준으로 빈 배정
	// 여기서는 예측값의 크기를 기준으로 함
	for _, val := range validations {
		// 예측값을 0-1 범위로 정규화 (대략적)
		// 실제로는 confidence score를 사용해야 함
		normalized := math.Abs(val.PredictedRet) * 10 // 대략적인 스케일링
		if normalized > 1 {
			normalized = 1
		}

		binIdx := int(normalized * float64(numBins-1))
		if binIdx >= numBins {
			binIdx = numBins - 1
		}
		bins[binIdx] = append(bins[binIdx], val)
	}

	// 빈별 통계 계산
	var result []risk.CalibrationBin
	for i, bin := range bins {
		if len(bin) == 0 {
			continue
		}

		var sumPred, sumActual float64
		var hitCount int
		for _, val := range bin {
			sumPred += val.PredictedRet
			sumActual += val.ActualRet
			if val.DirectionHit {
				hitCount++
			}
		}

		n := float64(len(bin))
		result = append(result, risk.CalibrationBin{
			ModelVersion: v.modelVersion,
			HorizonDays:  5,
			Bin:          i,
			SampleCount:  len(bin),
			AvgPredicted: sumPred / n,
			AvgActual:    sumActual / n,
			HitRate:      float64(hitCount) / n,
		})
	}

	return result
}

// Note: EventWithPerformance is defined in aggregator.go
