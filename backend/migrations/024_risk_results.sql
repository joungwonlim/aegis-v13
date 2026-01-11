-- Migration: 024_risk_results
-- Description: Create risk analysis result tables (Monte Carlo, VaR)
-- Date: 2026-01-11

-- Monte Carlo 시뮬레이션 결과 테이블
-- ⭐ config에 재현성 정보 포함 (num_simulations, seed 등)
CREATE TABLE IF NOT EXISTS analytics.montecarlo_results (
    run_id VARCHAR(50) PRIMARY KEY,
    run_date DATE NOT NULL,
    config JSONB NOT NULL,            -- MonteCarloConfig 전체 저장 (재현성)
    mean_return NUMERIC(10,6),        -- 평균 수익률
    std_dev NUMERIC(10,6),            -- 표준편차
    var_95 NUMERIC(10,6),             -- 95% VaR (손실=양수)
    var_99 NUMERIC(10,6),             -- 99% VaR (손실=양수)
    cvar_95 NUMERIC(10,6),            -- 95% CVaR (Expected Shortfall)
    cvar_99 NUMERIC(10,6),            -- 99% CVaR
    percentiles JSONB,                -- {1: -0.05, 5: -0.03, ..., 99: 0.08}
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 일별 VaR 스냅샷 (시계열 추적용)
CREATE TABLE IF NOT EXISTS analytics.var_daily_snapshots (
    snapshot_date DATE NOT NULL,
    portfolio_id VARCHAR(50) NOT NULL DEFAULT 'main',
    var_95 NUMERIC(10,6),
    var_99 NUMERIC(10,6),
    cvar_95 NUMERIC(10,6),
    cvar_99 NUMERIC(10,6),
    portfolio_value NUMERIC(15,2),    -- 당일 포트폴리오 가치
    var_95_amount NUMERIC(15,2),      -- VaR95 금액 (가치 * VaR%)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (snapshot_date, portfolio_id)
);

-- 스트레스 테스트 결과
CREATE TABLE IF NOT EXISTS analytics.stress_test_results (
    run_id VARCHAR(50) NOT NULL,
    scenario_name VARCHAR(50) NOT NULL,
    scenario_description TEXT,
    portfolio_loss NUMERIC(10,6),     -- 시나리오별 예상 손실률
    loss_amount NUMERIC(15,2),        -- 손실 금액
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (run_id, scenario_name)
);

-- 인덱스 생성
CREATE INDEX IF NOT EXISTS idx_montecarlo_results_date
    ON analytics.montecarlo_results(run_date);

CREATE INDEX IF NOT EXISTS idx_var_daily_date
    ON analytics.var_daily_snapshots(snapshot_date);
CREATE INDEX IF NOT EXISTS idx_var_daily_portfolio
    ON analytics.var_daily_snapshots(portfolio_id);

CREATE INDEX IF NOT EXISTS idx_stress_test_run
    ON analytics.stress_test_results(run_id);

-- 코멘트 추가
COMMENT ON TABLE analytics.montecarlo_results IS 'Monte Carlo 시뮬레이션 결과';
COMMENT ON COLUMN analytics.montecarlo_results.config IS 'MonteCarloConfig JSON (num_simulations, holding_period, method, seed 등) - 재현성 보장';
COMMENT ON COLUMN analytics.montecarlo_results.var_95 IS '95% VaR - 손실을 양수로 표현 (0.05 = 5% 손실)';

COMMENT ON TABLE analytics.var_daily_snapshots IS '일별 VaR 추이 기록 (리스크 모니터링용)';
COMMENT ON COLUMN analytics.var_daily_snapshots.var_95_amount IS '포트폴리오 가치 × VaR95 = 최대 손실 금액';

COMMENT ON TABLE analytics.stress_test_results IS '스트레스 테스트 시나리오별 결과';
COMMENT ON COLUMN analytics.stress_test_results.scenario_name IS '시나리오 이름 (covid_crash, interest_rate_shock 등)';

-- 검증
DO $$
BEGIN
    RAISE NOTICE 'Migration 024: risk analysis tables created successfully';
END $$;
