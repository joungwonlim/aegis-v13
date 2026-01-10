-- Migration: 020_forecast_tables
-- Description: Create forecast event detection and prediction tables
-- Date: 2026-01-10

-- Create analytics schema if not exists
CREATE SCHEMA IF NOT EXISTS analytics;

-- 이벤트 저장 테이블
CREATE TABLE IF NOT EXISTS analytics.forecast_events (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(20) NOT NULL,
    event_date DATE NOT NULL,
    event_type VARCHAR(20) NOT NULL,
    day_return NUMERIC(8,4),
    close_to_high NUMERIC(8,4),
    gap_ratio NUMERIC(8,4),
    volume_z_score NUMERIC(8,2),
    sector VARCHAR(50),
    market_cap_bucket VARCHAR(10),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(code, event_date, event_type)
);

-- 전방 성과 테이블
CREATE TABLE IF NOT EXISTS analytics.forward_performance (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT REFERENCES analytics.forecast_events(id) ON DELETE CASCADE,
    fwd_ret_1d NUMERIC(8,4),
    fwd_ret_2d NUMERIC(8,4),
    fwd_ret_3d NUMERIC(8,4),
    fwd_ret_5d NUMERIC(8,4),
    max_runup_5d NUMERIC(8,4),
    max_drawdown_5d NUMERIC(8,4),
    gap_hold_3d BOOLEAN,
    filled_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(event_id)
);

-- 통계 집계 테이블
CREATE TABLE IF NOT EXISTS analytics.forecast_stats (
    id BIGSERIAL PRIMARY KEY,
    level VARCHAR(10) NOT NULL,  -- SYMBOL/SECTOR/BUCKET/MARKET
    key VARCHAR(50) NOT NULL,
    event_type VARCHAR(20) NOT NULL,
    sample_count INT,
    avg_ret_1d NUMERIC(8,4),
    avg_ret_2d NUMERIC(8,4),
    avg_ret_3d NUMERIC(8,4),
    avg_ret_5d NUMERIC(8,4),
    win_rate_1d NUMERIC(5,4),
    win_rate_5d NUMERIC(5,4),
    p10_mdd NUMERIC(8,4),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(level, key, event_type)
);

-- 인덱스 생성
CREATE INDEX IF NOT EXISTS idx_forecast_events_code ON analytics.forecast_events(code);
CREATE INDEX IF NOT EXISTS idx_forecast_events_date ON analytics.forecast_events(event_date);
CREATE INDEX IF NOT EXISTS idx_forecast_events_type ON analytics.forecast_events(event_type);
CREATE INDEX IF NOT EXISTS idx_forecast_events_sector ON analytics.forecast_events(sector);
CREATE INDEX IF NOT EXISTS idx_forecast_events_bucket ON analytics.forecast_events(market_cap_bucket);
CREATE INDEX IF NOT EXISTS idx_forecast_stats_lookup ON analytics.forecast_stats(level, key, event_type);

-- 코멘트 추가
COMMENT ON TABLE analytics.forecast_events IS '가격 패턴 기반 이벤트 (E1: 급등, E2: 갭+급등)';
COMMENT ON TABLE analytics.forward_performance IS '이벤트 발생 후 5거래일 전방 성과';
COMMENT ON TABLE analytics.forecast_stats IS '4단계 폴백 계층 통계 (SYMBOL/SECTOR/BUCKET/MARKET)';

COMMENT ON COLUMN analytics.forecast_events.day_return IS '당일 수익률';
COMMENT ON COLUMN analytics.forecast_events.close_to_high IS '고가 대비 종가 위치 (0~1)';
COMMENT ON COLUMN analytics.forecast_events.gap_ratio IS '갭 비율 (E2 이벤트용)';
COMMENT ON COLUMN analytics.forecast_events.volume_z_score IS '거래량 z-score (20일 기준)';
COMMENT ON COLUMN analytics.forecast_events.market_cap_bucket IS 'small/mid/large';

COMMENT ON COLUMN analytics.forward_performance.max_runup_5d IS '5일간 최대 상승';
COMMENT ON COLUMN analytics.forward_performance.max_drawdown_5d IS '5일간 최대 하락';
COMMENT ON COLUMN analytics.forward_performance.gap_hold_3d IS '3일간 갭 유지 여부';

COMMENT ON COLUMN analytics.forecast_stats.p10_mdd IS '하위 10% 최대 낙폭';

-- 검증
DO $$
BEGIN
    RAISE NOTICE 'Migration 020: forecast tables created successfully';
END $$;
