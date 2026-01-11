'use client'

import { useState, useCallback, createContext, useContext, type ReactNode } from 'react'
import type { StockInfo } from '../types'

/**
 * StockDetail 상태 관리 Context
 * SSOT: 종목 상세 시트 상태 관리는 이 훅에서만
 */

interface StockDetailContextValue {
  /** 선택된 종목 정보 */
  selectedStock: StockInfo | null
  /** 시트 열림 상태 */
  isOpen: boolean
  /** 종목 상세 시트 열기 */
  openStockDetail: (stock: StockInfo) => void
  /** 종목 상세 시트 닫기 */
  closeStockDetail: () => void
  /** 시트 열림 상태 변경 핸들러 (Sheet의 onOpenChange용) */
  handleOpenChange: (open: boolean) => void
}

const StockDetailContext = createContext<StockDetailContextValue | null>(null)

interface StockDetailProviderProps {
  children: ReactNode
}

export function StockDetailProvider({ children }: StockDetailProviderProps) {
  const [selectedStock, setSelectedStock] = useState<StockInfo | null>(null)
  const [isOpen, setIsOpen] = useState(false)

  const openStockDetail = useCallback((stock: StockInfo) => {
    setSelectedStock(stock)
    setIsOpen(true)
  }, [])

  const closeStockDetail = useCallback(() => {
    setIsOpen(false)
    // 애니메이션이 끝난 후 selectedStock을 null로 설정
    setTimeout(() => setSelectedStock(null), 300)
  }, [])

  const handleOpenChange = useCallback((open: boolean) => {
    if (!open) {
      closeStockDetail()
    }
  }, [closeStockDetail])

  return (
    <StockDetailContext.Provider
      value={{
        selectedStock,
        isOpen,
        openStockDetail,
        closeStockDetail,
        handleOpenChange,
      }}
    >
      {children}
    </StockDetailContext.Provider>
  )
}

export function useStockDetail() {
  const context = useContext(StockDetailContext)
  if (!context) {
    throw new Error('useStockDetail must be used within a StockDetailProvider')
  }
  return context
}
