/**
 * Forecast Events Hook
 * SSOT: modules/forecast/hooks/useForecastEvents.ts
 */

import { useState, useEffect } from 'react'
import type { Event } from '../types'

interface UseForecastEventsOptions {
  autoFetch?: boolean
  limit?: number
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

export function useForecastEvents(
  symbol: string,
  options: UseForecastEventsOptions = {}
) {
  const { autoFetch = false, limit = 10 } = options

  const [data, setData] = useState<Event[] | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const fetchEvents = async () => {
    if (!symbol) {
      setError(new Error('Symbol is required'))
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const response = await fetch(
        `${API_BASE}/api/forecast/events/${symbol}?limit=${limit}`
      )

      if (!response.ok) {
        throw new Error(`Failed to fetch events: ${response.statusText}`)
      }

      const result = await response.json()
      setData(result.events || [])
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error'))
    } finally {
      setIsLoading(false)
    }
  }

  // Auto-fetch on mount if enabled
  useEffect(() => {
    if (autoFetch && symbol) {
      fetchEvents()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [symbol, autoFetch])

  return {
    data,
    isLoading,
    error,
    fetchEvents,
    refetch: fetchEvents,
  }
}
