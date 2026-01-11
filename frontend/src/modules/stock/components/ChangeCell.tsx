'use client'

/**
 * ChangeCell - 전일대비 표시
 * SSOT: 등락 표시 셀은 이 컴포넌트에서만
 * 형식: ▲ 2 (+0.96%) 또는 ▼ 29 (-3.07%)
 */

import { cn } from '@/shared/lib/utils'
import type { SizeVariant } from '../types'

interface ChangeCellProps {
  change?: number
  changeRate?: number
  size?: SizeVariant
  showIcon?: boolean
  className?: string
}

const sizeConfig = {
  sm: 'text-sm',
  md: 'text-base',
  lg: 'text-lg',
}

export function ChangeCell({
  change,
  changeRate,
  size = 'md',
  showIcon = true,
  className,
}: ChangeCellProps) {
  if (
    (change === undefined || change === null) &&
    (changeRate === undefined || changeRate === null)
  ) {
    return <span className={cn('text-muted-foreground', className)}>-</span>
  }

  const displayChange = change ?? 0
  const displayRate = changeRate ?? 0
  const isPositive = displayChange > 0
  const isNegative = displayChange < 0
  const icon = isPositive ? '▲' : isNegative ? '▼' : ''

  return (
    <span
      className={cn(
        'font-mono font-medium whitespace-nowrap',
        isPositive && 'text-positive',
        isNegative && 'text-negative',
        !isPositive && !isNegative && 'text-neutral',
        sizeConfig[size],
        className
      )}
    >
      {showIcon && icon && <span className="mr-1">{icon}</span>}
      {Math.abs(displayChange).toLocaleString('ko-KR')}
      {displayRate !== 0 && (
        <span className="ml-1">
          ({isPositive ? '+' : isNegative ? '-' : ''}
          {Math.abs(displayRate).toFixed(2)}%)
        </span>
      )}
    </span>
  )
}
