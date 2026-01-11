---
sidebar_position: 1
title: Folder Structure
description: Next.js 폴더 구조
---

# Frontend Folder Structure

> Next.js App Router 기반 구조

---

## SSOT 원칙

### ❌ 금지 패턴

```tsx
// 페이지에서 직접 fetch - 금지!
async function Page() {
    const data = await fetch('/api/stocks')  // ❌
}

// 페이지에서 컴포넌트 정의 - 금지!
function StockCard({ stock }) { ... }  // ❌
export default function Page() { ... }

// 인라인 타입 정의 - 금지!
interface Stock { ... }  // ❌
```

### ✅ 올바른 패턴

```tsx
// API는 모듈의 api.ts에서만
import { stockApi } from '@/modules/stock/api'

// 컴포넌트는 modules/*/components/에서
import { StockCard } from '@/modules/stock/components/StockCard'

// 타입은 types/에서
import type { Stock } from '@/modules/stock/types'
```

---

## 폴더 구조

```
frontend/
├── src/
│   ├── app/                      # Next.js App Router
│   │   ├── (dashboard)/          # 대시보드 레이아웃 그룹
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx          # 메인 대시보드
│   │   │   ├── portfolio/
│   │   │   │   └── page.tsx
│   │   │   ├── signals/
│   │   │   │   └── page.tsx
│   │   │   ├── execution/
│   │   │   │   └── page.tsx
│   │   │   └── audit/
│   │   │       └── page.tsx
│   │   ├── (auth)/               # 인증 레이아웃 그룹
│   │   │   ├── login/
│   │   │   └── layout.tsx
│   │   ├── layout.tsx            # 루트 레이아웃
│   │   └── globals.css
│   │
│   ├── modules/                  # ⭐ 도메인별 모듈
│   │   ├── dashboard/
│   │   │   ├── components/
│   │   │   ├── hooks/
│   │   │   └── api.ts
│   │   ├── portfolio/
│   │   │   ├── components/
│   │   │   ├── hooks/
│   │   │   ├── types/
│   │   │   └── api.ts
│   │   ├── signals/
│   │   │   ├── components/
│   │   │   ├── hooks/
│   │   │   └── api.ts
│   │   ├── execution/
│   │   │   ├── components/
│   │   │   ├── hooks/
│   │   │   └── api.ts
│   │   └── audit/
│   │       ├── components/
│   │       ├── hooks/
│   │       └── api.ts
│   │
│   ├── shared/                   # ⭐ 공용 코드
│   │   ├── components/
│   │   │   ├── ui/              # shadcn/ui
│   │   │   │   ├── button.tsx
│   │   │   │   ├── card.tsx
│   │   │   │   └── ...
│   │   │   └── layout/          # 레이아웃 컴포넌트
│   │   │       ├── Header.tsx
│   │   │       ├── Sidebar.tsx
│   │   │       └── ...
│   │   ├── hooks/
│   │   │   ├── useAuth.ts
│   │   │   └── useToast.ts
│   │   ├── api/
│   │   │   └── client.ts        # API 클라이언트
│   │   ├── types/
│   │   │   └── common.ts
│   │   └── utils/
│   │       ├── format.ts
│   │       └── date.ts
│   │
│   └── lib/                      # 외부 라이브러리 설정
│       ├── queryClient.ts
│       └── auth.ts
│
├── public/
├── tailwind.config.ts
├── next.config.js
└── package.json
```

---

## 모듈 구조 상세

### modules/portfolio/ 예시

```
modules/portfolio/
├── components/
│   ├── PortfolioSummary.tsx     # 포트폴리오 요약
│   ├── PositionList.tsx         # 보유 종목 목록
│   ├── PositionCard.tsx         # 개별 종목 카드
│   └── index.ts                 # barrel export
├── hooks/
│   ├── usePortfolio.ts          # 포트폴리오 조회
│   ├── usePositions.ts          # 포지션 조회
│   └── index.ts
├── types/
│   ├── portfolio.ts
│   └── index.ts
└── api.ts                       # API 호출
```

---

## API 호출 패턴

### shared/api/client.ts

```tsx
// 기본 API 클라이언트
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

export async function apiClient<T>(
    endpoint: string,
    options?: RequestInit
): Promise<T> {
    const res = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers,
        },
    })

    if (!res.ok) {
        throw new Error(`API Error: ${res.status}`)
    }

    return res.json()
}
```

### modules/portfolio/api.ts

```tsx
import { apiClient } from '@/shared/api/client'
import type { Portfolio, Position } from './types'

export const portfolioApi = {
    // 포트폴리오 조회
    getPortfolio: () =>
        apiClient<Portfolio>('/api/portfolio'),

    // 포지션 목록
    getPositions: () =>
        apiClient<Position[]>('/api/portfolio/positions'),

    // 목표 포트폴리오
    getTarget: () =>
        apiClient<Portfolio>('/api/portfolio/target'),
}
```

---

## Hook 패턴 (TanStack Query)

```tsx
// modules/portfolio/hooks/usePortfolio.ts

import { useQuery } from '@tanstack/react-query'
import { portfolioApi } from '../api'

export function usePortfolio() {
    return useQuery({
        queryKey: ['portfolio'],
        queryFn: portfolioApi.getPortfolio,
        staleTime: 1000 * 60,  // 1분
    })
}

export function usePositions() {
    return useQuery({
        queryKey: ['positions'],
        queryFn: portfolioApi.getPositions,
        refetchInterval: 1000 * 30,  // 30초마다
    })
}
```

---

## 컴포넌트 규칙

### 1. 숫자는 font-mono 필수

```tsx
// ✅ 올바름
<span className="font-mono">72,300원</span>
<span className="font-mono tabular-nums">+3.25%</span>

// ❌ 금지
<span>72,300원</span>
```

### 2. 색상은 시맨틱 클래스

```tsx
// ✅ 올바름
<span className="text-positive">+3.25%</span>
<span className="text-negative">-2.10%</span>

// ❌ 금지
<span className="text-red-500">-2.10%</span>
```

### 3. shadcn/ui 컴포넌트 사용

```tsx
// ✅ 올바름
import { Button } from '@/shared/components/ui/button'
import { Card } from '@/shared/components/ui/card'

// ❌ 금지
<button className="px-4 py-2 bg-blue-500">...</button>
```

---

## Import 별칭

```json
// tsconfig.json
{
  "compilerOptions": {
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
```

```tsx
// 사용
import { Button } from '@/shared/components/ui/button'
import { usePortfolio } from '@/modules/portfolio/hooks'
import type { Stock } from '@/modules/stock/types'
```

---

**Prev**: [Audit Layer](../backend/audit-layer.md)
**Next**: [Database Schema](../database/schema-design.md)
