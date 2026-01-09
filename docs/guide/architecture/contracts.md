# Contracts

> 핵심 타입과 인터페이스

---

## 4대 핵심 Contract

레이어 간 데이터 전달에 사용되는 핵심 타입입니다.

```
DataQualitySnapshot → Universe → SignalSet → TargetPortfolio
```

---

## 1. DataQualitySnapshot

**용도**: S0 → S1 데이터 품질 정보 전달

```go
// internal/contracts/data.go

type DataQualitySnapshot struct {
    Date         time.Time         `json:"date"`
    TotalStocks  int               `json:"total_stocks"`
    ValidStocks  int               `json:"valid_stocks"`
    Coverage     map[string]float64 `json:"coverage"`  // 데이터별 커버리지
    QualityScore float64           `json:"quality_score"`
}
```

---

## 2. Universe

**용도**: S1 → S2 투자 가능 종목 전달

```go
// internal/contracts/universe.go

type Universe struct {
    Date       time.Time         `json:"date"`
    Stocks     []string          `json:"stocks"`
    Excluded   map[string]string `json:"excluded"`  // 제외 종목: 사유
    TotalCount int               `json:"total_count"`
}
```

---

## 3. SignalSet

**용도**: S2 → S3/S4 시그널 데이터 전달

```go
// internal/contracts/signals.go

type SignalSet struct {
    Date    time.Time              `json:"date"`
    Signals map[string]StockSignal `json:"signals"`
}

type StockSignal struct {
    Code      string             `json:"code"`
    Factors   map[string]float64 `json:"factors"`
    Events    []EventSignal      `json:"events"`
    UpdatedAt time.Time          `json:"updated_at"`
}

type EventSignal struct {
    Type      string    `json:"type"`
    Score     float64   `json:"score"`
    Source    string    `json:"source"`
    Timestamp time.Time `json:"timestamp"`
}
```

---

## 4. TargetPortfolio

**용도**: S5 → S6 목표 포트폴리오 전달

```go
// internal/contracts/portfolio.go

type TargetPortfolio struct {
    Date      time.Time        `json:"date"`
    Positions []TargetPosition `json:"positions"`
    Cash      float64          `json:"cash"`
}

type TargetPosition struct {
    Code      string  `json:"code"`
    Name      string  `json:"name"`
    Weight    float64 `json:"weight"`
    TargetQty int     `json:"target_qty"`
    Action    Action  `json:"action"`
    Reason    string  `json:"reason"`
}

type Action string

const (
    ActionBuy  Action = "BUY"
    ActionSell Action = "SELL"
    ActionHold Action = "HOLD"
)
```

---

## 레이어 인터페이스

각 레이어가 구현해야 할 인터페이스입니다.

```go
// internal/contracts/interfaces.go

// S0: 데이터 품질 검증
type QualityGate interface {
    Check(ctx context.Context, date time.Time) (*DataQualitySnapshot, error)
}

// S1: 유니버스 생성
type UniverseBuilder interface {
    Build(ctx context.Context, snapshot *DataQualitySnapshot) (*Universe, error)
}

// S2: 시그널 생성
type SignalBuilder interface {
    Build(ctx context.Context, universe *Universe) (*SignalSet, error)
}

// S3: 스크리닝
type Screener interface {
    Screen(ctx context.Context, signals *SignalSet) ([]string, error)
}

// S4: 랭킹
type Ranker interface {
    Rank(ctx context.Context, codes []string, signals *SignalSet) ([]RankedStock, error)
}

// S5: 포트폴리오 구성
type PortfolioConstructor interface {
    Construct(ctx context.Context, ranked []RankedStock) (*TargetPortfolio, error)
}

// S6: 주문 실행
type ExecutionPlanner interface {
    Plan(ctx context.Context, target *TargetPortfolio) ([]Order, error)
}

// S7: 성과 분석
type Auditor interface {
    Analyze(ctx context.Context, period string) (*PerformanceReport, error)
}
```

---

## Brain Orchestrator

Brain은 **로직 없이** 위 인터페이스를 순서대로 호출만 합니다.

```go
// internal/brain/orchestrator.go

type Orchestrator struct {
    quality   QualityGate
    universe  UniverseBuilder
    signals   SignalBuilder
    screener  Screener
    ranker    Ranker
    portfolio PortfolioConstructor
    executor  ExecutionPlanner
}

func (o *Orchestrator) Run(ctx context.Context) error {
    // S0
    snapshot, err := o.quality.Check(ctx, time.Now())
    if err != nil { return err }

    // S1
    universe, err := o.universe.Build(ctx, snapshot)
    if err != nil { return err }

    // S2
    signals, err := o.signals.Build(ctx, universe)
    if err != nil { return err }

    // S3
    passed, err := o.screener.Screen(ctx, signals)
    if err != nil { return err }

    // S4
    ranked, err := o.ranker.Rank(ctx, passed, signals)
    if err != nil { return err }

    // S5
    target, err := o.portfolio.Construct(ctx, ranked)
    if err != nil { return err }

    // S6
    _, err = o.executor.Plan(ctx, target)
    return err
}
```

---

**Prev**: [Data Flow](./data-flow.md)
**Next**: [Backend Folder Structure](../backend/folder-structure.md)
