'use client'

/**
 * 일봉 데이터 조회 Hook
 * SSOT: 일봉 차트 데이터 조회
 * v13 백엔드 사용
 */

import { useQuery } from '@tanstack/react-query'
import type { DailyPrice } from '../types'

// v13 백엔드 사용
const STOCK_API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

interface DailyPricesResponse {
  success: boolean
  data: DailyPrice[]
}

async function fetchDailyPrices(code: string, days: number): Promise<DailyPrice[]> {
  if (!code) return []

  const res = await fetch(`${STOCK_API_BASE}/api/stocks/${code}/daily?days=${days}`)
  if (!res.ok) {
    throw new Error(`Failed to fetch daily prices: ${res.status}`)
  }

  const json: DailyPricesResponse = await res.json()
  return json.data ?? []
}

export function useDailyPrices(code: string, days: number = 365) {
  return useQuery({
    queryKey: ['stock', 'daily', code, days],
    queryFn: () => fetchDailyPrices(code, days),
    enabled: !!code,
    staleTime: 5 * 60 * 1000, // 5분
  })
}
