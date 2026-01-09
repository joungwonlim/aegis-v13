# Signals Layer

> S2: 시그널 생성

---

## 책임

Universe에 포함된 종목들의 **팩터/이벤트 시그널**을 계산

---

## 폴더 구조

```
internal/signals/
├── builder.go      # SignalBuilder 구현 (조합)
├── momentum.go     # 모멘텀 시그널
├── value.go        # 가치 시그널
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
// internal/signals/builder.go

type signalBuilder struct {
    momentum MomentumCalculator
    value    ValueCalculator
    event    EventCalculator
    db       *pgxpool.Pool
}

func (b *signalBuilder) Build(ctx context.Context, universe *contracts.Universe) (*contracts.SignalSet, error) {
    signalSet := &contracts.SignalSet{
        Date:    universe.Date,
        Signals: make(map[string]contracts.StockSignal),
    }

    for _, code := range universe.Stocks {
        signal := contracts.StockSignal{
            Code:    code,
            Factors: make(map[string]float64),
        }

        // 각 시그널 계산
        signal.Factors["momentum"] = b.momentum.Calculate(ctx, code)
        signal.Factors["value"] = b.value.Calculate(ctx, code)
        signal.Factors["quality"] = b.calculateQuality(ctx, code)

        // 이벤트 시그널
        signal.Events = b.event.GetEvents(ctx, code)
        signal.Factors["event"] = b.event.CalculateScore(signal.Events)

        signal.UpdatedAt = time.Now()
        signalSet.Signals[code] = signal
    }

    return signalSet, nil
}
```

---

## Momentum Signal

```go
// internal/signals/momentum.go

type MomentumCalculator interface {
    Calculate(ctx context.Context, code string) float64
}

type momentumCalculator struct {
    db *pgxpool.Pool
}

func (c *momentumCalculator) Calculate(ctx context.Context, code string) float64 {
    // 1. 수익률 계산
    ret1m := c.getReturn(ctx, code, 20)   // 1개월
    ret3m := c.getReturn(ctx, code, 60)   // 3개월
    ret6m := c.getReturn(ctx, code, 120)  // 6개월

    // 2. 가중 평균 (최근 비중 높게)
    score := ret1m*0.5 + ret3m*0.3 + ret6m*0.2

    // 3. Z-score 정규화 (-3 ~ +3 범위)
    return c.normalize(score)
}
```

---

## Value Signal

```go
// internal/signals/value.go

type ValueCalculator interface {
    Calculate(ctx context.Context, code string) float64
}

func (c *valueCalculator) Calculate(ctx context.Context, code string) float64 {
    fundamental := c.getFundamental(ctx, code)

    // PER 점수 (낮을수록 좋음, 역수)
    perScore := c.invertAndNormalize(fundamental.PER)

    // PBR 점수 (낮을수록 좋음, 역수)
    pbrScore := c.invertAndNormalize(fundamental.PBR)

    // PSR 점수
    psrScore := c.invertAndNormalize(fundamental.PSR)

    // 가중 평균
    return perScore*0.4 + pbrScore*0.4 + psrScore*0.2
}
```

---

## Event Signal

```go
// internal/signals/event.go

type EventCalculator interface {
    GetEvents(ctx context.Context, code string) []contracts.EventSignal
    CalculateScore(events []contracts.EventSignal) float64
}

func (c *eventCalculator) GetEvents(ctx context.Context, code string) []contracts.EventSignal {
    events := make([]contracts.EventSignal, 0)

    // 1. 공시 이벤트
    disclosures := c.getDisclosures(ctx, code, 30) // 최근 30일
    for _, d := range disclosures {
        events = append(events, contracts.EventSignal{
            Type:      d.Type,
            Score:     c.scoreDisclosure(d),
            Source:    "DART",
            Timestamp: d.Date,
        })
    }

    // 2. 뉴스 이벤트
    news := c.getNews(ctx, code, 7) // 최근 7일
    for _, n := range news {
        events = append(events, contracts.EventSignal{
            Type:      "NEWS",
            Score:     n.Sentiment,
            Source:    n.Source,
            Timestamp: n.Date,
        })
    }

    return events
}

func (c *eventCalculator) CalculateScore(events []contracts.EventSignal) float64 {
    if len(events) == 0 {
        return 0
    }

    var totalScore float64
    var totalWeight float64

    for _, e := range events {
        // 시간 감쇠 (최근 이벤트 가중치 높음)
        daysSince := time.Since(e.Timestamp).Hours() / 24
        weight := math.Exp(-daysSince / 7) // 7일 반감기

        totalScore += e.Score * weight
        totalWeight += weight
    }

    return totalScore / totalWeight
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

## DB 스키마

```sql
-- signals.factor_scores: 팩터 점수
CREATE TABLE signals.factor_scores (
    id          SERIAL PRIMARY KEY,
    date        DATE NOT NULL,
    code        VARCHAR(10) NOT NULL,
    momentum    DECIMAL(8,4),
    value       DECIMAL(8,4),
    quality     DECIMAL(8,4),
    event       DECIMAL(8,4),
    technical   DECIMAL(8,4),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, code)
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

CREATE INDEX idx_events_code_date ON signals.events(code, event_date);
```

---

**Prev**: [Data Layer](./data-layer.md)
**Next**: [Selection Layer](./selection-layer.md)
