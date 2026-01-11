-- Migration: 023_forecast_validation
-- Description: Create forecast validation tables for prediction vs actual comparison
-- Date: 2026-01-11

-- Forecast 검증 결과 테이블
-- ⭐ PK: (event_id, model_version) - 동일 이벤트에 대해 다중 모델 비교 가능
CREATE TABLE IF NOT EXISTS analytics.forecast_validations (
    event_id BIGINT NOT NULL REFERENCES analytics.forecast_events(id) ON DELETE CASCADE,
    model_version VARCHAR(20) NOT NULL,
    code VARCHAR(20) NOT NULL,
    event_type VARCHAR(20) NOT NULL,
    predicted_ret NUMERIC(8,4),       -- 예측 수익률 (5일)
    actual_ret NUMERIC(8,4),          -- 실제 수익률 (5일)
    error NUMERIC(8,4),               -- 오차 (actual - predicted)
    abs_error NUMERIC(8,4),           -- 절대 오차
    direction_hit BOOLEAN,            -- 방향성 적중 (부호 일치)
    validated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (event_id, model_version)
);

-- 정확도 리포트 테이블 (집계 수준별)
CREATE TABLE IF NOT EXISTS analytics.accuracy_reports (
    model_version VARCHAR(20) NOT NULL,
    level VARCHAR(20) NOT NULL,       -- ALL, EVENT_TYPE, CODE, SECTOR
    key VARCHAR(50) NOT NULL,         -- level에 따른 키 값
    event_type VARCHAR(20) NOT NULL DEFAULT 'ALL',
    sample_count INT NOT NULL,
    mae NUMERIC(8,4),                 -- Mean Absolute Error
    rmse NUMERIC(8,4),                -- Root Mean Squared Error
    hit_rate NUMERIC(5,4),            -- 방향성 적중률 (0~1)
    mean_error NUMERIC(8,4),          -- 편향 (bias)
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (model_version, level, key, event_type)
);

-- 캘리브레이션 빈 테이블 (신뢰도 다이어그램용)
CREATE TABLE IF NOT EXISTS analytics.calibration_bins (
    model_version VARCHAR(20) NOT NULL,
    horizon_days INT NOT NULL,        -- 예측 기간 (5, 10, 20일)
    bin INT NOT NULL,                 -- 빈 번호 (0-9)
    sample_count INT NOT NULL,
    avg_predicted NUMERIC(8,4),       -- 빈 내 평균 예측값
    avg_actual NUMERIC(8,4),          -- 빈 내 평균 실제값
    hit_rate NUMERIC(5,4),            -- 빈 내 적중률
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (model_version, horizon_days, bin)
);

-- 인덱스 생성
CREATE INDEX IF NOT EXISTS idx_forecast_validations_code
    ON analytics.forecast_validations(code);
CREATE INDEX IF NOT EXISTS idx_forecast_validations_model
    ON analytics.forecast_validations(model_version);
CREATE INDEX IF NOT EXISTS idx_forecast_validations_event_type
    ON analytics.forecast_validations(event_type);
CREATE INDEX IF NOT EXISTS idx_forecast_validations_validated_at
    ON analytics.forecast_validations(validated_at);

CREATE INDEX IF NOT EXISTS idx_accuracy_reports_model
    ON analytics.accuracy_reports(model_version);
CREATE INDEX IF NOT EXISTS idx_accuracy_reports_level
    ON analytics.accuracy_reports(level);

-- 코멘트 추가
COMMENT ON TABLE analytics.forecast_validations IS 'Forecast 예측 vs 실제 검증 결과 (모델 버전별)';
COMMENT ON COLUMN analytics.forecast_validations.model_version IS '모델 버전 (A/B 테스트용)';
COMMENT ON COLUMN analytics.forecast_validations.direction_hit IS '예측과 실제 수익률 부호 일치 여부';

COMMENT ON TABLE analytics.accuracy_reports IS '집계 수준별 정확도 리포트';
COMMENT ON COLUMN analytics.accuracy_reports.level IS 'ALL=전체, EVENT_TYPE=이벤트유형별, CODE=종목별';
COMMENT ON COLUMN analytics.accuracy_reports.mean_error IS '양수=과소예측, 음수=과대예측 (bias)';

COMMENT ON TABLE analytics.calibration_bins IS '신뢰도 캘리브레이션 빈 (reliability diagram)';
COMMENT ON COLUMN analytics.calibration_bins.bin IS '예측 신뢰도 구간 (0=최저, 9=최고)';

-- 검증
DO $$
BEGIN
    RAISE NOTICE 'Migration 023: forecast_validations tables created successfully';
END $$;
