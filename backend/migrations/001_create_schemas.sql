-- =====================================================
-- 001_create_schemas.sql
-- Phase 1: 6개 스키마 생성 (7단계 파이프라인)
-- =====================================================

-- S0(Quality), S1(Universe) 데이터
CREATE SCHEMA IF NOT EXISTS data;
COMMENT ON SCHEMA data IS 'S0(Data Quality) + S1(Universe): 원천 데이터 및 투자 가능 종목';

-- S2(Signals) 시그널 데이터
CREATE SCHEMA IF NOT EXISTS signals;
COMMENT ON SCHEMA signals IS 'S2(Signals): 팩터 시그널 (momentum, technical, value, quality, flow, event)';

-- S3(Screener), S4(Ranking) 선택 로직
CREATE SCHEMA IF NOT EXISTS selection;
COMMENT ON SCHEMA selection IS 'S3(Screener) + S4(Ranking): 스크리닝 및 랭킹 결과';

-- S5(Portfolio) 포트폴리오 관리
CREATE SCHEMA IF NOT EXISTS portfolio;
COMMENT ON SCHEMA portfolio IS 'S5(Portfolio): 목표 포트폴리오 및 리밸런싱';

-- S6(Execution) 주문 실행
CREATE SCHEMA IF NOT EXISTS execution;
COMMENT ON SCHEMA execution IS 'S6(Execution): 주문 실행 및 거래 기록';

-- S7(Audit) 성과 분석
CREATE SCHEMA IF NOT EXISTS audit;
COMMENT ON SCHEMA audit IS 'S7(Audit): 성과 분석 및 귀속 분석';

-- =====================================================
-- 스키마 생성 완료
-- 검증: SELECT schema_name FROM information_schema.schemata WHERE schema_name IN ('data', 'signals', 'selection', 'portfolio', 'execution', 'audit');
-- =====================================================
