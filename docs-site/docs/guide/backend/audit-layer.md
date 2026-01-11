---
sidebar_position: 7
title: Audit Layer
description: S7 성과 분석
---

# Audit Layer

> S7: 성과 분석

---

## 책임

트레이딩 성과 측정, 귀인 분석, 피드백 생성

---

## 구현 상태 (2026-01-11)

| 컴포넌트 | 상태 | 파일 |
|---------|------|------|
| **PerformanceAnalyzer** | ✅ 완료 | `internal/audit/performance.go` |
| **Attribution** | ✅ 완료 | `internal/audit/attribution.go` |
| **Snapshot** | ✅ 완료 | `internal/audit/snapshot.go` |
| **Repository** | ✅ 완료 | `internal/audit/repository.go` |
| **RiskReporter** | ✅ 완료 | `internal/audit/risk_report.go` |

### 리스크 모듈 (공용)

| 컴포넌트 | 상태 | 파일 |
|---------|------|------|
| **RiskEngine** | ✅ 완료 | `internal/risk/engine.go` |
| **VaR/CVaR** | ✅ 완료 | `internal/risk/var.go` |
| **Monte Carlo** | ✅ 완료 | `internal/risk/montecarlo.go` |
| **Risk Types** | ✅ 완료 | `internal/risk/types.go` |

---

## 프로세스 흐름

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Audit Pipeline                                      │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        ▼                           ▼                           ▼
┌───────────────────┐       ┌───────────────────┐       ┌───────────────────┐
│ 1. Daily Snapshot │       │ 2. Performance    │       │ 3. Attribution    │
│    (16:00 KST)    │       │    Analyzer       │       │    Analysis       │
├───────────────────┤       ├───────────────────┤       ├───────────────────┤
│ • 포트폴리오 상태 │       │ • Total Return    │       │ • Momentum 기여도  │
│ • 보유 종목       │       │ • Sharpe Ratio    │       │ • Flow 기여도      │
│ • 일별 수익률     │       │ • Max Drawdown    │       │ • Technical 기여도 │
│ • 누적 수익률     │       │ • Win Rate        │       │ • Value 기여도     │
│                   │       │ • Alpha/Beta      │       │ • Quality 기여도   │
└───────────────────┘       └───────────────────┘       └───────────────────┘
                                    │
                                    ▼
                    ┌───────────────────────────────────┐
                    │       PerformanceReport           │
                    ├───────────────────────────────────┤
                    │ 주요 지표:                        │
                    │ • Sharpe > 1.0                   │
                    │ • MDD < 20%                      │
                    │ • Win Rate > 50%                 │
                    │ • Alpha > 0                      │
                    └───────────────────────────────────┘
```

---

## 폴더 구조

```
internal/audit/
├── performance.go    # 성과 측정
├── attribution.go    # 귀인 분석
├── snapshot.go       # 스냅샷 저장
├── risk_report.go    # 리스크 리포팅 (NEW)
└── repository.go     # DB 저장소

