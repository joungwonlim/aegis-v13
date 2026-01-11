/**
 * Portfolio API
 * SSOT: 모든 포트폴리오 API 호출
 * v13 백엔드 연동 (KIS API)
 */

import type {
  BalanceResponse,
  PositionsResponse,
  TargetPortfolio,
  ApiResponse,
} from './types'

// v13 백엔드
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

async function portfolioFetch<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    cache: 'no-store',
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!res.ok) {
    throw new Error(`Portfolio API Error: ${res.status}`)
  }

  return res.json()
}

export const portfolioApi = {
  // KIS 계좌 잔고 조회 (잔고 + 보유종목)
  getBalance: () => {
    return portfolioFetch<BalanceResponse>('/api/trading/balance')
  },

  // KIS 보유 종목만 조회
  getPositions: () => {
    return portfolioFetch<PositionsResponse>('/api/trading/positions')
  },

  // 목표 포트폴리오 조회 (Legacy - 추후 구현)
  getTarget: (date?: string) => {
    const query = date ? `?date=${date}` : ''
    return portfolioFetch<ApiResponse<TargetPortfolio>>(
      `/api/portfolio/target${query}`
    )
  },

  // 포트폴리오 구성 실행 (Legacy - 추후 구현)
  construct: (params: {
    date: string
    config?: {
      max_positions?: number
      max_weight?: number
      min_weight?: number
      cash_reserve?: number
      weighting_mode?: 'equal' | 'score_based' | 'risk_parity'
    }
    constraints?: {
      max_sector_weight?: number
      blacklist?: string[]
    }
  }) => {
    return portfolioFetch<ApiResponse<{ job_id: string; status: string; result: TargetPortfolio }>>(
      '/api/portfolio/construct',
      {
        method: 'POST',
        body: JSON.stringify(params),
      }
    )
  },

  // 청산 모니터링 설정 변경
  updateExitMonitoring: (stockCode: string, enabled: boolean) => {
    return portfolioFetch<{ success: boolean; stock_code: string; enabled: boolean }>(
      `/api/trading/positions/${stockCode}/exit-monitoring`,
      {
        method: 'PATCH',
        body: JSON.stringify({ enabled }),
      }
    )
  },

  // 청산 모니터링 상태 조회
  getExitMonitoringStatus: () => {
    return portfolioFetch<{
      statuses: Array<{ stock_code: string; enabled: boolean }>
      count: number
    }>('/api/trading/exit-monitoring')
  },
}
