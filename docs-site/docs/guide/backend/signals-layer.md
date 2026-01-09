---
sidebar_position: 3
title: Signals Layer
description: S2 시그널 생성
---

# Signals Layer

> S2: 시그널 생성

---

## 책임

Universe에 포함된 종목들의 **팩터/이벤트 시그널**을 계산

---

## 폴더 구조

```
internal/s2_signals/
├── builder.go      # SignalBuilder 구현 (조합)
├── momentum.go     # 모멘텀 시그널
├── technical.go    # 기술적 시그널 (RSI, MACD)
├── value.go        # 가치 시그널
├── quality.go      # 퀄리티 시그널
├── flow.go         # 수급 시그널 ⭐
├── event.go        # 이벤트 시그널
└── repository.go   # DB 접근
```

---

## Signal Builder

### 인터페이스

```go
type SignalBuilder interface {
    Build(ctx context.Context, universe *Universe) (*SignalSet, error)
}
```

### 구현

```go
// internal/s2_signals/builder.go

type Builder struct {
    momentum       *MomentumCalculator
    technical      *TechnicalCalculator
    value          *ValueCalculator
    quality        *QualityCalculator
    flow           *FlowCalculator      // 수급 ⭐
    event          *EventCalculator
    priceRepo      contracts.PriceRepository
    flowRepo       contracts.InvestorFlowRepository
    financialRepo  contracts.FinancialRepository
    disclosureRepo contracts.DisclosureRepository
    logger         *logger.Logger
}

func (b *Builder) Build(ctx context.Context, universe *contracts.Universe, date time.Time) (*contracts.SignalSet, error) {
    signalSet := &contracts.SignalSet{
        Date:    date,
        Signals: make(map[string]*contracts.StockSignals),
    }

    for _, code := range universe.Stocks {
        signals, err := b.calculateStockSignals(ctx, code, date)
        if err != nil {
            b.logger.Warn("Failed to calculate signals", "code", code)
            continue
        }

        signalSet.Signals[code] = signals
    }

    return signalSet, nil
}

// 모든 시그널은 -1.0 ~ 1.0 범위로 정규화됨
```

---

## Momentum Signal

```go
// internal/s2_signals/momentum.go

type MomentumCalculator struct {
    logger *logger.Logger
}

func (c *MomentumCalculator) Calculate(ctx context.Context, code string, prices []PricePoint) (float64, contracts.SignalDetails, error) {
    // 1. 수익률 계산
    return1M := c.calculateReturn(prices, 20)   // 1개월 (20 거래일)
    return3M := c.calculateReturn(prices, 60)   // 3개월 (60 거래일)
    volumeRate := c.calculateVolumeGrowth(prices, 20)

    // 2. 가중 평균
    // Return1M: 40%, Return3M: 40%, VolumeRate: 20%
    score := return1M*0.4 + return3M*0.4 + volumeRate*0.2

    // 3. tanh 정규화 (-1.0 ~ 1.0 범위)
    normalizedScore := math.Tanh(score * 2)

    details := contracts.SignalDetails{
        Return1M:   return1M,
        Return3M:   return3M,
        VolumeRate: volumeRate,
    }

    return normalizedScore, details, nil
}
```

---

## Value Signal

```go
// internal/s2_signals/value.go

type ValueCalculator struct {
    logger *logger.Logger
}

func (c *ValueCalculator) Calculate(ctx context.Context, code string, metrics ValueMetrics) (float64, contracts.SignalDetails, error) {
    // PER 점수 (낮을수록 좋음)
    // 15 = 중립(0), 5 = 저평가(1.0), 25 = 고평가(-1.0)
    perScore := (15 - metrics.PER) / 15

    // PBR 점수 (낮을수록 좋음)
    // 1.5 = 중립(0), 0.5 = 저평가(1.0), 2.5 = 고평가(-1.0)
    pbrScore := (1.5 - metrics.PBR) / 1.5

    // PSR 점수
    // 2.0 = 중립(0), 0.5 = 저평가(1.0), 4.0 = 고평가(-1.0)
    psrScore := (2.0 - metrics.PSR) / 2.0

    // 가중 평균: PER 50%, PBR 30%, PSR 20%
    score := perScore*0.5 + pbrScore*0.3 + psrScore*0.2

    // tanh로 부드럽게 변환
    score = math.Tanh(score * 1.5)

    details := contracts.SignalDetails{
        PER: metrics.PER,
        PBR: metrics.PBR,
        PSR: metrics.PSR,
    }

    return score, details, nil
}
```

