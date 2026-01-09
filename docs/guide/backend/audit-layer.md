# Audit Layer

> S7: 성과 분석

---

## 책임

트레이딩 성과 측정, 귀인 분석, 피드백 생성

---

## 폴더 구조

```
internal/audit/
├── performance.go    # 성과 측정
├── attribution.go    # 귀인 분석
└── snapshot.go       # 스냅샷 저장
```

---

## Performance Analyzer

### 인터페이스

```go
type Auditor interface {
    Analyze(ctx context.Context, period string) (*PerformanceReport, error)
}
```

### 구현

```go
// internal/audit/performance.go

type auditor struct {
    db *pgxpool.Pool
}

type PerformanceReport struct {
    Period        string    `json:"period"`
    StartDate     time.Time `json:"start_date"`
    EndDate       time.Time `json:"end_date"`

    // 수익률
    TotalReturn   float64   `json:"total_return"`
    AnnualReturn  float64   `json:"annual_return"`

    // 리스크 지표
    Volatility    float64   `json:"volatility"`
    Sharpe        float64   `json:"sharpe"`
    Sortino       float64   `json:"sortino"`
    MaxDrawdown   float64   `json:"max_drawdown"`

    // 트레이딩 지표
    WinRate       float64   `json:"win_rate"`
    AvgWin        float64   `json:"avg_win"`
    AvgLoss       float64   `json:"avg_loss"`
    ProfitFactor  float64   `json:"profit_factor"`

    // 비교
    Benchmark     float64   `json:"benchmark"`      // KOSPI 수익률
    Alpha         float64   `json:"alpha"`
    Beta          float64   `json:"beta"`
}

func (a *auditor) Analyze(ctx context.Context, period string) (*PerformanceReport, error) {
    report := &PerformanceReport{Period: period}

    // 기간 파싱
    report.StartDate, report.EndDate = a.parsePeriod(period)

    // 일별 수익률 조회
    dailyReturns := a.getDailyReturns(ctx, report.StartDate, report.EndDate)

    // 수익률 계산
    report.TotalReturn = a.calculateTotalReturn(dailyReturns)
    report.AnnualReturn = a.annualize(report.TotalReturn, len(dailyReturns))

    // 리스크 지표
    report.Volatility = a.calculateVolatility(dailyReturns)
    report.Sharpe = a.calculateSharpe(report.AnnualReturn, report.Volatility)
    report.MaxDrawdown = a.calculateMaxDrawdown(dailyReturns)

    // 트레이딩 지표
    trades := a.getTrades(ctx, report.StartDate, report.EndDate)
    report.WinRate = a.calculateWinRate(trades)
    report.ProfitFactor = a.calculateProfitFactor(trades)

    // 벤치마크 비교
    report.Benchmark = a.getBenchmarkReturn(ctx, report.StartDate, report.EndDate)
    report.Alpha = report.TotalReturn - report.Benchmark

    return report, nil
}
```

---

## 귀인 분석 (Attribution)

어떤 요인이 수익에 기여했는지 분석:

```go
// internal/audit/attribution.go

type Attribution struct {
    Factor      string  `json:"factor"`
    Contribution float64 `json:"contribution"`  // 수익 기여도
    Exposure    float64  `json:"exposure"`       // 평균 노출도
}

func (a *auditor) Attribution(ctx context.Context, period string) ([]Attribution, error) {
    attrs := make([]Attribution, 0)

    // 팩터별 기여도 계산
    factors := []string{"momentum", "value", "quality", "event"}

    for _, factor := range factors {
        contrib := a.calculateFactorContribution(ctx, period, factor)
        attrs = append(attrs, Attribution{
            Factor:       factor,
            Contribution: contrib,
            Exposure:     a.getAverageExposure(ctx, period, factor),
        })
    }

    return attrs, nil
}
```

---

## 스냅샷 저장

매일 포트폴리오 상태 기록:

