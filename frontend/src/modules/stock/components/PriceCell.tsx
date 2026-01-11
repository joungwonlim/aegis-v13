'use client'

/**
 * PriceCell - 현재가 표시
 * SSOT: 가격 표시 셀은 이 컴포넌트에서만
 */

import { cn } from '@/shared/lib/utils'
import type { SizeVariant } from '../types'

interface PriceCellProps {
  price?: number
  size?: SizeVariant
  className?: string
}

const sizeConfig = {
  sm: 'text-sm',
  md: 'text-base',
  lg: 'text-lg',
}

export function PriceCell({ price, size = 'md', className }: PriceCellProps) {
  if (price === undefined || price === null || price === 0) {
    return <span className={cn('text-muted-foreground', className)}>-</span>
  }

  return (
    <span className={cn('font-mono font-medium', sizeConfig[size], className)}>
      {price.toLocaleString('ko-KR')}
    </span>
  )
}
