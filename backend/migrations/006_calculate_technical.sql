-- =====================================================
-- 006_calculate_technical.sql
-- Phase 3: 기술적 지표 계산 (MA, RSI, MACD, Bollinger)
--
-- data.daily_prices → signals.technical_details 변환
-- Window function을 사용하여 이동평균, RSI, MACD 계산
-- =====================================================

-- ============================================
-- 1. 이동평균 (MA5, MA10, MA20, MA60, MA120)
-- ============================================
WITH ma_calc AS (
    SELECT
        stock_code,
        trade_date,
        close_price,
        -- 이동평균
        AVG(close_price) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 4 PRECEDING AND CURRENT ROW
        ) AS ma5,
        AVG(close_price) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 9 PRECEDING AND CURRENT ROW
        ) AS ma10,
        AVG(close_price) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 19 PRECEDING AND CURRENT ROW
        ) AS ma20,
        AVG(close_price) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 59 PRECEDING AND CURRENT ROW
        ) AS ma60,
        AVG(close_price) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 119 PRECEDING AND CURRENT ROW
        ) AS ma120,
        -- 볼린저 밴드 (20일 기준)
        AVG(close_price) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 19 PRECEDING AND CURRENT ROW
        ) AS bb_middle,
        STDDEV(close_price) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 19 PRECEDING AND CURRENT ROW
        ) AS bb_stddev
    FROM data.daily_prices
    WHERE trade_date >= '2022-01-01'
),
-- ============================================
-- 2. RSI 계산 (14일 기준)
-- ============================================
price_change AS (
    SELECT
        stock_code,
        trade_date,
        close_price,
        close_price - LAG(close_price) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
        ) AS price_change
    FROM data.daily_prices
    WHERE trade_date >= '2022-01-01'
),
rsi_calc AS (
    SELECT
        stock_code,
        trade_date,
        close_price,
        AVG(CASE WHEN price_change > 0 THEN price_change ELSE 0 END) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 13 PRECEDING AND CURRENT ROW
        ) AS avg_gain,
        AVG(CASE WHEN price_change < 0 THEN ABS(price_change) ELSE 0 END) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 13 PRECEDING AND CURRENT ROW
        ) AS avg_loss
    FROM price_change
),
-- ============================================
-- 3. MACD 계산 (12, 26, 9)
-- ============================================
ema_calc AS (
    SELECT
        stock_code,
        trade_date,
        close_price,
        AVG(close_price) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 11 PRECEDING AND CURRENT ROW
        ) AS ema12,  -- 단순 평균으로 근사
        AVG(close_price) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 25 PRECEDING AND CURRENT ROW
        ) AS ema26
    FROM data.daily_prices
    WHERE trade_date >= '2022-01-01'
),
macd_calc AS (
    SELECT
        stock_code,
        trade_date,
        ema12 - ema26 AS macd,
        AVG(ema12 - ema26) OVER (
            PARTITION BY stock_code
            ORDER BY trade_date
            ROWS BETWEEN 8 PRECEDING AND CURRENT ROW
        ) AS macd_signal
    FROM ema_calc
),
-- ============================================
-- 4. 모든 지표 병합
-- ============================================
all_indicators AS (
    SELECT
        m.stock_code,
        m.trade_date AS calc_date,
        m.ma5,
        m.ma10,
        m.ma20,
        m.ma60,
        m.ma120,
        -- RSI
        CASE
            WHEN r.avg_loss = 0 THEN 100
            WHEN r.avg_gain = 0 THEN 0
            ELSE 100 - (100 / (1 + r.avg_gain / r.avg_loss))
        END AS rsi14,
        -- MACD
        mc.macd,
        mc.macd_signal,
        mc.macd - mc.macd_signal AS macd_hist,
        -- 볼린저 밴드
        m.bb_middle + (2 * m.bb_stddev) AS bb_upper,
        m.bb_middle AS bb_middle,
        m.bb_middle - (2 * m.bb_stddev) AS bb_lower
    FROM ma_calc m
    LEFT JOIN rsi_calc r ON m.stock_code = r.stock_code AND m.trade_date = r.trade_date
    LEFT JOIN macd_calc mc ON m.stock_code = mc.stock_code AND m.trade_date = mc.trade_date
)
-- ============================================
-- 5. INSERT
-- ============================================
INSERT INTO signals.technical_details (
    stock_code,
    calc_date,
    ma5, ma10, ma20, ma60, ma120,
    rsi14,
    macd, macd_signal, macd_hist,
    bb_upper, bb_middle, bb_lower,
    updated_at
)
SELECT
    stock_code,
    calc_date,
    ma5, ma10, ma20, ma60, ma120,
    rsi14,
    macd, macd_signal, macd_hist,
    bb_upper, bb_middle, bb_lower,
    NOW()
FROM all_indicators
ON CONFLICT (stock_code, calc_date) DO UPDATE SET
    ma5 = EXCLUDED.ma5,
    ma10 = EXCLUDED.ma10,
    ma20 = EXCLUDED.ma20,
    ma60 = EXCLUDED.ma60,
    ma120 = EXCLUDED.ma120,
    rsi14 = EXCLUDED.rsi14,
    macd = EXCLUDED.macd,
    macd_signal = EXCLUDED.macd_signal,
    macd_hist = EXCLUDED.macd_hist,
    bb_upper = EXCLUDED.bb_upper,
    bb_middle = EXCLUDED.bb_middle,
    bb_lower = EXCLUDED.bb_lower,
    updated_at = NOW();

-- =====================================================
-- 검증 쿼리
-- =====================================================

-- 1. 전체 레코드 수 확인
-- SELECT COUNT(*) FROM signals.technical_details;
-- 예상: ~3,035,230 rows (data.daily_prices와 동일)

-- 2. 샘플 데이터 확인 (특정 종목)
-- SELECT
--     stock_code,
--     calc_date,
--     ma5, ma20, ma120,
--     rsi14,
--     macd, macd_signal
-- FROM signals.technical_details
-- WHERE stock_code = '005930'  -- 삼성전자
-- ORDER BY calc_date DESC
-- LIMIT 10;

-- 3. MA120 데이터가 충분한지 확인 (120일 이후부터 정상값)
-- SELECT
--     stock_code,
--     MIN(calc_date) as first_date,
--     COUNT(*) as total_days
-- FROM signals.technical_details
-- WHERE ma120 IS NOT NULL
-- GROUP BY stock_code
-- ORDER BY total_days DESC
-- LIMIT 10;

-- =====================================================
