-- =====================================================
-- 002_create_data_tables.sql
-- Phase 2: data 스키마 테이블 생성
-- =====================================================

-- ============================================
-- 1. stocks (종목 마스터)
-- ============================================
CREATE TABLE data.stocks (
    code          VARCHAR(20) PRIMARY KEY,
    name          VARCHAR(200) NOT NULL,
    market        VARCHAR(20) NOT NULL,           -- KOSPI, KOSDAQ, KONEX
    sector        VARCHAR(100),
    listing_date  DATE NOT NULL,
    delisting_date DATE,
    status        VARCHAR(20) DEFAULT 'active',   -- active, delisted, suspended
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stocks_market ON data.stocks(market);
CREATE INDEX idx_stocks_sector ON data.stocks(sector);
CREATE INDEX idx_stocks_status ON data.stocks(status);

COMMENT ON TABLE data.stocks IS '종목 마스터 (SSOT)';
COMMENT ON COLUMN data.stocks.code IS '종목 코드 (6자리)';
COMMENT ON COLUMN data.stocks.market IS 'KOSPI, KOSDAQ, KONEX';

-- ============================================
-- 2. daily_prices (일봉 데이터) - PARTITIONED
-- ============================================
CREATE TABLE data.daily_prices (
    stock_code   VARCHAR(20) NOT NULL,
    trade_date   DATE NOT NULL,
    open_price   NUMERIC(12,2) NOT NULL,
    high_price   NUMERIC(12,2) NOT NULL,
    low_price    NUMERIC(12,2) NOT NULL,
    close_price  NUMERIC(12,2) NOT NULL,
    volume       BIGINT NOT NULL,
    trading_value NUMERIC(15,0),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
) PARTITION BY RANGE (trade_date);

-- 파티션 생성 (2022-2027, 반기별)
CREATE TABLE data.daily_prices_2022_h1 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2022-01-01') TO ('2022-07-01');
CREATE TABLE data.daily_prices_2022_h2 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2022-07-01') TO ('2023-01-01');
CREATE TABLE data.daily_prices_2023_h1 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2023-01-01') TO ('2023-07-01');
CREATE TABLE data.daily_prices_2023_h2 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2023-07-01') TO ('2024-01-01');
CREATE TABLE data.daily_prices_2024_h1 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2024-01-01') TO ('2024-07-01');
CREATE TABLE data.daily_prices_2024_h2 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2024-07-01') TO ('2025-01-01');
CREATE TABLE data.daily_prices_2025_h1 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2025-01-01') TO ('2025-07-01');
CREATE TABLE data.daily_prices_2025_h2 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2025-07-01') TO ('2026-01-01');
CREATE TABLE data.daily_prices_2026_h1 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2026-01-01') TO ('2026-07-01');
CREATE TABLE data.daily_prices_2026_h2 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2026-07-01') TO ('2027-01-01');
CREATE TABLE data.daily_prices_2027_h1 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2027-01-01') TO ('2027-07-01');

CREATE INDEX idx_daily_prices_date ON data.daily_prices(trade_date);
CREATE INDEX idx_daily_prices_stock ON data.daily_prices(stock_code);

COMMENT ON TABLE data.daily_prices IS '일봉 데이터 (파티션: 반기별)';

-- ============================================
-- 3. investor_flow (투자자별 수급) - PARTITIONED
-- ============================================
CREATE TABLE data.investor_flow (
    stock_code        VARCHAR(20) NOT NULL,
    trade_date        DATE NOT NULL,
    foreign_net_qty   BIGINT DEFAULT 0,
    foreign_net_value BIGINT DEFAULT 0,
    inst_net_qty      BIGINT DEFAULT 0,
    inst_net_value    BIGINT DEFAULT 0,
    indiv_net_qty     BIGINT DEFAULT 0,
    indiv_net_value   BIGINT DEFAULT 0,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
) PARTITION BY RANGE (trade_date);

-- 파티션 생성 (daily_prices와 동일)
CREATE TABLE data.investor_flow_2022_h1 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2022-01-01') TO ('2022-07-01');
CREATE TABLE data.investor_flow_2022_h2 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2022-07-01') TO ('2023-01-01');
CREATE TABLE data.investor_flow_2023_h1 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2023-01-01') TO ('2023-07-01');
CREATE TABLE data.investor_flow_2023_h2 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2023-07-01') TO ('2024-01-01');
CREATE TABLE data.investor_flow_2024_h1 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2024-01-01') TO ('2024-07-01');
CREATE TABLE data.investor_flow_2024_h2 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2024-07-01') TO ('2025-01-01');
CREATE TABLE data.investor_flow_2025_h1 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2025-01-01') TO ('2025-07-01');
CREATE TABLE data.investor_flow_2025_h2 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2025-07-01') TO ('2026-01-01');
CREATE TABLE data.investor_flow_2026_h1 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2026-01-01') TO ('2026-07-01');
CREATE TABLE data.investor_flow_2026_h2 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2026-07-01') TO ('2027-01-01');
CREATE TABLE data.investor_flow_2027_h1 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2027-01-01') TO ('2027-07-01');

CREATE INDEX idx_investor_flow_date ON data.investor_flow(trade_date);
CREATE INDEX idx_investor_flow_stock ON data.investor_flow(stock_code);

COMMENT ON TABLE data.investor_flow IS '투자자별 수급 (외국인, 기관, 개인)';

-- ============================================
-- 4. fundamentals (재무 데이터)
-- ============================================
CREATE TABLE data.fundamentals (
    stock_code    VARCHAR(20) NOT NULL,
    report_date   DATE NOT NULL,
    per           NUMERIC(10,2),
    pbr           NUMERIC(10,2),
    roe           NUMERIC(10,2),
    debt_ratio    NUMERIC(10,2),
    revenue       BIGINT,
    operating_profit BIGINT,
    net_profit    BIGINT,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, report_date)
);

