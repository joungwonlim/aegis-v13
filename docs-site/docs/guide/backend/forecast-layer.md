---
sidebar_position: 9
title: Forecast Layer
description: 이벤트 기반 예측 시스템
---

# Forecast Layer

> 가격 패턴 기반 이벤트 감지 및 예측 시스템

---

## Overview

Forecast 모듈은 v10의 이벤트 기반 예측 시스템을 v13 아키텍처에 맞게 재구현한 것입니다.

### 핵심 기능

| 기능 | 설명 |
|------|------|
| **이벤트 감지** | E1(급등), E2(갭+급등) 패턴 감지 |
| **전방 성과 추적** | 이벤트 후 5거래일 수익률/MDD 추적 |
| **통계 집계** | 4단계 폴백 계층 통계 |
| **예측 생성** | 베이지안 수축 기반 예측 |

---

## 이벤트 타입

### E1_SURGE (급등)

```
조건: dayReturn >= 3.5% AND closeToHigh >= 0.4
```

- `dayReturn`: 당일 수익률 `(close - prev_close) / prev_close`
- `closeToHigh`: 고가 대비 종가 위치 `(close - low) / (high - low)`

### E2_GAP_SURGE (갭+급등)

```
조건: E1 조건 AND gapRatio >= 1.5%
```

- `gapRatio`: 갭 비율 `(open - prev_close) / prev_close`

---

## 파일 구조

```
internal/forecast/
├── detector.go      # 이벤트 감지
├── tracker.go       # 전방 성과 추적
├── aggregator.go    # 통계 집계
├── predictor.go     # 예측 생성
├── validator.go     # 예측 검증 (S7 Audit용)
└── repository.go    # DB 저장소
```

---

## 타입 정의

### ForecastEvent

```go
type ForecastEvent struct {
    ID              int64
    Code            string
    Date            time.Time
    EventType       ForecastEventType  // E1_SURGE, E2_GAP_SURGE
    DayReturn       float64            // 당일 수익률
    CloseToHigh     float64            // 고가 대비 종가 (0~1)
    GapRatio        float64            // 갭 비율
    VolumeZScore    float64            // 거래량 z-score
    Sector          string
    MarketCapBucket string             // small/mid/large
}
```

### ForwardPerformance

```go
type ForwardPerformance struct {
    EventID       int64
    FwdRet1D      float64  // t+1 수익률
    FwdRet2D      float64  // t+2 수익률
    FwdRet3D      float64  // t+3 수익률
    FwdRet5D      float64  // t+5 수익률
    MaxRunup5D    float64  // 5일 최대 상승
    MaxDrawdown5D float64  // 5일 최대 하락
    GapHold3D     bool     // 3일간 갭 유지
}
```

### ForecastStats

```go
type ForecastStats struct {
    Level       ForecastStatsLevel  // SYMBOL/SECTOR/BUCKET/MARKET
    Key         string              // 종목코드/섹터명/버킷명/ALL
    EventType   ForecastEventType
    SampleCount int
    AvgRet1D    float64
    AvgRet5D    float64
    WinRate1D   float64  // 1일 후 양수 비율
    WinRate5D   float64  // 5일 후 양수 비율
    P10MDD      float64  // 하위 10% MDD
}
```

---

## 4단계 폴백 계층

예측 시 샘플 수가 부족하면 다음 레벨로 폴백합니다.

```
1. SYMBOL  → 해당 종목의 과거 이벤트 통계
2. SECTOR  → 같은 섹터 종목들의 통계
3. BUCKET  → 같은 시가총액 구간 (small/mid/large)
4. MARKET  → 전체 시장 평균
```

### 폴백 조건

- 샘플 수 < 5 → 다음 레벨로 폴백
- MARKET 레벨은 항상 존재

---

## 베이지안 수축

소표본 편향을 보정하기 위해 베이지안 수축을 적용합니다.

```go
// K = 10 (수축 강도)
weight := n / (n + K)
shrunkReturn := weight * sampleMean + (1-weight) * marketMean
```

