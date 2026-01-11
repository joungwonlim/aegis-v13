---
sidebar_position: 2
title: Layout Components
description: 레이아웃 컴포넌트 가이드
---

# Layout Components

> 페이지 레이아웃을 위한 재사용 컴포넌트

---

## 개요

모든 페이지는 **일관된 레이아웃**을 위해 아래 컴포넌트를 사용합니다.

```
shared/components/layout/
├── AppHeader.tsx      # 앱 상단 네비게이션
├── PageContainer.tsx  # 페이지 컨테이너 (여백, 최대폭)
├── PageHeader.tsx     # 페이지 제목/액션 영역
├── ThemeToggle.tsx    # 다크모드 토글
└── index.ts           # exports
```

---

## AppHeader

앱 전역 상단 네비게이션 헤더입니다.

### 기본 사용

```tsx
import { AppHeader } from '@/shared/components/layout'

// 기본 네비게이션 사용
<AppHeader />

// 커스텀 브랜드
<AppHeader brand="My App" />
```

### 커스텀 네비게이션

```tsx
import { AppHeader, type NavItem } from '@/shared/components/layout'

const customNav: NavItem[] = [
  { label: 'Home', href: '/' },
  { label: 'About', href: '/about' },
]

<AppHeader navItems={customNav} />
```

### 추가 액션

```tsx
<AppHeader
  actions={
    <Button variant="outline" size="sm">
      Settings
    </Button>
  }
/>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `brand` | `string` | `'Aegis v13'` | 로고/브랜드 텍스트 |
| `navItems` | `NavItem[]` | 기본 메뉴 | 네비게이션 아이템 |
| `actions` | `ReactNode` | - | 우측 추가 액션 |
| `className` | `string` | - | 추가 스타일 |

### 기본 네비게이션 항목

```tsx
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
```

---

## PageContainer

페이지 컨텐츠의 기본 컨테이너입니다. 중앙 정렬, 최대 너비, 좌우 패딩을 제공합니다.

### 기본 사용

```tsx
import { PageContainer } from '@/shared/components/layout'

<PageContainer>
  <h1>Page Title</h1>
  <p>Page content here...</p>
</PageContainer>
```

### 패딩 없이 (테이블 등)

```tsx
<PageContainer noPadding>
  <Table>...</Table>
</PageContainer>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `children` | `ReactNode` | - | 컨텐츠 |
| `noPadding` | `boolean` | `false` | 패딩 제거 |
| `className` | `string` | - | 추가 스타일 |

### 기본 스타일

```css
.container.mx-auto.px-6.py-6
```

---

## PageHeader

페이지 상단 제목 및 액션 영역입니다.

### 기본 사용

```tsx
import { PageHeader } from '@/shared/components/layout'

<PageHeader title="Portfolio" />
```

### 설명 포함

```tsx
<PageHeader
  title="Portfolio"
  description="보유 종목 현황을 확인합니다"
/>
```

### 액션 버튼

```tsx
<PageHeader
  title="Watchlist"
  actions={
    <>
      <Button variant="outline" size="sm">
        <RefreshCw className="h-4 w-4" />
      </Button>
      <Button size="sm">
        <Plus className="h-4 w-4 mr-1" />
        추가
      </Button>
    </>
  }
/>
```

### 뱃지 포함

```tsx
import { Badge } from '@/shared/components/ui/badge'

<PageHeader
  title="Universe"
  badge={<Badge variant="secondary">KOSPI + KOSDAQ</Badge>}
/>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `title` | `string` | - | 페이지 제목 (필수) |
| `description` | `string` | - | 페이지 설명 |
| `actions` | `ReactNode` | - | 우측 액션 버튼 |
| `badge` | `ReactNode` | - | 제목 옆 뱃지 |
| `className` | `string` | - | 추가 스타일 |

---

## 페이지 레이아웃 패턴

### 기본 페이지 구조

```tsx
// app/(dashboard)/example/page.tsx
import { PageHeader } from '@/shared/components/layout'
import { Card, CardContent } from '@/shared/components/ui/card'

export default function ExamplePage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Example"
        description="예시 페이지입니다"
      />

      <Card>
        <CardContent className="pt-6">
          {/* 페이지 내용 */}
        </CardContent>
      </Card>
    </div>
  )
}
```

### Dashboard Layout

```tsx
// app/(dashboard)/layout.tsx
import { AppHeader, PageContainer } from '@/shared/components/layout'

export default function DashboardLayout({ children }) {
  return (
    <div className="min-h-screen bg-background">
      <AppHeader />
      <main>
        <PageContainer>{children}</PageContainer>
      </main>
    </div>
  )
}
```

---

## 새 페이지 추가 가이드

### 1. 페이지 파일 생성

```tsx
// app/(dashboard)/new-page/page.tsx
'use client'

import { PageHeader } from '@/shared/components/layout'

export default function NewPage() {
  return (
    <div className="space-y-6">
      <PageHeader title="New Page" />
      {/* 페이지 내용 */}
    </div>
  )
}
```

### 2. 네비게이션 추가 (필요시)

`AppHeader`의 `defaultNavItems`에 새 항목을 추가하거나, 커스텀 `navItems`를 전달합니다.

```tsx
// shared/components/layout/AppHeader.tsx
const defaultNavItems: NavItem[] = [
  // ... existing items
  { label: 'New Page', href: '/new-page' },
]
```

---

## SSOT 규칙

| 컴포넌트 | 위치 | 용도 |
|----------|------|------|
| `AppHeader` | `shared/components/layout/` | 앱 전역 헤더 |
| `PageContainer` | `shared/components/layout/` | 페이지 컨테이너 |
| `PageHeader` | `shared/components/layout/` | 페이지 제목/액션 |

### 금지 패턴

```tsx
// ❌ 페이지에서 직접 container 스타일 정의
<div className="container mx-auto px-6 py-6">

// ✅ PageContainer 사용
<PageContainer>
```

```tsx
// ❌ 페이지에서 직접 헤더 스타일 정의
<div className="flex items-center justify-between mb-6">
  <h1 className="text-2xl font-bold">Title</h1>
</div>

// ✅ PageHeader 사용
<PageHeader title="Title" />
```
