---
sidebar_position: 1
title: Schema Design
description: PostgreSQL 스키마 설계
---

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
┌─────────────────────────────────────────────────────────────────────────┐
│                            data schema                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  stocks ─────────┬───────────────┬───────────────┬─────────────────────│
│    │             │               │               │                      │
│    ▼             ▼               ▼               ▼                      │
│  prices      market_cap    fundamentals    investor_flow               │
│    │             │               │               │                      │
│    └─────────────┴───────────────┴───────────────┘                      │
│                              │                                          │
│                              ▼                                          │
│              quality_snapshots ──────▶ universe_snapshots              │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          signals schema                                  │
├─────────────────────────────────────────────────────────────────────────┤
│  technical_scores ◀────┬────▶ flow_scores                              │
│         │              │           │                                    │
│         ▼              ▼           ▼                                    │
│       factor_scores ◀──────── events                                   │
│              │                                                          │
│              └────────────────▶ signal_snapshots                       │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         selection schema                                 │
├─────────────────────────────────────────────────────────────────────────┤
│  screened ────────────────────▶ ranked                                  │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         portfolio schema                                 │
├─────────────────────────────────────────────────────────────────────────┤
│  targets ◀─────────────────── holdings                                  │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         execution schema                                 │
├─────────────────────────────────────────────────────────────────────────┤
│  orders ──────────────────────▶ executions                              │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           audit schema                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  daily_snapshots         performance_reports         attributions       │
└─────────────────────────────────────────────────────────────────────────┘
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
    is_spac         BOOLEAN DEFAULT FALSE,
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stocks_market ON data.stocks(market);
CREATE INDEX idx_stocks_sector ON data.stocks(sector);
```

### prices (일별 가격)

```sql
CREATE TABLE data.prices (
    stock_code      VARCHAR(10) NOT NULL REFERENCES data.stocks(code),
    date            DATE NOT NULL,
    open            BIGINT,
    high            BIGINT,
    low             BIGINT,
    close           BIGINT NOT NULL,
    volume          BIGINT,               -- 거래량
    value           BIGINT,               -- 거래대금
    adj_close       DECIMAL(15,2),        -- 수정종가
    adj_factor      DECIMAL(10,6) DEFAULT 1.0,
    change_rate     DECIMAL(8,4),         -- 등락률
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, date)
);

CREATE INDEX idx_prices_date ON data.prices(date);
CREATE INDEX idx_prices_code_date ON data.prices(stock_code, date DESC);
```

### market_cap (시가총액)

```sql
CREATE TABLE data.market_cap (
    stock_code          VARCHAR(10) NOT NULL REFERENCES data.stocks(code),
    date                DATE NOT NULL,
    market_cap          BIGINT NOT NULL,          -- 시가총액 (원)
    shares_outstanding  BIGINT,                   -- 상장주식수
    shares_float        BIGINT,                   -- 유동주식수
    float_ratio         DECIMAL(5,2),             -- 유동비율
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, date)
);

CREATE INDEX idx_market_cap_date ON data.market_cap(date);
```

### investor_flow (투자자별 매매동향) ⭐ 수급 데이터

```sql
CREATE TABLE data.investor_flow (
    stock_code      VARCHAR(10) NOT NULL REFERENCES data.stocks(code),
    date            DATE NOT NULL,

    -- 순매수 금액 (백만원)
    foreign_net     BIGINT,               -- 외국인 순매수
    inst_net        BIGINT,               -- 기관계 순매수
    individual_net  BIGINT,               -- 개인 순매수

    -- 기관 상세
    financial_net   BIGINT,               -- 금융투자
    insurance_net   BIGINT,               -- 보험
    trust_net       BIGINT,               -- 투신
    pension_net     BIGINT,               -- 연기금
    bank_net        BIGINT,               -- 은행
    other_inst_net  BIGINT,               -- 기타법인

    -- 순매수 수량 (주)
    foreign_qty     BIGINT,
    inst_qty        BIGINT,
    individual_qty  BIGINT,

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, date)
);

