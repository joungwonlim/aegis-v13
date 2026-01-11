---
sidebar_position: 2
title: Data Flow
description: 7단계 파이프라인과 데이터 흐름
---

# Data Flow

> 7단계 파이프라인 상세

---

## Pipeline Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         AEGIS v13 PIPELINE                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  External    ┌────┐   ┌────┐   ┌────┐   ┌────┐   ┌────┐   ┌────┐       │
│  Sources ───→│ S0 │──→│ S1 │──→│ S2 │──→│ S3 │──→│ S4 │──→│ S5 │       │
│              └────┘   └────┘   └────┘   └────┘   └────┘   └────┘       │
│              Data     Universe Signals  Screen   Rank     Portfolio    │
│                                                              │         │
│                                                              ▼         │
│                                                           ┌────┐       │
│              ┌────┐                                       │ S6 │       │
│              │ S7 │◀──────────────────────────────────────┤    │       │
│              └────┘                                       └────┘       │
│              Audit                                        Execution    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 데이터 흐름 요약

| Stage | 입력 | 처리 | 출력 |
|-------|------|------|------|
| **S0** | 외부 API | 수집 + 품질검증 | `DataQualitySnapshot` |
| **S1** | DataQualitySnapshot | 종목 필터링 | `Universe` |
| **S2** | Universe + Raw Data | 시그널 생성 | `SignalSet` |
| **S3** | SignalSet | Hard Cut 필터 | `[]string` (통과 종목) |
| **S4** | 통과 종목 + SignalSet | 점수화 + 순위 | `[]RankedStock` |
| **S5** | RankedStock + 현재 포지션 | 최적화 | `TargetPortfolio` |
| **S6** | TargetPortfolio | 주문 실행 | `[]Order` |
| **S7** | 체결 내역 | 성과 분석 | `PerformanceReport` |

---

## Stage 0: Data (데이터 수집/품질)

### 수집 데이터

| 데이터 | 필수 | 커버리지 | 소스 | 갱신 |
|--------|------|----------|------|------|
| **가격 (OHLCV)** | ✅ | 100% | Naver | Daily |
| **거래량** | ✅ | 100% | Naver | Daily |
| **시가총액** | ✅ | 95%+ | Naver/KRX | Daily |
| **재무제표** | ⚠️ | 80%+ | DART/Naver | Quarterly |
| **투자자 수급** | ⚠️ | 80%+ | Naver | Daily |
| **공시** | ⚠️ | 70%+ | DART | Real-time |

### 출력: DataQualitySnapshot

```go
type DataQualitySnapshot struct {
    Date         time.Time     `json:"date"`
    TotalStocks  int           `json:"total_stocks"`
    ValidStocks  int           `json:"valid_stocks"`
    Coverage     DataCoverage  `json:"coverage"`
    QualityScore float64       `json:"quality_score"`  // 0.0 ~ 1.0
    PassedGate   bool          `json:"passed_gate"`
    Issues       []DataIssue   `json:"issues"`
}

type DataCoverage struct {
    Price      float64 `json:"price"`       // 가격 커버리지
    Volume     float64 `json:"volume"`      // 거래량 커버리지
    MarketCap  float64 `json:"market_cap"`  // 시총 커버리지
    Financials float64 `json:"financials"`  // 재무 커버리지
    Investor   float64 `json:"investor"`    // 수급 커버리지
    Disclosure float64 `json:"disclosure"`  // 공시 커버리지
}
```

### 품질 게이트 조건

```yaml
quality_gate:
  price: 1.0       # 100% 필수
  volume: 1.0      # 100% 필수
  market_cap: 0.95 # 95% 이상
  financials: 0.80 # 80% 이상
  investor: 0.80   # 80% 이상
  disclosure: 0.70 # 70% 이상 (선택)
```

---

## Stage 1: Universe (투자 유니버스)

