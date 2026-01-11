/**
 * StockList API
 * SSOT: 모든 관심종목 API 호출
 */

import type {
  StockListResponse,
  StockListItem,
  CreateStockListRequest,
  UpdateStockListRequest,
} from './types'

// API Base URL (v13 백엔드 사용)
const STOCKLIST_API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

async function stocklistFetch<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const url = `${STOCKLIST_API_BASE}${endpoint}`
  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!res.ok) {
    throw new Error(`API Error: ${res.status}`)
  }

  return res.json()
}

export const stocklistApi = {
  // 관심종목 목록 조회
  getList: () => {
    return stocklistFetch<StockListResponse>('/api/v1/watchlist')
  },

  // watch 목록만 조회
  getWatchList: () => {
    return stocklistFetch<{ success: boolean; data: StockListItem[] }>('/api/v1/watchlist/watch')
  },

  // candidate 목록만 조회
  getCandidateList: () => {
    return stocklistFetch<{ success: boolean; data: StockListItem[] }>('/api/v1/watchlist/candidate')
  },

  // 관심종목 추가
  create: (data: CreateStockListRequest) => {
    return stocklistFetch<{ success: boolean; data: StockListItem }>('/api/v1/watchlist', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  // 관심종목 수정
  update: (id: number, data: UpdateStockListRequest) => {
    return stocklistFetch<{ success: boolean; data: StockListItem }>(`/api/v1/watchlist/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },

  // 관심종목 삭제
  delete: (id: number) => {
    return stocklistFetch<{ success: boolean }>(`/api/v1/watchlist/${id}`, {
      method: 'DELETE',
    })
  },
}
