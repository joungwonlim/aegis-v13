/**
 * Universe 훅
 * SSOT: 유니버스/랭킹 데이터 조회 훅
 */

import { useQuery } from '@tanstack/react-query'
import { universeApi, type MarketType } from './api'
import type { RankingCategory } from './types'

/**
 * 랭킹 데이터 조회 훅
 * @param category 랭킹 카테고리
 * @param market 마켓 (KOSPI, KOSDAQ, ALL)
 */
export function useRanking(
  category: RankingCategory = 'top',
  market: MarketType = 'ALL'
) {
  return useQuery({
    queryKey: ['ranking', category, market],
    queryFn: () => universeApi.getRanking(category, market),
    staleTime: 60 * 1000, // 1분
  })
}

/**
 * 랭킹 상태 조회 훅
 */
export function useRankingStatus() {
  return useQuery({
    queryKey: ['ranking', 'status'],
    queryFn: () => universeApi.getRankingStatus(),
    staleTime: 5 * 60 * 1000, // 5분
  })
}

// ============================================
// Pipeline Hooks (S1-S5 실제 데이터)
// ============================================

/**
 * S1 유니버스 데이터 조회 훅 (Brain이 생성한 데이터)
 */
export function useUniverse(market: MarketType = 'ALL') {
  return useQuery({
    queryKey: ['pipeline', 'universe', market],
    queryFn: () => universeApi.getUniverse(market),
    staleTime: 60 * 1000, // 1분
  })
}

/**
 * S2 신호 데이터 조회 훅
 */
export function useSignals(market: MarketType = 'KOSPI') {
  return useQuery({
    queryKey: ['pipeline', 'signals', market],
    queryFn: () => universeApi.getSignals(market),
    staleTime: 60 * 1000, // 1분
  })
}

/**
 * S3 스크리닝 데이터 조회 훅 (Hard Cut 통과 종목)
 */
export function useScreened(market: MarketType = 'ALL') {
  return useQuery({
    queryKey: ['pipeline', 'screened', market],
    queryFn: () => universeApi.getScreened(market),
    staleTime: 60 * 1000, // 1분
  })
}

/**
 * S4 파이프라인 랭킹 조회 훅 (신호 점수 포함)
 */
export function usePipelineRanking(market: MarketType = 'KOSPI') {
  return useQuery({
    queryKey: ['pipeline', 'ranking', market],
    queryFn: () => universeApi.getPipelineRanking(market),
    staleTime: 60 * 1000, // 1분
  })
}

/**
 * S5 포트폴리오 조회 훅
 */
export function usePortfolio() {
  return useQuery({
    queryKey: ['pipeline', 'portfolio'],
    queryFn: () => universeApi.getPortfolio(),
    staleTime: 60 * 1000, // 1분
  })
}
