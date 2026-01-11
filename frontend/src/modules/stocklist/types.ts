/**
 * StockList Types
 * SSOT: 관심종목 관련 타입 정의
 */

// 관심종목 카테고리
export type StockListCategory = 'watch' | 'candidate'

// 관심종목 아이템
export interface StockListItem {
  id: number
  stock_code: string
  stock_name?: string
  category: StockListCategory
  alert_enabled: boolean
  grok_analysis?: string
  gemini_analysis?: string
  chatgpt_analysis?: string
  created_at: string
  updated_at: string
  // 실시간 데이터 (별도 조회)
  current_price?: number
  change_rate?: number
  market?: string
}

// 관심종목 목록 응답
export interface StockListResponse {
  success: boolean
  data: {
    watch: StockListItem[]
    candidate: StockListItem[]
  }
}

// 관심종목 생성 요청
export interface CreateStockListRequest {
  stock_code: string
  category: StockListCategory
  alert_enabled?: boolean
}

// 관심종목 수정 요청
export interface UpdateStockListRequest {
  category?: StockListCategory
  alert_enabled?: boolean
}
