---
sidebar_position: 2
title: Data Layer
description: S0 Data Collection + S1 Universe
---

# Data Layer

> S0: 데이터 수집/품질 + S1: 투자 유니버스

---

## 역할

| Stage | 역할 | 입력 | 출력 |
|-------|------|------|------|
| **S0** | 데이터 수집, 품질 검증 | 외부 API | `DataQualitySnapshot` |
| **S1** | 투자 가능 종목 필터링 | `DataQualitySnapshot` | `Universe` |

**데이터 요구사항**: [Data Requirements](../architecture/data-requirements.md) 참조

---

## 폴더 구조

```
internal/
├── s0_data/
│   ├── sources/           # 데이터 소스별 수집
│   │   ├── naver/
│   │   │   ├── client.go
│   │   │   ├── prices.go       # 가격/거래량
│   │   │   ├── market_cap.go   # 시가총액
│   │   │   ├── fundamentals.go # 재무제표
│   │   │   └── investor.go     # 투자자 수급
│   │   ├── dart/
│   │   │   ├── client.go
│   │   │   ├── disclosure.go   # 공시
│   │   │   └── financials.go   # 재무제표 상세
│   │   └── krx/
│   │       ├── client.go
│   │       ├── stocks.go       # 종목 마스터
│   │       └── market.go       # 시장 지표
│   ├── quality/
│   │   ├── validator.go        # 데이터 검증
│   │   └── coverage.go         # 커버리지 체크
│   ├── scheduler/
│   │   ├── scheduler.go        # 수집 스케줄러
│   │   └── job.go              # Job 정의
│   ├── repository.go           # DB 저장
│   └── contract.go             # DataQualitySnapshot
│
├── s1_universe/
│   ├── builder.go              # Universe 생성
│   ├── filters.go              # 필터 조건들
│   ├── repository.go           # DB 저장
│   └── contract.go             # Universe
```

---

## S0: Data Collection

### 수집 대상 데이터

| 데이터 | 소스 | 커버리지 목표 | 갱신 주기 |
|--------|------|--------------|----------|
| 가격/거래량 | Naver | 100% | Daily |
| 시가총액 | Naver | 95%+ | Daily |
| 투자자 수급 | Naver | 80%+ | Daily |
| 재무제표 | DART/Naver | 80%+ | Quarterly |
| 밸류에이션 | Naver/계산 | 80%+ | Daily |
| 공시 | DART | 70%+ | Real-time |
| 종목 마스터 | KRX | 100% | Daily |

### 소스 인터페이스

```go
// internal/s0_data/sources/source.go

type DataSource interface {
    Name() string
    Fetch(ctx context.Context, opts FetchOptions) error
}

type FetchOptions struct {
    Symbols   []string   // 특정 종목만 (nil = 전체)
    DateFrom  time.Time
    DateTo    time.Time
}
```

### Naver Source 구현

```go
// internal/s0_data/sources/naver/prices.go

type PriceSource struct {
    client *NaverClient
    repo   *repository.PriceRepository
}

func (s *PriceSource) Name() string { return "naver_prices" }

func (s *PriceSource) Fetch(ctx context.Context, opts FetchOptions) error {
    stocks := opts.Symbols
    if len(stocks) == 0 {
        stocks = s.repo.GetAllStockCodes(ctx)
    }

    for _, code := range stocks {
        data, err := s.client.FetchOHLCV(ctx, code, opts.DateFrom, opts.DateTo)
        if err != nil {
            s.logger.Warn("fetch failed", "code", code, "error", err)
            continue
        }

        if err := s.repo.SaveOHLCV(ctx, data); err != nil {
            return fmt.Errorf("save OHLCV failed: %w", err)
        }
    }

    return nil
}
```

### 투자자 수급 수집

