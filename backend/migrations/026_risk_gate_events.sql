-- Migration: 026_risk_gate_events
-- Description: Create risk gate events table for S6 Shadow Mode
-- Date: 2026-01-11

-- 리스크 게이트 이벤트 테이블 (Shadow Mode 분석용)
CREATE TABLE IF NOT EXISTS execution.risk_gate_events (
    id SERIAL PRIMARY KEY,
    run_id VARCHAR(50) NOT NULL,
    mode VARCHAR(20) NOT NULL,          -- 'shadow', 'enforce', 'off'
    passed BOOLEAN NOT NULL,
    would_block BOOLEAN NOT NULL,       -- Shadow 모드에서 차단됐을지 여부
    violation_count INT DEFAULT 0,
    var_95 NUMERIC(10,6),               -- 95% VaR
    var_99 NUMERIC(10,6),               -- 99% VaR
    message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 인덱스 생성
CREATE INDEX IF NOT EXISTS idx_risk_gate_events_run_id
    ON execution.risk_gate_events(run_id);

CREATE INDEX IF NOT EXISTS idx_risk_gate_events_created_at
    ON execution.risk_gate_events(created_at);

CREATE INDEX IF NOT EXISTS idx_risk_gate_events_mode
    ON execution.risk_gate_events(mode);

CREATE INDEX IF NOT EXISTS idx_risk_gate_events_would_block
    ON execution.risk_gate_events(would_block)
    WHERE would_block = true;

-- 코멘트 추가
COMMENT ON TABLE execution.risk_gate_events IS 'S6 리스크 게이트 이벤트 기록 (Shadow Mode 분석용)';
COMMENT ON COLUMN execution.risk_gate_events.mode IS '게이트 모드: shadow(로깅만), enforce(실제 차단), off(비활성화)';
COMMENT ON COLUMN execution.risk_gate_events.would_block IS 'Shadow 모드에서 실제로 차단됐을지 여부';
COMMENT ON COLUMN execution.risk_gate_events.violation_count IS '리스크 한도 위반 횟수';

-- 검증
DO $$
BEGIN
    RAISE NOTICE 'Migration 026: risk_gate_events table created successfully';
END $$;