CREATE INDEX idx_investor_flow_date ON data.investor_flow(date);
CREATE INDEX idx_investor_flow_foreign ON data.investor_flow(date, foreign_net DESC);
```

### fundamentals (재무 데이터)

```sql
CREATE TABLE data.fundamentals (
    stock_code      VARCHAR(10) NOT NULL REFERENCES data.stocks(code),
    date            DATE NOT NULL,        -- 기준일
    fiscal_year     INT,
    fiscal_quarter  INT,                  -- 1, 2, 3, 4

    -- 밸류에이션 지표
    per             DECIMAL(10,2),
    pbr             DECIMAL(10,2),
    psr             DECIMAL(10,2),
    pcr             DECIMAL(10,2),
    ev_ebitda       DECIMAL(10,2),
    div_yield       DECIMAL(8,4),         -- 배당수익률

    -- 주당 지표
    eps             DECIMAL(15,2),        -- 주당순이익
    bps             DECIMAL(15,2),        -- 주당순자산
    dps             DECIMAL(15,2),        -- 주당배당금

    -- 수익성
    roe             DECIMAL(8,4),
    roa             DECIMAL(8,4),
    npm             DECIMAL(8,4),         -- 순이익률
    opm             DECIMAL(8,4),         -- 영업이익률

    -- 재무 안정성
    debt_ratio      DECIMAL(10,2),
    current_ratio   DECIMAL(10,2),

    -- 성장성
    revenue_growth  DECIMAL(8,4),         -- 매출성장률 YoY
    profit_growth   DECIMAL(8,4),         -- 이익성장률 YoY

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, date)
);

