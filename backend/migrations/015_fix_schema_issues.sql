-- Migration: 015_fix_schema_issues
-- Description: Fix schema issues found during data collection
-- Date: 2026-01-10

-- 1. Add updated_at to market_cap
ALTER TABLE data.market_cap
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- 2. Create market_indicators table
CREATE TABLE IF NOT EXISTS data.market_indicators (
    id SERIAL PRIMARY KEY,
    indicator_date DATE NOT NULL,
    market_type VARCHAR(20) NOT NULL,  -- KOSPI, KOSDAQ
    indicator_name VARCHAR(50) NOT NULL,
    indicator_value NUMERIC(15,4),
    foreign_net_value BIGINT,
    inst_net_value BIGINT,
    indiv_net_value BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(indicator_date, market_type, indicator_name)
);

CREATE INDEX IF NOT EXISTS idx_market_indicators_date
ON data.market_indicators(indicator_date);

-- 3. Add unique constraint to disclosures for upsert
-- First, we need a unique constraint on (stock_code, disclosed_at, title)
CREATE UNIQUE INDEX IF NOT EXISTS idx_disclosures_unique
ON data.disclosures(stock_code, disclosed_at, title);

-- Grant permissions
GRANT ALL ON data.market_indicators TO aegis_v13;
GRANT USAGE, SELECT ON SEQUENCE data.market_indicators_id_seq TO aegis_v13;

-- Verify
DO $$
BEGIN
    RAISE NOTICE 'Migration 015: Schema issues fixed successfully';
END $$;