### 입력
- `DataQualitySnapshot` (S0에서 전달)

### 필터 조건

| 조건 | 기본값 | 설명 |
|------|--------|------|
| 거래정지 | 제외 | `is_halted = false` |
| 관리종목 | 제외 | `is_admin = false` |
| SPAC | 제외 | `is_spac = false` |
| 최소 시총 | 1,000억 | `market_cap >= 100B` |
| 최소 거래대금 | 5억 | `avg_volume >= 500M` |
| 최소 상장일 | 90일 | `listing_days >= 90` |
| 제외 섹터 | 금융, 보험 | 업종 필터 |

### 출력: Universe

```go
type Universe struct {
    Date       time.Time         `json:"date"`
    Stocks     []string          `json:"stocks"`     // 투자 가능 종목
    Excluded   map[string]string `json:"excluded"`   // 제외 종목: 사유
    TotalCount int               `json:"total_count"`
    Config     UniverseConfig    `json:"config"`
}
```

---

## Stage 2: Signals (시그널 생성)

### 입력
- `Universe` (S1에서 전달)
- Raw Data (가격, 재무, 수급 등)

### 시그널 유형

| 시그널 | 사용 데이터 | 지표 예시 |
|--------|------------|----------|
| **Momentum** | 가격, 거래량 | 수익률, 거래량 증가율 |
| **Technical** | 가격 | RSI, MACD, 이평선 |
| **Value** | 재무, 밸류에이션 | PER, PBR, PSR |
| **Quality** | 재무 | ROE, 부채비율, 성장률 |
| **Flow** | 투자자 수급 | 외국인/기관 순매수 |
| **Event** | 공시 | 실적, 지분변동, 자사주 |

### 출력: SignalSet

```go
type SignalSet struct {
    Date    time.Time                 `json:"date"`
    Signals map[string]*StockSignals  `json:"signals"`  // 종목별 시그널
}

type StockSignals struct {
    Code       string  `json:"code"`

    // 시그널 점수 (-1.0 ~ 1.0)
    Momentum   float64 `json:"momentum"`
    Technical  float64 `json:"technical"`
    Value      float64 `json:"value"`
    Quality    float64 `json:"quality"`
    Flow       float64 `json:"flow"`       // 수급 시그널
    Event      float64 `json:"event"`

    // 원본 데이터
    Details    SignalDetails `json:"details"`
}

type SignalDetails struct {
    // Momentum
    Return1M   float64 `json:"return_1m"`
    Return3M   float64 `json:"return_3m"`
    VolumeRate float64 `json:"volume_rate"`

    // Technical
    RSI        float64 `json:"rsi"`
    MACD       float64 `json:"macd"`
    MA20Cross  int     `json:"ma20_cross"`  // 1=상향돌파, -1=하향돌파

    // Value
    PER        float64 `json:"per"`
    PBR        float64 `json:"pbr"`
    PSR        float64 `json:"psr"`

    // Quality
    ROE        float64 `json:"roe"`
    DebtRatio  float64 `json:"debt_ratio"`

    // Flow (수급)
    ForeignNet5D  int64 `json:"foreign_net_5d"`   // 외국인 5일 순매수
    ForeignNet20D int64 `json:"foreign_net_20d"`  // 외국인 20일 순매수
    InstNet5D     int64 `json:"inst_net_5d"`      // 기관 5일 순매수
    InstNet20D    int64 `json:"inst_net_20d"`     // 기관 20일 순매수
}
```

---

## Stage 3: Screener (1차 필터링)

### 입력
- `SignalSet` (S2에서 전달)

### Hard Cut 조건

| 조건 | 기본값 | 목적 |
|------|--------|------|
| 최소 모멘텀 | -0.5 | 급락 종목 제외 |
| 최소 거래대금 | 10억 | 유동성 확보 |
| 최대 PER | 100 | 과대평가 제외 |
| 최소 ROE | 0% | 적자 기업 제외 |

