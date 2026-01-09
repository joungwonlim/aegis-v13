# Database Schema Design

> PostgreSQL 스키마 설계

---

## 스키마 구조

레이어별로 스키마를 분리하여 관리:

```sql
CREATE SCHEMA data;       -- S0-S1: 원천 데이터, 유니버스
CREATE SCHEMA signals;    -- S2: 시그널
CREATE SCHEMA selection;  -- S3-S4: 스크리닝, 랭킹
CREATE SCHEMA portfolio;  -- S5: 포트폴리오
CREATE SCHEMA execution;  -- S6: 주문/체결
CREATE SCHEMA audit;      -- S7: 성과 분석
```

---

## 전체 ERD

```
┌─────────────────────────────────────────────────────────────────────┐
│                           data schema                               │
├─────────────────────────────────────────────────────────────────────┤
│  stocks ─────────┬───────────────────────────────────────────────── │
│    │             │                                                  │
│    ▼             ▼                                                  │
│  prices      fundamentals                                           │
│    │                                                                │
│    └──────────▶ quality_snapshots ──▶ universe_snapshots           │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         signals schema                              │
├─────────────────────────────────────────────────────────────────────┤
│  factor_scores ◀────────────────────────────────────────────────── │
│       │                                                             │
│       │         events                                              │
│       │           │                                                 │
│       └───────────┴───────▶ signal_snapshots                       │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        selection schema                             │
├─────────────────────────────────────────────────────────────────────┤
│  screened ─────────────────▶ ranked                                │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        portfolio schema                             │
├─────────────────────────────────────────────────────────────────────┤
│  targets ◀──────────────── holdings                                │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        execution schema                             │
├─────────────────────────────────────────────────────────────────────┤
│  orders ─────────────────▶ executions                              │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          audit schema                               │
├─────────────────────────────────────────────────────────────────────┤
│  daily_snapshots          performance_reports        attributions  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## data 스키마

### stocks (종목 마스터)

```sql
CREATE TABLE data.stocks (
    code            VARCHAR(10) PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    name_en         VARCHAR(100),
    market          VARCHAR(10),          -- KOSPI, KOSDAQ
    sector          VARCHAR(50),
    industry        VARCHAR(100),
    listing_date    DATE,
    fiscal_month    INT DEFAULT 12,       -- 결산월
    is_halted       BOOLEAN DEFAULT FALSE,
    is_admin        BOOLEAN DEFAULT FALSE,
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stocks_market ON data.stocks(market);
CREATE INDEX idx_stocks_sector ON data.stocks(sector);
```

### prices (일별 가격)

```sql
CREATE TABLE data.prices (
    id              SERIAL PRIMARY KEY,
    code            VARCHAR(10) NOT NULL REFERENCES data.stocks(code),
    date            DATE NOT NULL,
    open            INT,
    high            INT,
    low             INT,
    close           INT NOT NULL,
    volume          BIGINT,
    value           BIGINT,               -- 거래대금
    market_cap      BIGINT,               -- 시가총액
    change_rate     DECIMAL(8,4),         -- 등락률
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(code, date)
);

CREATE INDEX idx_prices_date ON data.prices(date);
CREATE INDEX idx_prices_code_date ON data.prices(code, date DESC);
```

### fundamentals (재무 데이터)

```sql
CREATE TABLE data.fundamentals (
    id              SERIAL PRIMARY KEY,
    code            VARCHAR(10) NOT NULL REFERENCES data.stocks(code),
    date            DATE NOT NULL,        -- 기준일
    fiscal_year     INT,
    fiscal_quarter  INT,                  -- 1, 2, 3, 4

    -- 가치 지표
    per             DECIMAL(10,2),
    pbr             DECIMAL(10,2),
    psr             DECIMAL(10,2),
    pcr             DECIMAL(10,2),
    ev_ebitda       DECIMAL(10,2),

    -- 수익성
    roe             DECIMAL(8,4),
    roa             DECIMAL(8,4),
    npm             DECIMAL(8,4),         -- 순이익률
    opm             DECIMAL(8,4),         -- 영업이익률

    -- 재무 안정성
    debt_ratio      DECIMAL(10,2),
    current_ratio   DECIMAL(10,2),

    -- 성장성
    revenue_growth  DECIMAL(8,4),
    profit_growth   DECIMAL(8,4),

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(code, date)
);

CREATE INDEX idx_fundamentals_code_date ON data.fundamentals(code, date DESC);
```

---

## signals 스키마

### factor_scores (팩터 점수)

```sql
CREATE TABLE signals.factor_scores (
    id              SERIAL PRIMARY KEY,
    date            DATE NOT NULL,
    code            VARCHAR(10) NOT NULL,

    -- 팩터별 점수 (-3 ~ +3 정규화)
    momentum        DECIMAL(8,4),
    value           DECIMAL(8,4),
    quality         DECIMAL(8,4),
    growth          DECIMAL(8,4),
    volatility      DECIMAL(8,4),
    liquidity       DECIMAL(8,4),

    -- 이벤트 점수
    event           DECIMAL(8,4),

    -- 기술적 점수
    technical       DECIMAL(8,4),

    -- 메타
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, code)
);

