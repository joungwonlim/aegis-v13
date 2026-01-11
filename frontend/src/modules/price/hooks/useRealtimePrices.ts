/**
 * 실시간 가격 조회 Hook
 * SSOT: 모든 실시간 가격 조회는 이 hook을 통해서만
 */

import { useQuery } from '@tanstack/react-query'
import { api } from '@/shared/api/client'

export interface RealtimePrice {
  price: number
  change: number
  change_rate: number
  volume?: number
  updated_at?: string
}

interface PricesResponse {
  prices: Record<string, RealtimePrice>
}

/**
 * 실시간 가격 조회 Hook
 *
 * 우선순위:
 * 1. KIS WebSocket (실시간)
 * 2. KIS REST API (폴백)
 * 3. Naver Finance (백업)
 */
export function useRealtimePrices(
  symbols: string[],
  options?: { enabled?: boolean; refetchInterval?: number }
) {
  const { enabled = true, refetchInterval = 5000 } = options ?? {}

  return useQuery({
    queryKey: ['prices', 'realtime', symbols.sort().join(',')],
    queryFn: async (): Promise<Record<string, RealtimePrice>> => {
      if (symbols.length === 0) return {}

      const res = await api.get<PricesResponse>(
        `/api/trading/prices?symbols=${symbols.join(',')}`
      )
      return res.prices
    },
    enabled: enabled && symbols.length > 0,
    staleTime: 1000,
    refetchInterval,
    refetchIntervalInBackground: false,
  })
}
