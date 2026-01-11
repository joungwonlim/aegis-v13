/**
 * Forecast API
 * SSOT: modules/forecast/api.ts
 */

import type { ForecastPrediction } from './types'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

export const forecastApi = {
  /**
   * 특정 종목의 forecast 예측 조회
   */
  async getPrediction(stockCode: string): Promise<ForecastPrediction | null> {
    const res = await fetch(`${API_BASE}/api/forecast/predict/${stockCode}`)

    if (res.status === 404) {
      // 예측 데이터 없음 (최근 이벤트 없음)
      return null
    }

    if (!res.ok) {
      throw new Error(`Failed to fetch forecast: ${res.statusText}`)
    }

    const data = await res.json()
    return data.prediction
  },
}
