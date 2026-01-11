'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/card'
import { cn } from '@/shared/lib/utils'
import type { KISBalance, KISPosition } from '../types'

interface PortfolioSummaryProps {
  balance?: KISBalance
  positions?: KISPosition[]
  isLoading?: boolean
  className?: string
}

export function PortfolioSummary({
  balance,
  positions,
  isLoading,
  className,
}: PortfolioSummaryProps) {
  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>포트폴리오 요약</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="animate-pulse space-y-4">
            <div className="h-8 bg-muted rounded w-1/2" />
            <div className="h-4 bg-muted rounded w-1/3" />
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!balance) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>포트폴리오 요약</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">데이터가 없습니다</p>
        </CardContent>
      </Card>
    )
  }

  const positionCount = positions?.length ?? 0
  // 총자산 = 주문가능금액 + 평가금액 (실제 사용 가능한 자산)
  const totalAsset = balance.available_cash + balance.total_evaluation

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>포트폴리오 요약</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          {/* Row 1: 총자산, 총평가금액, 주문가능금액, 평가손익 */}

          {/* 총자산 */}
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">총자산</p>
            <p className="text-2xl font-mono font-bold">
              {totalAsset.toLocaleString('ko-KR')}
              <span className="text-sm font-normal text-muted-foreground ml-1">원</span>
            </p>
          </div>

          {/* 총 평가금액 */}
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">총 평가금액</p>
            <p className="text-2xl font-mono font-bold">
              {balance.total_evaluation.toLocaleString('ko-KR')}
              <span className="text-sm font-normal text-muted-foreground ml-1">원</span>
            </p>
          </div>

          {/* 주문가능금액 */}
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">주문가능금액</p>
            <p className="text-2xl font-mono font-bold">
              {balance.available_cash.toLocaleString('ko-KR')}
              <span className="text-sm font-normal text-muted-foreground ml-1">원</span>
            </p>
          </div>

          {/* 평가손익 */}
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">평가손익</p>
            <p
              className={cn(
                'text-2xl font-mono font-bold',
                balance.total_profit_loss > 0 && 'text-positive',
                balance.total_profit_loss < 0 && 'text-negative'
              )}
            >
              {balance.total_profit_loss > 0 ? '+' : ''}
              {balance.total_profit_loss.toLocaleString('ko-KR')}
              <span className="text-sm font-normal text-muted-foreground ml-1">원</span>
            </p>
          </div>

          {/* Row 2: 보유종목, 총매수금액, 예수금, 손익율 */}

          {/* 보유 종목 수 */}
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">보유 종목</p>
            <p className="text-2xl font-mono font-bold">
              {positionCount}
              <span className="text-sm font-normal text-muted-foreground ml-1">종목</span>
            </p>
          </div>

          {/* 총 매수금액 */}
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">총 매수금액</p>
            <p className="text-2xl font-mono font-bold">
              {balance.total_purchase.toLocaleString('ko-KR')}
              <span className="text-sm font-normal text-muted-foreground ml-1">원</span>
            </p>
          </div>

          {/* 예수금 */}
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">예수금</p>
            <p className="text-2xl font-mono font-bold">
              {balance.total_deposit.toLocaleString('ko-KR')}
              <span className="text-sm font-normal text-muted-foreground ml-1">원</span>
            </p>
          </div>

          {/* 손익율 */}
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">손익율</p>
            <p
              className={cn(
                'text-2xl font-mono font-bold',
                balance.profit_loss_rate > 0 && 'text-positive',
                balance.profit_loss_rate < 0 && 'text-negative'
              )}
            >
              {balance.profit_loss_rate > 0 ? '+' : ''}
              {balance.profit_loss_rate.toFixed(2)}
              <span className="text-sm font-normal text-muted-foreground ml-1">%</span>
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