internal/risk/        # 공용 리스크 엔진 (NEW)
├── engine.go         # RiskEngine 인터페이스/구현
├── var.go            # VaR/CVaR 계산
├── montecarlo.go     # Monte Carlo 시뮬레이터
└── types.go          # 공용 타입
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

    // 팩터별 기여도 계산 (SSOT: data-flow.md 기준)
    factors := []FactorInfo{
        {"momentum", 0.20},   // 20%
        {"technical", 0.20},  // 20%
        {"value", 0.15},      // 15%
        {"quality", 0.15},    // 15%
        {"flow", 0.25},       // 25% ⭐ 수급 (한국 시장 중요)
        {"event", 0.05},      // 5%
    }

    for _, factor := range factors {
        contrib := a.calculateFactorContribution(ctx, period, factor.Name)
        attrs = append(attrs, Attribution{
            Factor:       factor.Name,
            Contribution: contrib,
            Exposure:     a.getAverageExposure(ctx, period, factor.Name),
        })
    }

    return attrs, nil
}
```

---

## Risk Analysis (리스크 분석)

### 3단계 도입 계획

리스크 분석은 단계적으로 도입됩니다.

| Phase | 목표 | 상태 |
|-------|------|------|
| **Phase A: S7 Audit** | Monte Carlo/Forecast가 유용한지 검증 | ✅ 완료 |
| **Phase B: S6 Shadow** | 실운영에서 얼마나 자주 막히는지 관찰 | 🔜 예정 |
| **Phase C: S6 Enforce** | 실제 주문 거부/축소 | 🔜 예정 |

### 리스크 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│                        Risk Engine (공용)                        │
│                      internal/risk/engine.go                     │
└─────────────────────────────────────────────────────────────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                                 ▼
┌─────────────────────────────┐   ┌─────────────────────────────┐
│    S6 Execution (사전)      │   │    S7 Audit (사후)          │
│    Risk Gate                │   │    Risk Reporter            │
├─────────────────────────────┤   ├─────────────────────────────┤
│ • VaR/ES 한도 체크          │   │ • Monte Carlo 시뮬레이션    │
│ • 익스포저 한도             │   │ • Forecast 캘리브레이션     │
│ • 유동성 상한               │   │ • Decision tracing          │
│ • 빠름 (100-300ms)          │   │ • 무거움 (수 초)            │
│ • 결정적, 보수적            │   │ • 상세 분석                 │
└─────────────────────────────┘   └─────────────────────────────┘
```

### RiskEngine (공용)

S6(사전 게이트)와 S7(사후 리포트) 모두 사용하는 순수 계산 엔진입니다.

```go
// internal/risk/engine.go

type Engine struct{}

// NewEngine Engine 생성
func NewEngine() *Engine {
    return &Engine{}
}

// VaR 계산 (Historical Simulation)
func (e *Engine) VaR(returns []float64, confidence float64) VaRResult

// Monte Carlo 시뮬레이션
func (e *Engine) MonteCarlo(input PortfolioReturns, config MonteCarloConfig) (*MonteCarloResult, error)

// 스트레스 테스트
func (e *Engine) StressTest(weights map[string]float64, scenarios map[string]float64) map[string]float64
```

### VaR/CVaR

VaR(Value at Risk)는 특정 신뢰수준에서 예상되는 최대 손실입니다.
CVaR(Conditional VaR, Expected Shortfall)는 VaR를 초과하는 손실의 평균입니다.

```go
// internal/risk/var.go

// VaRResult VaR 계산 결과
type VaRResult struct {
    Confidence float64  // 신뢰수준 (0.95, 0.99)
    VaR        float64  // Value at Risk (손실=양수)
    CVaR       float64  // Expected Shortfall
}

// 해석 예시:
// VaR95 = 0.05 → "95% 신뢰수준에서 하루 최대 5% 손실"
// CVaR95 = 0.08 → "VaR를 초과하는 손실이 발생하면 평균 8% 손실"
```

**VaR 계산 방법 (Historical Simulation)**:
```go
// 과거 수익률을 정렬하여 percentile 계산
func (e *Engine) VaR(returns []float64, confidence float64) VaRResult {
    sorted := make([]float64, len(returns))
    copy(sorted, returns)
    sort.Float64s(sorted)

    idx := int((1.0 - confidence) * float64(len(sorted)))
    var result VaRResult
    result.Confidence = confidence
    result.VaR = -sorted[idx]  // 손실을 양수로 표현

    // CVaR: VaR 이하 수익률의 평균
    sum := 0.0
    for i := 0; i <= idx; i++ {
        sum += sorted[i]
    }
    result.CVaR = -sum / float64(idx+1)

    return result
}
```

### Monte Carlo 설정

