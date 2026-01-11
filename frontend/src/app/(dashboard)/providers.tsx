'use client'

import type { ReactNode } from 'react'
import { StockDetailProvider, StockDetailSheet } from '@/modules/stock'

interface DashboardProvidersProps {
  children: ReactNode
}

/**
 * Dashboard 클라이언트 프로바이더
 * StockDetailProvider 등 클라이언트 컨텍스트를 제공
 */
export function DashboardProviders({ children }: DashboardProvidersProps) {
  return (
    <StockDetailProvider>
      {children}
      {/* StockDetailSheet는 전역적으로 사용 가능하도록 여기에 배치 */}
      <StockDetailSheet />
    </StockDetailProvider>
  )
}
