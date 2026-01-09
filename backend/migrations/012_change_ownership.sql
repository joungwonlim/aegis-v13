-- =====================================================
-- 012_change_ownership.sql
-- 모든 스키마 및 테이블 소유자를 aegis_v13으로 변경
-- =====================================================

-- 1. 스키마 소유자 변경
ALTER SCHEMA data OWNER TO aegis_v13;
ALTER SCHEMA signals OWNER TO aegis_v13;
ALTER SCHEMA selection OWNER TO aegis_v13;
ALTER SCHEMA portfolio OWNER TO aegis_v13;
ALTER SCHEMA execution OWNER TO aegis_v13;
ALTER SCHEMA audit OWNER TO aegis_v13;

-- 2. data 스키마의 모든 테이블 소유자 변경
ALTER TABLE data.stocks OWNER TO aegis_v13;
ALTER TABLE data.daily_prices OWNER TO aegis_v13;
ALTER TABLE data.investor_flow OWNER TO aegis_v13;
ALTER TABLE data.fundamentals OWNER TO aegis_v13;
ALTER TABLE data.market_cap OWNER TO aegis_v13;
ALTER TABLE data.disclosures OWNER TO aegis_v13;
ALTER TABLE data.quality_snapshots OWNER TO aegis_v13;
ALTER TABLE data.universe_snapshots OWNER TO aegis_v13;

-- 3. signals 스키마의 모든 테이블 소유자 변경
ALTER TABLE signals.factor_scores OWNER TO aegis_v13;
ALTER TABLE signals.flow_details OWNER TO aegis_v13;
ALTER TABLE signals.technical_details OWNER TO aegis_v13;
ALTER TABLE signals.event_signals OWNER TO aegis_v13;

-- 4. selection 스키마의 모든 테이블 소유자 변경
ALTER TABLE selection.screening_results OWNER TO aegis_v13;
ALTER TABLE selection.ranking_results OWNER TO aegis_v13;

-- 5. portfolio 스키마의 모든 테이블 소유자 변경
ALTER TABLE portfolio.target_portfolios OWNER TO aegis_v13;
ALTER TABLE portfolio.portfolio_snapshots OWNER TO aegis_v13;
ALTER TABLE portfolio.rebalancing_history OWNER TO aegis_v13;

-- 6. execution 스키마의 모든 테이블 소유자 변경
ALTER TABLE execution.orders OWNER TO aegis_v13;
ALTER TABLE execution.trades OWNER TO aegis_v13;
ALTER TABLE execution.order_errors OWNER TO aegis_v13;

-- 7. audit 스키마의 모든 테이블 소유자 변경
ALTER TABLE audit.performance_reports OWNER TO aegis_v13;
ALTER TABLE audit.attribution_analysis OWNER TO aegis_v13;
ALTER TABLE audit.benchmark_data OWNER TO aegis_v13;
ALTER TABLE audit.daily_pnl OWNER TO aegis_v13;

-- 8. 시퀀스 소유자 변경
ALTER SEQUENCE data.disclosures_id_seq OWNER TO aegis_v13;
ALTER SEQUENCE signals.event_signals_id_seq OWNER TO aegis_v13;
ALTER SEQUENCE execution.orders_id_seq OWNER TO aegis_v13;
ALTER SEQUENCE execution.trades_id_seq OWNER TO aegis_v13;
ALTER SEQUENCE execution.order_errors_id_seq OWNER TO aegis_v13;
ALTER SEQUENCE portfolio.rebalancing_history_id_seq OWNER TO aegis_v13;

-- =====================================================
-- 검증: 스키마 소유자 확인
-- SELECT schema_name, schema_owner FROM information_schema.schemata WHERE schema_name IN ('data', 'signals', 'selection', 'portfolio', 'execution', 'audit');
-- =====================================================
