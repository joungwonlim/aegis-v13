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

## 구현 상태 (2026-01-10)

| 컴포넌트 | 상태 | 파일 |
|---------|------|------|
| **S0: Quality Gate** | ✅ 완료 | `internal/s0_data/quality/validator.go` |
| **S0: Repository** | ✅ 완료 | `internal/s0_data/repository.go` |
| **S0: Naver Client** | ✅ 완료 | `internal/external/naver/client.go` |
| **S0: DART Client** | ✅ 완료 | `internal/external/dart/client.go` |
| **S0: KRX Client** | ✅ 완료 | `internal/external/krx/client.go` |
| **S0: Data Collector** | ✅ 완료 | `internal/s0_data/collector/collector.go` |
| **S1: Universe Builder** | ✅ 완료 | `internal/s1_universe/builder.go` |
| **S1: Repository** | ✅ 완료 | `internal/s1_universe/repository.go` |
| Scheduler | ❌ TODO | `internal/s0_data/scheduler/` |

## 폴더 구조

```
internal/
├── s0_data/
│   ├── collector/
│   │   └── collector.go        # ✅ 데이터 수집 오케스트레이터
│   ├── quality/
│   │   ├── validator.go        # ✅ 데이터 검증
│   │   └── validator_test.go   # ✅ 테스트
│   └── repository.go           # ✅ DB 저장
│
├── s1_universe/
│   ├── builder.go              # ✅ Universe 생성
│   ├── builder_test.go         # ✅ 테스트
│   └── repository.go           # ✅ DB 저장 (universe_snapshots)
│
├── external/                   # ✅ 외부 API 클라이언트 (SSOT)
│   ├── naver/
│   │   ├── client.go           # ✅ HTTP 클라이언트
│   │   ├── prices.go           # ✅ 가격 데이터 수집
│   │   ├── prices_test.go      # ✅ 테스트
│   │   ├── investor.go         # ✅ 투자자 수급 수집
│   │   └── investor_test.go    # ✅ 테스트
│   ├── dart/
│   │   ├── client.go           # ✅ Legacy TLS 클라이언트
│   │   ├── client_test.go      # ✅ 테스트
│   │   └── disclosure.go       # ✅ 공시 데이터 수집
│   └── krx/
│       ├── client.go           # ✅ HTTP 클라이언트
│       ├── market_trend.go     # ✅ 시장 지표 수집
│       └── market_trend_test.go # ✅ 테스트
│
└── contracts/                  # ✅ 타입 정의 (SSOT)
    ├── data.go                 # DataQualitySnapshot
    └── universe.go             # Universe
```

**완료** (2026-01-10):
- ✅ DART/KRX 데이터를 Collector에 통합 완료
- ✅ Repository에 SaveDisclosures, SaveMarketTrend 메서드 추가

**TODO**:
- `scheduler/` (스케줄러 - 일정 관리)
- 시가총액 계산 로직 추가 (가격 × 발행주식수)
- 종목 마스터 데이터 수집 로직

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

### 데이터 수집 아키텍처

**구현 완료** (2026-01-10):

```
Collector (오케스트레이터)
    ↓
Naver Client (HTTP)
    ↓
Repository (DB 저장)
```

### Naver Client 구현

```go
// internal/external/naver/client.go

type Client struct {
    httpClient *httputil.Client  // SSOT: pkg/httputil 사용
    logger     *logger.Logger
    baseURL    string
}

// FetchPrices: JSON API 호출 + regex 파싱
func (c *Client) FetchPrices(ctx context.Context, stockCode string, from, to time.Time) ([]PriceData, error)

// FetchInvestorFlow: HTML 파싱 (goquery)
func (c *Client) FetchInvestorFlow(ctx context.Context, stockCode string, from, to time.Time) ([]InvestorFlowData, error)
```

**특징**:
- JSON + regex 듀얼 파싱 (안정성)
- HTML 파싱 with goquery (투자자 수급)
- 페이지네이션 지원 (최대 150 페이지)

