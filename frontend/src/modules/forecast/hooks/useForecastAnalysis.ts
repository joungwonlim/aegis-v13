/**
 * Forecast Analysis Hook
 * SSOT: modules/forecast/hooks/useForecastAnalysis.ts
 */

import { useState, useEffect } from 'react'
import type { ForecastResult } from '../types'

interface UseForecastAnalysisOptions {
  autoFetch?: boolean
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

export function useForecastAnalysis(
  symbol: string,
  options: UseForecastAnalysisOptions = {}
) {
  const { autoFetch = false } = options

  const [data, setData] = useState<ForecastResult | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const analyze = async () => {
    if (!symbol) {
      setError(new Error('Symbol is required'))
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const response = await fetch(`${API_BASE}/api/forecast/analyze/${symbol}`, {
        method: 'POST',
      })

      if (!response.ok) {
        throw new Error(`Failed to analyze forecast: ${response.statusText}`)
      }

      const result = await response.json()
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error'))
    } finally {
      setIsLoading(false)
    }
  }

  // Auto-fetch on mount if enabled
  useEffect(() => {
    if (autoFetch && symbol) {
      analyze()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [symbol, autoFetch])

  return {
    data,
    isLoading,
    error,
    analyze,
    refetch: analyze,
  }
}
