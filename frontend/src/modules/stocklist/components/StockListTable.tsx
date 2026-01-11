'use client'

/**
 * StockListTable - 관심종목 테이블
 * SSOT: 관심종목 표시는 이 컴포넌트에서만
 *
 * StockDataTable을 래핑하고 다음 기능 추가:
 * - 실시간 가격 자동 연동
 * - 포트폴리오(보유종목) 자동 표시
 * - 카테고리별 필터링
 */

import { useMemo } from 'react'
import { StockDataTable, useStockDetail } from '@/modules/stock'
import type { StockDataItem, StockDataColumn } from '@/modules/stock'
import { useRealtimePrices } from '@/modules/price'
import type { StockListItem } from '../types'

interface StockListTableProps {
  items: StockListItem[]
  /** 추가 컬럼 */
  extraColumns?: StockDataColumn[]
  /** 삭제 콜백 */
  onDelete?: (code: string) => void
  /** 빈 메시지 */
  emptyMessage?: string
  className?: string
}

export function StockListTable({
  items,
  extraColumns = [],
  onDelete,
  emptyMessage = '관심종목이 없습니다',
  className,
}: StockListTableProps) {
  const { openStockDetail } = useStockDetail()

  // 종목 코드 목록 추출
  const stockCodes = useMemo(
    () => items.map((item) => item.stock_code),
    [items]
  )

  // 실시간 가격 조회
  const { data: prices } = useRealtimePrices(stockCodes, {
    enabled: stockCodes.length > 0,
    refetchInterval: 5000,
  })

  // StockDataItem 형식으로 변환 (실시간 가격 병합)
  const tableData: StockDataItem[] = useMemo(() => {
    return items.map((item) => {
      const realtimePrice = prices?.[item.stock_code]
      return {
        code: item.stock_code,
        name: item.stock_name ?? item.stock_code,
        market: item.market,
        // 실시간 가격 우선, 없으면 저장된 가격
        price: realtimePrice?.price ?? item.current_price,
        change: realtimePrice?.change,
        changeRate: realtimePrice?.change_rate ?? item.change_rate,
        // 원본 데이터 보존 (추가 컬럼용)
        alertEnabled: item.alert_enabled,
        category: item.category,
        createdAt: item.created_at,
      }
    })
  }, [items, prices])

  // 종목 클릭 핸들러
  const handleRowClick = (item: StockDataItem) => {
    openStockDetail({
      code: item.code,
      name: item.name ?? item.code,
    })
  }

  return (
    <StockDataTable
      data={tableData}
      extraColumns={extraColumns}
      onRowClick={handleRowClick}
      onDelete={onDelete}
      emptyMessage={emptyMessage}
      className={className}
    />
  )
}
