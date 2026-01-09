-- =====================================================
-- 004_create_signals_tables.sql
-- Phase 3: signals 스키마 테이블 생성 (S2)
-- =====================================================

-- ============================================
-- 1. factor_scores (팩터 점수)
-- ============================================
CREATE TABLE signals.factor_scores (
    stock_code   VARCHAR(20) NOT NULL,
    calc_date    DATE NOT NULL,
    momentum     NUMERIC(5,4) NOT NULL DEFAULT 0.0,  -- 모멘텀 점수 (0~1)
    technical    NUMERIC(5,4) NOT NULL DEFAULT 0.0,  -- 기술적 점수 (0~1)
    value        NUMERIC(5,4) NOT NULL DEFAULT 0.0,  -- 가치 점수 (0~1)
    quality      NUMERIC(5,4) NOT NULL DEFAULT 0.0,  -- 퀄리티 점수 (0~1)
    flow         NUMERIC(5,4) NOT NULL DEFAULT 0.0,  -- 수급 점수 (0~1)
    event        NUMERIC(5,4) NOT NULL DEFAULT 0.0,  -- 이벤트 점수 (0~1)
    total_score  NUMERIC(5,4),                       -- 종합 점수 (가중평균)
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_factor_scores_date ON signals.factor_scores(calc_date);
CREATE INDEX idx_factor_scores_total ON signals.factor_scores(total_score DESC);

COMMENT ON TABLE signals.factor_scores IS 'S2: 팩터 점수 (6가지 시그널)';
COMMENT ON COLUMN signals.factor_scores.momentum IS '모멘텀 점수 (0~1): 가격 추세';
COMMENT ON COLUMN signals.factor_scores.technical IS '기술적 점수 (0~1): MA, RSI, MACD';
COMMENT ON COLUMN signals.factor_scores.value IS '가치 점수 (0~1): PER, PBR 등';
COMMENT ON COLUMN signals.factor_scores.quality IS '퀄리티 점수 (0~1): ROE, 부채비율';
COMMENT ON COLUMN signals.factor_scores.flow IS '수급 점수 (0~1): 외국인/기관 순매수';
COMMENT ON COLUMN signals.factor_scores.event IS '이벤트 점수 (0~1): 공시, 뉴스';

-- ============================================
-- 2. flow_details (수급 상세)
-- ============================================
CREATE TABLE signals.flow_details (
    stock_code        VARCHAR(20) NOT NULL,
    calc_date         DATE NOT NULL,
    -- 5일 누적
    foreign_net_5d    BIGINT DEFAULT 0,
    inst_net_5d       BIGINT DEFAULT 0,
    indiv_net_5d      BIGINT DEFAULT 0,
    -- 10일 누적
    foreign_net_10d   BIGINT DEFAULT 0,
    inst_net_10d      BIGINT DEFAULT 0,
    indiv_net_10d     BIGINT DEFAULT 0,
    -- 20일 누적
    foreign_net_20d   BIGINT DEFAULT 0,
    inst_net_20d      BIGINT DEFAULT 0,
    indiv_net_20d     BIGINT DEFAULT 0,
    updated_at        TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_flow_details_date ON signals.flow_details(calc_date);

COMMENT ON TABLE signals.flow_details IS '수급 상세 (5D/10D/20D 누적)';

-- ============================================
-- 3. technical_details (기술적 지표)
-- ============================================
CREATE TABLE signals.technical_details (
    stock_code   VARCHAR(20) NOT NULL,
    calc_date    DATE NOT NULL,
    -- 이동평균
    ma5          NUMERIC(12,2),
    ma10         NUMERIC(12,2),
    ma20         NUMERIC(12,2),
    ma60         NUMERIC(12,2),
    ma120        NUMERIC(12,2),
    -- RSI
    rsi14        NUMERIC(5,2),
    -- MACD
    macd         NUMERIC(12,4),
    macd_signal  NUMERIC(12,4),
    macd_hist    NUMERIC(12,4),
    -- 볼린저 밴드
    bb_upper     NUMERIC(12,2),
    bb_middle    NUMERIC(12,2),
    bb_lower     NUMERIC(12,2),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_technical_details_date ON signals.technical_details(calc_date);

COMMENT ON TABLE signals.technical_details IS '기술적 지표 (MA, RSI, MACD, Bollinger)';

-- ============================================
-- 4. event_signals (이벤트 시그널)
-- ============================================
CREATE TABLE signals.event_signals (
    id            SERIAL PRIMARY KEY,
    stock_code    VARCHAR(20) NOT NULL,
    event_date    DATE NOT NULL,
    event_type    VARCHAR(50) NOT NULL,      -- disclosure, news, earning
    event_subtype VARCHAR(50),
    title         TEXT,
    description   TEXT,
    impact_score  NUMERIC(5,4) DEFAULT 0.0,  -- 영향도 점수 (0~1)
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_event_signals_stock ON signals.event_signals(stock_code);
CREATE INDEX idx_event_signals_date ON signals.event_signals(event_date);
CREATE INDEX idx_event_signals_type ON signals.event_signals(event_type);

COMMENT ON TABLE signals.event_signals IS '이벤트 시그널 (공시, 뉴스, 실적)';

-- =====================================================
-- signals 스키마 테이블 생성 완료
-- 검증: SELECT table_name FROM information_schema.tables WHERE table_schema = 'signals' ORDER BY table_name;
-- =====================================================