```go
// internal/s0_data/sources/naver/investor.go

type InvestorSource struct {
    client *NaverClient
    repo   *repository.InvestorRepository
}

func (s *InvestorSource) Fetch(ctx context.Context, opts FetchOptions) error {
    stocks := opts.Symbols
    if len(stocks) == 0 {
        stocks = s.repo.GetAllStockCodes(ctx)
    }

    for _, code := range stocks {
        flow, err := s.client.FetchInvestorFlow(ctx, code, opts.DateFrom, opts.DateTo)
        if err != nil {
            continue // 일부 종목 실패 허용
        }

        if err := s.repo.SaveInvestorFlow(ctx, flow); err != nil {
            return fmt.Errorf("save investor flow failed: %w", err)
        }
    }

    return nil
}
```

---

## S0: Quality Gate

### Contract

```go
// internal/s0_data/contract.go

type DataQualitySnapshot struct {
    Date         time.Time          `json:"date"`
    TotalStocks  int                `json:"total_stocks"`
    ValidStocks  int                `json:"valid_stocks"`
    Coverage     DataCoverage       `json:"coverage"`
    QualityScore float64            `json:"quality_score"`
    Issues       []DataIssue        `json:"issues"`
    PassedGate   bool               `json:"passed_gate"`
}

type DataCoverage struct {
    Price       float64 `json:"price"`        // 가격 커버리지
    Volume      float64 `json:"volume"`       // 거래량 커버리지
    MarketCap   float64 `json:"market_cap"`   // 시가총액 커버리지
    Financials  float64 `json:"financials"`   // 재무제표 커버리지
    Valuation   float64 `json:"valuation"`    // 밸류에이션 커버리지
    Investor    float64 `json:"investor"`     // 수급 커버리지
    Disclosure  float64 `json:"disclosure"`   // 공시 커버리지
}

type DataIssue struct {
    DataType    string `json:"data_type"`
    StockCode   string `json:"stock_code"`
    IssueType   string `json:"issue_type"`   // missing, outlier, stale
    Description string `json:"description"`
}
```

### Quality Gate 구현

```go
// internal/s0_data/quality/validator.go

type QualityGate struct {
    db     *pgxpool.Pool
    config QualityConfig
}

type QualityConfig struct {
    MinPriceCoverage     float64 `yaml:"min_price_coverage"`     // 1.0 (100%)
    MinVolumeCoverage    float64 `yaml:"min_volume_coverage"`    // 1.0 (100%)
    MinMarketCapCoverage float64 `yaml:"min_market_cap_coverage"` // 0.95
    MinFinancialCoverage float64 `yaml:"min_financial_coverage"` // 0.80
    MinInvestorCoverage  float64 `yaml:"min_investor_coverage"`  // 0.80
    MinDisclosureCoverage float64 `yaml:"min_disclosure_coverage"` // 0.70
}

func (g *QualityGate) Check(ctx context.Context, date time.Time) (*DataQualitySnapshot, error) {
    snapshot := &DataQualitySnapshot{
        Date:   date,
        Issues: make([]DataIssue, 0),
    }

    // 1. 전체 종목 수
    snapshot.TotalStocks = g.countTotalStocks(ctx)

    // 2. 커버리지 체크
    snapshot.Coverage = DataCoverage{
        Price:      g.checkCoverage(ctx, date, "price"),
        Volume:     g.checkCoverage(ctx, date, "volume"),
        MarketCap:  g.checkCoverage(ctx, date, "market_cap"),
        Financials: g.checkCoverage(ctx, date, "financials"),
        Valuation:  g.checkCoverage(ctx, date, "valuation"),
        Investor:   g.checkCoverage(ctx, date, "investor"),
        Disclosure: g.checkCoverage(ctx, date, "disclosure"),
    }

    // 3. 이상치 탐지
    snapshot.Issues = append(snapshot.Issues, g.detectOutliers(ctx, date)...)

    // 4. 품질 점수 계산
    snapshot.QualityScore = g.calculateScore(snapshot.Coverage)
    snapshot.ValidStocks = int(float64(snapshot.TotalStocks) * snapshot.QualityScore)

    // 5. Gate 통과 여부
    snapshot.PassedGate = g.checkPassCriteria(snapshot.Coverage)

    return snapshot, nil
}

func (g *QualityGate) checkPassCriteria(c DataCoverage) bool {
    return c.Price >= g.config.MinPriceCoverage &&
           c.Volume >= g.config.MinVolumeCoverage &&
           c.MarketCap >= g.config.MinMarketCapCoverage &&
           c.Financials >= g.config.MinFinancialCoverage &&
           c.Investor >= g.config.MinInvestorCoverage
    // Disclosure는 선택적
}
```