CREATE INDEX idx_fundamentals_date ON data.fundamentals(report_date);

COMMENT ON TABLE data.fundamentals IS '재무 데이터 (분기 또는 연간)';

-- ============================================
-- 5. market_cap (시가총액)
-- ============================================
CREATE TABLE data.market_cap (
    stock_code    VARCHAR(20) NOT NULL,
    trade_date    DATE NOT NULL,
    market_cap    BIGINT NOT NULL,
    shares_outstanding BIGINT,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
);

CREATE INDEX idx_market_cap_date ON data.market_cap(trade_date);

COMMENT ON TABLE data.market_cap IS '시가총액 (일별)';

-- ============================================
-- 6. disclosures (공시)
-- ============================================
CREATE TABLE data.disclosures (
    id            SERIAL PRIMARY KEY,
    stock_code    VARCHAR(20) NOT NULL,
    disclosed_at  TIMESTAMPTZ NOT NULL,
    title         TEXT NOT NULL,
    category      VARCHAR(100),
    content       TEXT,
    url           TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_disclosures_stock ON data.disclosures(stock_code);
CREATE INDEX idx_disclosures_date ON data.disclosures(disclosed_at);

COMMENT ON TABLE data.disclosures IS 'DART 공시 데이터';

-- ============================================
-- 7. quality_snapshots (S0: 데이터 품질)
-- ============================================
CREATE TABLE data.quality_snapshots (
    snapshot_date DATE PRIMARY KEY,
    total_stocks  INT NOT NULL,
    valid_stocks  INT NOT NULL,
    coverage      JSONB,                  -- {price: 0.95, flow: 0.90, ...}
    quality_score NUMERIC(5,4) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE data.quality_snapshots IS 'S0: 데이터 품질 스냅샷 (일별)';

-- ============================================
-- 8. universe_snapshots (S1: 유니버스)
-- ============================================
CREATE TABLE data.universe_snapshots (
    snapshot_date DATE PRIMARY KEY,
    eligible_stocks JSONB NOT NULL,       -- [code1, code2, ...]
    total_count   INT NOT NULL,
    criteria      JSONB,                   -- {min_market_cap: 100억, ...}
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE data.universe_snapshots IS 'S1: 투자 가능 종목 유니버스 (일별)';

-- =====================================================
-- data 스키마 테이블 생성 완료
-- 검증: SELECT table_name FROM information_schema.tables WHERE table_schema = 'data' ORDER BY table_name;
-- =====================================================
