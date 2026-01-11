/**
 * Portfolio Hooks
 * SSOT: 모든 포트폴리오 관련 hooks
 * KIS API 연동
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { portfolioApi } from '../api'

// Query Keys
export const portfolioKeys = {
  all: ['portfolio'] as const,
  balance: () => [...portfolioKeys.all, 'balance'] as const,
  positions: () => [...portfolioKeys.all, 'positions'] as const,
  target: (date?: string) => [...portfolioKeys.all, 'target', date] as const,
}

// KIS 계좌 잔고 조회 (잔고 + 보유종목)
export function useBalance() {
  return useQuery({
    queryKey: portfolioKeys.balance(),
    queryFn: () => portfolioApi.getBalance(),
    refetchInterval: 30 * 1000, // 30초마다 갱신
    staleTime: 10 * 1000, // 10초간 fresh
  })
}

// KIS 보유 종목만 조회
export function usePositions() {
  return useQuery({
    queryKey: portfolioKeys.positions(),
    queryFn: () => portfolioApi.getPositions(),
    refetchInterval: 30 * 1000,
    staleTime: 10 * 1000,
  })
}

// 목표 포트폴리오 조회
export function useTargetPortfolio(date?: string) {
  return useQuery({
    queryKey: portfolioKeys.target(date),
    queryFn: () => portfolioApi.getTarget(date),
    select: (response) => response.data,
  })
}

// 포트폴리오 구성 실행
export function useConstructPortfolio() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: portfolioApi.construct,
    onSuccess: () => {
      // 관련 쿼리 무효화
      queryClient.invalidateQueries({ queryKey: portfolioKeys.all })
    },
  })
}

// 청산 모니터링 설정 변경
export function useUpdateExitMonitoring() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ stockCode, enabled }: { stockCode: string; enabled: boolean }) => {
      console.log('[Exit Monitoring] Calling API:', { stockCode, enabled })
      return portfolioApi.updateExitMonitoring(stockCode, enabled)
    },
    onSuccess: (data) => {
      console.log('[Exit Monitoring] Success:', data)
      // positions 쿼리 강제 refetch (staleTime 무시)
      queryClient.invalidateQueries({
        queryKey: portfolioKeys.positions(),
        refetchType: 'all',
      })
      queryClient.invalidateQueries({
        queryKey: portfolioKeys.balance(),
        refetchType: 'all',
      })
    },
    onError: (error) => {
      console.error('[Exit Monitoring] Error:', error)
    },
  })
}