---

## Flow Signal (수급)

```go
// internal/s2_signals/flow.go

type FlowCalculator struct {
    logger *logger.Logger
}

type FlowData struct {
    Date          string
    ForeignNet    int64  // 외국인 순매수
    InstNet       int64  // 기관 순매수
    IndividualNet int64  // 개인 순매수
}

func (c *FlowCalculator) Calculate(ctx context.Context, code string, flowData []FlowData) (float64, contracts.SignalDetails, error) {
    if len(flowData) < 20 {
        return 0.0, contracts.SignalDetails{}, nil
    }

    // 1. 5일/20일 누적 순매수 계산
    foreignNet5D := c.sumNetBuying(flowData[:5], "foreign")
    foreignNet20D := c.sumNetBuying(flowData[:20], "foreign")
    instNet5D := c.sumNetBuying(flowData[:5], "inst")
    instNet20D := c.sumNetBuying(flowData[:20], "inst")

    // 2. 정규화 (tanh 사용)
    // 10억 = 1.0, 50억(20일) = 1.0
    foreignScore5D := math.Tanh(float64(foreignNet5D) / 10_000_000_000)
    foreignScore20D := math.Tanh(float64(foreignNet20D) / 50_000_000_000)
    instScore5D := math.Tanh(float64(instNet5D) / 10_000_000_000)
    instScore20D := math.Tanh(float64(instNet20D) / 50_000_000_000)

    // 3. 가중 평균
    // 외국인: 60%, 기관: 40%
    // 단기(5D): 70%, 장기(20D): 30%
    foreignScore := foreignScore5D*0.7 + foreignScore20D*0.3
    instScore := instScore5D*0.7 + instScore20D*0.3
    score := foreignScore*0.6 + instScore*0.4

    details := contracts.SignalDetails{
        ForeignNet5D:  foreignNet5D,
        ForeignNet20D: foreignNet20D,
        InstNet5D:     instNet5D,
        InstNet20D:    instNet20D,
    }

    return score, details, nil
}
```

### 수급 점수 해석

| 점수 범위 | 해석 |
|-----------|------|
| 0.5 ~ 1.0 | 강한 매수세 (외국인+기관 동시 순매수) |
| 0.0 ~ 0.5 | 약한 매수세 |
| -0.5 ~ 0.0 | 약한 매도세 |
| -1.0 ~ -0.5 | 강한 매도세 (외국인+기관 동시 순매도) |

---

## Event Signal

```go
// internal/s2_signals/event.go

type EventCalculator struct {
    logger *logger.Logger
}

func (c *EventCalculator) Calculate(ctx context.Context, code string, events []contracts.EventSignal, currentDate time.Time) (float64, contracts.SignalDetails, error) {
    if len(events) == 0 {
        return 0.0, contracts.SignalDetails{}, nil
    }

    score := c.calculateScore(events, currentDate)
    return score, contracts.SignalDetails{}, nil
}

func (c *EventCalculator) calculateScore(events []contracts.EventSignal, currentDate time.Time) float64 {
    var weightedSum float64
    var totalWeight float64

    for _, event := range events {
        // 이벤트 점수 (이미 -1.0 ~ 1.0)
        score := event.Score

        // 시간 감쇠 적용
        daysSince := currentDate.Sub(event.Timestamp).Hours() / 24
        timeWeight := c.calculateTimeWeight(daysSince)

        weightedSum += score * timeWeight
        totalWeight += timeWeight
    }

    if totalWeight == 0 {
        return 0.0
    }

    finalScore := weightedSum / totalWeight

    // Clamp to -1.0 ~ 1.0
    if finalScore > 1.0 {
        finalScore = 1.0
    } else if finalScore < -1.0 {
        finalScore = -1.0
    }

    return finalScore
}

// calculateTimeWeight: 지수 감쇠
// 7일: ~100%, 30일: ~50%, 90일: ~25%
func (c *EventCalculator) calculateTimeWeight(daysSince float64) float64 {
    const decayRate = 0.023
    weight := math.Exp(-decayRate * daysSince)

    if weight < 0.1 {
        weight = 0.1  // 최소 가중치
    }

    return weight
}
```

