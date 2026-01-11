-- Migration: 018_fix_portfolio_snapshots
-- Description: Add missing columns to portfolio_snapshots for S5 compatibility
-- Date: 2026-01-10

-- Add missing columns
ALTER TABLE portfolio.portfolio_snapshots
ADD COLUMN IF NOT EXISTS total_positions INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS total_weight NUMERIC(5,4) DEFAULT 0,
ADD COLUMN IF NOT EXISTS cash_reserve NUMERIC(5,4) DEFAULT 0,
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Verify
DO $$
BEGIN
    RAISE NOTICE 'Migration 018: portfolio_snapshots columns added successfully';
END $$;
