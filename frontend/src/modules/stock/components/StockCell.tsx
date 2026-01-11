'use client'

/**
 * StockCell - 종목명 + 로고 + 상태점 표시
 * SSOT: 종목 정보 셀 렌더링은 이 컴포넌트에서만
 *
 * clickable=true 시 클릭하면 전역 StockDetailSheet가 열림
 */

import { useState, useCallback } from 'react'
import { cn } from '@/shared/lib/utils'
import type { SizeVariant } from '../types'
import { useStockDetail } from '../hooks/useStockDetail'

interface StockCellProps {
  code: string
  name?: string
  market?: string  // KOSPI, KOSDAQ 등
  size?: SizeVariant
  isHolding?: boolean
  isExitMonitoring?: boolean
  /** 클릭 시 StockDetailSheet 자동 열기 (기본값: true) */
  clickable?: boolean
  /** 커스텀 클릭 핸들러 (clickable보다 우선) */
  onClick?: (stock: { code: string; name: string; market?: string }) => void
  className?: string
}

const sizeConfig = {
  sm: { image: 'w-5 h-5', name: 'text-xs', code: 'text-[10px]' },
  md: { image: 'w-6 h-6', name: 'text-sm', code: 'text-xs' },
  lg: { image: 'w-8 h-8', name: 'text-base', code: 'text-sm' },
}

export function StockCell({
  code,
  name,
  market,
  size = 'md',
  isHolding = false,
  isExitMonitoring = false,
  clickable = true,
  onClick,
  className,
}: StockCellProps) {
  const [imageError, setImageError] = useState(false)
  const { openStockDetail } = useStockDetail()
  const config = sizeConfig[size]
  const displayName = name || code

  // 네이버 로고 URL
  const logoUrl = `https://ssl.pstatic.net/imgstock/fn/real/logo/stock/Stock${code}.svg`

  // 클릭 핸들러
  const handleClick = useCallback(() => {
    const stockInfo = { code, name: displayName, market }
    if (onClick) {
      onClick(stockInfo)
    } else if (clickable) {
      openStockDetail(stockInfo)
    }
  }, [code, displayName, market, onClick, clickable, openStockDetail])

  const isClickable = clickable || !!onClick

  return (
    <div
      className={cn(
        'flex items-center gap-2.5',
        isClickable && 'cursor-pointer hover:opacity-80 transition-opacity',
        className
      )}
      onClick={isClickable ? handleClick : undefined}
    >
      {/* 로고 */}
      {!imageError ? (
        <img
          src={logoUrl}
          alt={displayName}
          className={cn(config.image, 'rounded-full object-cover flex-shrink-0')}
          onError={() => setImageError(true)}
        />
      ) : (
        <div
          className={cn(
            config.image,
            'rounded-full bg-muted flex items-center justify-center text-[10px] text-muted-foreground font-medium'
          )}
        >
          {displayName.charAt(0)}
        </div>
      )}

      {/* 종목명 + 코드 */}
      <div className="flex flex-col min-w-0">
        <div className="flex items-center gap-1">
          <span className={cn('font-medium truncate', config.name)}>
            {displayName}
          </span>
          {/* 상태 점 표시 */}
          {isHolding && (
            <span
              className={cn(
                'w-1.5 h-1.5 rounded-full flex-shrink-0',
                isExitMonitoring ? 'bg-red-500' : 'bg-green-500'
              )}
              title={isExitMonitoring ? '자동청산 모니터링' : '보유 종목'}
            />
          )}
        </div>
        <div className="flex items-center gap-1">
          <span className={cn('text-muted-foreground', config.code)}>
            {code}
          </span>
          {market && (
            <span className={cn('text-muted-foreground', config.code)}>
              · {market}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}
