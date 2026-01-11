'use client'

/**
 * AppHeader - 앱 상단 네비게이션 헤더
 * SSOT: 앱 전역 헤더는 이 컴포넌트 사용
 *
 * 포함 요소:
 * - 로고/브랜드
 * - 메인 네비게이션
 * - 우측 액션 (테마 토글 등)
 */

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/shared/lib/utils'
import { ThemeToggle } from './ThemeToggle'
import type { ReactNode } from 'react'

export interface NavItem {
  label: string
  href: string
}

interface AppHeaderProps {
  /** 브랜드/로고 텍스트 */
  brand?: string
  /** 네비게이션 아이템 */
  navItems?: NavItem[]
  /** 우측 추가 액션 */
  actions?: ReactNode
  className?: string
}

const defaultNavItems: NavItem[] = [
  { label: 'Dashboard', href: '/' },
  { label: 'Portfolio', href: '/portfolio' },
  { label: 'Watchlist', href: '/watchlist' },
  { label: 'Universe', href: '/universe' },
  { label: 'Ranking', href: '/ranking' },
  { label: 'Signals', href: '/signals' },
  { label: 'Execution', href: '/execution' },
  { label: 'Audit', href: '/audit' },
]

export function AppHeader({
  brand = 'Aegis v13',
  navItems = defaultNavItems,
  actions,
  className,
}: AppHeaderProps) {
  const pathname = usePathname()

  const isActive = (href: string) => {
    if (href === '/') {
      return pathname === '/'
    }
    return pathname.startsWith(href)
  }

  return (
    <header
      className={cn(
        'sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60',
        className
      )}
    >
      <div className="container mx-auto px-6 flex h-14 items-center justify-between">
        <div className="flex">
          <Link href="/" className="mr-6 flex items-center space-x-2">
            <span className="font-bold">{brand}</span>
          </Link>
          <nav className="flex items-center space-x-6 text-sm font-medium">
            {navItems.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  'transition-colors hover:text-foreground/80',
                  isActive(item.href)
                    ? 'text-foreground'
                    : 'text-foreground/60'
                )}
              >
                {item.label}
              </Link>
            ))}
          </nav>
        </div>
        <div className="flex items-center space-x-2">
          {actions}
          <ThemeToggle />
        </div>
      </div>
    </header>
  )
}
