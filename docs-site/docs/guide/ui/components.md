---
sidebar_position: 2
title: Components
description: UI 컴포넌트 가이드
---

# Components

> shadcn/ui 기반 컴포넌트 시스템

---

## 개요

Aegis v13은 **shadcn/ui**를 기반으로 합니다.

- [shadcn/ui 공식 문서](https://ui.shadcn.com)
- Radix UI 기반 접근성 보장
- Tailwind CSS로 스타일링
- 복사-붙여넣기 방식 (의존성 최소화)

---

## 설치된 컴포넌트

```
shared/components/ui/
├── button.tsx
├── card.tsx
├── badge.tsx
├── input.tsx
├── select.tsx
├── table.tsx
├── dialog.tsx
├── dropdown-menu.tsx
├── tabs.tsx
├── toast.tsx
└── ...
```

---

## Button

### Variants

```tsx
import { Button } from '@/shared/components/ui/button'

<Button variant="default">Primary</Button>
<Button variant="secondary">Secondary</Button>
<Button variant="outline">Outline</Button>
<Button variant="ghost">Ghost</Button>
<Button variant="destructive">Destructive</Button>
```

### Sizes

```tsx
<Button size="sm">Small</Button>
<Button size="default">Default</Button>
<Button size="lg">Large</Button>
<Button size="icon"><IconPlus /></Button>
```

### Trading Buttons

```tsx
// 매수 버튼
<Button className="bg-positive hover:bg-positive/90">
  매수
</Button>

// 매도 버튼
<Button className="bg-negative hover:bg-negative/90">
  매도
</Button>
```

---

## Card

### 기본 사용

```tsx
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'

<Card>
  <CardHeader>
    <CardTitle>포트폴리오 요약</CardTitle>
  </CardHeader>
  <CardContent>
    {/* 내용 */}
  </CardContent>
</Card>
```

### Stock Card

```tsx
<Card>
  <CardContent className="p-4">
    <div className="flex justify-between items-center">
      <div>
        <p className="font-semibold">삼성전자</p>
        <p className="text-sm text-muted-foreground">005930</p>
      </div>
      <div className="text-right">
        <p className="font-mono text-lg">72,300</p>
        <p className="font-mono text-sm text-positive">+2.41%</p>
      </div>
    </div>
  </CardContent>
</Card>
```

---

## Badge

### Variants

```tsx
import { Badge } from '@/shared/components/ui/badge'

<Badge>Default</Badge>
<Badge variant="secondary">Secondary</Badge>
<Badge variant="outline">Outline</Badge>
<Badge variant="destructive">Destructive</Badge>
```

### Trading Badges

```tsx
// 상승
<Badge className="bg-positive/10 text-positive border-0">
  +3.25%
</Badge>

// 하락
<Badge className="bg-negative/10 text-negative border-0">
  -2.10%
</Badge>

// 보합
<Badge variant="secondary">
  0.00%
</Badge>
```

### Status Badges

```tsx
<Badge className="bg-blue-500/10 text-blue-500">매수 대기</Badge>
<Badge className="bg-green-500/10 text-green-500">체결 완료</Badge>
<Badge className="bg-yellow-500/10 text-yellow-500">부분 체결</Badge>
<Badge className="bg-red-500/10 text-red-500">주문 취소</Badge>
```

---

## Table

### 기본 사용

```tsx
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/shared/components/ui/table'

<Table>
  <TableHeader>
    <TableRow>
      <TableHead>종목</TableHead>
      <TableHead className="text-right">현재가</TableHead>
      <TableHead className="text-right">등락률</TableHead>
    </TableRow>
  </TableHeader>
  <TableBody>
    <TableRow>
      <TableCell>삼성전자</TableCell>
      <TableCell className="text-right font-mono">72,300</TableCell>
      <TableCell className="text-right font-mono text-positive">+2.41%</TableCell>
    </TableRow>
  </TableBody>
</Table>
```

### 정렬 규칙

| 데이터 타입 | 정렬 |
|------------|------|
| 텍스트 (종목명) | 좌측 |
| 숫자 (가격, 수량) | 우측 |
| 상태 (뱃지) | 중앙 |
| 액션 (버튼) | 우측 |

---

## Input

### 기본 사용

```tsx
import { Input } from '@/shared/components/ui/input'

<Input placeholder="종목 검색..." />
<Input type="number" className="font-mono" />
```

### 숫자 입력

```tsx
// 가격 입력
<div className="relative">
  <Input
    type="text"
    className="font-mono pr-8"
    placeholder="0"
  />
  <span className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground">
    원
  </span>
</div>

// 수량 입력
<Input
  type="number"
  className="font-mono"
  min={1}
  step={1}
/>
```

---

## Select

```tsx
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/shared/components/ui/select'

<Select>
  <SelectTrigger>
    <SelectValue placeholder="주문 유형" />
  </SelectTrigger>
  <SelectContent>
    <SelectItem value="limit">지정가</SelectItem>
    <SelectItem value="market">시장가</SelectItem>
  </SelectContent>
</Select>
```

---

## Dialog (Modal)

```tsx
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/shared/components/ui/dialog'

<Dialog>
  <DialogTrigger asChild>
    <Button>주문하기</Button>
  </DialogTrigger>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>주문 확인</DialogTitle>
      <DialogDescription>
        삼성전자 10주를 72,300원에 매수합니다.
      </DialogDescription>
    </DialogHeader>
    <DialogFooter>
      <Button variant="outline">취소</Button>
      <Button>확인</Button>
    </DialogFooter>
  </DialogContent>
</Dialog>
```

---

## Tabs

```tsx
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/shared/components/ui/tabs'

<Tabs defaultValue="portfolio">
  <TabsList>
    <TabsTrigger value="portfolio">포트폴리오</TabsTrigger>
    <TabsTrigger value="orders">주문내역</TabsTrigger>
    <TabsTrigger value="history">거래내역</TabsTrigger>
  </TabsList>
  <TabsContent value="portfolio">
    {/* 포트폴리오 내용 */}
  </TabsContent>
  <TabsContent value="orders">
    {/* 주문내역 내용 */}
  </TabsContent>
</Tabs>
```

---

## Toast

```tsx
import { useToast } from '@/shared/hooks/useToast'

function OrderButton() {
  const { toast } = useToast()

  const handleOrder = () => {
    toast({
      title: '주문 완료',
      description: '삼성전자 10주 매수 주문이 접수되었습니다.',
    })
  }

  return <Button onClick={handleOrder}>주문</Button>
}
```

### Variants

```tsx
// 성공
toast({ title: '주문 체결', variant: 'default' })

// 에러
toast({ title: '주문 실패', variant: 'destructive' })
```

---

## Watchlist (관심종목)

관심종목 테이블 컴포넌트입니다. 다크/라이트 테마 모두 지원합니다.

### 구조

```
┌─────────────────────────────────────────────────────────────┐
│  관심종목                    [+ 종목 추가] [↻] [∧]          │
├─────────────────────────────────────────────────────────────┤
│  순번   종목명              현재가           전일대비       │
├─────────────────────────────────────────────────────────────┤
│  1     🔴 에이비프로바이오    211     ▲ 2 (+0.96%)     🗑   │
│        195990                                               │
├─────────────────────────────────────────────────────────────┤
│  2     🔴 리튬포어스          916     ▼ 29 (-3.07%)    🗑   │
│        073570                                               │
└─────────────────────────────────────────────────────────────┘
```

### 기본 사용

```tsx
import { Watchlist } from '@/modules/watchlist/components/Watchlist'

const stocks = [
  {
    rank: 1,
    code: '195990',
    name: '에이비프로바이오',
    logo: '/logos/195990.png',
    price: 211,
    change: 2,
    changeRate: 0.96,
  },
  {
    rank: 2,
    code: '073570',
    name: '리튬포어스',
    logo: '/logos/073570.png',
    price: 916,
    change: -29,
    changeRate: -3.07,
  },
]

<Watchlist
  stocks={stocks}
  onAdd={() => openAddModal()}
  onRefresh={() => refetchData()}
  onDelete={(code) => removeStock(code)}
/>
```

### Props

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `stocks` | `WatchlistStock[]` | Yes | 종목 리스트 |
| `onAdd` | `() => void` | No | 종목 추가 클릭 핸들러 |
| `onRefresh` | `() => void` | No | 새로고침 클릭 핸들러 |
| `onDelete` | `(code: string) => void` | No | 삭제 클릭 핸들러 |
| `isCollapsible` | `boolean` | No | 접기/펼치기 기능 (기본: true) |
| `className` | `string` | No | 추가 스타일 클래스 |

### WatchlistStock Type

```tsx
interface WatchlistStock {
  rank: number           // 순번
  code: string           // 종목코드 (6자리)
  name: string           // 종목명
  logo?: string          // 로고 이미지 URL
  price: number          // 현재가
  change: number         // 전일대비 (원)
  changeRate: number     // 등락률 (%)
}
```

### 종목 로고 URL

네이버 증권에서 제공하는 SVG 로고를 사용합니다.

```tsx
// URL 패턴
const getStockLogoUrl = (code: string) =>
  `https://ssl.pstatic.net/imgstock/fn/real/logo/stock/Stock${code}.svg`

// 예시
getStockLogoUrl('005930')  // 삼성전자
// → https://ssl.pstatic.net/imgstock/fn/real/logo/stock/Stock005930.svg

getStockLogoUrl('195990')  // 에이비프로바이오
// → https://ssl.pstatic.net/imgstock/fn/real/logo/stock/Stock195990.svg
```

#### 사용 예시

```tsx
const stocks = [
  {
    rank: 1,
    code: '195990',
    name: '에이비프로바이오',
    logo: getStockLogoUrl('195990'),
    price: 211,
    change: 2,
    changeRate: 0.96,
  },
  {
    rank: 10,
    code: '005930',
    name: '삼성전자',
    logo: getStockLogoUrl('005930'),
    price: 139000,
    change: 200,
    changeRate: 0.14,
  },
]
```

#### 로고 컴포넌트

```tsx
// modules/stock/components/StockLogo.tsx

interface StockLogoProps {
  code: string
  name: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizeMap = {
  sm: 'h-6 w-6',
  md: 'h-8 w-8',
  lg: 'h-10 w-10',
}

export function StockLogo({ code, name, size = 'md', className }: StockLogoProps) {
  const logoUrl = `https://ssl.pstatic.net/imgstock/fn/real/logo/stock/Stock${code}.svg`

  return (
    <img
      src={logoUrl}
      alt={name}
      className={cn(sizeMap[size], 'rounded-full', className)}
      onError={(e) => {
        // 로고 없을 경우 기본 아이콘으로 대체
        e.currentTarget.src = '/icons/stock-default.svg'
      }}
    />
  )
}
```

### 스타일 가이드

#### 색상

| 상태 | Light Theme | Dark Theme |
|------|-------------|------------|
| 상승 (▲) | `text-positive` (#22c55e) | `text-positive` (#22c55e) |
| 하락 (▼) | `text-negative` (#ef4444) | `text-negative` (#ef4444) |
| 보합 | `text-muted-foreground` | `text-muted-foreground` |
| 배경 | `bg-card` (white) | `bg-card` (#1c1c1e) |
| 테두리 | `border` | `border` |

#### 폰트

```tsx
// 가격/등락률은 반드시 monospace
<span className="font-mono">139,000</span>
<span className="font-mono text-positive">▲ 200 (+0.14%)</span>

// 종목명은 기본 폰트
<span className="font-medium">삼성전자</span>

// 종목코드는 muted
<span className="text-sm text-muted-foreground">005930</span>
```

### 컴포넌트 구현

```tsx
// modules/watchlist/components/Watchlist.tsx

import { useState } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/shared/components/ui/table'
import { Plus, RefreshCw, ChevronUp, ChevronDown, Trash2 } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

interface WatchlistStock {
  rank: number
  code: string
  name: string
  logo?: string
  price: number
  change: number
  changeRate: number
}

interface WatchlistProps {
  stocks: WatchlistStock[]
  onAdd?: () => void
  onRefresh?: () => void
  onDelete?: (code: string) => void
  isCollapsible?: boolean
  className?: string
}

export function Watchlist({
  stocks,
  onAdd,
  onRefresh,
  onDelete,
  isCollapsible = true,
  className,
}: WatchlistProps) {
  const [isCollapsed, setIsCollapsed] = useState(false)

  const formatPrice = (price: number) => {
    return price.toLocaleString('ko-KR')
  }

  const formatChange = (change: number, rate: number) => {
    const sign = change >= 0 ? '▲' : '▼'
    const absChange = Math.abs(change)
    const absRate = Math.abs(rate)
    return `${sign} ${formatPrice(absChange)} (${change >= 0 ? '+' : '-'}${absRate.toFixed(2)}%)`
  }

  return (
    <Card className={className}>
      <CardHeader className="flex flex-row items-center justify-between py-4">
        <CardTitle className="text-lg font-semibold">관심종목</CardTitle>
        <div className="flex items-center gap-2">
          {onAdd && (
            <Button size="sm" onClick={onAdd}>
              <Plus className="h-4 w-4 mr-1" />
              종목 추가
            </Button>
          )}
          {onRefresh && (
            <Button variant="ghost" size="icon" onClick={onRefresh}>
              <RefreshCw className="h-4 w-4" />
            </Button>
          )}
          {isCollapsible && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setIsCollapsed(!isCollapsed)}
            >
              {isCollapsed ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronUp className="h-4 w-4" />
              )}
            </Button>
          )}
        </div>
      </CardHeader>

      {!isCollapsed && (
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16 text-center">순번</TableHead>
                <TableHead>종목명</TableHead>
                <TableHead className="text-right">현재가</TableHead>
                <TableHead className="text-right">전일대비</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {stocks.map((stock) => (
                <TableRow key={stock.code}>
                  <TableCell className="text-center text-muted-foreground">
                    {stock.rank}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      {stock.logo && (
                        <img
                          src={stock.logo}
                          alt={stock.name}
                          className="h-8 w-8 rounded-full"
                        />
                      )}
                      <div>
                        <p className="font-medium">{stock.name}</p>
                        <p className="text-sm text-muted-foreground">
                          {stock.code}
                        </p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatPrice(stock.price)}
                  </TableCell>
                  <TableCell
                    className={cn(
                      'text-right font-mono',
                      stock.change > 0 && 'text-positive',
                      stock.change < 0 && 'text-negative'
                    )}
                  >
                    {formatChange(stock.change, stock.changeRate)}
                  </TableCell>
                  <TableCell>
                    {onDelete && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onDelete(stock.code)}
                      >
                        <Trash2 className="h-4 w-4 text-muted-foreground" />
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      )}
    </Card>
  )
}
```

### 테마 지원

Tailwind CSS와 CSS 변수를 사용하여 다크/라이트 테마를 자동으로 지원합니다.

```css
/* globals.css */
:root {
  --positive: 142 76% 36%;  /* green-500 */
  --negative: 0 84% 60%;    /* red-500 */
}

.dark {
  --positive: 142 71% 45%;
  --negative: 0 91% 71%;
}
```

```tsx
// tailwind.config.ts
theme: {
  extend: {
    colors: {
      positive: 'hsl(var(--positive))',
      negative: 'hsl(var(--negative))',
    }
  }
}
```

---

## 컴포넌트 사용 규칙

### ✅ 올바른 사용

```tsx
import { Button } from '@/shared/components/ui/button'
import { Card } from '@/shared/components/ui/card'
import { StockCard } from '@/modules/stock/components/StockCard'
import { Watchlist } from '@/modules/watchlist/components/Watchlist'
```

### ❌ 금지

```tsx
// 직접 스타일링 금지
<button className="px-4 py-2 bg-blue-500">...</button>

// 인라인 스타일 금지
<div style={{ padding: '16px' }}>...</div>
```

---

**Prev**: [Foundation](./foundation)
