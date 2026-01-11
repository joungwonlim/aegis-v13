'use client'

/**
 * PositionList - 보유 종목 리스트
 * StockDataTable 기반으로 리팩토링
 */

import { useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/card'
import { cn } from '@/shared/lib/utils'
import {
  StockDataTable,
  PriceCell,
  type StockDataColumn,
  type StockDataItem,
} from '@/modules/stock/components'
import type { KISPosition } from '../types'

interface PositionListProps {
  positions?: KISPosition[]
  isLoading?: boolean
  className?: string
}

// KISPosition을 StockDataItem으로 확장
interface PositionDataItem extends StockDataItem {
  availableQty?: number
  purchaseAmount?: number
  weight?: number
}

export function PositionList({
  positions,
  isLoading,
  className,
}: PositionListProps) {
  // 총 평가금액 계산 (비중 계산용)
  const totalEvaluation = useMemo(() => {
    return positions?.reduce((sum, p) => sum + p.eval_amount, 0) ?? 0
  }, [positions])

  // KISPosition → StockDataItem 변환
  const data: PositionDataItem[] = useMemo(() => {
    if (!positions) return []
    return positions.map((p) => ({
      code: p.stock_code,
      name: p.stock_name,
      market: p.market,
      price: p.current_price,
      quantity: p.quantity,
      availableQty: p.available_quantity,
      avgPrice: p.avg_buy_price,
      purchaseAmount: p.purchase_amount,
      evalAmount: p.eval_amount,
      profitLoss: p.profit_loss,
      profitLossRate: p.profit_loss_rate,
      weight: totalEvaluation > 0 ? (p.eval_amount / totalEvaluation) * 100 : 0,
    }))
  }, [positions, totalEvaluation])

  // 포트폴리오용 추가 컬럼 정의 (기본 컬럼: 순번, 종목명, 현재가, 전일대비)
  // Note: 보유종목 녹색점/청산모니터링 빨간점은 StockDataTable 내부에서 자동 처리
  const portfolioExtraColumns: StockDataColumn[] = useMemo(
    () => [
      {
        key: 'quantity',
        label: '보유수량',
        width: 'w-20',
        align: 'right',
        render: (item) => (
          <span className="font-mono">
            {item.quantity?.toLocaleString('ko-KR') ?? '-'}
          </span>
        ),
      },
      {
        key: 'availableQty',
        label: '매도가능',
        width: 'w-20',
        align: 'right',
        render: (item: PositionDataItem) => (
          <span className="font-mono text-muted-foreground">
            {item.availableQty?.toLocaleString('ko-KR') ?? '-'}
          </span>
        ),
      },
      {
        key: 'avgPrice',
        label: '평균매입가',
        width: 'w-24',
        align: 'right',
        render: (item) => <PriceCell price={item.avgPrice} size="sm" />,
      },
      {
        key: 'purchaseAmount',
        label: '매입금액',
        width: 'w-28',
        align: 'right',
        render: (item: PositionDataItem) => (
          <span className="font-mono">
            {item.purchaseAmount?.toLocaleString('ko-KR') ?? '-'}
          </span>
        ),
      },
      {
        key: 'evalAmount',
        label: '평가금액',
        width: 'w-28',
        align: 'right',
        render: (item) => (
          <span className="font-mono">
            {item.evalAmount?.toLocaleString('ko-KR') ?? '-'}
          </span>
        ),
      },
      {
        key: 'profitLoss',
        label: '평가손익',
        width: 'w-28',
        align: 'right',
        render: (item) => {
          const pl = item.profitLoss ?? 0
          return (
            <span
              className={cn(
                'font-mono',
                pl > 0 && 'text-positive',
                pl < 0 && 'text-negative'
              )}
            >
              {pl > 0 ? '+' : ''}
              {pl.toLocaleString('ko-KR')}
            </span>
          )
        },
      },
      {
        key: 'profitLossRate',
        label: '수익률',
        width: 'w-20',
        align: 'right',
        render: (item) => {
          const rate = item.profitLossRate ?? 0
          return (
            <span
              className={cn(
                'font-mono',
                rate > 0 && 'text-positive',
                rate < 0 && 'text-negative'
              )}
            >
              {rate > 0 ? '+' : ''}
              {rate.toFixed(2)}%
            </span>
          )
        },
      },
      {
        key: 'weight',
        label: '비중',
        width: 'w-16',
        align: 'right',
        render: (item: PositionDataItem) => (
          <span className="font-mono text-muted-foreground">
            {item.weight?.toFixed(1) ?? '0.0'}%
          </span>
        ),
      },
    ],
    []
  )

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>보유 종목</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="animate-pulse space-y-3">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="h-12 bg-muted rounded" />
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={className}>
      <CardHeader className="pb-2">
        <CardTitle>보유 종목 ({data.length})</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <StockDataTable
          data={data}
          extraColumns={portfolioExtraColumns}
          showIndex={false}
          emptyMessage="보유 종목이 없습니다"
        />
      </CardContent>
    </Card>
  )
}