CREATE INDEX idx_fundamentals_code_date ON data.fundamentals(stock_code, date DESC);
```

### disclosures (공시)

```sql
CREATE TABLE data.disclosures (
    id              SERIAL PRIMARY KEY,
    stock_code      VARCHAR(10) NOT NULL REFERENCES data.stocks(code),
    disclosure_id   VARCHAR(50) UNIQUE,   -- DART 고유번호
    date            DATE NOT NULL,
    title           TEXT NOT NULL,
    type            VARCHAR(50),          -- earnings, material, stake, treasury
    importance      INT DEFAULT 2,        -- 1=높음, 2=중간, 3=낮음
    url             TEXT,
    parsed_data     JSONB,                -- 파싱된 주요 내용
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_disclosures_code_date ON data.disclosures(stock_code, date DESC);
CREATE INDEX idx_disclosures_type ON data.disclosures(type);
```

### quality_snapshots (품질 스냅샷)

```sql
CREATE TABLE data.quality_snapshots (
    id              SERIAL PRIMARY KEY,
    date            DATE NOT NULL UNIQUE,
    total_stocks    INT,
    valid_stocks    INT,
    coverage        JSONB,                -- 데이터별 커버리지
    quality_score   DECIMAL(5,4),
    passed_gate     BOOLEAN,
    issues          JSONB,                -- 품질 이슈 목록
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- coverage JSONB 예시:
-- {
--   "price": 1.0,
--   "volume": 1.0,
--   "market_cap": 0.97,
--   "financials": 0.85,
--   "investor": 0.82,
--   "disclosure": 0.75
-- }
```

### universe_snapshots (유니버스 스냅샷)

```sql
CREATE TABLE data.universe_snapshots (
    id              SERIAL PRIMARY KEY,
    date            DATE NOT NULL UNIQUE,
    stocks          TEXT[],               -- 투자 가능 종목 코드
    excluded        JSONB,                -- 제외 종목: 사유
    total_count     INT,
    config          JSONB,                -- 유니버스 설정
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

---

## signals 스키마

### factor_scores (통합 팩터 점수)

```sql
CREATE TABLE signals.factor_scores (
    stock_code      VARCHAR(10) NOT NULL,
    date            DATE NOT NULL,

    -- 팩터별 점수 (-1.0 ~ 1.0 정규화)
    momentum        DECIMAL(8,4),         -- 모멘텀
    technical       DECIMAL(8,4),         -- 기술적 지표
    value           DECIMAL(8,4),         -- 가치
    quality         DECIMAL(8,4),         -- 퀄리티
    growth          DECIMAL(8,4),         -- 성장
    flow            DECIMAL(8,4),         -- 수급 ⭐
    event           DECIMAL(8,4),         -- 이벤트

    -- 통합 점수
    total_score     DECIMAL(8,4),

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (date, stock_code)
);

-- PK가 (date, stock_code)이므로 date 단독 인덱스는 불필요
CREATE INDEX idx_factor_scores_total ON signals.factor_scores(date, total_score DESC);
```

### flow_details (수급 상세) ⭐

```sql
CREATE TABLE signals.flow_details (
    stock_code      VARCHAR(10) NOT NULL,
    date            DATE NOT NULL,

    -- 외국인 누적 순매수
    foreign_net_5d   BIGINT,              -- 5일 누적
    foreign_net_10d  BIGINT,              -- 10일 누적
    foreign_net_20d  BIGINT,              -- 20일 누적

    -- 기관 누적 순매수
    inst_net_5d      BIGINT,
    inst_net_10d     BIGINT,
    inst_net_20d     BIGINT,

    -- 연속 순매수일
    foreign_streak   INT,                 -- 외국인 연속 순매수 일수 (+/-로 방향)
    inst_streak      INT,                 -- 기관 연속 순매수 일수

    -- 시그널
    foreign_signal   DECIMAL(8,4),        -- -1.0 ~ 1.0
    inst_signal      DECIMAL(8,4),
    combined_signal  DECIMAL(8,4),        -- 통합 수급 시그널

    created_at       TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (date, stock_code)
);

-- PK가 (date, stock_code)이므로 date 단독 인덱스는 불필요
```

### technical_details (기술적 지표 상세)

```sql
CREATE TABLE signals.technical_details (
    stock_code      VARCHAR(10) NOT NULL,
    date            DATE NOT NULL,

    -- 이동평균
    ma5             DECIMAL(15,2),
    ma20            DECIMAL(15,2),
    ma60            DECIMAL(15,2),
    ma120           DECIMAL(15,2),

    -- 모멘텀
    rsi_14          DECIMAL(8,4),
    macd            DECIMAL(15,4),
    macd_signal     DECIMAL(15,4),
    macd_hist       DECIMAL(15,4),

    -- 볼린저밴드
    bb_upper        DECIMAL(15,2),
    bb_middle       DECIMAL(15,2),
    bb_lower        DECIMAL(15,2),

    -- 추세
    trend_5d        INT,                  -- 1=상승, 0=횡보, -1=하락
    trend_20d       INT,

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (date, stock_code)
);
```

### events (이벤트 로그)

```sql
CREATE TABLE signals.events (
    id              SERIAL PRIMARY KEY,
    stock_code      VARCHAR(10) NOT NULL,
    event_date      DATE NOT NULL,
    event_type      VARCHAR(50) NOT NULL, -- earnings, disclosure, stake, etc
    title           TEXT,
    score           DECIMAL(5,2),         -- 이벤트 점수
    source          VARCHAR(30),          -- DART, NEWS
    raw_data        JSONB,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_events_code_date ON signals.events(stock_code, event_date DESC);
CREATE INDEX idx_events_type ON signals.events(event_type);
```

---

## selection 스키마

### ranked (랭킹 결과)

```sql
CREATE TABLE selection.ranked (
    date            DATE NOT NULL,
    stock_code      VARCHAR(10) NOT NULL,
    rank            INT NOT NULL,
    total_score     DECIMAL(8,4),

    -- 팩터별 기여도
    scores          JSONB,
    -- {
    --   "momentum": 0.15,
    --   "technical": 0.08,
    --   "value": 0.12,
    --   "quality": 0.10,
    --   "flow": 0.18,
    --   "event": 0.02
    -- }

    -- 스크리닝 통과 여부
    screened        BOOLEAN DEFAULT TRUE,
    screen_reason   TEXT,                 -- 제외 사유 (if screened=false)

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (date, stock_code)
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

    stock_code      VARCHAR(10) NOT NULL,
    -- stock_name 제거: data.stocks와 JOIN으로 조회 (정규화)
    side            VARCHAR(4) NOT NULL,  -- BUY, SELL

    quantity        INT NOT NULL,
    price           INT,                  -- 주문가 (0=시장가)
    order_type      VARCHAR(10),          -- LIMIT, MARKET

    -- 상태
    status          VARCHAR(20) DEFAULT 'PENDING',
    filled_qty      INT DEFAULT 0,
    filled_price    INT,

    -- 의사결정 근거
    scores          JSONB,                -- Brain 점수
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
    benchmark_return DECIMAL(8,6),        -- KOSPI 대비

    -- 포지션
    positions       JSONB,

    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

### signal_attribution (시그널 기여도 분석)

```sql
CREATE TABLE audit.signal_attribution (
    id              SERIAL PRIMARY KEY,
    date            DATE NOT NULL,
    period          VARCHAR(20),          -- daily, weekly, monthly

    -- 시그널별 수익 기여도
    momentum_contrib   DECIMAL(8,6),
    technical_contrib  DECIMAL(8,6),
    value_contrib      DECIMAL(8,6),
    quality_contrib    DECIMAL(8,6),
    flow_contrib       DECIMAL(8,6),      -- 수급 기여도 ⭐
    event_contrib      DECIMAL(8,6),

    -- 메타
    details         JSONB,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_attribution_date ON audit.signal_attribution(date);
```

---

## 데이터 요구사항 매핑

| 데이터 | 테이블 | 커버리지 |
|--------|--------|----------|
| 가격 (OHLCV) | `data.prices` | 100% |
| 거래량 | `data.prices.volume` | 100% |
| 시가총액 | `data.market_cap` | 95%+ |
| 재무제표 | `data.fundamentals` | 80%+ |
| **투자자 수급** | `data.investor_flow` | **80%+** |
| 공시 | `data.disclosures` | 70%+ |

---

## 네이밍 규칙

| 구분 | 규칙 | 예시 |
|------|------|------|
| 스키마 | 소문자, 단수 | `data`, `signals` |
| 테이블 | 소문자, 스네이크 | `investor_flow`, `factor_scores` |
| 컬럼 | snake_case | `created_at`, `total_score` |
| 인덱스 | `idx_테이블_컬럼` | `idx_prices_code_date` |
| PK | `(stock_code, date)` 복합키 | 시계열 데이터 표준 |

---

## 마이그레이션 전략

### v10 데이터 현황

| 항목 | 값 |
|------|------|
| **데이터 기간** | 2022-01-03 ~ 2026-01-09 |
| **총 영업일** | 983일 (약 4년) |
| **종목 수** | 2,835개 (활성) |
| **가격 레코드** | 3,035,230 rows |
| **수급 레코드** | 2,438,932 rows |

### 마이그레이션 순서

#### Phase 1: 스키마 생성
```sql
-- 001_create_schemas.sql
CREATE SCHEMA data;
CREATE SCHEMA signals;
CREATE SCHEMA selection;
CREATE SCHEMA portfolio;
CREATE SCHEMA execution;
CREATE SCHEMA audit;
```

#### Phase 2: data 스키마 (v10 직접 이전)
```sql
-- 002_create_data_tables.sql
-- 테이블 생성 (stocks, prices, investor_flow, fundamentals 등)

-- 003_migrate_from_v10.sql
-- v10 데이터 이전 (INSERT INTO ... SELECT FROM ...)
```

**v10 → v13 매핑**:
- `market.stocks` → `data.stocks`
- `market.daily_prices` → `data.prices`
- `market.investor_trading` → `data.investor_flow` ⭐
- `analysis.fundamentals` → `data.fundamentals`

#### Phase 3: signals 스키마 (계산 필요)
```sql
-- 004_create_signals_tables.sql
-- 005_calculate_flow_details.sql  (investor_flow → 5D/20D 누적)
-- 006_calculate_technical.sql     (prices → MA, RSI, MACD)
```

#### Phase 4: 나머지 스키마
```sql
-- 007_create_selection_tables.sql
-- 008_create_portfolio_tables.sql
-- 009_create_execution_tables.sql
-- 010_create_audit_tables.sql
```

### 마이그레이션 실행

```bash
# 1. 스키마 생성
psql -U aegis_v13 -d aegis_v13 -f backend/migrations/001_create_schemas.sql

# 2. 테이블 생성
psql -U aegis_v13 -d aegis_v13 -f backend/migrations/002_create_data_tables.sql

# 3. 데이터 이전 (v10 → v13)
psql -U aegis_v13 -d aegis_v13 -f backend/migrations/003_migrate_from_v10.sql

# 4. 시그널 계산
psql -U aegis_v13 -d aegis_v13 -f backend/migrations/004_create_signals_tables.sql
psql -U aegis_v13 -d aegis_v13 -f backend/migrations/005_calculate_flow_details.sql

# 5. 나머지 스키마
psql -U aegis_v13 -d aegis_v13 -f backend/migrations/006_*.sql
```

### 검증

```sql
-- 레코드 수 확인
SELECT 'stocks' as table_name, COUNT(*) FROM data.stocks
UNION ALL
SELECT 'prices', COUNT(*) FROM data.prices
UNION ALL
SELECT 'investor_flow', COUNT(*) FROM data.investor_flow;

-- 예상 결과:
-- stocks: 2,835
-- prices: 3,035,230
-- investor_flow: 2,438,932
```

---

**Prev**: [Frontend Folder Structure](../frontend/folder-structure.md)
**Next**: [Getting Started](../overview/getting-started.md)
