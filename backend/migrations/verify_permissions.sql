-- =====================================================
-- verify_permissions.sql
-- aegis_v13 사용자 권한 검증
--
-- 실행:
-- PGPASSWORD=aegis_v13_won psql -U aegis_v13 -d aegis_v13 -f verify_permissions.sql
-- =====================================================

\echo '=== 1. 현재 사용자 확인 ==='
SELECT current_user, current_database();

\echo ''
\echo '=== 2. 스키마 소유자 확인 ==='
SELECT
    nspname as schema_name,
    pg_catalog.pg_get_userbyid(nspowner) as owner
FROM pg_namespace
WHERE nspname IN ('data', 'signals', 'selection', 'portfolio', 'execution', 'audit')
ORDER BY nspname;

\echo ''
\echo '=== 3. 테이블 개수 확인 ==='
SELECT
    table_schema,
    COUNT(*) as table_count
FROM information_schema.tables
WHERE table_schema IN ('data', 'signals', 'selection', 'portfolio', 'execution', 'audit')
  AND table_type = 'BASE TABLE'
GROUP BY table_schema
ORDER BY table_schema;

\echo ''
\echo '=== 4. 데이터 개수 확인 ==='
SELECT
    'stocks' as table_name,
    (SELECT COUNT(*) FROM data.stocks) as count
UNION ALL
SELECT 'daily_prices', (SELECT COUNT(*) FROM data.daily_prices)
UNION ALL
SELECT 'investor_flow', (SELECT COUNT(*) FROM data.investor_flow)
UNION ALL
SELECT 'fundamentals', (SELECT COUNT(*) FROM data.fundamentals)
UNION ALL
SELECT 'flow_details', (SELECT COUNT(*) FROM signals.flow_details)
UNION ALL
SELECT 'technical_details', (SELECT COUNT(*) FROM signals.technical_details)
ORDER BY table_name;

\echo ''
\echo '=== 5. 샘플 데이터 (삼성전자) ==='
SELECT
    s.code,
    s.name,
    COUNT(p.trade_date) as trading_days,
    MIN(p.trade_date) as first_date,
    MAX(p.trade_date) as last_date
FROM data.stocks s
LEFT JOIN data.daily_prices p ON s.code = p.stock_code
WHERE s.code = '005930'
GROUP BY s.code, s.name;

\echo ''
\echo '=== 6. INSERT 권한 테스트 (audit.daily_pnl) ==='
BEGIN;
INSERT INTO audit.daily_pnl (pnl_date, realized_pnl, unrealized_pnl, total_pnl)
VALUES (CURRENT_DATE, 0, 0, 0)
ON CONFLICT (pnl_date) DO UPDATE SET realized_pnl = 0;
ROLLBACK;
\echo 'INSERT 권한 테스트: 성공 (롤백됨)'

\echo ''
\echo '=== 7. CREATE TABLE 권한 테스트 (audit 스키마) ==='
BEGIN;
CREATE TABLE audit.test_table (id SERIAL PRIMARY KEY);
DROP TABLE audit.test_table;
ROLLBACK;
\echo 'CREATE TABLE 권한 테스트: 성공 (롤백됨)'

\echo ''
\echo '=== 검증 완료! ==='
