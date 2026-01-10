-- =====================================================
-- 013_create_decision_snapshots.sql
-- audit 스키마: 의사결정 스냅샷 테이블
-- =====================================================

-- 스키마 생성 (없으면)
CREATE SCHEMA IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.decision_snapshots (
    id              SERIAL PRIMARY KEY,
    config_hash     VARCHAR(64) NOT NULL,
    config_yaml     TEXT NOT NULL,
    strategy_id     VARCHAR(50) NOT NULL,
    git_commit      VARCHAR(40),
    data_snapshot_id VARCHAR(50),
    decision_date   DATE NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT uq_decision_snapshot UNIQUE (strategy_id, decision_date)
);

CREATE INDEX IF NOT EXISTS idx_decision_snapshots_hash ON audit.decision_snapshots(config_hash);
CREATE INDEX IF NOT EXISTS idx_decision_snapshots_date ON audit.decision_snapshots(decision_date);

COMMENT ON TABLE audit.decision_snapshots IS '의사결정 스냅샷 (설정 재현성 보장)';
COMMENT ON COLUMN audit.decision_snapshots.config_hash IS 'SHA256(canonical JSON)';
COMMENT ON COLUMN audit.decision_snapshots.config_yaml IS '원본 YAML 전문';

-- =====================================================
-- 검증: SELECT * FROM audit.decision_snapshots LIMIT 1;
-- =====================================================
