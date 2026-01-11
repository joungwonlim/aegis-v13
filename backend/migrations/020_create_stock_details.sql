-- ============================================
-- Stock Details Table
-- ============================================
-- Purpose: 종목 상세 설명 (기업개요) 저장
-- SSOT: 이 테이블은 StockDetailSheet 실행 시에만 조회됨
-- Performance: stocks 테이블과 분리하여 일반 쿼리 성능 보호
-- ============================================

-- UP Migration
CREATE TABLE IF NOT EXISTS data.stock_details (
    code         VARCHAR(20) PRIMARY KEY REFERENCES data.stocks(code) ON DELETE CASCADE,
    description  TEXT,                             -- 기업개요 (Naver Finance에서 수집)
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stock_details_updated ON data.stock_details(updated_at);

COMMENT ON TABLE data.stock_details IS '종목 상세 설명 (기업개요)';
COMMENT ON COLUMN data.stock_details.code IS '종목 코드 (FK to stocks)';
COMMENT ON COLUMN data.stock_details.description IS 'Naver Finance 기업개요';
COMMENT ON COLUMN data.stock_details.updated_at IS '최종 업데이트 시각';

-- Sample data (optional, for testing)
-- INSERT INTO data.stock_details (code, description) VALUES
-- ('005930', '삼성전자는 반도체, 스마트폰, 가전 등을 제조하는 글로벌 IT 기업입니다.');

-- DOWN Migration (commented out, uncomment when needed)
-- DROP TABLE IF EXISTS data.stock_details;