CREATE INDEX idx_factor_scores_date ON signals.factor_scores(date);
```

### events (이벤트 로그)

```sql
CREATE TABLE signals.events (
    id              SERIAL PRIMARY KEY,
    code            VARCHAR(10) NOT NULL,
    event_date      DATE NOT NULL,
    event_type      VARCHAR(50) NOT NULL,

    -- 이벤트 정보
    title           TEXT,
    score           DECIMAL(5,2),         -- 이벤트 점수
    source          VARCHAR(30),          -- DART, NEWS, ETC

    -- 원본 데이터
    raw_data        JSONB,

    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_events_code_date ON signals.events(code, event_date DESC);
CREATE INDEX idx_events_type ON signals.events(event_type);
```

---

## selection 스키마

### ranked (랭킹 결과)

```sql
CREATE TABLE selection.ranked (
    id              SERIAL PRIMARY KEY,
    date            DATE NOT NULL,
    code            VARCHAR(10) NOT NULL,

    rank            INT NOT NULL,
    total_score     DECIMAL(8,4),

    -- 팩터별 기여도
    factors         JSONB,

    -- 스크리닝 통과 여부
    screened        BOOLEAN DEFAULT TRUE,

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, code)
);

CREATE INDEX idx_ranked_date_rank ON selection.ranked(date, rank);
```

---

## execution 스키마

### orders (주문)

```sql
CREATE TABLE execution.orders (
    id              SERIAL PRIMARY KEY,
    order_id        VARCHAR(50) UNIQUE,   -- 증권사 주문번호

    code            VARCHAR(10) NOT NULL,
    name            VARCHAR(100),
    side            VARCHAR(4) NOT NULL,  -- BUY, SELL

    quantity        INT NOT NULL,
    price           INT,                  -- 주문가 (0=시장가)
    order_type      VARCHAR(10),          -- LIMIT, MARKET

    -- 상태
    status          VARCHAR(20) DEFAULT 'PENDING',
    filled_qty      INT DEFAULT 0,
    filled_price    INT,

    -- Brain 점수 (의사결정 근거)
    brain_scores    JSONB,

    -- 메타
    reason          TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_orders_status ON execution.orders(status);
CREATE INDEX idx_orders_created ON execution.orders(created_at);
```

---

## audit 스키마

### daily_snapshots (일별 스냅샷)

```sql
CREATE TABLE audit.daily_snapshots (
    id              SERIAL PRIMARY KEY,
    date            DATE NOT NULL UNIQUE,

    -- 자산
    total_value     DECIMAL(15,2),
    cash            DECIMAL(15,2),
    stock_value     DECIMAL(15,2),

    -- 수익률
    daily_return    DECIMAL(8,6),
    cum_return      DECIMAL(8,6),

    -- 포지션
    positions       JSONB,

    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_snapshots_date ON audit.daily_snapshots(date);
```

---

## 네이밍 규칙

| 구분 | 규칙 | 예시 |
|------|------|------|
| 스키마 | 소문자, 단수 | `data`, `signals` |
| 테이블 | 소문자, 복수 | `stocks`, `orders` |
| 컬럼 | snake_case | `created_at`, `total_score` |
| 인덱스 | `idx_테이블_컬럼` | `idx_prices_code_date` |
| FK | `fk_테이블_참조테이블` | `fk_prices_stocks` |

---

**Prev**: [Frontend Folder Structure](../frontend/folder-structure.md)
**Next**: [Getting Started](../overview/getting-started.md)
