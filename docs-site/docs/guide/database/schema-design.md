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
CREATE SCHEMA analytics;  -- 이벤트 예측 (Forecast)
```

---

## 전체 ERD

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            data schema (8 tables)                        │
├─────────────────────────────────────────────────────────────────────────┤
│  stocks ─────────┬───────────────┬───────────────┬─────────────────────│
│    │             │               │               │                      │
│    ▼             ▼               ▼               ▼                      │
│ daily_prices  market_cap   fundamentals   investor_flow (PARTITIONED)  │
│    │             │               │               │                      │
│    └─────────────┴───────────────┴───────────────┘                      │
│                              │                                          │
│              disclosures     ▼                                          │
│                   quality_snapshots ──────▶ universe_snapshots         │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          signals schema (4 tables)                       │
├─────────────────────────────────────────────────────────────────────────┤
│  technical_details ◀────┬────▶ flow_details                            │
│         │               │           │                                   │
│         ▼               ▼           ▼                                   │
│       factor_scores ◀──────── event_signals                            │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         selection schema (2 tables)                      │
├─────────────────────────────────────────────────────────────────────────┤
│  screening_results ──────────▶ ranking_results                          │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         portfolio schema (3 tables)                      │
├─────────────────────────────────────────────────────────────────────────┤
│  target_portfolios ◀──── portfolio_snapshots                            │
│         │                                                               │
│         └────────────────▶ rebalancing_history                          │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         execution schema (3 tables)                      │
├─────────────────────────────────────────────────────────────────────────┤
│  orders ──────────────────────▶ trades                                  │
│    │                                                                    │
│    └──────────────────────────▶ order_errors                            │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           audit schema (4 tables)                        │
├─────────────────────────────────────────────────────────────────────────┤
│  daily_pnl     performance_reports     attribution_analysis             │
│                         │                                               │
│                         └──────────▶ benchmark_data                     │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                        analytics schema (3 tables)                       │
├─────────────────────────────────────────────────────────────────────────┤
│  forecast_events ──────────▶ forward_performance                        │
│         │                                                               │
│         └──────────────────▶ forecast_stats                             │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## data 스키마

### stocks (종목 마스터)

```sql
CREATE TABLE data.stocks (
    code           VARCHAR(20) PRIMARY KEY,
    name           VARCHAR(200) NOT NULL,
    market         VARCHAR(20) NOT NULL,    -- KOSPI, KOSDAQ, KONEX
    sector         VARCHAR(100),
    listing_date   DATE NOT NULL,
    delisting_date DATE,
    status         VARCHAR(20) DEFAULT 'active',  -- active, delisted, suspended
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stocks_market ON data.stocks(market);
CREATE INDEX idx_stocks_sector ON data.stocks(sector);
CREATE INDEX idx_stocks_status ON data.stocks(status);
```

### daily_prices (일봉 데이터) - PARTITIONED

```sql
CREATE TABLE data.daily_prices (
    stock_code    VARCHAR(20) NOT NULL,
    trade_date    DATE NOT NULL,
    open_price    NUMERIC(12,2) NOT NULL,
    high_price    NUMERIC(12,2) NOT NULL,
    low_price     NUMERIC(12,2) NOT NULL,
    close_price   NUMERIC(12,2) NOT NULL,
    volume        BIGINT NOT NULL,
    trading_value NUMERIC(15,0),
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
) PARTITION BY RANGE (trade_date);

-- 파티션: 반기별 (2022-2027)
CREATE TABLE data.daily_prices_2022_h1 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2022-01-01') TO ('2022-07-01');
-- ... (2022_h2, 2023_h1, 2023_h2, ..., 2027_h1)

CREATE INDEX idx_daily_prices_date ON data.daily_prices(trade_date);
CREATE INDEX idx_daily_prices_stock ON data.daily_prices(stock_code);
```

### investor_flow (투자자별 수급) ⭐ - PARTITIONED

```sql
CREATE TABLE data.investor_flow (
    stock_code        VARCHAR(20) NOT NULL,
    trade_date        DATE NOT NULL,
    foreign_net_qty   BIGINT DEFAULT 0,     -- 외국인 순매수 수량
    foreign_net_value BIGINT DEFAULT 0,     -- 외국인 순매수 금액
    inst_net_qty      BIGINT DEFAULT 0,     -- 기관 순매수 수량
    inst_net_value    BIGINT DEFAULT 0,     -- 기관 순매수 금액
    indiv_net_qty     BIGINT DEFAULT 0,     -- 개인 순매수 수량
    indiv_net_value   BIGINT DEFAULT 0,     -- 개인 순매수 금액
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
) PARTITION BY RANGE (trade_date);

-- 파티션: 반기별 (daily_prices와 동일)

CREATE INDEX idx_investor_flow_date ON data.investor_flow(trade_date);
CREATE INDEX idx_investor_flow_stock ON data.investor_flow(stock_code);
```

### fundamentals (재무 데이터)

```sql
CREATE TABLE data.fundamentals (
    stock_code       VARCHAR(20) NOT NULL,
    year             INT NOT NULL,           -- 회계연도
    quarter          INT NOT NULL,           -- 분기 (1, 2, 3, 4)
    revenue          BIGINT,                 -- 매출액
    operating_profit BIGINT,                 -- 영업이익
    net_profit       BIGINT,                 -- 순이익
    assets           BIGINT,                 -- 자산총계
    equity           BIGINT,                 -- 자본총계
    debt             BIGINT,                 -- 부채총계
    roe              NUMERIC(10,2),          -- Return on Equity
    debt_ratio       NUMERIC(10,2),          -- 부채비율
    per              NUMERIC(10,2),          -- Price to Earnings Ratio
    pbr              NUMERIC(10,2),          -- Price to Book Ratio
    psr              NUMERIC(10,2),          -- Price to Sales Ratio
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, year, quarter)
);

