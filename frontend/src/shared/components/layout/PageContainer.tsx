'use client'

/**
 * PageContainer - 페이지 컨테이너 레이아웃
 * SSOT: 모든 페이지의 기본 컨테이너는 이 컴포넌트 사용
 *
 * 기본 스타일: container mx-auto px-6 py-6
 */

import { cn } from '@/shared/lib/utils'
import type { ReactNode } from 'react'

interface PageContainerProps {
  children: ReactNode
  className?: string
  /** 패딩 없이 full-width 컨텐츠 (테이블 등) */
  noPadding?: boolean
}

export function PageContainer({
  children,
  className,
  noPadding = false,
}: PageContainerProps) {
  return (
    <div
      className={cn(
        'container mx-auto',
        !noPadding && 'px-6 py-6',
        className
      )}
    >
      {children}
    </div>
  )
}
