-- =====================================================
-- 022_create_exit_monitoring_table.sql
-- 자동청산 모니터링 설정 테이블
-- =====================================================

CREATE TABLE portfolio.exit_monitoring (
    id              SERIAL PRIMARY KEY,
    stock_code      VARCHAR(20) NOT NULL UNIQUE,
    enabled         BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_exit_monitoring_stock_code ON portfolio.exit_monitoring(stock_code);
CREATE INDEX idx_exit_monitoring_enabled ON portfolio.exit_monitoring(enabled);

COMMENT ON TABLE portfolio.exit_monitoring IS '자동청산 모니터링 설정';
COMMENT ON COLUMN portfolio.exit_monitoring.enabled IS 'true: 자동청산 모니터링 활성화';

-- =====================================================
-- exit_monitoring 테이블 생성 완료
-- =====================================================