### 출력

```go
// 통과한 종목 코드 리스트
[]string{"005930", "000660", "035420", ...}
```

**목적**: S4 Ranking 전에 빠르게 후보군 축소

---

## Stage 4: Ranking (순위 산출)

### 입력
- 통과 종목 (S3에서 전달)
- `SignalSet` (S2에서 전달)

### 가중치 설정

```yaml
weights:
  momentum: 0.25
  technical: 0.15
  value: 0.20
  quality: 0.15
  flow: 0.20      # 수급
  event: 0.05
```

### 출력: []RankedStock

```go
type RankedStock struct {
    Code       string       `json:"code"`
    Name       string       `json:"name"`
    Rank       int          `json:"rank"`
    TotalScore float64      `json:"total_score"`
    Scores     ScoreDetail  `json:"scores"`
}

type ScoreDetail struct {
    Momentum  float64 `json:"momentum"`
    Technical float64 `json:"technical"`
    Value     float64 `json:"value"`
    Quality   float64 `json:"quality"`
    Flow      float64 `json:"flow"`
    Event     float64 `json:"event"`
}
```

---

## Stage 5: Portfolio (포트폴리오 구성)

### 입력
- `[]RankedStock` (S4에서 전달)
- 현재 포지션

### 제약 조건

```yaml
constraints:
  max_stocks: 20           # 최대 종목 수
  max_weight: 0.10         # 종목당 최대 비중
  min_weight: 0.02         # 종목당 최소 비중
  max_sector_weight: 0.30  # 섹터당 최대 비중
  turnover_limit: 0.30     # 일일 회전율 제한
```

### 출력: TargetPortfolio

```go
type TargetPortfolio struct {
    Date      time.Time        `json:"date"`
    Positions []TargetPosition `json:"positions"`
    Summary   PortfolioSummary `json:"summary"`
}

type TargetPosition struct {
    Code        string  `json:"code"`
    Name        string  `json:"name"`
    Weight      float64 `json:"weight"`       // 목표 비중
    TargetValue int64   `json:"target_value"` // ⭐ 목표 금액 (수량은 S6에서 계산)
    CurrentQty  int     `json:"current_qty"`  // 현재 수량
    Action      string  `json:"action"`       // BUY, SELL, HOLD
    Reason      string  `json:"reason"`       // 편입/편출 사유
}

type PortfolioSummary struct {
    TotalStocks   int     `json:"total_stocks"`
    TotalBuys     int     `json:"total_buys"`
    TotalSells    int     `json:"total_sells"`
    TurnoverRate  float64 `json:"turnover_rate"`
    ExpectedCost  float64 `json:"expected_cost"`  // 예상 거래비용
}
```

---

## Stage 6: Execution (주문 실행)

### 입력
- `TargetPortfolio` (S5에서 전달)

### 주문 생성

```go
type Order struct {
    ID        string    `json:"id"`
    Code      string    `json:"code"`
    Side      string    `json:"side"`      // BUY, SELL
    OrderType string    `json:"order_type"` // LIMIT, MARKET
    Quantity  int       `json:"quantity"`
    Price     int       `json:"price"`
    Status    string    `json:"status"`    // PENDING, FILLED, REJECTED
    CreatedAt time.Time `json:"created_at"`
    FilledAt  time.Time `json:"filled_at"`
    FilledQty int       `json:"filled_qty"`
    FilledAvg int       `json:"filled_avg"` // 체결 평균가
}
```

### 실행 전략

| 전략 | 설명 |
|------|------|
| TWAP | 시간 분산 체결 |
| VWAP | 거래량 가중 분산 |
| Market | 즉시 시장가 체결 |

---

## Stage 7: Audit (성과 분석)

### 입력
- 체결 내역
- 시장 데이터

### 출력: PerformanceReport