---

## S1: Universe Builder

### Contract

```go
// internal/s1_universe/contract.go

type Universe struct {
    Date        time.Time            `json:"date"`
    Stocks      []string             `json:"stocks"`      // 포함 종목
    Excluded    map[string]string    `json:"excluded"`    // 제외 종목: 사유
    TotalCount  int                  `json:"total_count"`
    Config      UniverseConfig       `json:"config"`
}

type UniverseConfig struct {
    MinMarketCap   int64  `yaml:"min_market_cap"`    // 최소 시총 (억)
    MinVolume      int64  `yaml:"min_volume"`        // 최소 거래대금 (백만)
    MinListingDays int    `yaml:"min_listing_days"`  // 최소 상장일수
    ExcludeAdmin   bool   `yaml:"exclude_admin"`     // 관리종목 제외
    ExcludeHalt    bool   `yaml:"exclude_halt"`      // 거래정지 제외
    ExcludeSPAC    bool   `yaml:"exclude_spac"`      // SPAC 제외
    ExcludeSectors []string `yaml:"exclude_sectors"` // 제외 섹터
}
```

### Universe Builder 구현

```go
// internal/s1_universe/builder.go

type UniverseBuilder struct {
    db     *pgxpool.Pool
    config UniverseConfig
}

func (b *UniverseBuilder) Build(ctx context.Context, snapshot *DataQualitySnapshot) (*Universe, error) {
    if !snapshot.PassedGate {
        return nil, fmt.Errorf("data quality gate not passed: score=%.2f", snapshot.QualityScore)
    }

    universe := &Universe{
        Date:     snapshot.Date,
        Stocks:   make([]string, 0),
        Excluded: make(map[string]string),
        Config:   b.config,
    }

    // 전체 종목 조회
    allStocks, err := b.getAllStocks(ctx)
    if err != nil {
        return nil, err
    }

    for _, stock := range allStocks {
        reason := b.checkExclusion(ctx, stock)
        if reason != "" {
            universe.Excluded[stock.Code] = reason
            continue
        }
        universe.Stocks = append(universe.Stocks, stock.Code)
    }

    universe.TotalCount = len(universe.Stocks)
    return universe, nil
}

func (b *UniverseBuilder) checkExclusion(ctx context.Context, stock Stock) string {
    // 우선순위 순서로 체크
    if b.config.ExcludeHalt && stock.IsHalted {
        return "거래정지"
    }
    if b.config.ExcludeAdmin && stock.IsAdmin {
        return "관리종목"
    }
    if b.config.ExcludeSPAC && stock.IsSPAC {
        return "SPAC"
    }
    if stock.MarketCap < b.config.MinMarketCap*100_000_000 { // 억 → 원
        return fmt.Sprintf("시가총액 미달 (%d억)", stock.MarketCap/100_000_000)
    }
    if stock.AvgVolume < b.config.MinVolume*1_000_000 { // 백만 → 원
        return fmt.Sprintf("거래대금 미달 (%d백만)", stock.AvgVolume/1_000_000)
    }
    if stock.ListingDays < b.config.MinListingDays {
        return fmt.Sprintf("상장일수 미달 (%d일)", stock.ListingDays)
    }
    for _, sector := range b.config.ExcludeSectors {
        if stock.Sector == sector {
            return fmt.Sprintf("제외 섹터 (%s)", sector)
        }
    }
    return "" // 통과
}
```

---

## 설정

