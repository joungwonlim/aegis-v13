/**
 * Forecast Prediction Hook
 * SSOT: modules/forecast/hooks/useForecastPrediction.ts
 */

import { useQuery } from '@tanstack/react-query'
import { forecastApi } from '../api'

export function useForecastPrediction(stockCode: string, enabled = true) {
  return useQuery({
    queryKey: ['forecast', 'prediction', stockCode],
    queryFn: () => forecastApi.getPrediction(stockCode),
    enabled: enabled && !!stockCode,
    staleTime: 1000 * 60 * 60, // 1시간
    gcTime: 1000 * 60 * 60 * 2, // 2시간
  })
}