### Data Collector 구현 ⭐ 업데이트

```go
// internal/s0_data/collector/collector.go

type Collector struct {
    naverClient *naver.Client
    dartClient  *dart.Client   // ✅ 추가
    krxClient   *krx.Client    // ✅ 추가
    repo        *s0_data.Repository
    logger      *logger.Logger
}

// Worker pool 패턴으로 병렬 수집
func (c *Collector) FetchAllPrices(ctx context.Context, from, to time.Time, cfg Config) ([]FetchResult, error)
func (c *Collector) FetchAllInvestorFlow(ctx context.Context, from, to time.Time, cfg Config) ([]FetchResult, error)

// 가격 + 수급 동시 수집
func (c *Collector) FetchAll(ctx context.Context, from, to time.Time, cfg Config) error

// ✅ 공시 데이터 수집 (DART)
func (c *Collector) FetchDisclosures(ctx context.Context, from, to time.Time) error

// ✅ 시장 지표 수집 (KRX)
func (c *Collector) FetchMarketTrends(ctx context.Context) error
```

**특징**:
- Worker pool으로 동시 처리
- 에러 허용 (일부 종목 실패 시 계속 진행)
- 진행 상태 로깅
- **다중 소스 통합**: Naver + DART + KRX

### DART Client 구현

```go
// internal/external/dart/client.go

type Client struct {
    httpClient *http.Client  // Legacy TLS 지원
    apiKey     string
    baseURL    string
    logger     *logger.Logger
}

// FetchDisclosures: 특정 기업의 공시 조회 (with retry)
func (c *Client) FetchDisclosures(ctx context.Context, corpCode string, from, to time.Time) ([]Disclosure, error)

// FetchDisclosuresForPage: 전체 기업 공시 페이지별 조회
func (c *Client) FetchDisclosuresForPage(ctx context.Context, from, to time.Time, page int) ([]Disclosure, int, error)
```

**특징**:
- **Legacy TLS 지원**: Go 1.22+에서 DART API 호환을 위한 RSA 키 교환 활성화
- **Exponential backoff retry**: 네트워크 오류 시 자동 재시도 (최대 3회)
- **Retryable error 판별**: EOF, timeout, connection reset 등 재시도 가능한 오류 구분

```go
// Legacy TLS Configuration (DART API 호환)
func newLegacyCompatibleClient(timeout time.Duration) *http.Client {
    tlsCfg := &tls.Config{
        MinVersion: tls.VersionTLS12,
        MaxVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_RSA_WITH_AES_128_GCM_SHA256,  // DART API 필수
            // ... 추가 cipher suites
        },
    }
}
```

### KRX Client 구현

```go
// internal/external/krx/client.go

type Client struct {
    httpClient *httputil.Client
    logger     *logger.Logger
    baseURL    string  // Naver API proxy
}

// FetchMarketTrend: 시장 전체 투자자 수급 데이터
func (c *Client) FetchMarketTrend(ctx context.Context, indexName string) (*MarketTrendData, error)

type MarketTrendData struct {
    TradeDate      time.Time
    ForeignNet     float64  // 외국인 순매수
    InstitutionNet float64  // 기관 순매수
    IndividualNet  float64  // 개인 순매수
}
```

**특징**:
- **Naver API 프록시 사용**: `https://m.stock.naver.com/api/index/{index}/trend`
- **복잡한 포맷 파싱**: "+1,459,781", "-1,240,182" 형식 처리
- **시장 지표 제공**: KOSPI, KOSDAQ별 투자자 수급

---

## S0: Quality Gate

### Contract (✅ 구현됨)

