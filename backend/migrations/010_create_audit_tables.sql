-- =====================================================
-- 010_create_audit_tables.sql
-- Phase 4: audit 스키마 테이블 생성 (S7)
-- =====================================================

-- ============================================
-- 1. performance_reports (성과 보고서)
-- ============================================
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

COMMENT ON TABLE audit.performance_reports IS 'S7: 성과 분석 보고서 (주간/월간/연간)';

-- ============================================
-- 2. attribution_analysis (귀속 분석)
-- ============================================
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
    flow_contrib      NUMERIC(10,6),
    event_contrib     NUMERIC(10,6),
    -- 섹터별 기여도
    sector_contrib    JSONB,                  -- {sector1: 0.02, sector2: -0.01, ...}
    -- 종목별 기여도
    stock_contrib     JSONB,                  -- {code1: 0.05, code2: -0.02, ...}
    created_at        TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE audit.attribution_analysis IS '성과 귀속 분석 (팩터별/섹터별/종목별)';

-- ============================================
-- 3. benchmark_data (벤치마크 데이터)
-- ============================================
CREATE TABLE audit.benchmark_data (
    benchmark_date DATE NOT NULL,
    benchmark_code VARCHAR(20) NOT NULL,     -- KOSPI, KOSDAQ
    close_price    NUMERIC(12,2) NOT NULL,
    daily_return   NUMERIC(10,6),
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (benchmark_date, benchmark_code)
);

CREATE INDEX idx_benchmark_data_date ON audit.benchmark_data(benchmark_date);
CREATE INDEX idx_benchmark_data_code ON audit.benchmark_data(benchmark_code);

COMMENT ON TABLE audit.benchmark_data IS '벤치마크 데이터 (KOSPI, KOSDAQ)';

-- ============================================
-- 4. daily_pnl (일별 손익)
-- ============================================
CREATE TABLE audit.daily_pnl (
    pnl_date         DATE PRIMARY KEY,
    realized_pnl     BIGINT DEFAULT 0,       -- 실현 손익
    unrealized_pnl   BIGINT DEFAULT 0,       -- 미실현 손익
    total_pnl        BIGINT,
    daily_return     NUMERIC(10,6),
    cumulative_return NUMERIC(10,6),
    portfolio_value  BIGINT,
    cash_balance     BIGINT,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_daily_pnl_date ON audit.daily_pnl(pnl_date);

COMMENT ON TABLE audit.daily_pnl IS '일별 손익 기록';

-- =====================================================
-- audit 스키마 테이블 생성 완료
-- 검증: SELECT table_name FROM information_schema.tables WHERE table_schema = 'audit' ORDER BY table_name;
-- =====================================================
