/**
 * Universe 모듈 타입 정의
 */

export interface RankingItem {
  ID: number
  SnapshotDate: string
  SnapshotTime: string
  Category: string
  Market: string
  RankPosition: number
  StockCode: string
  StockName: string
  CurrentPrice: number
  PriceChange: number
  ChangeRate: number
  Volume: number
  TradingValue: number
  HighPrice: number
  LowPrice: number
  MarketCap: number
  CreatedAt: string
}

export interface RankingResponse {
  success: boolean
  data: {
    category: string
    count: number
    items: RankingItem[]
  }
}

export interface RankingStatusResponse {
  success: boolean
  data: {
    categories: string[]
    is_running: boolean
    last_collection: string
    markets: string[]
    schedule_times: string[]
  }
}

export type RankingCategory =
  | 'top'
  | 'trading'
  | 'capitalization'
  | 'quantHigh'
  | 'quantLow'
  | 'priceTop'
  | 'upper'
  | 'lower'
  | 'new'
  | 'high52week'
  | 'low52week'