```go
// internal/contracts/data.go

type DataQualitySnapshot struct {
    Date         time.Time          `json:"date"`
    TotalStocks  int                `json:"total_stocks"`
    ValidStocks  int                `json:"valid_stocks"`
    Coverage     map[string]float64 `json:"coverage"`     // 실제 구현
    QualityScore float64            `json:"quality_score"`
}

// IsValid checks if the snapshot passed the quality gate
func (s *DataQualitySnapshot) IsValid() bool {
    return s.QualityScore >= 0.70 // 70% 이상
}

// Coverage keys:
// - "price": 가격 커버리지
// - "volume": 거래량 커버리지
// - "market_cap": 시가총액 커버리지
// - "fundamentals": 재무제표 커버리지
// - "investor": 수급 커버리지
```

### Quality Gate 구현 (✅ 구현됨)

```go
// internal/s0_data/quality/validator.go

type QualityGate struct {
    db     *pgxpool.Pool
    config Config
}

type Config struct {
    MinPriceCoverage      float64 `yaml:"min_price_coverage"`
    MinVolumeCoverage     float64 `yaml:"min_volume_coverage"`
    MinMarketCapCoverage  float64 `yaml:"min_market_cap_coverage"`
    MinFinancialCoverage  float64 `yaml:"min_financial_coverage"`
    MinInvestorCoverage   float64 `yaml:"min_investor_coverage"`
    MinDisclosureCoverage float64 `yaml:"min_disclosure_coverage"`
}

func NewQualityGate(db *pgxpool.Pool, config Config) *QualityGate {
    return &QualityGate{db: db, config: config}
}

// Check validates data quality for a given date
// ⭐ SSOT: S0 → S1 품질 검증
func (g *QualityGate) Check(ctx context.Context, date time.Time) (*contracts.DataQualitySnapshot, error) {
    snapshot := &contracts.DataQualitySnapshot{
        Date:     date,
        Coverage: make(map[string]float64),
    }

    // 1. 전체 종목 수
    totalStocks, err := g.countTotalStocks(ctx)
    if err != nil {
        return nil, fmt.Errorf("count total stocks: %w", err)
    }
    snapshot.TotalStocks = totalStocks

    // 2. 커버리지 체크
    coverage, err := g.checkCoverage(ctx, date)
    if err != nil {
        return nil, fmt.Errorf("check coverage: %w", err)
    }
    snapshot.Coverage = coverage

    // 3. 품질 점수 계산 (가중 평균)
    snapshot.QualityScore = g.calculateScore(coverage)
    snapshot.ValidStocks = int(float64(totalStocks) * snapshot.QualityScore)

    return snapshot, nil
}

// calculateScore calculates overall quality score using weighted average
func (g *QualityGate) calculateScore(coverage map[string]float64) float64 {
    weights := map[string]float64{
        "price":        0.30, // 가격 데이터 필수
        "volume":       0.30, // 거래량 데이터 필수
        "market_cap":   0.15, // 시가총액
        "fundamentals": 0.15, // 재무제표
        "investor":     0.10, // 수급
    }

    score := 0.0
    for key, weight := range weights {
        if cov, exists := coverage[key]; exists {
            score += cov * weight
        }
    }
    return score
}
```

**테스트 결과** (2026-01-08 기준):
- 전체 종목: 922개
- 유효 종목: 737개
- 품질 점수: **80.04%** ✅
- 커버리지:
  - Price: 98.81%
  - Volume: 95.55%
  - Fundamentals: 90.24%
  - Investor: 82.00%
  - Market Cap: 0.00% (데이터 누락)

---

## S1: Universe Builder

### Contract (✅ 구현됨)

```go
// internal/contracts/universe.go

type Universe struct {
    Date       time.Time         `json:"date"`
    Stocks     []string          `json:"stocks"`   // 포함 종목
    Excluded   map[string]string `json:"excluded"` // 제외 종목: 사유
    TotalCount int               `json:"total_count"`
}
```

### Config (✅ 구현됨)

