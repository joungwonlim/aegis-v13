'use client'

import { useState } from 'react'
import {
  ExternalLink,
  TrendingUp,
  TrendingDown,
  Star,
  LogOut,
} from 'lucide-react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/shared/components/ui/sheet'
import { Button } from '@/shared/components/ui/button'
import { Badge } from '@/shared/components/ui/badge'
import { cn } from '@/shared/lib/utils'
import { useStockDetail } from '../hooks/useStockDetail'
import { useDailyPrices } from '../hooks/useDailyPrices'
import { useInvestorTrading } from '../hooks/useInvestorTrading'
import { useStockDescription } from '../hooks/useStockDescription'
import { useRealtimePrices } from '@/modules/price'
import { useStockList, useAddStock, useDeleteStock } from '@/modules/stocklist'
import { usePositions, useUpdateExitMonitoring } from '@/modules/portfolio'
import { ForecastTab } from '@/modules/forecast'
import { PriceChart } from './PriceChart'
import { InvestorTradingChart } from './InvestorTradingChart'

/**
 * StockDetailSheet - 종목 상세 정보 사이드 시트
 * SSOT: 종목 상세 정보 표시는 이 컴포넌트에서만
 *
 * 전역적으로 사용되며, StockDetailProvider 내부에서 사용해야 함
 */

interface ExternalLinkItem {
  label: string
  href: string
}

