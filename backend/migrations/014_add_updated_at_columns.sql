-- Migration: 014_add_updated_at_columns
-- Description: Add updated_at columns to tables that need them
-- Date: 2026-01-10

-- Add updated_at to data.daily_prices
ALTER TABLE data.daily_prices
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Add updated_at to data.investor_flow
ALTER TABLE data.investor_flow
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Add updated_at to data.quality_snapshots if not exists
ALTER TABLE data.quality_snapshots
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Note: data.market_caps and data.financials tables will be added when needed

-- Verify
DO $$
BEGIN
    RAISE NOTICE 'Migration 014: updated_at columns added successfully';
END $$;