### 신뢰도 계산

```go
confidence := min(1.0, sampleCount / 30.0)
```

---

## Forecast Validation (S7 Audit)

예측 정확도를 검증하고 모델 품질을 측정합니다. 이 검증 시스템은 S7 Audit 레이어의 핵심 구성요소로, 예측 모델의 품질을 지속적으로 모니터링합니다.

### 검증 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│                    Forecast Validation Pipeline                  │
└─────────────────────────────────────────────────────────────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼                      ▼                      ▼
┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐
│ 1. Prediction     │  │ 2. Actual Result  │  │ 3. Comparison     │
│    Retrieval      │  │    Collection     │  │    & Scoring      │
├───────────────────┤  ├───────────────────┤  ├───────────────────┤
│ • 과거 예측값     │  │ • 5일 후 실제값   │  │ • 오차 계산       │
│ • 모델 버전별     │  │ • forward_perf    │  │ • 방향성 체크     │
│ • 이벤트 타입별   │  │   테이블 조회     │  │ • 메트릭 집계     │
└───────────────────┘  └───────────────────┘  └───────────────────┘
                               │
                               ▼
                ┌───────────────────────────────────┐
                │       AccuracyReport              │
                ├───────────────────────────────────┤
                │ • MAE (평균 절대 오차)            │
                │ • RMSE (평균 제곱근 오차)         │
                │ • Hit Rate (방향성 적중률)        │
                │ • Mean Error (편향)               │
                │ • 이벤트 타입별 분석              │
                └───────────────────────────────────┘
```

### Validator

```go
// internal/forecast/validator.go

type Validator struct {
    repo         *Repository
    predictor    *Predictor
    modelVersion string          // 다중 모델 비교용 (A/B 테스트)
    log          zerolog.Logger
}

// NewValidator Validator 생성
func NewValidator(repo *Repository, predictor *Predictor, modelVersion string, log zerolog.Logger) *Validator {
    return &Validator{
        repo:         repo,
        predictor:    predictor,
        modelVersion: modelVersion,
        log:          log.With().Str("component", "forecast_validator").Logger(),
    }
}

// ValidateAll 전체 검증 - 완료된 이벤트 모두 검증
func (v *Validator) ValidateAll(ctx context.Context) ([]risk.ValidationResult, error)

// ValidateRange 기간별 검증
func (v *Validator) ValidateRange(ctx context.Context, from, to time.Time) ([]risk.ValidationResult, error)

// CalculateAccuracy 정확도 리포트 생성
func (v *Validator) CalculateAccuracy(ctx context.Context, validations []risk.ValidationResult) *risk.AccuracyReport

// CalculateCalibrationBins 캘리브레이션 빈 계산
func (v *Validator) CalculateCalibrationBins(ctx context.Context, validations []risk.ValidationResult, numBins int) []risk.CalibrationBin
```

### 검증 메트릭

| 메트릭 | 설명 | 계산 방법 | 목표 |
|--------|------|-----------|------|
| **MAE** | Mean Absolute Error | `Σ|actual - predicted| / n` | < 2% |
| **RMSE** | Root Mean Squared Error | `√(Σ(actual - predicted)² / n)` | < 3% |
| **Hit Rate** | 방향성 적중률 | `(부호 일치 수) / n` | > 55% |
| **Mean Error** | 편향 (bias) | `Σ(actual - predicted) / n` | ~0% |

**메트릭 해석**:
- **MAE**: 예측 오차의 평균 크기. 낮을수록 좋음
- **RMSE**: 큰 오차에 더 큰 패널티. MAE보다 이상치에 민감
- **Hit Rate**: 상승/하락 방향만 맞췄는지. 55% 이상이면 유의미한 예측력
- **Mean Error**: 양수면 과소예측, 음수면 과대예측. 0에 가까워야 함

### ValidationResult

```go
// internal/risk/types.go

