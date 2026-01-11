import type { ReactNode } from 'react'
import { AppHeader, PageContainer } from '@/shared/components/layout'
import { DashboardProviders } from './providers'

interface DashboardLayoutProps {
  children: ReactNode
}

export default function DashboardLayout({ children }: DashboardLayoutProps) {
  return (
    <DashboardProviders>
      <div className="min-h-screen bg-background">
        <AppHeader />
        <main>
          <PageContainer>{children}</PageContainer>
        </main>
      </div>
    </DashboardProviders>
  )
}