CREATE INDEX idx_fundamentals_year_quarter ON data.fundamentals(year, quarter);
```

### market_cap (시가총액)

```sql
CREATE TABLE data.market_cap (
    stock_code         VARCHAR(20) NOT NULL,
    trade_date         DATE NOT NULL,
    market_cap         BIGINT NOT NULL,
    shares_outstanding BIGINT,
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
);

CREATE INDEX idx_market_cap_date ON data.market_cap(trade_date);
```

### disclosures (공시)

```sql
CREATE TABLE data.disclosures (
    id           SERIAL PRIMARY KEY,
    stock_code   VARCHAR(20) NOT NULL,
    disclosed_at TIMESTAMPTZ NOT NULL,
    title        TEXT NOT NULL,
    category     VARCHAR(100),
    content      TEXT,
    url          TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_disclosures_stock ON data.disclosures(stock_code);
CREATE INDEX idx_disclosures_date ON data.disclosures(disclosed_at);
```

### quality_snapshots (S0: 데이터 품질)

```sql
CREATE TABLE data.quality_snapshots (
    snapshot_date DATE PRIMARY KEY,
    total_stocks  INT NOT NULL,
    valid_stocks  INT NOT NULL,
    coverage      JSONB,                  -- {price: 0.95, flow: 0.90, ...}
    quality_score NUMERIC(5,4) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

### universe_snapshots (S1: 유니버스)

```sql
CREATE TABLE data.universe_snapshots (
    snapshot_date   DATE PRIMARY KEY,
    eligible_stocks JSONB NOT NULL,       -- [code1, code2, ...]
    total_count     INT NOT NULL,
    criteria        JSONB,                -- {min_market_cap: 100억, ...}
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

---

## signals 스키마

### factor_scores (팩터 점수)

```sql
CREATE TABLE signals.factor_scores (
    stock_code   VARCHAR(20) NOT NULL,
    calc_date    DATE NOT NULL,
    momentum     NUMERIC(5,4) DEFAULT 0.0,  -- 모멘텀 점수 (0~1)
    technical    NUMERIC(5,4) DEFAULT 0.0,  -- 기술적 점수 (0~1)
    value        NUMERIC(5,4) DEFAULT 0.0,  -- 가치 점수 (0~1)
    quality      NUMERIC(5,4) DEFAULT 0.0,  -- 퀄리티 점수 (0~1)
    flow         NUMERIC(5,4) DEFAULT 0.0,  -- 수급 점수 (0~1) ⭐
    event        NUMERIC(5,4) DEFAULT 0.0,  -- 이벤트 점수 (0~1)
    total_score  NUMERIC(5,4),              -- 종합 점수 (가중평균)
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_factor_scores_date ON signals.factor_scores(calc_date);
CREATE INDEX idx_factor_scores_total ON signals.factor_scores(total_score DESC);
```

### flow_details (수급 상세) ⭐

```sql
CREATE TABLE signals.flow_details (
    stock_code        VARCHAR(20) NOT NULL,
    calc_date         DATE NOT NULL,
    -- 5일 누적
    foreign_net_5d    BIGINT DEFAULT 0,
    inst_net_5d       BIGINT DEFAULT 0,
    indiv_net_5d      BIGINT DEFAULT 0,
    -- 10일 누적
    foreign_net_10d   BIGINT DEFAULT 0,
    inst_net_10d      BIGINT DEFAULT 0,
    indiv_net_10d     BIGINT DEFAULT 0,
    -- 20일 누적
    foreign_net_20d   BIGINT DEFAULT 0,
    inst_net_20d      BIGINT DEFAULT 0,
    indiv_net_20d     BIGINT DEFAULT 0,
    updated_at        TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_flow_details_date ON signals.flow_details(calc_date);
```

### technical_details (기술적 지표)

```sql
CREATE TABLE signals.technical_details (
    stock_code   VARCHAR(20) NOT NULL,
    calc_date    DATE NOT NULL,
    -- 이동평균
    ma5          NUMERIC(12,2),
    ma10         NUMERIC(12,2),
    ma20         NUMERIC(12,2),
    ma60         NUMERIC(12,2),
    ma120        NUMERIC(12,2),
    -- RSI
    rsi14        NUMERIC(5,2),
    -- MACD
    macd         NUMERIC(12,4),
    macd_signal  NUMERIC(12,4),
    macd_hist    NUMERIC(12,4),
    -- 볼린저 밴드
    bb_upper     NUMERIC(12,2),
    bb_middle    NUMERIC(12,2),
    bb_lower     NUMERIC(12,2),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_technical_details_date ON signals.technical_details(calc_date);
```

### event_signals (이벤트 시그널)

```sql
CREATE TABLE signals.event_signals (
    id            SERIAL PRIMARY KEY,
    stock_code    VARCHAR(20) NOT NULL,
    event_date    DATE NOT NULL,
    event_type    VARCHAR(50) NOT NULL,    -- disclosure, news, earning
    event_subtype VARCHAR(50),
    title         TEXT,
    description   TEXT,
    impact_score  NUMERIC(5,4) DEFAULT 0.0,  -- 영향도 점수 (0~1)
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_event_signals_stock ON signals.event_signals(stock_code);
CREATE INDEX idx_event_signals_date ON signals.event_signals(event_date);
CREATE INDEX idx_event_signals_type ON signals.event_signals(event_type);
```

---

## selection 스키마

### screening_results (S3: 스크리닝)

```sql
CREATE TABLE selection.screening_results (
    screen_date   DATE PRIMARY KEY,
    passed_stocks JSONB NOT NULL,           -- [code1, code2, ...]
    total_count   INT NOT NULL,
    criteria      JSONB,                     -- {min_score: 0.5, ...}
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

### ranking_results (S4: 랭킹)

```sql
CREATE TABLE selection.ranking_results (
    stock_code   VARCHAR(20) NOT NULL,
    rank_date    DATE NOT NULL,
    rank         INT NOT NULL,               -- 1-based ranking
    total_score  NUMERIC(5,4) NOT NULL,
    momentum     NUMERIC(5,4),
    technical    NUMERIC(5,4),
    value        NUMERIC(5,4),
    quality      NUMERIC(5,4),
    flow         NUMERIC(5,4),
    event        NUMERIC(5,4),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, rank_date)
);

CREATE INDEX idx_ranking_results_date ON selection.ranking_results(rank_date);
CREATE INDEX idx_ranking_results_rank ON selection.ranking_results(rank, rank_date);
CREATE INDEX idx_ranking_results_score ON selection.ranking_results(total_score DESC);
```

---

## portfolio 스키마

### target_portfolios (S5: 목표 포트폴리오)

```sql
CREATE TABLE portfolio.target_portfolios (
    portfolio_date DATE PRIMARY KEY,
    positions      JSONB NOT NULL,          -- [{code, name, weight, target_qty, action, reason}, ...]
    cash_weight    NUMERIC(5,4) DEFAULT 0.0,
    total_weight   NUMERIC(5,4),
    position_count INT,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);
```

### portfolio_snapshots (실제 포트폴리오)

```sql
CREATE TABLE portfolio.portfolio_snapshots (
    snapshot_date    DATE PRIMARY KEY,
    total_deposit    BIGINT NOT NULL,
    total_evaluation BIGINT NOT NULL,
    cash_balance     BIGINT NOT NULL,
    profit_loss      BIGINT,
    profit_loss_rate NUMERIC(10,6),
    positions        JSONB NOT NULL,        -- [{code, qty, avg_price, current_price, value}, ...]
    position_count   INT,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
```

### rebalancing_history (리밸런싱 기록)

```sql
CREATE TABLE portfolio.rebalancing_history (
    id              SERIAL PRIMARY KEY,
    rebalance_date  DATE NOT NULL,
    from_positions  JSONB,                  -- 변경 전 포지션
    to_positions    JSONB,                  -- 변경 후 목표 포지션
    required_orders JSONB,                  -- 필요한 주문들
    status          VARCHAR(20) DEFAULT 'pending',  -- pending, in_progress, completed, failed
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_rebalancing_history_date ON portfolio.rebalancing_history(rebalance_date);
CREATE INDEX idx_rebalancing_history_status ON portfolio.rebalancing_history(status);
```

---

## execution 스키마

### orders (주문)

```sql
CREATE TABLE execution.orders (
    id              SERIAL PRIMARY KEY,
    stock_code      VARCHAR(20) NOT NULL,
    stock_name      VARCHAR(200),
    order_date      DATE NOT NULL,
    order_time      TIMESTAMPTZ DEFAULT NOW(),
    order_action    VARCHAR(10) NOT NULL,     -- BUY, SELL
    order_type      VARCHAR(20) NOT NULL,     -- MARKET, LIMIT, STOP
    order_price     NUMERIC(12,2),
    order_qty       INT NOT NULL,
    filled_qty      INT DEFAULT 0,
    filled_price    NUMERIC(12,2),
    status          VARCHAR(20) DEFAULT 'pending',  -- pending, submitted, filled, partial, cancelled, rejected
    broker_order_no VARCHAR(50),
    reason          TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_orders_stock ON execution.orders(stock_code);
CREATE INDEX idx_orders_date ON execution.orders(order_date);
CREATE INDEX idx_orders_status ON execution.orders(status);
CREATE INDEX idx_orders_broker ON execution.orders(broker_order_no);
```

### trades (체결)

```sql
CREATE TABLE execution.trades (
    id              SERIAL PRIMARY KEY,
    order_id        INT NOT NULL REFERENCES execution.orders(id),
    stock_code      VARCHAR(20) NOT NULL,
    trade_date      DATE NOT NULL,
    trade_time      TIMESTAMPTZ NOT NULL,
    trade_action    VARCHAR(10) NOT NULL,     -- BUY, SELL
    trade_price     NUMERIC(12,2) NOT NULL,
    trade_qty       INT NOT NULL,
    trade_amount    BIGINT NOT NULL,
    commission      BIGINT DEFAULT 0,
    tax             BIGINT DEFAULT 0,
    broker_trade_no VARCHAR(50),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_trades_order ON execution.trades(order_id);
CREATE INDEX idx_trades_stock ON execution.trades(stock_code);
CREATE INDEX idx_trades_date ON execution.trades(trade_date);
```

### order_errors (주문 오류)

```sql
CREATE TABLE execution.order_errors (
    id              SERIAL PRIMARY KEY,
    order_id        INT REFERENCES execution.orders(id),
    stock_code      VARCHAR(20),
    error_time      TIMESTAMPTZ DEFAULT NOW(),
    error_code      VARCHAR(50),
    error_message   TEXT NOT NULL,
    broker_response JSONB,
    retry_count     INT DEFAULT 0,
    resolved        BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_order_errors_order ON execution.order_errors(order_id);
CREATE INDEX idx_order_errors_resolved ON execution.order_errors(resolved);
```

---

## audit 스키마

### performance_reports (S7: 성과 보고서)

```sql
CREATE TABLE audit.performance_reports (
    report_date       DATE PRIMARY KEY,
    period_start      DATE NOT NULL,
    period_end        DATE NOT NULL,
    total_return      NUMERIC(10,6),
    benchmark_return  NUMERIC(10,6),
    alpha             NUMERIC(10,6),
    beta              NUMERIC(10,6),
    sharpe_ratio      NUMERIC(10,6),
    volatility        NUMERIC(10,6),
    max_drawdown      NUMERIC(10,6),
    win_rate          NUMERIC(5,4),
    avg_win           NUMERIC(10,6),
    avg_loss          NUMERIC(10,6),
    profit_factor     NUMERIC(10,6),
    total_trades      INT,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);
```

### attribution_analysis (귀속 분석)

```sql
CREATE TABLE audit.attribution_analysis (
    analysis_date     DATE PRIMARY KEY,
    period_start      DATE NOT NULL,
    period_end        DATE NOT NULL,
    total_return      NUMERIC(10,6),
    -- 팩터별 기여도
    momentum_contrib  NUMERIC(10,6),
    technical_contrib NUMERIC(10,6),
    value_contrib     NUMERIC(10,6),
    quality_contrib   NUMERIC(10,6),
    flow_contrib      NUMERIC(10,6),        -- 수급 기여도 ⭐
    event_contrib     NUMERIC(10,6),
    -- 섹터별/종목별 기여도
    sector_contrib    JSONB,                  -- {sector1: 0.02, ...}
    stock_contrib     JSONB,                  -- {code1: 0.05, ...}
    created_at        TIMESTAMPTZ DEFAULT NOW()
);
```

### benchmark_data (벤치마크)

```sql
CREATE TABLE audit.benchmark_data (
    benchmark_date DATE NOT NULL,
    benchmark_code VARCHAR(20) NOT NULL,     -- KOSPI, KOSDAQ
    close_price    NUMERIC(12,2) NOT NULL,
    daily_return   NUMERIC(10,6),
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (benchmark_date, benchmark_code)
);

CREATE INDEX idx_benchmark_data_date ON audit.benchmark_data(benchmark_date);
```

### daily_pnl (일별 손익)

```sql
CREATE TABLE audit.daily_pnl (
    pnl_date          DATE PRIMARY KEY,
    realized_pnl      BIGINT DEFAULT 0,       -- 실현 손익
    unrealized_pnl    BIGINT DEFAULT 0,       -- 미실현 손익
    total_pnl         BIGINT,
    daily_return      NUMERIC(10,6),
    cumulative_return NUMERIC(10,6),
    portfolio_value   BIGINT,
    cash_balance      BIGINT,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);
```

---

## analytics 스키마

> 이벤트 기반 예측 시스템 (Forecast 모듈)

### forecast_events (이벤트 감지)

```sql
CREATE TABLE analytics.forecast_events (
    id                BIGSERIAL PRIMARY KEY,
    code              VARCHAR(20) NOT NULL,
    event_date        DATE NOT NULL,
    event_type        VARCHAR(20) NOT NULL,     -- E1_SURGE, E2_GAP_SURGE
    day_return        NUMERIC(8,4),             -- 당일 수익률
    close_to_high     NUMERIC(8,4),             -- 고가 대비 종가 (0~1)
    gap_ratio         NUMERIC(8,4),             -- 갭 비율
    volume_z_score    NUMERIC(8,2),             -- 거래량 z-score
    sector            VARCHAR(50),
    market_cap_bucket VARCHAR(10),              -- small/mid/large
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(code, event_date, event_type)
);

CREATE INDEX idx_forecast_events_code ON analytics.forecast_events(code);
CREATE INDEX idx_forecast_events_date ON analytics.forecast_events(event_date);
```

### forward_performance (전방 성과)

```sql
CREATE TABLE analytics.forward_performance (
    id              BIGSERIAL PRIMARY KEY,
    event_id        BIGINT REFERENCES analytics.forecast_events(id),
    fwd_ret_1d      NUMERIC(8,4),             -- t+1 수익률
    fwd_ret_2d      NUMERIC(8,4),
    fwd_ret_3d      NUMERIC(8,4),
    fwd_ret_5d      NUMERIC(8,4),             -- t+5 수익률
    max_runup_5d    NUMERIC(8,4),             -- 5일 최대 상승
    max_drawdown_5d NUMERIC(8,4),             -- 5일 최대 하락
    gap_hold_3d     BOOLEAN,                  -- 3일간 갭 유지
    filled_at       TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(event_id)
);
```

### forecast_stats (통계 집계)

```sql
CREATE TABLE analytics.forecast_stats (
    id            BIGSERIAL PRIMARY KEY,
    level         VARCHAR(10) NOT NULL,       -- SYMBOL/SECTOR/BUCKET/MARKET
    key           VARCHAR(50) NOT NULL,       -- 종목코드/섹터명/버킷명/ALL
    event_type    VARCHAR(20) NOT NULL,
    sample_count  INT,
    avg_ret_1d    NUMERIC(8,4),
    avg_ret_2d    NUMERIC(8,4),
    avg_ret_3d    NUMERIC(8,4),
    avg_ret_5d    NUMERIC(8,4),
    win_rate_1d   NUMERIC(5,4),               -- 1일 후 양수 비율
    win_rate_5d   NUMERIC(5,4),               -- 5일 후 양수 비율
    p10_mdd       NUMERIC(8,4),               -- 하위 10% MDD
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(level, key, event_type)
);

CREATE INDEX idx_forecast_stats_lookup ON analytics.forecast_stats(level, key, event_type);
```

**이벤트 타입**:
- `E1_SURGE`: 급등 (dayReturn ≥ 3.5% AND closeToHigh ≥ 0.4)
- `E2_GAP_SURGE`: 갭+급등 (E1 + gapRatio ≥ 1.5%)

**4단계 폴백 계층**:
1. `SYMBOL`: 해당 종목의 과거 이벤트 통계
2. `SECTOR`: 같은 섹터 종목들의 통계
3. `BUCKET`: 같은 시가총액 구간 (small/mid/large)
4. `MARKET`: 전체 시장 평균

---

## 데이터 요구사항 매핑

| 데이터 | 테이블 | 커버리지 |
|--------|--------|----------|
| 가격 (OHLCV) | `data.daily_prices` | 100% |
| 거래량 | `data.daily_prices.volume` | 100% |
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
- `market.daily_prices` → `data.daily_prices`
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
SELECT 'daily_prices', COUNT(*) FROM data.daily_prices
UNION ALL
SELECT 'investor_flow', COUNT(*) FROM data.investor_flow;

-- 예상 결과:
-- stocks: 2,835
-- daily_prices: 3,035,230
-- investor_flow: 2,438,932
```

---

## 향후 확장 계획

> 시스템 안정화 후 추가 예정인 기능들

### 1. investor_flow 상세 분류

현재 3개 그룹(외인/기관/개인)에서 9개 투자자 유형으로 확장:

```sql
-- 현재 (v13.0)
foreign_net_qty, foreign_net_value,
inst_net_qty, inst_net_value,
indiv_net_qty, indiv_net_value

-- 향후 확장 (v13.x)
foreign_net, foreign_qty,      -- 외국인
inst_net, inst_qty,            -- 기관 합계
individual_net, individual_qty, -- 개인
financial_net,                  -- 금융투자
insurance_net,                  -- 보험
trust_net,                      -- 투신
pension_net,                    -- 연기금
bank_net,                       -- 은행
other_inst_net                  -- 기타기관
```

**필요 조건**: 상세 수급 데이터 소스 확보 (KRX/증권사 API)

### 2. fundamentals 확장 지표

현재 7개 핵심 지표에서 20+ 지표로 확장:

```sql
-- 현재 (v13.0)
per, pbr, roe, debt_ratio, revenue, operating_profit, net_profit

-- 향후 확장 (v13.x)
-- 밸류에이션
psr,                    -- Price to Sales
pcr,                    -- Price to Cash Flow
ev_ebitda,              -- EV/EBITDA
div_yield,              -- 배당수익률

-- 주당 지표
eps,                    -- 주당순이익
bps,                    -- 주당순자산
dps,                    -- 주당배당금

-- 수익성
roa,                    -- 총자산이익률
npm,                    -- 순이익률
opm,                    -- 영업이익률

-- 안정성
current_ratio,          -- 유동비율

-- 성장성
revenue_growth,         -- 매출 성장률
profit_growth           -- 이익 성장률
```

**필요 조건**: DART API 연동 완료, 재무제표 파싱 로직 구현

### 3. 실시간 데이터 지원

```sql
-- 향후 추가 테이블
CREATE TABLE data.realtime_prices (
    stock_code    VARCHAR(20) NOT NULL,
    timestamp     TIMESTAMPTZ NOT NULL,
    price         NUMERIC(12,2),
    volume        BIGINT,
    bid_price     NUMERIC(12,2),
    ask_price     NUMERIC(12,2),
    PRIMARY KEY (stock_code, timestamp)
);

CREATE TABLE data.realtime_orderbook (
    stock_code    VARCHAR(20) NOT NULL,
    timestamp     TIMESTAMPTZ NOT NULL,
    bid_prices    NUMERIC(12,2)[],
    bid_volumes   BIGINT[],
    ask_prices    NUMERIC(12,2)[],
    ask_volumes   BIGINT[],
    PRIMARY KEY (stock_code, timestamp)
);
```

**필요 조건**: 실시간 시세 API 연동 (KIS WebSocket)

### 4. 백테스트 스키마

```sql
-- 향후 추가 스키마
CREATE SCHEMA backtest;

CREATE TABLE backtest.simulations (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(200),
    start_date      DATE,
    end_date        DATE,
    initial_capital BIGINT,
    strategy_params JSONB,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE backtest.simulation_trades (
    simulation_id   INT REFERENCES backtest.simulations(id),
    trade_date      DATE,
    stock_code      VARCHAR(20),
    action          VARCHAR(10),
    price           NUMERIC(12,2),
    qty             INT,
    PRIMARY KEY (simulation_id, trade_date, stock_code)
);

CREATE TABLE backtest.simulation_results (
    simulation_id   INT PRIMARY KEY REFERENCES backtest.simulations(id),
    total_return    NUMERIC(10,6),
    sharpe_ratio    NUMERIC(10,6),
    max_drawdown    NUMERIC(10,6),
    metrics         JSONB
);
```

**필요 조건**: 백테스트 엔진 구현

### 5. 네이버 랭킹 데이터 활용

실시간 시장 모니터링 및 시그널 보조 데이터로 활용:

```yaml
활용 가능한 랭킹:
  - 거래량 상위: /sise/lastsearch2.naver
  - 상승률 상위: /sise/sise_rise.naver
  - 외국인 순매수: /sise/sise_deal.naver
  - 기관 순매수: /sise/sise_deal.naver
  - 신고가: /sise/sise_high.naver
  - 저PER/PBR: /sise/field_submit.naver

활용 방안:
  - 실시간 급등주/거래량 상위 모니터링
  - 우리 시스템 랭킹과 크로스 체크 (검증용)
  - Event 시그널 강화 (신고가 돌파, 급등 탐지)

구현 위치: external/naver/ranking.go
```

**필요 조건**: Phase 1 데이터 레이어 안정화 후 검토

### 6. 시장 지수 데이터

시장 상황 판단 및 리스크 관리용:

```yaml
데이터:
  - 국내: KOSPI, KOSDAQ
  - 해외: NASDAQ, S&P500, VIX (변동성)

활용 방안:
  - 시장 레짐 판단 (상승장/하락장/횡보장)
  - 리스크 관리 (하락장 → 현금 비중 증가)
  - 상대 강도 비교 (종목 수익률 vs 시장 수익률)
  - 베타 계산

데이터 소스:
  - 국내: 네이버 금융, KRX
  - 해외: Yahoo Finance, investing.com
```

**필요 조건**: 시장 레짐 판단 로직 설계

### 7. 종목 뉴스 및 시장 심리

Event 시그널 강화 및 투자 타이밍 조절:

```yaml
종목 뉴스:
  - 네이버 뉴스 (종목별 크롤링)
  - DART 공시 (이미 있음)
  - 증권사 리포트 (선택)

활용 방안:
  - 호재/악재 감지 → Event 시그널 강화
  - 뉴스 심리 분석 (긍정/부정 키워드)
  - 실적 발표, M&A, 경영권 변동 탐지

시장 심리 지표:
  - 공포탐욕지수
  - 신용잔고
  - 투자자 예탁금

구현 위치:
  - external/naver/news.go
  - signals/event.go (강화)
  - signals/market_context.go (신규)
```

**필요 조건**: NLP 감성 분석 또는 키워드 기반 분류 로직

### 8. 나스닥 반영 2단계 의사결정

미국 시장 마감 결과를 반영한 최종 투자 결정:

```yaml
배경:
  - 나스닥은 한국 시장에 큰 영향을 미침
  - 현재 계획: 전일 17시 결정 → 나스닥 결과 반영 불가

2단계 의사결정 구조:
  1차 분석 (전일 17:00):
    - 한국 종목 분석
    - 후보군 선정
    - 예비 포트폴리오 생성

  2차 조정 (당일 08:00):
    - 나스닥 마감 결과 확인 (한국 06:00)
    - 시장 심리 조정
    - 최종 포트폴리오 확정

나스닥 영향 반영:
  strong_up: "+1.5% 이상"   # 공격적 매수
  up: "+0.5% ~ +1.5%"       # 정상 진행
  neutral: "-0.5% ~ +0.5%"  # 유지
  down: "-1.5% ~ -0.5%"     # 보수적 (현금 비중 ↑)
  strong_down: "-1.5% 이하" # 매수 보류/방어 모드

타임라인 변경:
  현재: "15:30 마감 → 17:10 주문생성 → 익일 09:00 실행"
  변경: "15:30 마감 → 17:00 1차분석 → 06:00 나스닥마감 → 08:00 2차조정 → 09:00 실행"

구현 위치:
  - external/yahoo/index.go (나스닥 데이터 수집)
  - brain/orchestrator.go (2단계 결정 로직)
  - config/market_sentiment.yaml (임계값 설정)
```

**필요 조건**: 해외 지수 데이터 수집 연동 (Yahoo Finance), 2단계 의사결정 로직 설계

### 9. 청산 로직 (Exit Strategy v1.1)

v10에서 검증된 3단 익절 + HWM 트레일링 전략:

```yaml
# 3단 익절 (분할 매도)
익절 (Take-Profit):
  TP1:
    트리거: +6.0%
    매도비중: 25%
    효과: 초기 수익 확보
  TP2:
    트리거: +10.0%
    매도비중: 25%
    효과: 추가 수익 실현
  TP3:
    트리거: +15.0%
    매도비중: 20%
    효과: 목표 수익 달성
  잔여:
    방식: HWM 트레일링
    매도비중: 30%
    효과: 추세 추종으로 수익 극대화

# 손절 (ATR 연동 동적 스탑)
손절매 (Stop-Loss):
  진입직후:
    기본: -5%
    대안: 1.8 × ATR (변동성 적응)
  TP1_이후:
    스탑: 진입가 + 0.6%
    효과: 본전 보호 (익절 후 손실 방지)
  TP2_이후:
    스탑: HWM - 1.2 × ATR
    효과: 최고가 기반 트레일링

# 보유기간
보유기간:
  최대: 7 영업일
  조기청산: 모멘텀 약화 시

# 장중 긴급 청산
긴급청산:
  - 거래정지 예고
  - 상한가/하한가 도달
  - 급격한 거래량 이상

# 핵심 개선점 (v1.0 → v1.1)
개선사항:
  - 분할횟수_축소: "무한 반복 → 3회 + 트레일 (비용 감소)"
  - ATR_연동: "고정 % → 종목별 변동성 적응"
  - 본전_보호: "TP1 이후 스탑 = 진입가 + 0.6%"
  - HWM_트레일링: "최고가 기반 추세 추종 (정상 눌림 견딤)"

구현 위치:
  - portfolio/exit_rules.go
  - execution/stop_loss.go
  - execution/trailing_stop.go
  - config/exit_strategy.yaml
```

**필요 조건**: ATR 계산 로직, HWM 추적 시스템, 실시간 모니터링

---

**Prev**: [Frontend Folder Structure](../frontend/folder-structure.md)
**Next**: [Getting Started](../overview/getting-started.md)
