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