---

## 이벤트 타입별 점수

| 이벤트 | 기본 점수 | 설명 |
|--------|----------|------|
| 자사주매입 | +2.0 | 강한 긍정 |
| 배당 증가 | +1.5 | 긍정 |
| 실적 서프라이즈 | +1.0 ~ +2.0 | 정도에 따라 |
| 대규모 계약 | +1.0 | 긍정 |
| 유상증자 | -1.0 | 부정 |
| CB/BW 발행 | -0.5 | 약한 부정 |
| 횡령/배임 | -2.0 | 강한 부정 |

---

## Signal Repository

```go
// internal/s2_signals/repository.go

type SignalRepository struct {
    pool *pgxpool.Pool
}

func (r *SignalRepository) Save(ctx context.Context, signalSet *contracts.SignalSet) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    for code, signals := range signalSet.Signals {
        // 1. factor_scores 저장
        if err := r.saveFactorScores(ctx, tx, code, signalSet.Date, signals); err != nil {
            return err
        }

        // 2. flow_details 저장
        // 3. technical_details 저장
        if err := r.saveSignalDetails(ctx, tx, code, signalSet.Date, signals); err != nil {
            return err
        }
    }

    return tx.Commit(ctx)
}

func (r *SignalRepository) GetByDate(ctx context.Context, date time.Time) (*contracts.SignalSet, error) {
    // factor_scores에서 모든 종목의 시그널 조회
    // flow_details, technical_details 조인하여 상세 정보 로드
}
```

**Total Score 계산 (가중 평균):**
- Flow: 25%
- Momentum: 20%
- Technical: 20%
- Value: 15%
- Quality: 15%
- Event: 5%

---

## DB 스키마

```sql
-- signals.factor_scores: 팩터 점수
CREATE TABLE signals.factor_scores (
    id          SERIAL PRIMARY KEY,
    date        DATE NOT NULL,
    code        VARCHAR(10) NOT NULL,
    momentum    DECIMAL(8,4),
    technical   DECIMAL(8,4),
    value       DECIMAL(8,4),
    quality     DECIMAL(8,4),
    flow        DECIMAL(8,4),       -- 수급 시그널 ⭐
    event       DECIMAL(8,4),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, code)
);

-- signals.flow_details: 수급 상세 (5D/20D 누적) ⭐
CREATE TABLE signals.flow_details (
    id              SERIAL PRIMARY KEY,
    date            DATE NOT NULL,
    stock_code      VARCHAR(10) NOT NULL,
    foreign_net_5d  BIGINT,         -- 외국인 5일 순매수
    foreign_net_20d BIGINT,         -- 외국인 20일 순매수
    inst_net_5d     BIGINT,         -- 기관 5일 순매수
    inst_net_20d    BIGINT,         -- 기관 20일 순매수
    flow_score      DECIMAL(8,4),   -- 종합 수급 점수
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, stock_code)
);

-- signals.events: 이벤트 로그
CREATE TABLE signals.events (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(10) NOT NULL,
    event_type  VARCHAR(50) NOT NULL,
    score       DECIMAL(5,2),
    source      VARCHAR(20),
    event_date  DATE NOT NULL,
    raw_data    JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_factor_scores_date ON signals.factor_scores(date, code);
CREATE INDEX idx_flow_details_date ON signals.flow_details(date, stock_code);
CREATE INDEX idx_events_code_date ON signals.events(code, event_date);
```

---

**Prev**: [Data Layer](./data-layer.md)
**Next**: [Selection Layer](./selection-layer.md)
