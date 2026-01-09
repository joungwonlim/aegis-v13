-- =====================================================
-- 003_migrate_from_v10_fdw.sql
-- Phase 2: v10 데이터 마이그레이션 (postgres_fdw 사용)
--
-- 실행 전 조건:
-- 1. postgres_fdw 확장이 설치되어 있어야 함
-- 2. v10 데이터베이스가 같은 PostgreSQL 인스턴스에 있어야 함
-- =====================================================

-- 1. postgres_fdw 확장 설치
CREATE EXTENSION IF NOT EXISTS postgres_fdw;

-- 2. v10 데이터베이스로의 foreign server 생성
CREATE SERVER IF NOT EXISTS aegis_v10_server
    FOREIGN DATA WRAPPER postgres_fdw
    OPTIONS (host 'localhost', dbname 'aegis_v10', port '5432');

-- 3. user mapping 생성 (현재 사용자)
DO $$
BEGIN
    -- Check if user mapping exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_user_mappings
        WHERE srvname = 'aegis_v10_server'
        AND usename = current_user
    ) THEN
        EXECUTE format('CREATE USER MAPPING FOR %I SERVER aegis_v10_server OPTIONS (user %L)', current_user, current_user);
    END IF;
END $$;

-- 4. v10 테이블을 foreign table로 import
IMPORT FOREIGN SCHEMA market
    LIMIT TO (stocks, daily_prices, investor_trading)
    FROM SERVER aegis_v10_server
    INTO public;

IMPORT FOREIGN SCHEMA analysis
    LIMIT TO (fundamentals)
    FROM SERVER aegis_v10_server
    INTO public;

-- ============================================
-- 5. stocks 마이그레이션
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
    TRIM(stock_code),
    stock_name,
    market,
    sector,
    listed_date,
    delisted_date,
    CASE WHEN is_active THEN 'active' ELSE 'delisted' END,
    COALESCE(created_at, NOW()),
    COALESCE(updated_at, NOW())
FROM stocks
WHERE is_active = true
  AND listed_date IS NOT NULL  -- NOT NULL 제약 만족
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    market = EXCLUDED.market,
    sector = EXCLUDED.sector,
    updated_at = NOW();

-- 검증
DO $$
DECLARE
    row_count INT;
BEGIN
    SELECT COUNT(*) INTO row_count FROM data.stocks;
    RAISE NOTICE 'stocks 마이그레이션 완료: % rows', row_count;
END $$;

-- ============================================
-- 6. daily_prices 마이그레이션 (대용량, 시간 소요)
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
    TRIM(stock_code),
    trade_date,
    open_price,
    high_price,
    low_price,
    close_price,
    volume,
    value,
    COALESCE(created_at, NOW())
FROM daily_prices
WHERE trade_date >= '2022-01-01'
  AND open_price IS NOT NULL
  AND high_price IS NOT NULL
  AND low_price IS NOT NULL
  AND close_price IS NOT NULL
  AND volume IS NOT NULL  -- NOT NULL 제약 만족
ON CONFLICT (stock_code, trade_date) DO UPDATE SET
    open_price = EXCLUDED.open_price,
    high_price = EXCLUDED.high_price,
    low_price = EXCLUDED.low_price,
    close_price = EXCLUDED.close_price,
    volume = EXCLUDED.volume,
    trading_value = EXCLUDED.trading_value;

-- 검증
DO $$
DECLARE
    row_count INT;
BEGIN
    SELECT COUNT(*) INTO row_count FROM data.daily_prices;
    RAISE NOTICE 'daily_prices 마이그레이션 완료: % rows', row_count;
END $$;

-- ============================================
-- 7. investor_flow 마이그레이션 (대용량)
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
    TRIM(stock_code),
    trade_date,
    COALESCE(foreign_net_volume, 0),
    COALESCE(foreign_net_value, 0),
    COALESCE(inst_net_volume, 0),
    COALESCE(inst_net_value, 0),
    COALESCE(indiv_net_volume, 0),
    COALESCE(indiv_net_value, 0),
    NOW()
FROM investor_trading
WHERE trade_date >= '2022-01-01'
ON CONFLICT (stock_code, trade_date) DO UPDATE SET
    foreign_net_qty = EXCLUDED.foreign_net_qty,
    foreign_net_value = EXCLUDED.foreign_net_value,
    inst_net_qty = EXCLUDED.inst_net_qty,
    inst_net_value = EXCLUDED.inst_net_value,
    indiv_net_qty = EXCLUDED.indiv_net_qty,
    indiv_net_value = EXCLUDED.indiv_net_value;

-- 검증
DO $$
DECLARE
    row_count INT;
BEGIN
    SELECT COUNT(*) INTO row_count FROM data.investor_flow;
    RAISE NOTICE 'investor_flow 마이그레이션 완료: % rows', row_count;
END $$;

-- ============================================
-- 8. fundamentals 마이그레이션
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
    TRIM(stock_code),
    as_of_date,
    per,
    pbr,
    roe,
    debt_ratio,
    NULL,
    NULL,
    NULL,
    COALESCE(created_at, NOW())
FROM fundamentals
WHERE as_of_date >= '2022-01-01'
ON CONFLICT (stock_code, report_date) DO UPDATE SET
    per = EXCLUDED.per,
    pbr = EXCLUDED.pbr,
    roe = EXCLUDED.roe,
    debt_ratio = EXCLUDED.debt_ratio;

-- 검증
DO $$
DECLARE
    row_count INT;
BEGIN
    SELECT COUNT(*) INTO row_count FROM data.fundamentals;
    RAISE NOTICE 'fundamentals 마이그레이션 완료: % rows', row_count;
END $$;

-- ============================================
-- 9. 전체 검증
-- ============================================
SELECT
    (SELECT COUNT(*) FROM data.stocks) as stocks,
    (SELECT COUNT(*) FROM data.daily_prices) as prices,
    (SELECT COUNT(*) FROM data.investor_flow) as flow,
    (SELECT COUNT(*) FROM data.fundamentals) as fundamentals;

-- ============================================
-- 10. Foreign table 정리 (선택사항)
-- ============================================
-- DROP FOREIGN TABLE IF EXISTS stocks;
-- DROP FOREIGN TABLE IF EXISTS daily_prices;
-- DROP FOREIGN TABLE IF EXISTS investor_trading;
-- DROP FOREIGN TABLE IF EXISTS fundamentals;

-- =====================================================
