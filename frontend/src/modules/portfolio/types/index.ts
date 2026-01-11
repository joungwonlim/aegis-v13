/**
 * Portfolio Types
 * SSOT: v13 KIS API 응답 타입
 */

// KIS Balance (계좌 잔고) - v13 API 응답 형식
export interface KISBalance {
  total_deposit: number       // 예수금
  available_cash: number      // 출금가능금액
  total_purchase: number      // 매입금액합계
  total_evaluation: number    // 평가금액합계
  total_profit_loss: number   // 평가손익합계
  profit_loss_rate: number    // 수익률
  total_asset?: number        // 총자산
}

// KIS Position (보유 종목) - v13 API 응답 형식
export interface KISPosition {
  id?: number                  // DB ID (exit monitoring용)
  stock_code: string
  stock_name: string
  market?: string
  quantity: number             // 보유수량
  available_quantity: number   // 매도가능수량
  avg_buy_price: number        // 평균매입가
  current_price: number        // 현재가
  eval_amount: number          // 평가금액
  evaluation_amount?: number   // 평가금액 (alias)
  purchase_amount: number      // 매입금액
  profit_loss: number          // 평가손익
  profit_loss_rate: number     // 수익률 (%)
  exit_monitoring_enabled?: boolean
}

// Balance API 응답 (v13)
export interface BalanceResponse {
  balance: KISBalance
  positions: KISPosition[]
}

// Positions API 응답 (v13)
export interface PositionsResponse {
  positions: KISPosition[]
  count: number
}

// ========================================
// Legacy Types (문서 호환용)
// ========================================

// 목표 포트폴리오 포지션
export interface TargetPosition {
  code: string
  name: string
  weight: number
  target_qty: number
  action: 'BUY' | 'SELL' | 'HOLD'
  reason: string
}

// 목표 포트폴리오
export interface TargetPortfolio {
  date: string
  total_positions: number
  total_weight: number
  cash_reserve: number
  positions: TargetPosition[]
  config: PortfolioConfig
}

// 포트폴리오 설정
export interface PortfolioConfig {
  max_positions: number
  max_weight: number
  min_weight: number
  weighting_mode: 'equal' | 'score_based' | 'risk_parity'
}

// API 응답 타입
export interface ApiResponse<T> {
  success: boolean
  data: T
  error?: string
}
