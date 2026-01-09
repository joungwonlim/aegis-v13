# Contracts

> 7단계 파이프라인 타입 및 인터페이스 SSOT

---

## 개요

`internal/contracts`는 Aegis v13의 7단계 퀀트 파이프라인에서 사용되는 **모든 타입과 인터페이스의 단일 진실 원천(SSOT)**입니다.

---

## 4대 핵심 Contract

레이어 간 데이터 전달에 사용되는 핵심 타입:

```
DataQualitySnapshot → Universe → SignalSet → TargetPortfolio
```

| Contract | 파일 | 용도 |
|----------|------|------|
| `DataQualitySnapshot` | data.go | S0 → S1 데이터 품질 정보 |
| `Universe` | universe.go | S1 → S2 투자 가능 종목 |
| `SignalSet` | signals.go | S2 → S3/S4 시그널 데이터 |
| `RankedStock` | ranked.go | S4 → S5 랭킹 결과 |
| `TargetPortfolio` | portfolio.go | S5 → S6 목표 포트폴리오 |
| `Order` | order.go | S6 → Broker 주문 정보 |
| `PerformanceReport` | audit.go | S7 성과 분석 결과 |

---

## 7단계 인터페이스

각 레이어가 구현해야 할 인터페이스 (interfaces.go):

| 인터페이스 | 단계 | 책임 |
|-----------|------|------|
| `QualityGate` | S0 | 데이터 품질 검증 |
| `UniverseBuilder` | S1 | 투자 가능 유니버스 생성 |
| `SignalBuilder` | S2 | 시그널 생성 |
| `Screener` | S3 | 1차 필터링 |
| `Ranker` | S4 | 종합 랭킹 |
| `PortfolioConstructor` | S5 | 포트폴리오 구성 |
| `ExecutionPlanner` | S6 | 주문 실행 계획 |
| `Auditor` | S7 | 성과 분석 |

---

## 사용 예시

### 1. DataQualitySnapshot (S0 → S1)

```go
import "github.com/wonny/aegis/v13/backend/internal/contracts"

snapshot := &contracts.DataQualitySnapshot{
    Date:         time.Now(),
    TotalStocks:  2500,
    ValidStocks:  2300,
    QualityScore: 0.92,
    Coverage: map[string]float64{
        "price":  0.98,
        "volume": 0.95,
        "market": 0.90,
    },
}

if snapshot.IsValid() {
    fmt.Println("Data quality check passed")
}
```

### 2. Universe (S1 → S2)

```go
universe := &contracts.Universe{
    Date:   time.Now(),
    Stocks: []string{"005930", "000660", "035420"},
    Excluded: map[string]string{
        "999999": "No trading volume",
    },
}

if universe.Contains("005930") {
    fmt.Println("Samsung is in universe")
}
```

### 3. SignalSet (S2 → S3/S4)

```go
signalSet := &contracts.SignalSet{
    Date: time.Now(),
    Signals: map[string]*contracts.StockSignals{
        "005930": {
            Code:      "005930",
            Momentum:  0.8,
            Technical: 0.6,
            Value:     0.5,
            Quality:   0.7,
            Flow:      0.4,
            Event:     0.3,
        },
    },
}

if signals, exists := signalSet.Get("005930"); exists {
    fmt.Printf("Total Score: %.2f\n", signals.TotalScore())
    fmt.Printf("Is Positive: %v\n", signals.IsPositive())
}
```

### 4. TargetPortfolio (S5 → S6)

```go
portfolio := &contracts.TargetPortfolio{
    Date: time.Now(),
    Positions: []contracts.TargetPosition{
        {
            Code:      "005930",
            Name:      "Samsung",
            Weight:    0.30,
            TargetQty: 100,
            Action:    contracts.ActionBuy,
            Reason:    "Strong momentum + quality",
        },
    },
    Cash: 0.25,
}

fmt.Printf("Total Weight: %.2f\n", portfolio.TotalWeight())
fmt.Printf("Position Count: %d\n", portfolio.Count())
```

---

## 인터페이스 구현 예시

### QualityGate (S0)

```go
type MyQualityGate struct {
    db     *pgxpool.Pool
    logger *logger.Logger
}

func (q *MyQualityGate) Check(ctx context.Context, date time.Time) (*contracts.DataQualitySnapshot, error) {
    // 데이터 품질 검증 로직
    return &contracts.DataQualitySnapshot{
        Date:         date,
        TotalStocks:  2500,
        ValidStocks:  2300,
        QualityScore: 0.92,
    }, nil
}
```

---

## 테스트

```bash
# 전체 테스트 실행
go test ./internal/contracts/... -v

# 특정 테스트 실행
go test ./internal/contracts/... -v -run TestStockSignals_TotalScore

# 커버리지 확인
go test ./internal/contracts/... -cover
```

---

## SSOT 규칙

### ✅ 허용

```go
// contracts 타입 사용
import "github.com/wonny/aegis/v13/backend/internal/contracts"

func Process(signals *contracts.SignalSet) error {
    // OK
}
```

### ❌ 금지

```go
// 레이어에서 중복 타입 정의
type SignalSet struct {  // 금지!
    Date    time.Time
    Signals map[string]*StockSignals
}
```

**모든 타입과 인터페이스는 `internal/contracts`에서만 정의합니다.**

---

## 참고 문서

- [Contracts 아키텍처](../../../docs-site/docs/guide/architecture/contracts.md)
- [Data Flow](../../../docs-site/docs/guide/architecture/data-flow.md)
- [Backend Folder Structure](../../../docs-site/docs/guide/backend/folder-structure.md)
