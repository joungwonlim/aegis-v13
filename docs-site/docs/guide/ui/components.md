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

## 컴포넌트 사용 규칙

### ✅ 올바른 사용

```tsx
import { Button } from '@/shared/components/ui/button'
import { Card } from '@/shared/components/ui/card'
import { StockCard } from '@/modules/stock/components/StockCard'
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