export function StockDetailSheet() {
  const { selectedStock, isOpen, handleOpenChange } = useStockDetail()
  const [imageError, setImageError] = useState(false)

  // 실시간 가격
  const symbols = selectedStock ? [selectedStock.code] : []
  const { data: prices } = useRealtimePrices(symbols, {
    enabled: isOpen && !!selectedStock,
    refetchInterval: 3000,
  })

  // 일봉 데이터
  const { data: dailyData, isLoading: dailyLoading } = useDailyPrices(
    isOpen && selectedStock ? selectedStock.code : '',
    365
  )

  // 투자자별 매매동향
  const { data: investorData, isLoading: investorLoading } = useInvestorTrading(
    isOpen && selectedStock ? selectedStock.code : '',
    365
  )

  // 종목 상세 설명
  const { data: description } = useStockDescription(
    isOpen && selectedStock ? selectedStock.code : ''
  )

  // 관심종목 관련
  const { data: stocklistData } = useStockList()
  const addStock = useAddStock()
  const deleteStock = useDeleteStock()

  // 보유종목 조회 (평단가 표시용)
  const { data: positionsData } = usePositions()

  // 청산 모니터링 토글
  const updateExitMonitoring = useUpdateExitMonitoring()

  // selectedStock이 바뀔 때마다 imageError 리셋
  if (!selectedStock) {
    if (imageError) setImageError(false)
    return null
  }

  const { code, name, market } = selectedStock

  // 실시간 가격 정보
  const price = prices?.[code]
  const currentPrice = price?.price ?? 0
  const changeAmount = price?.change ?? 0
  const changeRate = price?.change_rate ?? 0
  const isPositive = changeRate >= 0

  // 네이버 로고 URL
  const logoUrl = `https://ssl.pstatic.net/imgstock/fn/real/logo/stock/Stock${code}.svg`

  // 관심종목 여부 확인
  const watchlistItem = stocklistData?.data
    ? [...(stocklistData.data.watch || []), ...(stocklistData.data.candidate || [])].find(
        (item) => item.stock_code === code
      )
    : null
  const isInWatchlist = !!watchlistItem

  // 보유종목 여부 확인 및 평단가 추출
  const position = positionsData?.positions?.find((p) => p.stock_code === code)
  const avgBuyPrice = position?.avg_buy_price
  const isExitMonitoringEnabled = position?.exit_monitoring_enabled ?? false

  // 청산 모니터링 토글 - stockCode 기반
  const handleToggleExitMonitoring = async () => {
    console.log('[Exit Toggle] position:', position)
    console.log('[Exit Toggle] stock_code:', code)
    console.log('[Exit Toggle] isExitMonitoringEnabled:', isExitMonitoringEnabled)
    if (!position) {
      console.log('[Exit Toggle] No position, returning early')
      return
    }
    console.log('[Exit Toggle] Calling mutate with:', {
      stockCode: code,
      enabled: !isExitMonitoringEnabled,
    })
    updateExitMonitoring.mutate({
      stockCode: code,
      enabled: !isExitMonitoringEnabled,
    })
  }

  // 관심종목 토글
  const handleToggleWatchlist = async () => {
    if (isInWatchlist && watchlistItem) {
      deleteStock.mutate(watchlistItem.id)
    } else {
      addStock.mutate({
        stock_code: code,
        category: 'watch',
      })
    }
  }

  // 외부 링크
  const externalLinks: ExternalLinkItem[] = [
    {
      label: '네이버 증권',
      href: `https://finance.naver.com/item/main.naver?code=${code}`,
    },
    {
      label: '네이버 토론',
      href: `https://finance.naver.com/item/board.naver?code=${code}`,
    },
    {
      label: 'DART 공시',
      href: `https://dart.fss.or.kr/dsab001/main.do?autoSearch=true&textCrpNm=${encodeURIComponent(name)}`,
    },
    {
      label: '증권플러스',
      href: `https://m.stockplus.com/m/stocks/KOREA-A${code}/community`,
    },
  ]

  return (
    <Sheet open={isOpen} onOpenChange={handleOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-2xl overflow-y-auto p-0">
        {/* 헤더 */}
        <SheetHeader className="sticky top-0 z-10 bg-background border-b p-4">
          <div className="flex items-center gap-3">
            {/* 로고 */}
            {!imageError ? (
              <img
                src={logoUrl}
                alt={name}
                className="w-12 h-12 rounded-full object-cover flex-shrink-0"
                onError={() => setImageError(true)}
              />
            ) : (
              <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center text-lg text-muted-foreground font-medium">
                {name.charAt(0)}
              </div>
            )}

            <div className="flex flex-col flex-1 min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <SheetTitle className="text-xl truncate">{name}</SheetTitle>
                <span className="text-sm text-muted-foreground font-mono">{code}</span>
                {market && (
                  <Badge variant="secondary" className="text-xs">
                    {market}
                  </Badge>
                )}
                {/* 보유종목 표시 */}
                {position && (
                  <Badge variant="default" className="text-xs bg-green-600">
                    보유
                  </Badge>
                )}
                {/* 청산 모니터링 표시 */}
                {isExitMonitoringEnabled && (
                  <Badge variant="destructive" className="text-xs">
                    EXIT
                  </Badge>
                )}
              </div>

              {/* 종목 상세 설명 */}
              {description && (
                <p className="text-xs text-muted-foreground mt-1.5 leading-relaxed">
                  {description}
                </p>
              )}

              {/* 현재가 */}
              {currentPrice > 0 ? (
                <div className="flex items-baseline gap-2 mt-1">
                  <span
                    className={cn(
                      'text-2xl font-bold font-mono',
                      isPositive ? 'text-red-500' : 'text-blue-500'
                    )}
                  >
                    {currentPrice.toLocaleString()}
                  </span>
                  <span className="text-sm text-muted-foreground">원</span>
                  <div
                    className={cn(
                      'flex items-center gap-1 ml-2',
                      isPositive ? 'text-red-500' : 'text-blue-500'
                    )}
                  >
                    {isPositive ? (
                      <TrendingUp className="w-4 h-4" />
                    ) : (
                      <TrendingDown className="w-4 h-4" />
                    )}
                    <span className="text-sm font-mono">
                      {changeAmount >= 0 ? '+' : ''}
                      {changeAmount.toLocaleString()}
                    </span>
                    <span className="text-sm font-mono">
                      ({isPositive ? '+' : ''}
                      {changeRate.toFixed(2)}%)
                    </span>
                  </div>
                </div>
              ) : (
                <div className="animate-pulse mt-1">
                  <div className="h-8 bg-muted rounded w-40" />
                </div>
              )}
            </div>

            {/* 액션 버튼들 */}
            <div className="flex items-center gap-1">
              {/* 관심종목 버튼 */}
              <Button
                variant="ghost"
                size="icon"
                onClick={handleToggleWatchlist}
                disabled={addStock.isPending || deleteStock.isPending}
                title={isInWatchlist ? '관심종목 해제' : '관심종목 추가'}
              >
                <Star
                  className={cn(
                    'w-5 h-5 transition-colors',
                    isInWatchlist
                      ? 'fill-yellow-500 text-yellow-500'
                      : 'fill-transparent text-muted-foreground'
                  )}
                />
              </Button>

              {/* 청산 모니터링 버튼 - 보유종목만 활성화 */}
              <Button
                variant="ghost"
                size="icon"
                onClick={handleToggleExitMonitoring}
                disabled={!position || updateExitMonitoring.isPending}
                title={
                  !position
                    ? '보유종목만 청산 모니터링 설정 가능'
                    : isExitMonitoringEnabled
                      ? '청산 모니터링 비활성화 (클릭하면 자동 청산 중지)'
                      : '청산 모니터링 활성화 (클릭하면 Exit Rule 적용)'
                }
              >
                <LogOut
                  className={cn(
                    'w-5 h-5 transition-colors',
                    isExitMonitoringEnabled
                      ? 'text-red-500'
                      : 'text-muted-foreground'
                  )}
                />
              </Button>

              {/* 네이버 링크 */}
              <Button variant="ghost" size="icon" asChild>
                <a
                  href={`https://stock.naver.com/domestic/stock/${code}/price`}
                  target="_blank"
                  rel="noopener noreferrer"
                  title="네이버 증권"
                >
                  <ExternalLink className="w-4 h-4 text-muted-foreground" />
                </a>
              </Button>
            </div>
          </div>
        </SheetHeader>

        {/* 컨텐츠 영역 */}
        <div className="flex flex-col gap-4 p-4">
          {/* 일봉 차트 */}
          <PriceChart
            data={dailyData ?? []}
            isLoading={dailyLoading}
            avgBuyPrice={avgBuyPrice}
          />

          {/* 투자자별 매매동향 */}
          <InvestorTradingChart data={investorData ?? []} isLoading={investorLoading} />

          {/* Forecast 예측 */}
          <ForecastTab symbol={code} />

          {/* 외부 링크 */}
          <section>
            <h3 className="text-sm font-medium text-muted-foreground mb-3">외부 링크</h3>
            <div className="grid grid-cols-2 gap-2">
              {externalLinks.map((link) => (
                <Button
                  key={link.href}
                  variant="outline"
                  size="sm"
                  className="justify-start"
                  asChild
                >
                  <a href={link.href} target="_blank" rel="noopener noreferrer">
                    <ExternalLink className="w-3.5 h-3.5 mr-2" />
                    {link.label}
                  </a>
                </Button>
              ))}
            </div>
          </section>
        </div>
      </SheetContent>
    </Sheet>
  )
}