type ValidationResult struct {
    EventID      int64     `json:"event_id"`
    ModelVersion string    `json:"model_version"`   // A/B 테스트용
    Code         string    `json:"code"`
    EventType    string    `json:"event_type"`
    PredictedRet float64   `json:"predicted_ret"`   // 예측 수익률 (5일)
    ActualRet    float64   `json:"actual_ret"`      // 실제 수익률 (5일)
    Error        float64   `json:"error"`           // 오차 (actual - predicted)
    AbsError     float64   `json:"abs_error"`       // 절대 오차
    DirectionHit bool      `json:"direction_hit"`   // 방향성 적중 (부호 일치)
    ValidatedAt  time.Time `json:"validated_at"`
}
```

### AccuracyReport

```go
// internal/risk/types.go

type AccuracyReport struct {
    ModelVersion string    `json:"model_version"`
    Level        string    `json:"level"`           // ALL, EVENT_TYPE, CODE
    Key          string    `json:"key"`             // level에 따른 키
    EventType    string    `json:"event_type"`
    SampleCount  int       `json:"sample_count"`
    MAE          float64   `json:"mae"`             // Mean Absolute Error
    RMSE         float64   `json:"rmse"`            // Root Mean Squared Error
    HitRate      float64   `json:"hit_rate"`        // 방향성 적중률 (0~1)
    MeanError    float64   `json:"mean_error"`      // 편향 (bias)
    UpdatedAt    time.Time `json:"updated_at"`
}
```

### 캘리브레이션 (Reliability Diagram)

예측 신뢰도와 실제 적중률의 일치도를 측정합니다. 잘 캘리브레이션된 모델은 "80% 신뢰도 예측의 80%가 맞아야" 합니다.

```go
// CalibrationBin 캘리브레이션 빈
type CalibrationBin struct {
    Bin          int     `json:"bin"`           // 빈 번호 (0-9)
    SampleCount  int     `json:"sample_count"`  // 샘플 수
    AvgPredicted float64 `json:"avg_predicted"` // 빈 내 평균 예측값
    AvgActual    float64 `json:"avg_actual"`    // 빈 내 평균 실제값
    HitRate      float64 `json:"hit_rate"`      // 빈 내 적중률
}

// 캘리브레이션 분석 예시
bins := validator.CalculateCalibrationBins(ctx, validations, 10)

// 이상적인 결과 (잘 캘리브레이션됨):
// Bin 0 (0-10% 신뢰도): HitRate ~10%
// Bin 5 (50-60% 신뢰도): HitRate ~55%
// Bin 9 (90-100% 신뢰도): HitRate ~95%
```

### 모델 버전 관리 (A/B 테스트)

동일 이벤트에 대해 여러 모델 버전의 예측을 비교할 수 있습니다.

```go
// PK: (event_id, model_version) - 동일 이벤트, 다른 모델 비교 가능

// 예: v1.0.0 vs v2.0.0 비교
v1Validator := forecast.NewValidator(repo, predictor, "v1.0.0", log)
v2Validator := forecast.NewValidator(repo, predictor, "v2.0.0", log)

v1Results, _ := v1Validator.ValidateAll(ctx)
v2Results, _ := v2Validator.ValidateAll(ctx)

v1Report := v1Validator.CalculateAccuracy(ctx, v1Results)
v2Report := v2Validator.CalculateAccuracy(ctx, v2Results)

// v2가 더 나은지 비교
if v2Report.HitRate > v1Report.HitRate {
    // v2 모델 채택
}
```

### 검증 결과 예시

```
══════════════════════════════════════════════════════════════════
              Forecast Validation Report (v1.0.0)
══════════════════════════════════════════════════════════════════

📊 Summary
  Events Validated: 146,792
  Model Version: v1.0.0

📈 Accuracy Metrics
  MAE (Mean Absolute Error): 3.42%
  RMSE (Root Mean Square Error): 5.18%
  Hit Rate (Direction Accuracy): 55.46%
  Mean Error (Bias): +0.12%