```go
// internal/risk/types.go

type MonteCarloConfig struct {
    Mode             MonteCarloMode   // portfolio_univariate (빠름) / asset_multivariate (정밀)
    ReturnType       ReturnType       // simple / log
    NumSimulations   int              // 기본: 10000
    HoldingPeriod    int              // 보유 기간 (일), 기본: 5
    ConfidenceLevels []float64        // [0.95, 0.99]
    Method           MonteCarloMethod // historical / normal / t
    LookbackDays     int              // 기본: 200
    Seed             int64            // 재현성용 (0=랜덤)
    MinSamples       int              // 최소 샘플 수 (기본: 30, fail-closed)
}
```

**설정 옵션 상세**:

| 옵션 | 값 | 설명 |
|------|-----|------|
| **Mode** | `portfolio_univariate` | 포트폴리오 전체 수익률로 시뮬레이션 (빠름) |
| | `asset_multivariate` | 개별 자산별 시뮬레이션 (상관관계 고려, 정밀) |
| **Method** | `historical` | 과거 수익률 Bootstrap 샘플링 |
| | `normal` | 정규분포 가정 |
| | `t` | Student-t 분포 (fat tail 반영) |
| **Seed** | `0` | 랜덤 (매번 다른 결과) |
| | `42` | 고정 시드 (재현성 보장) |

### Monte Carlo 결과

```go
// internal/risk/types.go

type MonteCarloResult struct {
    RunID               string              `json:"run_id"`
    RunDate             time.Time           `json:"run_date"`
    DecisionSnapshotID  *int64              `json:"decision_snapshot_id"`  // 재현성용
    Config              MonteCarloConfig    `json:"config"`                // 전체 설정 기록
    InputSampleCount    int                 `json:"input_sample_count"`
    MeanReturn          float64             `json:"mean_return"`
    StdDev              float64             `json:"std_dev"`
    VaR95               float64             `json:"var_95"`
    VaR99               float64             `json:"var_99"`
    CVaR95              float64             `json:"cvar_95"`
    CVaR99              float64             `json:"cvar_99"`
    Percentiles         map[int]float64     `json:"percentiles"`  // {1, 5, 10, 25, 50, 75, 90, 95, 99}
}
```

### RiskReporter (S7)

```go
// internal/audit/risk_report.go

type RiskReporter struct {
    engine *risk.Engine
    repo   *Repository
    log    zerolog.Logger
}

// NewRiskReporter RiskReporter 생성
func NewRiskReporter(engine *risk.Engine, repo *Repository, log zerolog.Logger) *RiskReporter

// GenerateReport 전체 리스크 리포트 생성
func (r *RiskReporter) GenerateReport(ctx context.Context, input RiskReportInput) (*RiskReport, error)

// SaveMonteCarloResult Monte Carlo 결과 저장
func (r *RiskReporter) SaveMonteCarloResult(ctx context.Context, result *risk.MonteCarloResult) error
```

### CLI 명령어

```bash
# Monte Carlo 시뮬레이션
go run ./cmd/quant audit montecarlo                        # 기본 설정
go run ./cmd/quant audit montecarlo --simulations 50000    # 시뮬레이션 횟수
go run ./cmd/quant audit montecarlo --holding 5            # 보유 기간 (일)
go run ./cmd/quant audit montecarlo --method t             # Student-t 분포 (fat tail)
go run ./cmd/quant audit montecarlo --seed 42              # 재현성용 시드
go run ./cmd/quant audit montecarlo --output json          # JSON 출력

# 데모 모드 (포트폴리오 스냅샷 없을 때)
go run ./cmd/quant audit montecarlo --demo                 # 샘플 포트폴리오로 테스트
go run ./cmd/quant audit montecarlo --demo --method t --seed 42

# 리스크 리포트
go run ./cmd/quant audit risk-report                       # 기본
go run ./cmd/quant audit risk-report --date 2024-01-15     # 특정 날짜
go run ./cmd/quant audit risk-report --demo                # 데모 모드
go run ./cmd/quant audit risk-report --output json         # JSON 출력
```

### Demo 모드

