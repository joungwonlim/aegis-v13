-- Migration: 017_create_target_positions
-- Description: Create portfolio.target_positions table for S5 portfolio construction
-- Date: 2026-01-10

CREATE TABLE IF NOT EXISTS portfolio.target_positions (
    id SERIAL PRIMARY KEY,
    target_date DATE NOT NULL,
    stock_code VARCHAR(20) NOT NULL,
    stock_name VARCHAR(100),
    weight NUMERIC(5,4) NOT NULL,  -- 비중 (0.0 ~ 1.0)
    target_qty INTEGER NOT NULL,    -- 목표 수량
    action VARCHAR(20) NOT NULL,    -- BUY, SELL, HOLD
    reason TEXT,                    -- 변경 사유
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_target_positions_date
ON portfolio.target_positions(target_date);

CREATE INDEX IF NOT EXISTS idx_target_positions_code
ON portfolio.target_positions(stock_code);

-- Grant permissions
GRANT ALL ON portfolio.target_positions TO aegis_v13;
GRANT USAGE, SELECT ON SEQUENCE portfolio.target_positions_id_seq TO aegis_v13;

-- Verify
DO $$
BEGIN
    RAISE NOTICE 'Migration 017: portfolio.target_positions created successfully';
END $$;
