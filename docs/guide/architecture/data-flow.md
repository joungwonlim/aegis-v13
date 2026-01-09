# Data Flow

> 7단계 파이프라인 상세

---

## Pipeline Overview

```
외부 데이터 → [S0] → [S1] → [S2] → [S3] → [S4] → [S5] → [S6] → 체결
                                                          ↓
                                                        [S7] → 피드백
```

---

## Stage 0: Data Quality (데이터 품질)

**입력**: 외부 API (KIS, DART, Naver)
**출력**: `DataQualitySnapshot`

```go
type DataQualitySnapshot struct {
    Date         time.Time
    TotalStocks  int
    ValidStocks  int
    MissingData  []string  // 누락된 데이터 목록
    QualityScore float64   // 0.0 ~ 1.0
}
```

**책임**:
- 원천 데이터 수집
- 결측치/이상치 탐지
- 품질 점수 산출

---

## Stage 1: Universe (투자 유니버스)

**입력**: `DataQualitySnapshot`
**출력**: `Universe`

```go
type Universe struct {
    Date    time.Time
    Stocks  []string  // 종목 코드 리스트
    Reason  map[string]string  // 제외 사유
}
```

**필터 조건**:
- 거래정지 종목 제외
- 관리종목 제외
- 최소 거래대금 미달 제외
- 상장 후 N일 미만 제외

---

## Stage 2: Signals (시그널 생성)

**입력**: `Universe`
**출력**: `SignalSet`

```go
type SignalSet struct {
    Date     time.Time
    Signals  map[string]StockSignals  // 종목별 시그널
}

type StockSignals struct {
    Momentum    float64  // 모멘텀 점수
    Value       float64  // 가치 점수
    Quality     float64  // 퀄리티 점수
    Event       float64  // 이벤트 점수
    Technical   float64  // 기술적 점수
}
```

**시그널 종류**:
- Momentum: 가격/거래량 모멘텀
- Value: PER, PBR 등 가치 지표
- Quality: ROE, 부채비율 등
- Event: 공시, 뉴스 이벤트
- Technical: 이동평균, RSI 등

---

## Stage 3: Screener (1차 필터링)

**입력**: `SignalSet`
**출력**: `[]string` (통과 종목)

```go
// Hard Cut 조건
type ScreenerConfig struct {
    MinMomentum  float64  // 최소 모멘텀
    MinVolume    int64    // 최소 거래대금
    MaxPER       float64  // 최대 PER
}
```

**목적**: AI 분석 전 빠른 필터링으로 비용 절감

---

## Stage 4: Ranking (순위 산출)

**입력**: 통과 종목 + `SignalSet`
**출력**: `[]RankedStock`

```go
type RankedStock struct {
    Code        string
    TotalScore  float64
    Rank        int
    Scores      StockSignals  // 세부 점수
}
```

**가중치 예시**:
```yaml
weights:
  momentum: 0.3
  value: 0.2
  quality: 0.2
  event: 0.2
  technical: 0.1
```

---

## Stage 5: Portfolio (포트폴리오 구성)

**입력**: `[]RankedStock`
**출력**: `TargetPortfolio`

```go
type TargetPortfolio struct {
    Date      time.Time
    Positions []TargetPosition
}

type TargetPosition struct {
    Code       string
    Weight     float64  // 비중 (0.0 ~ 1.0)
    TargetQty  int      // 목표 수량
    Action     string   // "BUY" | "SELL" | "HOLD"
}
```

---

## Stage 6: Execution (주문 실행)

**입력**: `TargetPortfolio`
**출력**: `[]Order`

```go
type Order struct {
    Code      string
    Side      string  // "BUY" | "SELL"
    Quantity  int
    Price     int
    Status    string  // "PENDING" | "FILLED" | "REJECTED"
}
```

---

## Stage 7: Audit (성과 분석)

**입력**: 체결 내역 + 시장 데이터
**출력**: 성과 리포트

```go
type PerformanceReport struct {
    Period       string
    TotalReturn  float64
    Sharpe       float64
    MaxDrawdown  float64
    WinRate      float64
}
```

---

**Prev**: [System Overview](./system-overview.md)
**Next**: [Contracts](./contracts.md)