포트폴리오 스냅샷이 없을 때 샘플 포트폴리오로 테스트할 수 있습니다.

```go
// 샘플 포트폴리오 (대형주 10종목, 균등 비중)
weights := map[string]float64{
    "005930": 0.10,  // 삼성전자
    "000660": 0.10,  // SK하이닉스
    "035420": 0.10,  // NAVER
    "035720": 0.10,  // 카카오
    "051910": 0.10,  // LG화학
    "006400": 0.10,  // 삼성SDI
    "005380": 0.10,  // 현대차
    "000270": 0.10,  // 기아
    "068270": 0.10,  // 셀트리온
    "105560": 0.10,  // KB금융
}
```

### 결과 예시

```
══════════════════════════════════════════════════════════════════
                Monte Carlo Simulation Results
══════════════════════════════════════════════════════════════════

📊 Configuration
  Run ID: mc_20260111_143052_abc123
  Simulations: 10,000
  Holding Period: 5 days
  Method: historical_bootstrap
  Seed: 42

📈 Input Data
  Sample Count: 487 days
  Portfolio Stocks: 10

📉 Risk Metrics
  Mean Return: +0.42%
  Std Dev: 2.18%
  VaR 95%: 3.21% (5일 최대 손실)
  VaR 99%: 5.67%
  CVaR 95%: 4.35% (Expected Shortfall)
  CVaR 99%: 6.89%

📊 Percentiles
  1%:  -6.12%
  5%:  -3.21%
  10%: -2.15%
  25%: -0.87%
  50%: +0.38% (median)
  75%: +1.65%
  90%: +2.89%
  95%: +3.54%
  99%: +5.21%

══════════════════════════════════════════════════════════════════
```

### 재현성 검증

동일한 seed를 사용하면 동일한 결과가 나옵니다.

```bash
# 첫 번째 실행
$ go run ./cmd/quant audit montecarlo --demo --seed 42
VaR 95%: 3.21%

# 두 번째 실행 (동일 seed)
$ go run ./cmd/quant audit montecarlo --demo --seed 42
VaR 95%: 3.21%  # ✅ 동일한 결과
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

-- analytics.montecarlo_results: Monte Carlo 결과
CREATE TABLE analytics.montecarlo_results (
    run_id              VARCHAR(50) PRIMARY KEY,
    run_date            DATE NOT NULL,
    decision_snapshot_id BIGINT REFERENCES audit.decision_snapshots(id),
    config              JSONB NOT NULL,       -- 재현성용
    input_sample_count  INT NOT NULL,
    mean_return         DECIMAL(10,6),
    std_dev             DECIMAL(10,6),
    var_95              DECIMAL(10,6),
    var_99              DECIMAL(10,6),
    cvar_95             DECIMAL(10,6),
    cvar_99             DECIMAL(10,6),
    percentiles         JSONB,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

-- analytics.var_daily_snapshots: 일별 VaR 추이
CREATE TABLE analytics.var_daily_snapshots (
    snapshot_date  DATE NOT NULL,
    portfolio_id   VARCHAR(50) DEFAULT 'main',
    var_95         DECIMAL(10,6),
    var_99         DECIMAL(10,6),
    cvar_95        DECIMAL(10,6),
    cvar_99        DECIMAL(10,6),
    portfolio_value DECIMAL(15,2),
    var_95_amount  DECIMAL(15,2),  -- 최대 손실 금액
    PRIMARY KEY (snapshot_date, portfolio_id)
);

-- analytics.stress_test_results: 스트레스 테스트
CREATE TABLE analytics.stress_test_results (
    run_id         VARCHAR(50),
    scenario_name  VARCHAR(50),
    portfolio_loss DECIMAL(10,6),
    loss_amount    DECIMAL(15,2),
    PRIMARY KEY (run_id, scenario_name)
);
```

---

**Prev**: [Execution Layer](./execution-layer.md)
**Next**: [Infrastructure](./infrastructure.md)
