# Database Migrations

Aegis v13 데이터베이스 마이그레이션 파일

---

## 실행 순서

```bash
# PostgreSQL 접속 (aegis_v13 사용자)
PGPASSWORD=aegis_v13_won psql -U aegis_v13 -d aegis_v13

# 또는 superuser로 초기 설정 시
# psql -U wonny -d aegis_v13

# Phase 1: 스키마 생성
\i 001_create_schemas.sql

# Phase 2: data 스키마
\i 002_create_data_tables.sql
\i 003_migrate_from_v10.sql

# Phase 3: signals 스키마
\i 004_create_signals_tables.sql
\i 005_calculate_flow_details.sql
\i 006_calculate_technical.sql

# Phase 4: 나머지 스키마
\i 007_create_selection_tables.sql
\i 008_create_portfolio_tables.sql
\i 009_create_execution_tables.sql
\i 010_create_audit_tables.sql

# Phase 5: 권한 설정 (superuser 권한 필요)
\i 011_grant_permissions.sql
\i 012_change_ownership.sql
```

---

## 파일 설명

| 파일 | 단계 | 설명 |
|------|------|------|
| 001_create_schemas.sql | Phase 1 | 6개 스키마 생성 |
| 002_create_data_tables.sql | Phase 2 | data 스키마 테이블 (8개) |
| 003_migrate_from_v10.sql | Phase 2 | v10 데이터 마이그레이션 |
| 004_create_signals_tables.sql | Phase 3 | signals 스키마 테이블 (4개) |
| 005_calculate_flow_details.sql | Phase 3 | 수급 누적 계산 (5D/10D/20D) |
| 006_calculate_technical.sql | Phase 3 | 기술적 지표 계산 (MA, RSI, MACD) |
| 007_create_selection_tables.sql | Phase 4 | selection 스키마 테이블 (2개) |
| 008_create_portfolio_tables.sql | Phase 4 | portfolio 스키마 테이블 (3개) |
| 009_create_execution_tables.sql | Phase 4 | execution 스키마 테이블 (3개) |
| 010_create_audit_tables.sql | Phase 4 | audit 스키마 테이블 (4개) |
| 011_grant_permissions.sql | Phase 5 | aegis_v13 사용자 권한 부여 |
| 012_change_ownership.sql | Phase 5 | 스키마/테이블 소유자 변경 |

---

## v10 마이그레이션 전 준비

003_migrate_from_v10.sql 실행 전 확인:

```sql
-- v10 데이터베이스 존재 확인
SELECT datname FROM pg_database WHERE datname = 'aegis_v10';

-- v10 데이터 개수 확인
SELECT
    (SELECT COUNT(*) FROM aegis_v10.market.stocks WHERE status = 'active') as stocks,
    (SELECT COUNT(*) FROM aegis_v10.market.daily_prices) as prices,
    (SELECT COUNT(*) FROM aegis_v10.market.investor_trading) as flow,
    (SELECT COUNT(*) FROM aegis_v10.analysis.fundamentals) as fundamentals;
```

---

## 검증 쿼리

### 전체 스키마 확인
```sql
SELECT schema_name
FROM information_schema.schemata
WHERE schema_name IN ('data', 'signals', 'selection', 'portfolio', 'execution', 'audit')
ORDER BY schema_name;
```

### 전체 테이블 개수
```sql
SELECT
    table_schema,
    COUNT(*) as table_count
FROM information_schema.tables
WHERE table_schema IN ('data', 'signals', 'selection', 'portfolio', 'execution', 'audit')
GROUP BY table_schema
ORDER BY table_schema;
```

### 데이터 개수 확인
```sql
SELECT
    (SELECT COUNT(*) FROM data.stocks) as stocks,
    (SELECT COUNT(*) FROM data.daily_prices) as prices,
    (SELECT COUNT(*) FROM data.investor_flow) as flow,
    (SELECT COUNT(*) FROM signals.flow_details) as flow_details,
    (SELECT COUNT(*) FROM signals.technical_details) as technical;
```

---

## 롤백

마이그레이션 롤백이 필요한 경우:

```sql
-- 전체 스키마 삭제 (주의!)
DROP SCHEMA IF EXISTS audit CASCADE;
DROP SCHEMA IF EXISTS execution CASCADE;
DROP SCHEMA IF EXISTS portfolio CASCADE;
DROP SCHEMA IF EXISTS selection CASCADE;
DROP SCHEMA IF EXISTS signals CASCADE;
DROP SCHEMA IF EXISTS data CASCADE;
```

---

## 참고

- 문서: `docs-site/docs/guide/database/schema-design.md`
- 실행 로그: 각 SQL 파일의 검증 쿼리 실행 결과 저장 권장