```go
// internal/audit/snapshot.go

type DailySnapshot struct {
    Date         time.Time              `json:"date"`
    TotalValue   float64                `json:"total_value"`
    Cash         float64                `json:"cash"`
    Positions    []PositionSnapshot     `json:"positions"`
    DailyReturn  float64                `json:"daily_return"`
    CumReturn    float64                `json:"cum_return"`
}

type PositionSnapshot struct {
    Code        string  `json:"code"`
    Quantity    int     `json:"quantity"`
    Price       int     `json:"price"`
    Value       float64 `json:"value"`
    Weight      float64 `json:"weight"`
    DailyPnL    float64 `json:"daily_pnl"`
}

func (a *auditor) SaveSnapshot(ctx context.Context) error {
    snapshot := &DailySnapshot{
        Date: time.Now(),
    }

    // 현재 잔고 조회
    balance := a.broker.GetBalance(ctx)
    snapshot.TotalValue = balance.TotalValue
    snapshot.Cash = balance.Cash

    // 포지션 스냅샷
    for _, pos := range balance.Positions {
        snapshot.Positions = append(snapshot.Positions, PositionSnapshot{
            Code:     pos.Code,
            Quantity: pos.Quantity,
            Price:    pos.CurrentPrice,
            Value:    float64(pos.Quantity * pos.CurrentPrice),
            Weight:   float64(pos.Quantity*pos.CurrentPrice) / snapshot.TotalValue,
        })
    }

    // 수익률 계산
    prevSnapshot := a.getPreviousSnapshot(ctx)
    if prevSnapshot != nil {
        snapshot.DailyReturn = (snapshot.TotalValue - prevSnapshot.TotalValue) / prevSnapshot.TotalValue
    }

    // 저장
    return a.saveSnapshot(ctx, snapshot)
}
```

---

## 주요 지표 설명

| 지표 | 설명 | 목표 |
|------|------|------|
| **Sharpe** | (수익률 - 무위험) / 변동성 | > 1.0 |
| **Sortino** | (수익률 - 무위험) / 하락변동성 | > 1.5 |
| **MDD** | 최대 낙폭 | < 20% |
| **Win Rate** | 승률 | > 50% |
| **Profit Factor** | 총이익 / 총손실 | > 1.5 |
| **Alpha** | 벤치마크 대비 초과수익 | > 0 |

---

## 설정 예시 (YAML)

```yaml
# config/audit.yaml

audit:
  # 스냅샷
  snapshot:
    enabled: true
    time: "16:00"  # 매일 오후 4시

  # 벤치마크
  benchmark: "KOSPI"  # KOSPI, KOSDAQ

  # 알림 기준
  alerts:
    max_drawdown: -0.10    # MDD -10% 도달 시
    daily_loss: -0.03      # 일 손실 -3% 시
```

---

## DB 스키마

```sql
-- audit.daily_snapshots: 일별 스냅샷
CREATE TABLE audit.daily_snapshots (
    id           SERIAL PRIMARY KEY,
    date         DATE NOT NULL UNIQUE,
    total_value  DECIMAL(15,2),
    cash         DECIMAL(15,2),
    positions    JSONB,
    daily_return DECIMAL(8,6),
    cum_return   DECIMAL(8,6),
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- audit.performance_reports: 성과 리포트
CREATE TABLE audit.performance_reports (
    id            SERIAL PRIMARY KEY,
    period        VARCHAR(20),      -- "1M", "3M", "YTD", "1Y"
    start_date    DATE,
    end_date      DATE,
    total_return  DECIMAL(8,6),
    sharpe        DECIMAL(6,3),
    max_drawdown  DECIMAL(8,6),
    win_rate      DECIMAL(5,4),
    report_data   JSONB,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- audit.attributions: 귀인 분석
CREATE TABLE audit.attributions (
    id           SERIAL PRIMARY KEY,
    period       VARCHAR(20),
    factor       VARCHAR(20),
    contribution DECIMAL(8,6),
    exposure     DECIMAL(5,4),
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_snapshots_date ON audit.daily_snapshots(date);
```

---

**Prev**: [Execution Layer](./execution-layer.md)
**Next**: [Frontend Folder Structure](../frontend/folder-structure.md)
