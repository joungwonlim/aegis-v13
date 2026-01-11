'use client'

/**
 * 종목 상세 설명 조회 Hook
 * SSOT: 종목 기업개요 조회
 * v13 백엔드 사용
 */

import { useQuery } from '@tanstack/react-query'

// v13 백엔드 사용
const STOCK_API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

interface StockDescriptionResponse {
  success: boolean
  description: string
}

async function fetchStockDescription(code: string): Promise<string> {
  if (!code) return ''

  const res = await fetch(`${STOCK_API_BASE}/api/stocks/${code}/description`)
  if (!res.ok) {
    throw new Error(`Failed to fetch stock description: ${res.status}`)
  }

  const json: StockDescriptionResponse = await res.json()
  return json.description ?? ''
}

export function useStockDescription(code: string) {
  return useQuery({
    queryKey: ['stock', 'description', code],
    queryFn: () => fetchStockDescription(code),
    enabled: !!code,
    staleTime: 24 * 60 * 60 * 1000, // 24시간 (기업개요는 자주 바뀌지 않음)
  })
}
