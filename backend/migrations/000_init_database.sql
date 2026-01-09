-- =====================================================
-- 000_init_database.sql
-- 데이터베이스 초기 설정 (postgres DB에서 실행)
--
-- 실행:
-- psql -U wonny -d postgres -f 000_init_database.sql
-- =====================================================

-- 1. aegis_v13 역할 생성 (이미 있으면 무시)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'aegis_v13') THEN
        CREATE ROLE aegis_v13 WITH LOGIN PASSWORD 'aegis_v13_won' CREATEDB;
        RAISE NOTICE 'Role aegis_v13 created';
    ELSE
        RAISE NOTICE 'Role aegis_v13 already exists';
    END IF;
END $$;

-- 2. aegis_v13 데이터베이스 생성 (이미 있으면 무시)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'aegis_v13') THEN
        PERFORM dblink_exec('dbname=postgres', 'CREATE DATABASE aegis_v13 OWNER aegis_v13');
        RAISE NOTICE 'Database aegis_v13 created';
    ELSE
        RAISE NOTICE 'Database aegis_v13 already exists';
    END IF;
END $$;

-- 주의: 위 블록은 dblink extension이 필요합니다
-- 대신 아래 방법 사용:

-- 2-1. 수동으로 데이터베이스 생성 (이미 있으면 에러 무시)
-- CREATE DATABASE aegis_v13 OWNER aegis_v13;

-- 3. template1 접근 권한
GRANT CONNECT ON DATABASE template1 TO aegis_v13;

-- 4. 현재 권한 확인
\du aegis_v13
\l aegis_v13

-- =====================================================
-- 다음 단계:
-- psql -U aegis_v13 -d aegis_v13 -f 001_create_schemas.sql
-- =====================================================
