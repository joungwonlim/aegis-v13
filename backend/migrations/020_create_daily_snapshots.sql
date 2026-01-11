-- 020_create_daily_snapshots.sql
-- audit.daily_snapshots 테이블 생성 (S7 Performance Analysis용)

CREATE TABLE IF NOT EXISTS audit.daily_snapshots (
    date           DATE PRIMARY KEY,
    total_value    NUMERIC(20,2) NOT NULL,
    cash           NUMERIC(20,2) NOT NULL,
    positions      JSONB NOT NULL DEFAULT '[]',
    daily_return   NUMERIC(10,6) DEFAULT 0,
    cum_return     NUMERIC(10,6) DEFAULT 0,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_daily_snapshots_date ON audit.daily_snapshots(date);

COMMENT ON TABLE audit.daily_snapshots IS '일별 포트폴리오 스냅샷 (성과 분석용)';
COMMENT ON COLUMN audit.daily_snapshots.date IS '스냅샷 날짜';
COMMENT ON COLUMN audit.daily_snapshots.total_value IS '총 포트폴리오 가치';
COMMENT ON COLUMN audit.daily_snapshots.cash IS '현금 잔고';
COMMENT ON COLUMN audit.daily_snapshots.positions IS '포지션 목록 (JSON)';
COMMENT ON COLUMN audit.daily_snapshots.daily_return IS '일일 수익률';
COMMENT ON COLUMN audit.daily_snapshots.cum_return IS '누적 수익률';