📋 By Event Type
  E1_SURGE:
    Count: 98,234  |  MAE: 3.28%  |  Hit Rate: 56.12%
  E2_GAP_SURGE:
    Count: 48,558  |  MAE: 3.71%  |  Hit Rate: 54.13%

✅ Model Quality: ACCEPTABLE (Hit Rate > 55%)
══════════════════════════════════════════════════════════════════
```

---

## CLI 명령어

### 전체 파이프라인

```bash
go run ./cmd/quant forecast run --from 2024-01-01
```

### 개별 단계

```bash
# 1. 이벤트 감지
go run ./cmd/quant forecast detect --from 2024-01-01 --to 2024-12-31

# 2. 전방 성과 채우기
go run ./cmd/quant forecast fill-forward

# 3. 통계 집계
go run ./cmd/quant forecast aggregate
```

### 예측 조회

```bash
go run ./cmd/quant forecast predict --code 005930
```

### 예측 검증 (S7 Audit)

```bash
# 전체 기간 검증
go run ./cmd/quant forecast validate

# 날짜 범위 지정
go run ./cmd/quant forecast validate --from 2024-01-01 --to 2024-06-30

# 모델 버전 지정 (A/B 테스트)
go run ./cmd/quant forecast validate --model v2.0.0

# 집계 레벨별 리포트
go run ./cmd/quant forecast validate --level EVENT_TYPE

# JSON 출력
go run ./cmd/quant forecast validate --output json
```

---

## 스케줄러 등록

Forecast 파이프라인은 스케줄러에 `forecast_pipeline` 작업으로 등록되어 있습니다.

### 스케줄

| 작업명 | 실행 시간 | 설명 |
|--------|----------|------|
| `forecast_pipeline` | **매일 18:30** | Universe 생성 후 실행 |

### 실행 순서

```
16:00 - data_collection (데이터 수집)
17:00 - investor_flow (투자자 수급)
18:00 - universe_generation (Universe 생성)
18:30 - forecast_pipeline (이벤트 감지/예측) ⭐
```

### 스케줄러 명령어

```bash
# 스케줄러 시작 (모든 작업 등록)
go run ./cmd/quant scheduler start

# 등록된 작업 목록 확인
go run ./cmd/quant scheduler list

# forecast_pipeline 즉시 실행
go run ./cmd/quant scheduler run forecast_pipeline

# 작업 상태 확인
go run ./cmd/quant scheduler status
```

### Job 구현

```go
// internal/scheduler/jobs/forecast.go
type ForecastJob struct {
    pool   *pgxpool.Pool
    logger *logger.Logger
}

func (j *ForecastJob) Name() string {
    return "forecast_pipeline"
}

func (j *ForecastJob) Schedule() string {
    return "0 30 18 * * *"  // 매일 18:30
}

