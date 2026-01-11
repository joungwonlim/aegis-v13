-- =====================================================
-- 021_create_watchlist_table.sql
-- 관심종목 (watchlist) 테이블 생성
-- =====================================================

-- ============================================
-- 1. watchlist (관심종목)
-- ============================================
CREATE TABLE portfolio.watchlist (
    id              SERIAL PRIMARY KEY,
    stock_code      VARCHAR(20) NOT NULL,
    category        VARCHAR(20) NOT NULL DEFAULT 'watch',  -- 'watch' or 'candidate'
    alert_enabled   BOOLEAN DEFAULT FALSE,
    grok_analysis   TEXT,
    gemini_analysis TEXT,
    chatgpt_analysis TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT watchlist_category_check CHECK (category IN ('watch', 'candidate')),
    CONSTRAINT watchlist_stock_code_unique UNIQUE (stock_code)
);

-- 인덱스
CREATE INDEX idx_watchlist_category ON portfolio.watchlist(category);
CREATE INDEX idx_watchlist_stock_code ON portfolio.watchlist(stock_code);

COMMENT ON TABLE portfolio.watchlist IS '관심종목 목록';
COMMENT ON COLUMN portfolio.watchlist.category IS 'watch: 관심종목, candidate: 후보종목';

-- =====================================================
-- watchlist 테이블 생성 완료
-- 검증: SELECT * FROM portfolio.watchlist;
-- =====================================================