```go
// internal/s1_universe/builder.go

type Config struct {
    MinMarketCap   int64    `yaml:"min_market_cap"`    // 최소 시총 (억)
    MinVolume      int64    `yaml:"min_volume"`        // 최소 거래대금 (백만)
    MinListingDays int      `yaml:"min_listing_days"`  // 최소 상장일수
    ExcludeAdmin   bool     `yaml:"exclude_admin"`     // 관리종목 제외
    ExcludeHalt    bool     `yaml:"exclude_halt"`      // 거래정지 제외
    ExcludeSPAC    bool     `yaml:"exclude_spac"`      // SPAC 제외
    ExcludeSectors []string `yaml:"exclude_sectors"`   // 제외 섹터
}

type Stock struct {
    Code        string
    Name        string
    Market      string
    Sector      string
    ListingDate time.Time
    MarketCap   int64 // 시가총액 (원)
    AvgVolume   int64 // 평균 거래대금 (원)
    ListingDays int   // 상장일수
    IsAdmin     bool  // 관리종목 여부
    IsHalted    bool  // 거래정지 여부
    IsSPAC      bool  // SPAC 여부
}
```

### Universe Builder 구현 (✅ 구현됨)

```go
// internal/s1_universe/builder.go

type Builder struct {
    db     *pgxpool.Pool
    config Config
}

func NewBuilder(db *pgxpool.Pool, config Config) *Builder {
    return &Builder{db: db, config: config}
}

// Build constructs the investable universe based on quality snapshot
// ⭐ SSOT: S1 → S2 유니버스 생성
func (b *Builder) Build(ctx context.Context, snapshot *contracts.DataQualitySnapshot) (*contracts.Universe, error) {
    // Quality gate 통과 확인
    if !snapshot.IsValid() {
        return nil, fmt.Errorf("data quality gate not passed: score=%.2f", snapshot.QualityScore)
    }

    universe := &contracts.Universe{
        Date:     snapshot.Date,
        Stocks:   make([]string, 0),
        Excluded: make(map[string]string),
    }

    // 전체 종목 조회
    stocks, err := b.getAllStocks(ctx, snapshot.Date)
    if err != nil {
        return nil, fmt.Errorf("get all stocks: %w", err)
    }

    // 필터링
    for _, stock := range stocks {
        reason := b.checkExclusion(stock)
        if reason != "" {
            universe.Excluded[stock.Code] = reason
            continue
        }
        universe.Stocks = append(universe.Stocks, stock.Code)
    }

    universe.TotalCount = len(universe.Stocks)
    return universe, nil
}

// checkExclusion checks if a stock should be excluded and returns the reason
func (b *Builder) checkExclusion(stock Stock) string {
    // 우선순위 순서로 체크

    // 1. 거래정지
    if b.config.ExcludeHalt && stock.IsHalted {
        return "거래정지"
    }

    // 2. 관리종목
    if b.config.ExcludeAdmin && stock.IsAdmin {
        return "관리종목"
    }

    // 3. SPAC
    if b.config.ExcludeSPAC && stock.IsSPAC {
        return "SPAC"
    }

    // 4. 시가총액 미달
    minMarketCap := b.config.MinMarketCap * 100_000_000 // 억 → 원
    if stock.MarketCap < minMarketCap {
        return fmt.Sprintf("시가총액 미달 (%d억)", stock.MarketCap/100_000_000)
    }

    // 5. 거래대금 미달
    minVolume := b.config.MinVolume * 1_000_000 // 백만 → 원
    if stock.AvgVolume < minVolume {
        return fmt.Sprintf("거래대금 미달 (%d백만)", stock.AvgVolume/1_000_000)
    }

    // 6. 상장일수 미달
    if stock.ListingDays < b.config.MinListingDays {
        return fmt.Sprintf("상장일수 미달 (%d일)", stock.ListingDays)
    }

    // 7. 제외 섹터
    for _, sector := range b.config.ExcludeSectors {
        if stock.Sector == sector {
            return fmt.Sprintf("제외 섹터 (%s)", sector)
        }
    }

    return "" // 통과
}
```

**테스트 결과** (2026-01-08 기준):
- 투자 가능 유니버스: **911종목** ✅
- 제외된 종목: 11개 (상장일수 30일 미만)
- 필터링 조건: 상장 30일 이상

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