```yaml
# config/data.yaml

# S0: 데이터 수집
data:
  sources:
    naver:
      rate_limit: 10  # requests per second
      timeout: 30s
    dart:
      api_key: ${DART_API_KEY}
      rate_limit: 100  # requests per minute
    krx:
      timeout: 60s

  quality:
    min_price_coverage: 1.0
    min_volume_coverage: 1.0
    min_market_cap_coverage: 0.95
    min_financial_coverage: 0.80
    min_investor_coverage: 0.80
    min_disclosure_coverage: 0.70

# S1: 유니버스
universe:
  min_market_cap: 1000      # 1000억 이상
  min_volume: 500           # 5억 이상
  min_listing_days: 90      # 상장 90일 이상
  exclude_admin: true
  exclude_halt: true
  exclude_spac: true
  exclude_sectors:
    - "금융"
    - "보험"
```

---

## 스케줄

| 작업 | 시간 | 설명 |
|------|------|------|
| 종목 마스터 | 08:00 | 장 시작 전 |
| 가격/거래량 | 15:45 | 장 마감 후 |
| 시가총액 | 15:50 | 가격 수집 후 |
| 투자자 수급 | 16:05 | 집계 완료 후 |
| 품질 체크 | 16:30 | 수집 완료 후 |
| Universe 생성 | 16:35 | 품질 통과 후 |

```go
// internal/s0_data/scheduler/scheduler.go

type DailySchedule struct {
    tasks []ScheduledTask
}

func DefaultDailySchedule() *DailySchedule {
    return &DailySchedule{
        tasks: []ScheduledTask{
            {Time: "08:00", Task: "stocks"},
            {Time: "15:45", Task: "prices"},
            {Time: "15:50", Task: "market_cap"},
            {Time: "16:05", Task: "investor"},
            {Time: "16:30", Task: "quality_check"},
            {Time: "16:35", Task: "universe_build"},
        },
    }
}
```

---

## DB 스키마

```sql
-- 데이터 스키마
CREATE SCHEMA IF NOT EXISTS data;

-- 가격 데이터
CREATE TABLE data.prices (
    stock_code  VARCHAR(10) NOT NULL,
    date        DATE NOT NULL,
    open        BIGINT,
    high        BIGINT,
    low         BIGINT,
    close       BIGINT NOT NULL,
    volume      BIGINT,
    value       BIGINT,
    adj_close   DECIMAL(15,2),
    adj_factor  DECIMAL(10,6) DEFAULT 1.0,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, date)
);

-- 투자자 수급
CREATE TABLE data.investor_flow (
    stock_code    VARCHAR(10) NOT NULL,
    date          DATE NOT NULL,
    foreign_net   BIGINT,      -- 외국인 순매수
    inst_net      BIGINT,      -- 기관 순매수
    individual_net BIGINT,     -- 개인 순매수
    financial_net BIGINT,      -- 금융투자
    insurance_net BIGINT,      -- 보험
    trust_net     BIGINT,      -- 투신
    pension_net   BIGINT,      -- 연기금
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, date)
);

-- 품질 스냅샷
CREATE TABLE data.quality_snapshots (
    id            SERIAL PRIMARY KEY,
    date          DATE NOT NULL UNIQUE,
    total_stocks  INT,
    valid_stocks  INT,
    coverage      JSONB,
    quality_score DECIMAL(5,4),
    passed_gate   BOOLEAN,
    issues        JSONB,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- 유니버스 스냅샷
CREATE TABLE data.universe_snapshots (
    id          SERIAL PRIMARY KEY,
    date        DATE NOT NULL UNIQUE,
    stocks      TEXT[],
    excluded    JSONB,
    total_count INT,
    config      JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 인덱스
CREATE INDEX idx_prices_date ON data.prices(date);
CREATE INDEX idx_investor_flow_date ON data.investor_flow(date);
```

---

**Prev**: [Folder Structure](./folder-structure.md)
**Next**: [Signals Layer](./signals-layer.md)
