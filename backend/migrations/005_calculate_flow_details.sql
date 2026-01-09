-- =====================================================
-- 005_calculate_flow_details.sql
-- Phase 3: 수급 누적 계산 (5D/10D/20D)
--
-- data.investor_flow → signals.flow_details 변환
-- Window function을 사용하여 이동 합계 계산
-- =====================================================

INSERT INTO signals.flow_details (
    stock_code,
    calc_date,
    foreign_net_5d,
    inst_net_5d,
    indiv_net_5d,
    foreign_net_10d,
    inst_net_10d,
    indiv_net_10d,
    foreign_net_20d,
    inst_net_20d,
    indiv_net_20d,
    updated_at
)
SELECT
    stock_code,
    trade_date AS calc_date,
    -- 5일 누적 (현재 포함)
    SUM(foreign_net_value) OVER (
        PARTITION BY stock_code
        ORDER BY trade_date
        ROWS BETWEEN 4 PRECEDING AND CURRENT ROW
    ) AS foreign_net_5d,
    SUM(inst_net_value) OVER (
        PARTITION BY stock_code
        ORDER BY trade_date
        ROWS BETWEEN 4 PRECEDING AND CURRENT ROW
    ) AS inst_net_5d,
    SUM(indiv_net_value) OVER (
        PARTITION BY stock_code
        ORDER BY trade_date
        ROWS BETWEEN 4 PRECEDING AND CURRENT ROW
    ) AS indiv_net_5d,
    -- 10일 누적
    SUM(foreign_net_value) OVER (
        PARTITION BY stock_code
        ORDER BY trade_date
        ROWS BETWEEN 9 PRECEDING AND CURRENT ROW
    ) AS foreign_net_10d,
    SUM(inst_net_value) OVER (
        PARTITION BY stock_code
        ORDER BY trade_date
        ROWS BETWEEN 9 PRECEDING AND CURRENT ROW
    ) AS inst_net_10d,
    SUM(indiv_net_value) OVER (
        PARTITION BY stock_code
        ORDER BY trade_date
        ROWS BETWEEN 9 PRECEDING AND CURRENT ROW
    ) AS indiv_net_10d,
    -- 20일 누적
    SUM(foreign_net_value) OVER (
        PARTITION BY stock_code
        ORDER BY trade_date
        ROWS BETWEEN 19 PRECEDING AND CURRENT ROW
    ) AS foreign_net_20d,
    SUM(inst_net_value) OVER (
        PARTITION BY stock_code
        ORDER BY trade_date
        ROWS BETWEEN 19 PRECEDING AND CURRENT ROW
    ) AS inst_net_20d,
    SUM(indiv_net_value) OVER (
        PARTITION BY stock_code
        ORDER BY trade_date
        ROWS BETWEEN 19 PRECEDING AND CURRENT ROW
    ) AS indiv_net_20d,
    NOW() AS updated_at
FROM data.investor_flow
WHERE trade_date >= '2022-01-01'
ON CONFLICT (stock_code, calc_date) DO UPDATE SET
    foreign_net_5d = EXCLUDED.foreign_net_5d,
    inst_net_5d = EXCLUDED.inst_net_5d,
    indiv_net_5d = EXCLUDED.indiv_net_5d,
    foreign_net_10d = EXCLUDED.foreign_net_10d,
    inst_net_10d = EXCLUDED.inst_net_10d,
    indiv_net_10d = EXCLUDED.indiv_net_10d,
    foreign_net_20d = EXCLUDED.foreign_net_20d,
    inst_net_20d = EXCLUDED.inst_net_20d,
    indiv_net_20d = EXCLUDED.indiv_net_20d,
    updated_at = NOW();

-- =====================================================
-- 검증 쿼리
-- =====================================================

-- 1. 전체 레코드 수 확인
-- SELECT COUNT(*) FROM signals.flow_details;
-- 예상: ~2,438,932 rows (data.investor_flow와 동일)

-- 2. 샘플 데이터 확인 (특정 종목)
-- SELECT
--     stock_code,
--     calc_date,
--     foreign_net_5d,
--     foreign_net_10d,
--     foreign_net_20d
-- FROM signals.flow_details
-- WHERE stock_code = '005930'  -- 삼성전자
-- ORDER BY calc_date DESC
-- LIMIT 10;

-- 3. 20일 데이터가 충분한지 확인 (20일 이후부터 정상값)
-- SELECT
--     stock_code,
--     MIN(calc_date) as first_date,
--     COUNT(*) as total_days
-- FROM signals.flow_details
-- GROUP BY stock_code
-- ORDER BY total_days DESC
-- LIMIT 10;

-- =====================================================
