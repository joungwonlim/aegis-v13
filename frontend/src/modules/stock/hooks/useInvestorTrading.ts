'use client'

/**
 * 투자자별 매매동향 조회 Hook
 * SSOT: 외국인/기관/개인 매매동향 데이터 조회
 * v13 백엔드 사용
 */

import { useQuery } from '@tanstack/react-query'
import type { InvestorTrading } from '../types'

// v13 백엔드 사용
const STOCK_API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

interface InvestorTradingResponse {
  success: boolean
  data: InvestorTrading[]
}

async function fetchInvestorTrading(code: string, days: number): Promise<InvestorTrading[]> {
  if (!code) return []

  const res = await fetch(`${STOCK_API_BASE}/api/stocks/${code}/investor-trading?days=${days}`)
  if (!res.ok) {
    throw new Error(`Failed to fetch investor trading: ${res.status}`)
  }

  const json: InvestorTradingResponse = await res.json()
  return json.data ?? []
}

export function useInvestorTrading(code: string, days: number = 365) {
  return useQuery({
    queryKey: ['stock', 'investor-trading', code, days],
    queryFn: () => fetchInvestorTrading(code, days),
    enabled: !!code,
    staleTime: 5 * 60 * 1000, // 5분
  })
}
