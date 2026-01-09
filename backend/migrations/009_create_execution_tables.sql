-- =====================================================
-- 009_create_execution_tables.sql
-- Phase 4: execution 스키마 테이블 생성 (S6)
-- =====================================================

-- ============================================
-- 1. orders (주문 테이블)
-- ============================================
CREATE TABLE execution.orders (
    id              SERIAL PRIMARY KEY,
    stock_code      VARCHAR(20) NOT NULL,
    stock_name      VARCHAR(200),
    order_date      DATE NOT NULL,
    order_time      TIMESTAMPTZ DEFAULT NOW(),
    order_action    VARCHAR(10) NOT NULL,      -- BUY, SELL
    order_type      VARCHAR(20) NOT NULL,      -- MARKET, LIMIT, STOP
    order_price     NUMERIC(12,2),
    order_qty       INT NOT NULL,
    filled_qty      INT DEFAULT 0,
    filled_price    NUMERIC(12,2),
    status          VARCHAR(20) DEFAULT 'pending',  -- pending, submitted, filled, partial, cancelled, rejected
    broker_order_no VARCHAR(50),
    reason          TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_orders_stock ON execution.orders(stock_code);
CREATE INDEX idx_orders_date ON execution.orders(order_date);
CREATE INDEX idx_orders_status ON execution.orders(status);
CREATE INDEX idx_orders_broker ON execution.orders(broker_order_no);

COMMENT ON TABLE execution.orders IS 'S6: 주문 테이블';

-- ============================================
-- 2. trades (체결 테이블)
-- ============================================
CREATE TABLE execution.trades (
    id              SERIAL PRIMARY KEY,
    order_id        INT NOT NULL REFERENCES execution.orders(id),
    stock_code      VARCHAR(20) NOT NULL,
    trade_date      DATE NOT NULL,
    trade_time      TIMESTAMPTZ NOT NULL,
    trade_action    VARCHAR(10) NOT NULL,      -- BUY, SELL
    trade_price     NUMERIC(12,2) NOT NULL,
    trade_qty       INT NOT NULL,
    trade_amount    BIGINT NOT NULL,
    commission      BIGINT DEFAULT 0,
    tax             BIGINT DEFAULT 0,
    broker_trade_no VARCHAR(50),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_trades_order ON execution.trades(order_id);
CREATE INDEX idx_trades_stock ON execution.trades(stock_code);
CREATE INDEX idx_trades_date ON execution.trades(trade_date);

COMMENT ON TABLE execution.trades IS '체결 내역';

-- ============================================
-- 3. order_errors (주문 오류 로그)
-- ============================================
CREATE TABLE execution.order_errors (
    id              SERIAL PRIMARY KEY,
    order_id        INT REFERENCES execution.orders(id),
    stock_code      VARCHAR(20),
    error_time      TIMESTAMPTZ DEFAULT NOW(),
    error_code      VARCHAR(50),
    error_message   TEXT NOT NULL,
    broker_response JSONB,
    retry_count     INT DEFAULT 0,
    resolved        BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_order_errors_order ON execution.order_errors(order_id);
CREATE INDEX idx_order_errors_stock ON execution.order_errors(stock_code);
CREATE INDEX idx_order_errors_resolved ON execution.order_errors(resolved);

COMMENT ON TABLE execution.order_errors IS '주문 오류 로그';

-- =====================================================
-- execution 스키마 테이블 생성 완료
-- 검증: SELECT table_name FROM information_schema.tables WHERE table_schema = 'execution' ORDER BY table_name;
-- =====================================================