```go
type PerformanceReport struct {
    Period       string  `json:"period"`        // daily, weekly, monthly
    StartDate    string  `json:"start_date"`
    EndDate      string  `json:"end_date"`

    // 수익률
    TotalReturn  float64 `json:"total_return"`
    BenchReturn  float64 `json:"bench_return"`  // KOSPI 대비
    ActiveReturn float64 `json:"active_return"` // 초과 수익

    // 리스크
    Volatility   float64 `json:"volatility"`
    MaxDrawdown  float64 `json:"max_drawdown"`
    Sharpe       float64 `json:"sharpe"`

    // 거래
    WinRate      float64 `json:"win_rate"`
    TurnoverRate float64 `json:"turnover_rate"`
    TotalCost    float64 `json:"total_cost"`

    // 상세
    ByStock      []StockPerformance `json:"by_stock"`
    BySignal     []SignalAttribution `json:"by_signal"`
}

type SignalAttribution struct {
    SignalType  string  `json:"signal_type"`   // momentum, value, flow...
    Contribution float64 `json:"contribution"` // 수익 기여도
}
```

---

## 데이터 의존성 맵

```
┌────────────────────────────────────────────────────────────────┐
│                    DATA DEPENDENCY MAP                         │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  External Data                                                 │
│  ┌─────────┬─────────┬─────────┬─────────┬─────────┐         │
│  │  Price  │ Volume  │ Market  │ Finance │ Investor│         │
│  │  OHLCV  │         │   Cap   │   Data  │  Flow   │         │
│  └────┬────┴────┬────┴────┬────┴────┬────┴────┬────┘         │
│       │         │         │         │         │               │
│       ▼         ▼         ▼         ▼         ▼               │
│  ┌─────────────────────────────────────────────────┐          │
│  │                    S0: Data                      │          │
│  │            DataQualitySnapshot                   │          │
│  └──────────────────────┬──────────────────────────┘          │
│                         │                                      │
│                         ▼                                      │
│  ┌─────────────────────────────────────────────────┐          │
│  │                  S1: Universe                    │          │
│  │                    Universe                      │          │
│  └──────────────────────┬──────────────────────────┘          │
│                         │                                      │
│       ┌─────────────────┼─────────────────┐                   │
│       │                 │                 │                    │
│       ▼                 ▼                 ▼                    │
│  ┌─────────┐      ┌─────────┐      ┌─────────┐               │
│  │Technical│      │  Value  │      │  Flow   │               │
│  │ Signal  │      │ Signal  │      │ Signal  │               │
│  └────┬────┘      └────┬────┘      └────┬────┘               │
│       │                │                │                     │
│       └────────────────┼────────────────┘                     │
│                        ▼                                       │
│  ┌─────────────────────────────────────────────────┐          │
│  │                  S2: Signals                     │          │
│  │                    SignalSet                     │          │
│  └──────────────────────┬──────────────────────────┘          │
│                         │                                      │
│                         ▼                                      │
│                    S3 → S4 → S5 → S6 → S7                     │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

---

## 일일 처리 타임라인

| 시간 | Stage | 작업 |
|------|-------|------|
| 08:00 | S0 | 종목 마스터 갱신 |
| 15:30 | - | 장 마감 |
| 15:45 | S0 | 가격/거래량 수집 |
| 15:50 | S0 | 시가총액 수집 |
| 16:05 | S0 | 투자자 수급 수집 |
| 16:30 | S0 | 데이터 품질 검증 |
| 16:35 | S1 | Universe 생성 |
| 16:40 | S2 | 시그널 생성 |
| 16:50 | S3 | Screening |
| 16:55 | S4 | Ranking |
| 17:00 | S5 | 포트폴리오 생성 |
| 17:10 | S6 | 주문 생성 (익일 실행) |
| 17:30 | S7 | 금일 성과 분석 |

---

**Prev**: [System Overview](./system-overview.md)
**Next**: [Contracts](./contracts.md)
