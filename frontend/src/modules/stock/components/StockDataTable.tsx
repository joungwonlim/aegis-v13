'use client'

/**
 * StockDataTable - 종목 데이터 통합 테이블
 * SSOT: 모든 종목 리스트 테이블은 이 컴포넌트 기반
 *
 * 기본 컬럼 (항상 표시): 순번, 종목명, 현재가, 전일대비
 * 추가 컬럼: extraColumns prop으로 전달
 *
 * 실시간 가격: useRealtimePrices 훅으로 자동 갱신
 * 정렬: 헤더 클릭으로 정렬 (종목명, 현재가, 전일대비)
 */

import { useMemo, useState } from 'react'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/shared/components/ui/table'
import { Button } from '@/shared/components/ui/button'
import { cn } from '@/shared/lib/utils'
import { StockCell } from './StockCell'
import { PriceCell } from './PriceCell'
import { ChangeCell } from './ChangeCell'
import { Trash2, ChevronUp, ChevronDown, LogOut } from 'lucide-react'
import { useRealtimePrices } from '@/modules/price'
import { usePositions, useUpdateExitMonitoring } from '@/modules/portfolio'

type SortDirection = 'asc' | 'desc'
type SortColumn = string | null  // 모든 컬럼 키를 지원

export interface StockDataItem {
  code: string
  name?: string
  market?: string  // KOSPI, KOSDAQ 등
  price?: number
  change?: number
  changeRate?: number
  // 확장 가능한 필드들
  quantity?: number
  avgPrice?: number
  profitLoss?: number
  profitLossRate?: number
  evalAmount?: number
  score?: number
  rank?: number
  volume?: number
  marketCap?: number
  /** React 렌더링용 고유 키 (market+code 등) */
  uniqueKey?: string
  [key: string]: unknown
}

export interface StockDataColumn {
  key: string
  label: string
  align?: 'left' | 'center' | 'right'
  width?: string
  render?: (item: StockDataItem, index: number) => React.ReactNode
  /** 정렬 가능 여부 (숫자 컬럼 권장) */
  sortable?: boolean
  /** 합계 행에서 사용할 집계 방식 */
  aggregation?: 'sum' | 'avg' | 'count'
}

interface StockDataTableProps {
  data: StockDataItem[]
  extraColumns?: StockDataColumn[]  // 추가 컬럼 (기본 컬럼 뒤에 표시)
  showIndex?: boolean
  showSummary?: boolean  // 합계 행 표시 여부
  onRowClick?: (item: StockDataItem) => void
  onDelete?: (code: string) => void
  emptyMessage?: string
  className?: string
}

