/**
 * useStockList Hook
 * SSOT: 관심종목 CRUD 훅
 */

'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { stocklistApi } from '../api'
import type { StockListCategory, CreateStockListRequest, UpdateStockListRequest } from '../types'

// Query Keys
export const stocklistKeys = {
  all: ['stocklist'] as const,
  list: () => [...stocklistKeys.all, 'list'] as const,
  watch: () => [...stocklistKeys.all, 'watch'] as const,
  candidate: () => [...stocklistKeys.all, 'candidate'] as const,
}

/**
 * 관심종목 전체 목록 조회 (watch + candidate)
 */
export function useStockList() {
  return useQuery({
    queryKey: stocklistKeys.list(),
    queryFn: stocklistApi.getList,
    staleTime: 30 * 1000, // 30초
  })
}

/**
 * Watch 목록만 조회
 */
export function useWatchList() {
  return useQuery({
    queryKey: stocklistKeys.watch(),
    queryFn: stocklistApi.getWatchList,
    staleTime: 30 * 1000,
  })
}

/**
 * Candidate 목록만 조회
 */
export function useCandidateList() {
  return useQuery({
    queryKey: stocklistKeys.candidate(),
    queryFn: stocklistApi.getCandidateList,
    staleTime: 30 * 1000,
  })
}

/**
 * 관심종목 추가
 */
export function useAddStock() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateStockListRequest) => stocklistApi.create(data),
    onSuccess: () => {
      // 전체 stocklist 캐시 무효화
      queryClient.invalidateQueries({ queryKey: stocklistKeys.all })
    },
  })
}

/**
 * 관심종목 수정 (카테고리 변경, 알림 설정 등)
 */
export function useUpdateStock() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateStockListRequest }) =>
      stocklistApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: stocklistKeys.all })
    },
  })
}

/**
 * 관심종목 삭제
 */
export function useDeleteStock() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => stocklistApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: stocklistKeys.all })
    },
  })
}

/**
 * 카테고리 간 이동 (watch <-> candidate)
 */
export function useMoveStock() {
  const updateStock = useUpdateStock()

  return {
    ...updateStock,
    mutate: (id: number, newCategory: StockListCategory) => {
      updateStock.mutate({ id, data: { category: newCategory } })
    },
    mutateAsync: (id: number, newCategory: StockListCategory) => {
      return updateStock.mutateAsync({ id, data: { category: newCategory } })
    },
  }
}
