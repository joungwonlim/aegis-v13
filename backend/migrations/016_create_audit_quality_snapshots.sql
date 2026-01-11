-- Migration: 016_create_audit_quality_snapshots
-- Description: Create audit.data_quality_snapshots table for S0 quality gate
-- Date: 2026-01-10

CREATE TABLE IF NOT EXISTS audit.data_quality_snapshots (
    snapshot_date DATE PRIMARY KEY,
    quality_score NUMERIC(5,4) NOT NULL,
    total_stocks INTEGER NOT NULL,
    valid_stocks INTEGER NOT NULL,
    price_coverage NUMERIC(5,4) NOT NULL DEFAULT 0,
    volume_coverage NUMERIC(5,4) NOT NULL DEFAULT 0,
    marketcap_coverage NUMERIC(5,4) NOT NULL DEFAULT 0,
    fundamentals_coverage NUMERIC(5,4) NOT NULL DEFAULT 0,
    investor_coverage NUMERIC(5,4) NOT NULL DEFAULT 0,
    passed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_quality_snapshots_date
ON audit.data_quality_snapshots(snapshot_date DESC);

-- Grant permissions
GRANT ALL ON audit.data_quality_snapshots TO aegis_v13;

-- Verify
DO $$
BEGIN
    RAISE NOTICE 'Migration 016: audit.data_quality_snapshots created successfully';
END $$;