export function StockDataTable({
  data,
  extraColumns = [],
  showIndex = true,
  showSummary = false,
  onRowClick,
  onDelete,
  emptyMessage = '종목이 없습니다',
  className,
}: StockDataTableProps) {
  // 정렬 상태
  const [sortColumn, setSortColumn] = useState<SortColumn>(null)
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc')

  // 모든 종목 코드 추출
  const stockCodes = useMemo(() => data.map((item) => item.code), [data])

  // 실시간 가격 조회 (3초 간격)
  const { data: realtimePrices } = useRealtimePrices(stockCodes, {
    enabled: stockCodes.length > 0,
    refetchInterval: 3000,
  })

  // 포트폴리오 데이터 자동 조회 (내부에서 처리)
  const { data: positionsData } = usePositions()
  const updateExitMonitoring = useUpdateExitMonitoring()

  // 보유종목 코드 Set (자동 생성)
  const holdingCodes = useMemo(() => {
    const codes = positionsData?.positions?.map((p) => p.stock_code) ?? []
    return new Set(codes)
  }, [positionsData])

  // 청산 모니터링 종목 코드 Set (자동 생성)
  const exitMonitoringCodes = useMemo(() => {
    if (!positionsData?.positions) return new Set<string>()
    return new Set(
      positionsData.positions
        .filter((p) => p.exit_monitoring_enabled)
        .map((p) => p.stock_code)
    )
  }, [positionsData])

  // Exit 토글 핸들러 (내부 처리) - stockCode 기반
  const handleToggleExitMonitoring = (code: string) => {
    // 보유종목인 경우에만 exit monitoring 토글 가능
    if (holdingCodes.has(code)) {
      const isCurrentlyEnabled = exitMonitoringCodes.has(code)
      updateExitMonitoring.mutate({ stockCode: code, enabled: !isCurrentlyEnabled })
    }
  }

  // 정렬된 데이터
  const sortedData = useMemo(() => {
    if (!sortColumn) return data

    return [...data].sort((a, b) => {
      let aValue: string | number | undefined
      let bValue: string | number | undefined

      switch (sortColumn) {
        case 'name':
          aValue = a.name || a.code
          bValue = b.name || b.code
          break
        case 'price':
          aValue = realtimePrices?.[a.code]?.price ?? a.price ?? 0
          bValue = realtimePrices?.[b.code]?.price ?? b.price ?? 0
          break
        case 'changeRate':
          aValue = realtimePrices?.[a.code]?.change_rate ?? a.changeRate ?? 0
          bValue = realtimePrices?.[b.code]?.change_rate ?? b.changeRate ?? 0
          break
        default:
          // extraColumns에서 값 가져오기
          aValue = a[sortColumn] as string | number | undefined
          bValue = b[sortColumn] as string | number | undefined
          break
      }

      // null/undefined 처리
      if (aValue == null) return 1
      if (bValue == null) return -1

      // 비교
      if (typeof aValue === 'number' && typeof bValue === 'number') {
        return sortDirection === 'asc' ? aValue - bValue : bValue - aValue
      }

      const comparison = String(aValue).localeCompare(String(bValue), 'ko')
      return sortDirection === 'asc' ? comparison : -comparison
    })
  }, [data, sortColumn, sortDirection, realtimePrices])

  // 합계 계산
  const summaryData = useMemo(() => {
    if (!showSummary) return null

    const summary: Record<string, number> = {}

    extraColumns.forEach((col) => {
      if (col.aggregation) {
        const values = data
          .map((item) => item[col.key])
          .filter((v): v is number => typeof v === 'number')

        if (values.length > 0) {
          switch (col.aggregation) {
            case 'sum':
              summary[col.key] = values.reduce((a, b) => a + b, 0)
              break
            case 'avg':
              summary[col.key] = values.reduce((a, b) => a + b, 0) / values.length
              break
            case 'count':
              summary[col.key] = values.length
              break
          }
        }
      }
    })

    return summary
  }, [data, extraColumns, showSummary])

  // 정렬 핸들러
  const handleSort = (column: SortColumn) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortColumn(column)
      setSortDirection('asc')
    }
  }

  // 정렬 아이콘
  const SortIcon = ({ column }: { column: SortColumn }) => {
    if (sortColumn !== column) return null
    return sortDirection === 'asc' ? (
      <ChevronUp className="w-3 h-3 inline-block ml-1" />
    ) : (
      <ChevronDown className="w-3 h-3 inline-block ml-1" />
    )
  }

  if (data.length === 0) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        {emptyMessage}
      </div>
    )
  }

  const getAlignment = (align?: 'left' | 'center' | 'right') => {
    switch (align) {
      case 'center':
        return 'text-center'
      case 'right':
        return 'text-right'
      default:
        return 'text-left'
    }
  }

  // 보유종목이 테이블에 있는지 확인 (Exit 컬럼 표시 여부)
  const hasHoldingStocks = sortedData.some((item) => holdingCodes.has(item.code))

  return (
    <Table className={className}>
      <TableHeader>
        <TableRow>
          {/* 기본 컬럼: 순번 */}
          {showIndex && (
            <TableHead className="w-14 text-center">순번</TableHead>
          )}
          {/* 기본 컬럼: 종목명 (정렬 가능) */}
          <TableHead
            className="w-40 cursor-pointer hover:text-foreground transition-colors select-none"
            onClick={() => handleSort('name')}
          >
            종목명
            <SortIcon column="name" />
          </TableHead>
          {/* 기본 컬럼: 현재가 (정렬 가능) */}
          <TableHead
            className="w-28 text-right cursor-pointer hover:text-foreground transition-colors select-none"
            onClick={() => handleSort('price')}
          >
            현재가
            <SortIcon column="price" />
          </TableHead>
          {/* 기본 컬럼: 전일대비 (정렬 가능) */}
          <TableHead
            className="w-36 text-right cursor-pointer hover:text-foreground transition-colors select-none"
            onClick={() => handleSort('changeRate')}
          >
            전일대비
            <SortIcon column="changeRate" />
          </TableHead>
          {/* 추가 컬럼 */}
          {extraColumns.map((column) => (
            <TableHead
              key={column.key}
              className={cn(
                column.width,
                getAlignment(column.align),
                column.sortable && 'cursor-pointer hover:text-foreground transition-colors select-none'
              )}
              onClick={column.sortable ? () => handleSort(column.key) : undefined}
            >
              {column.label}
              {column.sortable && <SortIcon column={column.key} />}
            </TableHead>
          ))}
          {/* Exit 토글 버튼 (보유종목이 있을 때만) */}
          {hasHoldingStocks && <TableHead className="w-12" />}
          {/* 삭제 버튼 */}
          {onDelete && <TableHead className="w-12" />}
        </TableRow>
      </TableHeader>
      <TableBody>
        {sortedData.map((item, index) => (
          <TableRow
            key={item.uniqueKey ?? item.code}
            className={cn(onRowClick && 'cursor-pointer')}
            onClick={() => onRowClick?.(item)}
          >
            {/* 기본 컬럼: 순번 */}
            {showIndex && (
              <TableCell className="text-center text-muted-foreground font-mono">
                {item.rank ?? index + 1}
              </TableCell>
            )}
            {/* 기본 컬럼: 종목명 */}
            <TableCell>
              <StockCell
                code={item.code}
                name={item.name}
                market={item.market}
                size="sm"
                isHolding={holdingCodes.has(item.code)}
                isExitMonitoring={exitMonitoringCodes.has(item.code)}
              />
            </TableCell>
            {/* 기본 컬럼: 현재가 (실시간 우선, 폴백으로 정적 값) */}
            <TableCell className="text-right">
              <PriceCell
                price={realtimePrices?.[item.code]?.price ?? item.price}
                size="sm"
              />
            </TableCell>
            {/* 기본 컬럼: 전일대비 (실시간 우선, 폴백으로 정적 값) */}
            <TableCell className="text-right">
              <ChangeCell
                change={realtimePrices?.[item.code]?.change ?? item.change}
                changeRate={realtimePrices?.[item.code]?.change_rate ?? item.changeRate}
                size="sm"
              />
            </TableCell>
            {/* 추가 컬럼 */}
            {extraColumns.map((column) => (
              <TableCell
                key={column.key}
                className={cn(getAlignment(column.align))}
              >
                {column.render ? column.render(item, index) : null}
              </TableCell>
            ))}
            {/* Exit 토글 버튼 (보유종목만) */}
            {hasHoldingStocks && holdingCodes.has(item.code) && (
              <TableCell>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleToggleExitMonitoring(item.code)
                  }}
                  title={exitMonitoringCodes.has(item.code) ? '자동청산 비활성화' : '자동청산 활성화'}
                >
                  <LogOut
                    className={cn(
                      'h-4 w-4',
                      exitMonitoringCodes.has(item.code)
                        ? 'text-red-500'
                        : 'text-muted-foreground'
                    )}
                  />
                </Button>
              </TableCell>
            )}
            {hasHoldingStocks && !holdingCodes.has(item.code) && (
              <TableCell />
            )}
            {/* 삭제 버튼 */}
            {onDelete && (
              <TableCell>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={(e) => {
                    e.stopPropagation()
                    onDelete(item.code)
                  }}
                >
                  <Trash2 className="h-4 w-4 text-muted-foreground" />
                </Button>
              </TableCell>
            )}
          </TableRow>
        ))}
        {/* 합계 행 */}
        {showSummary && summaryData && (
          <TableRow className="bg-muted/50 font-medium border-t-2">
            {showIndex && (
              <TableCell className="text-center text-muted-foreground">-</TableCell>
            )}
            <TableCell className="font-semibold">합계 / 평균</TableCell>
            <TableCell className="text-right">-</TableCell>
            <TableCell className="text-right">-</TableCell>
            {extraColumns.map((column) => (
              <TableCell
                key={column.key}
                className={cn(getAlignment(column.align), 'font-mono')}
              >
                {summaryData[column.key] !== undefined
                  ? column.aggregation === 'avg'
                    ? summaryData[column.key].toFixed(2)
                    : summaryData[column.key].toFixed(2)
                  : '-'}
              </TableCell>
            ))}
            {hasHoldingStocks && <TableCell />}
            {onDelete && <TableCell />}
          </TableRow>
        )}
      </TableBody>
    </Table>
  )
}
