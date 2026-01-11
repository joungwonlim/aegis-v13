'use client'

/**
 * PageHeader - 페이지 상단 헤더
 * SSOT: 페이지 제목, 설명, 액션 버튼 영역
 *
 * 사용 예시:
 * <PageHeader
 *   title="Portfolio"
 *   description="보유 종목 현황"
 *   actions={<Button>새로고침</Button>}
 * />
 */

import { cn } from '@/shared/lib/utils'
import type { ReactNode } from 'react'

interface PageHeaderProps {
  /** 페이지 제목 */
  title: string
  /** 페이지 설명 (선택) */
  description?: string
  /** 우측 액션 버튼 영역 */
  actions?: ReactNode
  /** 제목 옆 뱃지/태그 등 */
  badge?: ReactNode
  className?: string
}

export function PageHeader({
  title,
  description,
  actions,
  badge,
  className,
}: PageHeaderProps) {
  return (
    <div className={cn('flex items-center justify-between mb-6', className)}>
      <div className="space-y-1">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold tracking-tight">{title}</h1>
          {badge}
        </div>
        {description && (
          <p className="text-sm text-muted-foreground">{description}</p>
        )}
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  )
}
