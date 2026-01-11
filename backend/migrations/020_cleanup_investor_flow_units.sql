-- Migration: 020_cleanup_investor_flow_units.sql
-- Description: Remove investor flow data with wrong units (금액 대신 주식수 단위로 통일)
--
-- Problem: Before 2025-12-24, investor flow data was stored in KRW (금액)
--          After 2025-12-24, data is stored in share count (주식수)
-- Solution: Delete old data with wrong units to ensure consistency

-- Step 1: Check current data status (for debugging)
-- SELECT
--     trade_date,
--     COUNT(*),
--     AVG(ABS(foreign_net_value)) as avg_foreign_abs
-- FROM data.investor_flow
-- GROUP BY trade_date
-- ORDER BY trade_date DESC
-- LIMIT 30;

-- Step 2: Delete data before 2025-12-24 which has wrong units (금액)
DELETE FROM data.investor_flow
WHERE trade_date < '2025-12-24';

-- Step 3: Verify data is now consistent
-- SELECT
--     MIN(trade_date) as earliest_date,
--     MAX(trade_date) as latest_date,
--     COUNT(DISTINCT stock_code) as stock_count,
--     COUNT(*) as total_records
-- FROM data.investor_flow;
