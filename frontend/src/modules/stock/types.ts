/**
 * Stock 모듈 타입 정의
 * SSOT: 종목 관련 타입은 여기서만 정의
 */

export interface StockInfo {
  code: string
  name: string
  market?: string
}

export interface StockPrice {
  price: number
  change: number
  change_rate: number
  volume?: number
}

export type SizeVariant = 'sm' | 'md' | 'lg'

// 일봉 데이터
export interface DailyPrice {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

// 투자자별 매매동향 (v13 백엔드 응답 형식)
export interface InvestorTrading {
  date: string
  close_price: number
  price_change: number
  change_rate: number
  volume: number
  foreign_net: number
  inst_net: number
  indiv_net: number
}
