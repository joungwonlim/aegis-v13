-- =====================================================
-- 008_create_portfolio_tables.sql
-- Phase 4: portfolio 스키마 테이블 생성 (S5)
-- =====================================================

-- ============================================
-- 1. target_portfolios (목표 포트폴리오)
-- ============================================
CREATE TABLE portfolio.target_portfolios (
    portfolio_date DATE PRIMARY KEY,
    positions      JSONB NOT NULL,          -- [{code, name, weight, target_qty, action, reason}, ...]
    cash_weight    NUMERIC(5,4) DEFAULT 0.0,
    total_weight   NUMERIC(5,4),
    position_count INT,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE portfolio.target_portfolios IS 'S5: 목표 포트폴리오 (일별)';

-- ============================================
-- 2. portfolio_snapshots (실제 포트폴리오)
-- ============================================
CREATE TABLE portfolio.portfolio_snapshots (
    snapshot_date   DATE PRIMARY KEY,
    total_deposit   BIGINT NOT NULL,
    total_evaluation BIGINT NOT NULL,
    cash_balance    BIGINT NOT NULL,
    profit_loss     BIGINT,
    profit_loss_rate NUMERIC(10,6),
    positions       JSONB NOT NULL,         -- [{code, qty, avg_price, current_price, value}, ...]
    position_count  INT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE portfolio.portfolio_snapshots IS '실제 포트폴리오 스냅샷 (일별)';

-- ============================================
-- 3. rebalancing_history (리밸런싱 기록)
-- ============================================
CREATE TABLE portfolio.rebalancing_history (
    id              SERIAL PRIMARY KEY,
    rebalance_date  DATE NOT NULL,
    from_positions  JSONB,                  -- 변경 전 포지션
    to_positions    JSONB,                  -- 변경 후 목표 포지션
    required_orders JSONB,                  -- 필요한 주문들
    status          VARCHAR(20) DEFAULT 'pending',  -- pending, in_progress, completed, failed
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_rebalancing_history_date ON portfolio.rebalancing_history(rebalance_date);
CREATE INDEX idx_rebalancing_history_status ON portfolio.rebalancing_history(status);

COMMENT ON TABLE portfolio.rebalancing_history IS '리밸런싱 실행 기록';

-- =====================================================
-- portfolio 스키마 테이블 생성 완료
-- 검증: SELECT table_name FROM information_schema.tables WHERE table_schema = 'portfolio' ORDER BY table_name;
-- =====================================================
