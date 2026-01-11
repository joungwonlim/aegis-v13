-- Migration: 025_target_qty_to_value
-- Description: Rename target_qty to target_value in target_positions table
-- ⭐ P0 수정: Portfolio는 목표 금액(TargetValue)만 산출, Execution이 수량 계산
-- Date: 2026-01-11

-- 1. 기존 컬럼 이름 변경 (target_qty → target_value)
-- 기존 데이터는 수량이었지만, 앞으로는 금액으로 사용
ALTER TABLE portfolio.target_positions
    RENAME COLUMN target_qty TO target_value;

-- 2. 컬럼 타입 변경 (INT → BIGINT, 금액은 더 큰 숫자 필요)
ALTER TABLE portfolio.target_positions
    ALTER COLUMN target_value TYPE BIGINT;

-- 3. 코멘트 업데이트
COMMENT ON COLUMN portfolio.target_positions.target_value IS
    '목표 금액 (원화). Execution에서 현재가로 수량 계산. TargetValue = Weight × PortfolioValue';

-- 4. 기존 데이터 변환 (선택적 - 기존 데이터가 수량이면 금액으로 변환)
-- 기존 데이터가 수량(예: 100주)이면, 대략적인 금액으로 변환 (주당 50,000원 가정)
-- 운영 환경에서는 데이터 검증 후 실행
-- UPDATE portfolio.target_positions
-- SET target_value = target_value * 50000
-- WHERE target_value < 100000;  -- 10만원 미만이면 수량으로 간주

-- 검증
DO $$
BEGIN
    RAISE NOTICE 'Migration 025: target_qty renamed to target_value successfully';
END $$;