func (j *ForecastJob) Run(ctx context.Context) error {
    // 1. Event Detection
    // 2. Fill Forward Performance
    // 3. Aggregate Statistics
}
```

---

## DB 스키마

### analytics.forecast_events

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `id` | BIGSERIAL | PK |
| `code` | VARCHAR(20) | 종목코드 |
| `event_date` | DATE | 이벤트 발생일 |
| `event_type` | VARCHAR(20) | E1_SURGE, E2_GAP_SURGE |
| `day_return` | NUMERIC(8,4) | 당일 수익률 |
| `close_to_high` | NUMERIC(8,4) | 고가 대비 종가 |
| `gap_ratio` | NUMERIC(8,4) | 갭 비율 |
| `volume_z_score` | NUMERIC(8,2) | 거래량 z-score |
| `sector` | VARCHAR(50) | 섹터 |
| `market_cap_bucket` | VARCHAR(10) | small/mid/large |

### analytics.forward_performance

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `event_id` | BIGINT | FK → forecast_events |
| `fwd_ret_1d` ~ `fwd_ret_5d` | NUMERIC(8,4) | 전방 수익률 |
| `max_runup_5d` | NUMERIC(8,4) | 5일 최대 상승 |
| `max_drawdown_5d` | NUMERIC(8,4) | 5일 최대 하락 |
| `gap_hold_3d` | BOOLEAN | 갭 유지 여부 |

### analytics.forecast_stats

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `level` | VARCHAR(10) | SYMBOL/SECTOR/BUCKET/MARKET |
| `key` | VARCHAR(50) | 레벨별 키 |
| `event_type` | VARCHAR(20) | 이벤트 타입 |
| `sample_count` | INT | 샘플 수 |
| `avg_ret_*` | NUMERIC(8,4) | 평균 수익률 |
| `win_rate_*` | NUMERIC(5,4) | 승률 |
| `p10_mdd` | NUMERIC(8,4) | 하위 10% MDD |

### analytics.forecast_validations

예측 검증 결과 (모델 버전별)

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `event_id` | BIGINT | FK → forecast_events (PK) |
| `model_version` | VARCHAR(20) | 모델 버전 (PK) |
| `code` | VARCHAR(20) | 종목코드 |
| `event_type` | VARCHAR(20) | 이벤트 타입 |
| `predicted_ret` | NUMERIC(8,4) | 예측 수익률 |
| `actual_ret` | NUMERIC(8,4) | 실제 수익률 |
| `error` | NUMERIC(8,4) | 오차 |
| `abs_error` | NUMERIC(8,4) | 절대 오차 |
| `direction_hit` | BOOLEAN | 방향성 적중 |

### analytics.accuracy_reports

집계 수준별 정확도 리포트

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `model_version` | VARCHAR(20) | 모델 버전 (PK) |
| `level` | VARCHAR(20) | ALL/EVENT_TYPE/CODE (PK) |
| `key` | VARCHAR(50) | 레벨별 키 (PK) |
| `event_type` | VARCHAR(20) | 이벤트 타입 (PK) |
| `mae` | NUMERIC(8,4) | Mean Absolute Error |
| `rmse` | NUMERIC(8,4) | Root Mean Squared Error |
| `hit_rate` | NUMERIC(5,4) | 방향성 적중률 |
| `mean_error` | NUMERIC(8,4) | 편향 (bias) |

### analytics.calibration_bins

신뢰도 캘리브레이션 빈 (reliability diagram 용)

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `model_version` | VARCHAR(20) | 모델 버전 (PK) |
| `horizon_days` | INT | 예측 기간 5/10/20일 (PK) |
| `bin` | INT | 빈 번호 0-9 (PK) |
| `sample_count` | INT | 샘플 수 |
| `avg_predicted` | NUMERIC(8,4) | 빈 내 평균 예측값 |
| `avg_actual` | NUMERIC(8,4) | 빈 내 평균 실제값 |
| `hit_rate` | NUMERIC(5,4) | 빈 내 적중률 |

---

## 사용 예시

### 이벤트 감지

```go
detector := forecast.NewDetector(log)
events := detector.DetectEvents(ctx, priceDataList, volumeStatsMap)
```

### 전방 성과 계산

```go
tracker := forecast.NewTracker(log)
perf := tracker.CalculateForwardPerformance(ctx, eventID, baseClose, forwardPrices)
```

### 통계 집계

```go
aggregator := forecast.NewAggregator(log)
stats := aggregator.AggregateAll(ctx, eventsWithPerformance)
```

### 예측 생성

```go
predictor := forecast.NewPredictor(repository, log)
prediction, _ := predictor.Predict(ctx, event)
```

---

## S3/S4 통합 (선택)

### S3 Screener 필터

```go
// P10 MDD가 -10% 이내인 종목만 통과
MinP10MDD: -0.10
```

### S4 Ranker 점수

```go
// 기대 수익률 기반 가산점
EventForecastWeight: 0.10  // 10% 가중치
```

---

**Prev**: [Audit Layer](./audit-layer.md)
