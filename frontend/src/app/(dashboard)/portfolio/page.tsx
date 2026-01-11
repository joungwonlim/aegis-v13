'use client'

import { PortfolioSummary, PositionList } from '@/modules/portfolio/components'
import { useBalance, usePositions } from '@/modules/portfolio/hooks'

export default function PortfolioPage() {
  const { data: balanceData, isLoading: balanceLoading, error: balanceError } = useBalance()
  const { data: positionsData, isLoading: positionsLoading } = usePositions()

  const isLoading = balanceLoading || positionsLoading
  const error = balanceError

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">포트폴리오</h1>
        </div>
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-destructive">
            데이터를 불러오는데 실패했습니다. 백엔드 서버 연결을 확인해주세요.
          </p>
          <p className="text-sm text-muted-foreground mt-2">
            {error instanceof Error ? error.message : '알 수 없는 오류'}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">포트폴리오</h1>
      </div>

      <PortfolioSummary
        balance={balanceData?.balance}
        positions={positionsData?.positions}
        isLoading={isLoading}
      />

      <PositionList positions={positionsData?.positions} isLoading={isLoading} />
    </div>
  )
}
