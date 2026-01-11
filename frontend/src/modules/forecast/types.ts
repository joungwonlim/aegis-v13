/**
 * Forecast Module Types
 * SSOT: modules/forecast/types.ts
 * v10과 동일한 타입 구조
 */

export type EventType = 'E1' | 'E2'
export type ScopeType = 'SYMBOL' | 'SECTOR' | 'BUCKET' | 'MARKET'

export interface EventCharacteristics {
  trade_date: string
  ret: number
  gap: number
  close_to_high: number
  vol_z: number
}

export interface Prediction {
  expected_ret_5d: number
  win_rate_5d: number
  p10_mdd_5d: number
  p90_runup_5d: number
  gap_hold_win_rate?: number
  gap_break_win_rate?: number
}

export interface DailyGapStatus {
  day: number
  trade_date: string
  low: number
  is_held: boolean
}

export interface GapMonitoring {
  gap_zone_low: number
  days_elapsed: number
  daily_status: DailyGapStatus[]
  is_confirmed: boolean
  final_status?: boolean
}

export type PredictionType = 'EVENT_BASED' | 'GENERAL' | 'NONE'
export type QualityLevel = 'HIGH' | 'MEDIUM' | 'LOW' | 'UNKNOWN'

export interface ForecastResult {
  symbol: string
  analyzed_at: string
  event_detected: boolean
  event_type?: EventType
  current_event?: EventCharacteristics
  prediction?: Prediction
  gap_monitoring?: GapMonitoring
  fallback_level: ScopeType
  sample_size: number
  // Quality indicators
  prediction_type: PredictionType
  quality: QualityLevel
  warnings: string[]
}

export interface Event {
  id: number
  symbol: string
  trade_date: string
  event_type: EventType
  ret: number
  gap: number
  close_to_high: number
  vol_z: number
  fwd_ret_1d?: number
  fwd_ret_2d?: number
  fwd_ret_3d?: number
  fwd_ret_5d?: number
  max_runup_5d?: number
  max_drawdown_5d?: number
  gap_hold_3d?: boolean
  created_at: string
  updated_at: string
}

// Legacy types for backward compatibility
export type ForecastEventType = 'E1_SURGE' | 'E2_GAP_SURGE'
export type ForecastStatsLevel = 'SYMBOL' | 'SECTOR' | 'BUCKET' | 'MARKET'

export interface ForecastPrediction {
  stock_code: string
  event_type: ForecastEventType
  expected_return_1d: number
  expected_return_5d: number
  win_rate_1d: number
  win_rate_5d: number
  p10_mdd: number
  confidence: number
  sample_count: number
  level: ForecastStatsLevel
  fallback_reason?: string
}
