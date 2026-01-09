# Data Layer

> S0: Data Quality + S1: Universe

---

## 책임

| Stage | 역할 | 출력 |
|-------|------|------|
| S0 | 원천 데이터 수집, 품질 검증 | `DataQualitySnapshot` |
| S1 | 투자 가능 종목 필터링 | `Universe` |

---

## 폴더 구조

```
internal/data/
├── quality.go      # QualityGate 구현
├── universe.go     # UniverseBuilder 구현
└── repository.go   # DB 접근
```

---

## S0: Quality Gate

### 인터페이스

```go
type QualityGate interface {
    Check(ctx context.Context, date time.Time) (*DataQualitySnapshot, error)
}
```

### 구현

```go
// internal/data/quality.go

type qualityGate struct {
    db     *pgxpool.Pool
    logger *zerolog.Logger
}

func NewQualityGate(db *pgxpool.Pool, logger *zerolog.Logger) QualityGate {
    return &qualityGate{db: db, logger: logger}
}

func (g *qualityGate) Check(ctx context.Context, date time.Time) (*contracts.DataQualitySnapshot, error) {
    snapshot := &contracts.DataQualitySnapshot{
        Date:     date,
        Coverage: make(map[string]float64),
    }

    // 1. 전체 종목 수
    snapshot.TotalStocks = g.countTotalStocks(ctx, date)

    // 2. 데이터별 커버리지 체크
    snapshot.Coverage["price"] = g.checkPriceCoverage(ctx, date)
    snapshot.Coverage["volume"] = g.checkVolumeCoverage(ctx, date)
    snapshot.Coverage["fundamental"] = g.checkFundamentalCoverage(ctx, date)

    // 3. 품질 점수 계산
    snapshot.QualityScore = g.calculateQualityScore(snapshot.Coverage)
    snapshot.ValidStocks = int(float64(snapshot.TotalStocks) * snapshot.QualityScore)

    return snapshot, nil
}
```

### 품질 기준

| 데이터 | 필수 | 커버리지 목표 |
|--------|------|---------------|
| 가격 | ✅ | 100% |
| 거래량 | ✅ | 100% |
| 시가총액 | ✅ | 95%+ |
| 재무제표 | ⚠️ | 80%+ |
| 공시 | ⚠️ | 70%+ |

---

## S1: Universe Builder

### 인터페이스

```go
type UniverseBuilder interface {
    Build(ctx context.Context, snapshot *DataQualitySnapshot) (*Universe, error)
}
```

### 구현

```go
// internal/data/universe.go

type universeBuilder struct {
    db     *pgxpool.Pool
    config UniverseConfig
}

type UniverseConfig struct {
    MinMarketCap   int64   // 최소 시가총액 (억)
    MinVolume      int64   // 최소 거래대금 (백만)
    MinListingDays int     // 최소 상장일수
    ExcludeAdmin   bool    // 관리종목 제외
    ExcludeHalt    bool    // 거래정지 제외
}

func (b *universeBuilder) Build(ctx context.Context, snapshot *contracts.DataQualitySnapshot) (*contracts.Universe, error) {
    universe := &contracts.Universe{
        Date:     snapshot.Date,
        Stocks:   make([]string, 0),
        Excluded: make(map[string]string),
    }

    // 모든 종목 조회
    allStocks := b.getAllStocks(ctx)

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

func (b *universeBuilder) checkExclusion(ctx context.Context, stock Stock) string {
    if b.config.ExcludeHalt && stock.IsHalted {
        return "거래정지"
    }
    if b.config.ExcludeAdmin && stock.IsAdmin {
        return "관리종목"
    }
    if stock.MarketCap < b.config.MinMarketCap {
        return "시가총액 미달"
    }
    if stock.AvgVolume < b.config.MinVolume {
        return "거래대금 미달"
    }
    if stock.ListingDays < b.config.MinListingDays {
        return "상장일수 미달"
    }
    return ""  // 통과
}
```

---

## 설정 예시 (YAML)

```yaml
# config/universe.yaml

universe:
  min_market_cap: 1000      # 1000억 이상
  min_volume: 500           # 5억 이상
  min_listing_days: 90      # 상장 90일 이상
  exclude_admin: true
  exclude_halt: true

  # 섹터 필터 (선택)
  exclude_sectors:
    - "금융"
    - "보험"
```

---

## DB 스키마

```sql
-- data.stocks: 종목 마스터
CREATE TABLE data.stocks (
    code         VARCHAR(10) PRIMARY KEY,
    name         VARCHAR(100) NOT NULL,
    market       VARCHAR(10),  -- KOSPI, KOSDAQ
    sector       VARCHAR(50),
    listing_date DATE,
    is_halted    BOOLEAN DEFAULT FALSE,
    is_admin     BOOLEAN DEFAULT FALSE,
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

-- data.quality_snapshots: 품질 스냅샷
CREATE TABLE data.quality_snapshots (
    id            SERIAL PRIMARY KEY,
    date          DATE NOT NULL,
    total_stocks  INT,
    valid_stocks  INT,
    coverage      JSONB,
    quality_score DECIMAL(5,4),
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- data.universe_snapshots: 유니버스 스냅샷
CREATE TABLE data.universe_snapshots (
    id          SERIAL PRIMARY KEY,
    date        DATE NOT NULL,
    stocks      TEXT[],           -- 종목 코드 배열
    excluded    JSONB,            -- 제외 사유
    total_count INT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

---

**Prev**: [Folder Structure](./folder-structure.md)
**Next**: [Signals Layer](./signals-layer.md)
