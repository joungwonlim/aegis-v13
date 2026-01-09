-- =====================================================
-- 003_migrate_from_v10.sql
-- Phase 2: v10 데이터 마이그레이션
--
-- 실행 전 조건:
-- 1. v10 데이터베이스 (aegis_v10)가 같은 PostgreSQL 인스턴스에 있어야 함
-- 2. 또는 postgres_fdw를 통해 접근 가능해야 함
--
-- 데이터 규모 (v10):
-- - 데이터 기간: 2022-01-03 ~ 2026-01-09 (983일)
-- - 종목 수: 2,835개
-- - 가격 레코드: 3,035,230 rows
-- - 수급 레코드: 2,438,932 rows
-- - 재무 레코드: 12,432 rows
-- =====================================================

-- ============================================
-- 1. stocks 마이그레이션 (1:1 매핑)
-- ============================================
INSERT INTO data.stocks (
    code,
    name,
    market,
    sector,
    listing_date,
    delisting_date,
    status,
    created_at,
    updated_at
)
SELECT
    code,
    name,
    market,
    sector,
    listing_date,
    delisting_date,
    COALESCE(status, 'active'),
    COALESCE(created_at, NOW()),
    COALESCE(updated_at, NOW())
FROM aegis_v10.market.stocks
WHERE status = 'active'  -- 활성 종목만
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    market = EXCLUDED.market,
    sector = EXCLUDED.sector,
    updated_at = NOW();

-- 검증: SELECT COUNT(*) FROM data.stocks;
-- 예상: ~2,835 rows

-- ============================================
-- 2. daily_prices 마이그레이션 (1:1 매핑)
-- ============================================
INSERT INTO data.daily_prices (
    stock_code,
    trade_date,
    open_price,
    high_price,
    low_price,
    close_price,
    volume,
    trading_value,
    created_at
)
SELECT
    stock_code,
    trade_date,
    open_price,
    high_price,
    low_price,
    close_price,
    volume,
    trading_value,
    COALESCE(created_at, NOW())
FROM aegis_v10.market.daily_prices
WHERE trade_date >= '2022-01-01'
ON CONFLICT (stock_code, trade_date) DO UPDATE SET
    open_price = EXCLUDED.open_price,
    high_price = EXCLUDED.high_price,
    low_price = EXCLUDED.low_price,
    close_price = EXCLUDED.close_price,
    volume = EXCLUDED.volume,
    trading_value = EXCLUDED.trading_value;

-- 검증: SELECT COUNT(*) FROM data.daily_prices;
-- 예상: ~3,035,230 rows

-- ============================================
-- 3. investor_flow 마이그레이션 (컬럼명 변경)
-- ============================================
INSERT INTO data.investor_flow (
    stock_code,
    trade_date,
    foreign_net_qty,
    foreign_net_value,
    inst_net_qty,
    inst_net_value,
    indiv_net_qty,
    indiv_net_value,
    created_at
)
SELECT
    stock_code,
    trade_date,
    COALESCE(foreign_net_qty, 0),
    COALESCE(foreign_net_value, 0),
    COALESCE(inst_net_qty, 0),
    COALESCE(inst_net_value, 0),
    COALESCE(indiv_net_qty, 0),
    COALESCE(indiv_net_value, 0),
    COALESCE(created_at, NOW())
FROM aegis_v10.market.investor_trading
WHERE trade_date >= '2022-01-01'
ON CONFLICT (stock_code, trade_date) DO UPDATE SET
    foreign_net_qty = EXCLUDED.foreign_net_qty,
    foreign_net_value = EXCLUDED.foreign_net_value,
    inst_net_qty = EXCLUDED.inst_net_qty,
    inst_net_value = EXCLUDED.inst_net_value,
    indiv_net_qty = EXCLUDED.indiv_net_qty,
    indiv_net_value = EXCLUDED.indiv_net_value;

-- 검증: SELECT COUNT(*) FROM data.investor_flow;
-- 예상: ~2,438,932 rows

-- ============================================
-- 4. fundamentals 마이그레이션 (1:1 매핑)
-- ============================================
INSERT INTO data.fundamentals (
    stock_code,
    report_date,
    per,
    pbr,
    roe,
    debt_ratio,
    revenue,
    operating_profit,
    net_profit,
    created_at
)
SELECT
    stock_code,
    report_date,
    per,
    pbr,
    roe,
    debt_ratio,
    revenue,
    operating_profit,
    net_profit,
    COALESCE(created_at, NOW())
FROM aegis_v10.analysis.fundamentals
WHERE report_date >= '2022-01-01'
ON CONFLICT (stock_code, report_date) DO UPDATE SET
    per = EXCLUDED.per,
    pbr = EXCLUDED.pbr,
    roe = EXCLUDED.roe,
    debt_ratio = EXCLUDED.debt_ratio,
    revenue = EXCLUDED.revenue,
    operating_profit = EXCLUDED.operating_profit,
    net_profit = EXCLUDED.net_profit;

-- 검증: SELECT COUNT(*) FROM data.fundamentals;
-- 예상: ~12,432 rows

-- =====================================================
-- 마이그레이션 완료
--
-- 전체 검증:
-- SELECT
--     (SELECT COUNT(*) FROM data.stocks) as stocks,
--     (SELECT COUNT(*) FROM data.daily_prices) as prices,
--     (SELECT COUNT(*) FROM data.investor_flow) as flow,
--     (SELECT COUNT(*) FROM data.fundamentals) as fundamentals;
--
-- 예상 결과:
-- stocks: ~2,835
-- prices: ~3,035,230
-- flow: ~2,438,932
-- fundamentals: ~12,432
-- =====================================================
