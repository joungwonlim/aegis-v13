/**
 * Universe API
 * SSOT: 유니버스/랭킹 API 호출
 * v13 백엔드 사용
 */

import type {
  RankingResponse,
  RankingStatusResponse,
  RankingCategory,
  RankingItem,
} from './types'

// v13 백엔드 사용
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

export type MarketType = 'KOSPI' | 'KOSDAQ' | 'ALL'

/**
 * 랭킹 데이터 조회 (단일 마켓)
 */
async function getRankingSingle(
  category: RankingCategory,
  market: 'KOSPI' | 'KOSDAQ'
): Promise<RankingResponse> {
  const response = await fetch(
    `${API_BASE}/api/v1/ranking/${category}?market=${market}`
  )
  if (!response.ok) {
    throw new Error(`Failed to fetch ranking: ${response.status}`)
  }
  return response.json()
}

/**
 * 랭킹 데이터 조회 (KOSPI + KOSDAQ 통합 지원)
 */
export async function getRanking(
  category: RankingCategory = 'top',
  market: MarketType = 'ALL'
): Promise<RankingResponse> {
  if (market === 'ALL') {
    // KOSPI + KOSDAQ 모두 조회 후 병합
    const [kospiRes, kosdaqRes] = await Promise.all([
      getRankingSingle(category, 'KOSPI'),
      getRankingSingle(category, 'KOSDAQ'),
    ])

    const mergedItems: RankingItem[] = [
      ...(kospiRes.data?.items ?? []),
      ...(kosdaqRes.data?.items ?? []),
    ]

    // RankPosition 기준 정렬
    mergedItems.sort((a, b) => a.RankPosition - b.RankPosition)

    return {
      success: true,
      data: {
        category,
        count: mergedItems.length,
        items: mergedItems,
      },
    }
  }

  return getRankingSingle(category, market)
}

/**
 * 랭킹 상태 조회
 */
export async function getRankingStatus(): Promise<RankingStatusResponse> {
  const response = await fetch(`${API_BASE}/api/v1/ranking/status`)
  if (!response.ok) {
    throw new Error(`Failed to fetch ranking status: ${response.status}`)
  }
  return response.json()
}

// ============================================
// Pipeline API (S1-S5 실제 데이터)
// ============================================

// S1: Universe 데이터
export interface UniverseItem {
  stockCode: string
  stockName: string
  market: string
  currentPrice: number
  changeRate: number
  volume: number
  marketCap: number
}

export interface UniverseResponse {
  success: boolean
  data: {
    date: string
    market: string
    count: number
    totalCount: number
    items: UniverseItem[]
  }
}

/**
 * S1 유니버스 데이터 조회 (Brain이 생성한 데이터)
 */
export async function getUniverse(market: MarketType = 'ALL'): Promise<UniverseResponse> {
  const response = await fetch(`${API_BASE}/api/v1/pipeline/universe?market=${market}`)
  if (!response.ok) {
    throw new Error(`Failed to fetch universe: ${response.status}`)
  }
  return response.json()
}

// S2: Signal 데이터
export interface SignalItem {
  stockCode: string
  stockName: string
  market: string
  calcDate: string
  momentum: number
  technical: number
  value: number
  quality: number
  flow: number
  event: number
  totalScore: number
}

export interface SignalsResponse {
  success: boolean
  data: {
    date: string
    market: string
    count: number
    items: SignalItem[]
  }
}

export interface RankedItem {
  stockCode: string
  stockName: string
  market: string
  rank: number
  totalScore: number
  momentum: number
  technical: number
  value: number
  quality: number
  flow: number
  event: number
  currentPrice: number
  changeRate: number
}

export interface PipelineRankingResponse {
  success: boolean
  data: {
    date: string
    market: string
    count: number
    items: RankedItem[]
  }
}

export interface PortfolioItem {
  stockCode: string
  stockName: string
  market: string
  weight: number
  targetQty: number
  action: string
  reason: string
  currentPrice: number
  changeRate: number
}

export interface PortfolioResponse {
  success: boolean
  data: {
    date: string
    count: number
    totalWeight: number
    cash: number
    positions: PortfolioItem[]
  }
}

// S3: Screened 데이터 (Hard Cut 통과)
export interface ScreenedItem {
  stockCode: string
  stockName: string
  market: string
  calcDate: string
  momentum: number
  technical: number
  value: number
  quality: number
  flow: number
  event: number
  totalScore: number
  passedAll: boolean
}

export interface ScreenedResponse {
  success: boolean
  data: {
    date: string
    market: string
    count: number
    hardCutConditions: {
      momentum: string
      technical: string
      flow: string
    }
    items: ScreenedItem[]
  }
}

/**
 * S2 신호 데이터 조회
 */
export async function getSignals(market: MarketType = 'KOSPI'): Promise<SignalsResponse> {
  const response = await fetch(`${API_BASE}/api/v1/pipeline/signals?market=${market}`)
  if (!response.ok) {
    throw new Error(`Failed to fetch signals: ${response.status}`)
  }
  return response.json()
}

/**
 * S4 랭킹 데이터 조회 (신호 점수 포함)
 */
export async function getPipelineRanking(market: MarketType = 'KOSPI'): Promise<PipelineRankingResponse> {
  const response = await fetch(`${API_BASE}/api/v1/pipeline/ranking?market=${market}`)
  if (!response.ok) {
    throw new Error(`Failed to fetch pipeline ranking: ${response.status}`)
  }
  return response.json()
}

/**
 * S5 포트폴리오 데이터 조회
 */
export async function getPortfolio(): Promise<PortfolioResponse> {
  const response = await fetch(`${API_BASE}/api/v1/pipeline/portfolio`)
  if (!response.ok) {
    throw new Error(`Failed to fetch portfolio: ${response.status}`)
  }
  return response.json()
}

/**
 * S3 스크리닝 데이터 조회 (Hard Cut 통과 종목)
 */
export async function getScreened(market: MarketType = 'ALL'): Promise<ScreenedResponse> {
  const response = await fetch(`${API_BASE}/api/v1/pipeline/screened?market=${market}`)
  if (!response.ok) {
    throw new Error(`Failed to fetch screened: ${response.status}`)
  }
  return response.json()
}

export const universeApi = {
  getRanking,
  getRankingStatus,
  getUniverse,
  getSignals,
  getScreened,
  getPipelineRanking,
  getPortfolio,
}
