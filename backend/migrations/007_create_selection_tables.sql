-- =====================================================
-- 007_create_selection_tables.sql
-- Phase 4: selection 스키마 테이블 생성 (S3 + S4)
-- =====================================================

-- ============================================
-- 1. screening_results (S3: 스크리닝 결과)
-- ============================================
CREATE TABLE selection.screening_results (
    screen_date   DATE PRIMARY KEY,
    passed_stocks JSONB NOT NULL,           -- [code1, code2, ...]
    total_count   INT NOT NULL,
    criteria      JSONB,                     -- {min_score: 0.5, ...}
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE selection.screening_results IS 'S3: 스크리닝 통과 종목 (일별)';

-- ============================================
-- 2. ranking_results (S4: 랭킹 결과)
-- ============================================
CREATE TABLE selection.ranking_results (
    stock_code   VARCHAR(20) NOT NULL,
    rank_date    DATE NOT NULL,
    rank         INT NOT NULL,               -- 1-based ranking
    total_score  NUMERIC(5,4) NOT NULL,
    momentum     NUMERIC(5,4),
    technical    NUMERIC(5,4),
    value        NUMERIC(5,4),
    quality      NUMERIC(5,4),
    flow         NUMERIC(5,4),
    event        NUMERIC(5,4),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, rank_date)
);

CREATE INDEX idx_ranking_results_date ON selection.ranking_results(rank_date);
CREATE INDEX idx_ranking_results_rank ON selection.ranking_results(rank, rank_date);
CREATE INDEX idx_ranking_results_score ON selection.ranking_results(total_score DESC);

COMMENT ON TABLE selection.ranking_results IS 'S4: 종목별 랭킹 (일별)';

-- =====================================================
-- selection 스키마 테이블 생성 완료
-- 검증: SELECT table_name FROM information_schema.tables WHERE table_schema = 'selection' ORDER BY table_name;
-- =====================================================
