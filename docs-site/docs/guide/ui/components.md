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

## Watchlist (관심종목) - 모듈화 설계

관심종목 테이블 컴포넌트입니다. **code만 추가하면 자동으로 실시간 가격이 연동됩니다.**

### 핵심 설계 원칙

```
code만 입력 → 자동으로 name, logo, price, change 연동
```

| 데이터 | 소스 | 폴백 |
|--------|------|------|
| 종목명 | DB (stocks 테이블) | - |
| 로고 | Naver (ssl.pstatic.net) | 이니셜 표시 |
| 현재가 | KIS WebSocket → REST → Naver | 마지막 저장값 |
| 전일대비 | KIS WebSocket → REST → Naver | 마지막 저장값 |

### 상태 표시 (Dot Indicator)

| 상태 | 색상 | 의미 |
|------|------|------|
| 🟢 녹색점 | `bg-green-500` | 포트폴리오 보유 종목 |
| 🔴 빨간점 | `bg-red-500` | 자동청산 모니터링 중 |
| (없음) | - | 관심종목만 (미보유) |

### 간단 사용법

```tsx
// ✅ code만 추가하면 나머지는 자동!
const codes = ['195990', '073570', '005930']

<StockListTable codes={codes} />
```

---

## 모듈 구조

```
modules/
├── price/
│   ├── hooks/
│   │   └── useRealtimePrices.ts   # 실시간 가격 Hook
│   ├── providers/
│   │   └── PriceProvider.tsx      # 가격 Context
│   └── types.ts
│
├── stock/
│   ├── components/
│   │   ├── StockCell.tsx          # 종목명 + 로고 + 상태점 (클릭 시 시트 열림)
│   │   ├── PriceCell.tsx          # 실시간 현재가
│   │   ├── ChangeCell.tsx         # 실시간 전일대비
│   │   ├── StockDataTable.tsx     # 통합 테이블
│   │   └── StockDetailSheet.tsx   # 종목 상세 시트 (전역)
│   ├── hooks/
│   │   └── useStockDetail.tsx     # 종목 상세 시트 상태 + Provider
│   └── types.ts
│
└── stocklist/
    ├── components/
    │   └── StockListTable.tsx     # StockDataTable 래핑 + 포트폴리오 자동 연동
    └── hooks/
        └── useStockList.ts        # 관심종목 CRUD
```

---

## 1. useRealtimePrices (실시간 가격 Hook)

```tsx
// modules/price/hooks/useRealtimePrices.ts

import { useQuery } from '@tanstack/react-query'

interface RealtimePrice {
  price: number
  change: number
  change_rate: number
  volume: number
  updated_at: string
}

/**
 * 실시간 가격 조회 Hook
 *
 * 우선순위:
 * 1. KIS WebSocket (실시간)
 * 2. KIS REST API (폴백)
 * 3. Naver Finance (백업)
 */
export function useRealtimePrices(
  symbols: string[],
  options?: { enabled?: boolean; refetchInterval?: number }
) {
  const { enabled = true, refetchInterval = 1000 } = options ?? {}

  return useQuery({
    queryKey: ['prices', 'realtime', symbols.sort().join(',')],
    queryFn: async (): Promise<Record<string, RealtimePrice>> => {
      if (symbols.length === 0) return {}

      const res = await fetch(`/api/prices?symbols=${symbols.join(',')}`)
      const data = await res.json()
      return data.prices
    },
    enabled: enabled && symbols.length > 0,
    staleTime: 500,
    refetchInterval,
    refetchIntervalInBackground: false,
  })
}
```

### Backend API (가격 조회)

```go
// GET /api/prices?symbols=005930,195990

// 우선순위:
// 1. KIS WebSocket 캐시 (메모리)
// 2. KIS REST API
// 3. Naver 크롤링 (백업)

func (h *PriceHandler) GetPrices(w http.ResponseWriter, r *http.Request) {
    symbols := strings.Split(r.URL.Query().Get("symbols"), ",")

    prices := make(map[string]RealtimePrice)
    for _, symbol := range symbols {
        // 1. WebSocket 캐시 확인
        if price, ok := h.wsCache.Get(symbol); ok {
            prices[symbol] = price
            continue
        }

        // 2. KIS REST API
        price, err := h.kisClient.GetCurrentPrice(ctx, symbol)
        if err == nil {
            prices[symbol] = price
            continue
        }

        // 3. Naver 백업
        price, _ = h.naverClient.GetPrice(symbol)
        prices[symbol] = price
    }

    json.NewEncoder(w).Encode(map[string]any{"prices": prices})
}
```

---

## 2. StockCell (종목 셀)

```tsx
// modules/stock/components/StockCell.tsx

interface StockCellProps {
  code: string
  name?: string           // 없으면 자동 조회
  size?: 'sm' | 'md' | 'lg'
  layout?: 'horizontal' | 'vertical'
  isHolding?: boolean     // 🟢 녹색점
  isExitMonitoring?: boolean  // 🔴 빨간점
  onClick?: (stock: { code: string; name: string }) => void
}

const sizeConfig = {
  sm: { image: 'w-5 h-5', name: 'text-xs', code: 'text-[10px]' },
  md: { image: 'w-6 h-6', name: 'text-sm', code: 'text-xs' },
  lg: { image: 'w-8 h-8', name: 'text-base', code: 'text-sm' },
}

export function StockCell({
  code,
  name,
  size = 'md',
  layout = 'vertical',
  isHolding = false,
  isExitMonitoring = false,
  onClick,
}: StockCellProps) {
  const [imageError, setImageError] = useState(false)
  const config = sizeConfig[size]
  const displayName = name || code

  // 네이버 로고 URL
  const logoUrl = `https://ssl.pstatic.net/imgstock/fn/real/logo/stock/Stock${code}.svg`

  return (
    <div
      className={cn(
        'flex items-center gap-2.5',
        onClick && 'cursor-pointer hover:opacity-80 transition-opacity'
      )}
      onClick={() => onClick?.({ code, name: displayName })}
    >
      {/* 로고 */}
      {!imageError ? (
        <img
          src={logoUrl}
          alt={displayName}
          className={cn(config.image, 'rounded-full object-cover flex-shrink-0')}
          onError={() => setImageError(true)}
        />
      ) : (
        <div className={cn(
          config.image,
          'rounded-full bg-muted flex items-center justify-center text-[10px] text-muted-foreground'
        )}>
          {displayName.charAt(0)}
        </div>
      )}

      {/* 종목명 + 코드 */}
      <div className="flex flex-col min-w-0">
        <div className="flex items-center gap-1">
          <span className={cn('font-medium truncate', config.name)}>
            {displayName}
          </span>
          {/* 상태 점 표시 */}
          {isHolding && (
            <span
              className={cn(
                'w-1.5 h-1.5 rounded-full flex-shrink-0',
                isExitMonitoring ? 'bg-red-500' : 'bg-green-500'
              )}
              title={isExitMonitoring ? '자동청산 모니터링' : '보유 종목'}
            />
          )}
        </div>
        <span className={cn('text-muted-foreground truncate', config.code)}>
          {code}
        </span>
      </div>
    </div>
  )
}
```

---

## 3. PriceCell (현재가 셀)

```tsx
// modules/stock/components/PriceCell.tsx

interface PriceCellProps {
  code: string
  fallbackPrice?: number
  size?: 'sm' | 'md' | 'lg'
}

export function PriceCell({ code, fallbackPrice, size = 'md' }: PriceCellProps) {
  const { data: prices } = useRealtimePrices([code], { refetchInterval: 1000 })

  const price = prices?.[code]?.price ?? fallbackPrice ?? 0

  if (price === 0) {
    return <span className="text-muted-foreground">-</span>
  }

  return (
    <span className={cn('font-mono font-medium', sizeConfig[size])}>
      {price.toLocaleString('ko-KR')}
    </span>
  )
}
```

---

## 4. ChangeCell (전일대비 셀)

```tsx
// modules/stock/components/ChangeCell.tsx

interface ChangeCellProps {
  code: string
  size?: 'sm' | 'md' | 'lg'
  showIcon?: boolean
}

export function ChangeCell({ code, size = 'md', showIcon = true }: ChangeCellProps) {
  const { data: prices } = useRealtimePrices([code], { refetchInterval: 1000 })

  const price = prices?.[code]
  const change = price?.change ?? 0
  const changeRate = price?.change_rate ?? 0

  if (change === 0 && changeRate === 0) {
    return <span className="text-muted-foreground">-</span>
  }

  const isPositive = change >= 0
  const icon = isPositive ? '▲' : '▼'

  return (
    <div className={cn(
      'flex items-center justify-end gap-1 font-mono font-medium',
      isPositive ? 'text-positive' : 'text-negative',
      sizeConfig[size]
    )}>
      {showIcon && <span>{icon}</span>}
      <span>
        {Math.abs(change).toLocaleString()}
        <span className="ml-1">
          ({isPositive ? '+' : ''}{changeRate.toFixed(2)}%)
        </span>
      </span>
    </div>
  )
}
```

---

## 5. StockDataTable (통합 테이블)

**SSOT**: 모든 종목 리스트 테이블은 이 컴포넌트 기반으로 구현합니다.

### 핵심 설계

- **기본 컬럼 (항상 표시)**: 순번, 종목명, 현재가, 전일대비
- **추가 컬럼**: `extraColumns` prop으로 페이지별 필요한 컬럼 추가

```tsx
// modules/stock/components/StockDataTable.tsx

import { StockDataTable, type StockDataColumn } from '@/modules/stock/components'

interface StockDataItem {
  code: string
  name?: string
  price?: number
  change?: number
  changeRate?: number
  // 확장 가능한 필드들
  quantity?: number
  avgPrice?: number
  profitLoss?: number
  score?: number
  rank?: number
  [key: string]: unknown
}

interface StockDataTableProps {
  data: StockDataItem[]
  extraColumns?: StockDataColumn[]   // 추가 컬럼 (기본 컬럼 뒤에 표시)
  holdingCodes?: Set<string>         // 🟢 녹색점 표시할 종목
  exitMonitoringCodes?: Set<string>  // 🔴 빨간점 표시할 종목
  showIndex?: boolean
  onRowClick?: (item: StockDataItem) => void
  onDelete?: (code: string) => void
  emptyMessage?: string
}
```

### 기본 사용

```tsx
// 기본 컬럼만 사용 (순번, 종목명, 현재가, 전일대비)
<StockDataTable
  data={stocks}
  emptyMessage="종목이 없습니다"
/>
```

### 추가 컬럼 사용

```tsx
// 유니버스 페이지: 적합도 컬럼 추가
const extraColumns: StockDataColumn[] = [
  {
    key: 'score',
    label: '적합도',
    width: 'w-20',
    align: 'right',
    render: (item) => (
      <span className="font-mono">{item.score?.toFixed(1) ?? '-'}</span>
    ),
  },
]

<StockDataTable
  data={universeStocks}
  extraColumns={extraColumns}
/>
```

### 포트폴리오 페이지 예시

```tsx
// 포트폴리오: 보유수량, 평균매입가, 수익률 등 추가
const portfolioColumns: StockDataColumn[] = [
  {
    key: 'quantity',
    label: '보유수량',
    width: 'w-20',
    align: 'right',
    render: (item) => (
      <span className="font-mono">{item.quantity?.toLocaleString('ko-KR')}</span>
    ),
  },
  {
    key: 'avgPrice',
    label: '평균매입가',
    width: 'w-24',
    align: 'right',
    render: (item) => <PriceCell price={item.avgPrice} size="sm" />,
  },
  {
    key: 'profitLossRate',
    label: '수익률',
    width: 'w-20',
    align: 'right',
    render: (item) => {
      const rate = item.profitLossRate ?? 0
      return (
        <span className={cn('font-mono', rate > 0 && 'text-positive', rate < 0 && 'text-negative')}>
          {rate > 0 ? '+' : ''}{rate.toFixed(2)}%
        </span>
      )
    },
  },
]

<StockDataTable
  data={positions}
  extraColumns={portfolioColumns}
  showIndex={false}
/>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `data` | `StockDataItem[]` | - | 종목 데이터 배열 (필수) |
| `extraColumns` | `StockDataColumn[]` | `[]` | 추가 컬럼 정의 |
| `holdingCodes` | `Set<string>` | `new Set()` | 보유 종목 (🟢 녹색점) |
| `exitMonitoringCodes` | `Set<string>` | `new Set()` | 청산 모니터링 (🔴 빨간점) |
| `showIndex` | `boolean` | `true` | 순번 컬럼 표시 |
| `onRowClick` | `function` | - | 행 클릭 핸들러 |
| `onDelete` | `function` | - | 삭제 버튼 핸들러 |
| `emptyMessage` | `string` | `'종목이 없습니다'` | 빈 상태 메시지 |

---

## 6. StockDetailSheet (종목 상세 시트)

**SSOT**: 종목명 클릭 시 열리는 상세 정보 시트입니다. 전역적으로 사용 가능합니다.

### 핵심 설계

- **전역 Context**: `StockDetailProvider`가 dashboard layout에 통합
- **자동 연동**: `StockCell`의 `clickable=true`(기본값)로 자동 연결
- **외부 링크**: 네이버 증권, DART 공시 등 바로가기 제공

```
종목명 클릭 → StockDetailSheet 자동 열림
```

### Provider 설정 (layout에 이미 포함됨)

```tsx
// app/(dashboard)/providers.tsx
import { StockDetailProvider, StockDetailSheet } from '@/modules/stock'

export function DashboardProviders({ children }: { children: ReactNode }) {
  return (
    <StockDetailProvider>
      {children}
      <StockDetailSheet />
    </StockDetailProvider>
  )
}
```

### StockCell 자동 연동

```tsx
// 기본적으로 클릭 가능 (clickable=true)
<StockCell code="005930" name="삼성전자" />

// 클릭 비활성화
<StockCell code="005930" name="삼성전자" clickable={false} />

// 커스텀 클릭 핸들러 (StockDetailSheet 대신 커스텀 동작)
<StockCell
  code="005930"
  name="삼성전자"
  onClick={(stock) => console.log(stock)}
/>
```

### 직접 호출 (useStockDetail)

```tsx
import { useStockDetail } from '@/modules/stock'

function MyComponent() {
  const { openStockDetail, closeStockDetail, isOpen, selectedStock } = useStockDetail()

  const handleOpenSheet = () => {
    openStockDetail({ code: '005930', name: '삼성전자' })
  }

  return (
    <Button onClick={handleOpenSheet}>삼성전자 상세보기</Button>
  )
}
```

### StockDetailSheet Props

StockDetailSheet는 props 없이 사용됩니다. Context에서 상태를 가져옵니다.

```tsx
// dashboard layout에서 한 번만 렌더링
<StockDetailSheet />
```

### useStockDetail 반환값

| 반환값 | Type | Description |
|--------|------|-------------|
| `selectedStock` | `StockInfo \| null` | 선택된 종목 정보 |
| `isOpen` | `boolean` | 시트 열림 상태 |
| `openStockDetail` | `(stock: StockInfo) => void` | 시트 열기 |
| `closeStockDetail` | `() => void` | 시트 닫기 |
| `handleOpenChange` | `(open: boolean) => void` | Sheet의 onOpenChange용 |

### 외부 링크

StockDetailSheet에서 제공하는 외부 링크:

| 링크 | URL 패턴 |
|------|----------|
| 네이버 증권 | `https://finance.naver.com/item/main.naver?code={code}` |
| 네이버 토론 | `https://finance.naver.com/item/board.naver?code={code}` |
| DART 공시 | `https://dart.fss.or.kr/dsab001/main.do?autoSearch=true&textCrpNm={name}` |
| 증권플러스 커뮤 | `https://m.stockplus.com/m/stocks/KOREA-A{code}/community` |

### 확장 계획

향후 추가 예정 기능:
- 실시간 가격 정보
- 일봉/주봉 차트
- 뉴스/공시 탭
- 재무 정보 탭
- 관심종목 추가/삭제 버튼

---

## 7. StockListTable (종목 리스트 테이블)

```tsx
// modules/stocklist/components/StockListTable.tsx

interface StockListTableProps {
  codes: string[]
  onDelete?: (code: string) => void
}

export function StockListTable({ codes, onDelete }: StockListTableProps) {
  // 포트폴리오 보유 종목 조회
  const { data: positions } = usePositions()

  const holdingCodes = useMemo(() =>
    new Set(positions?.map(p => p.stock_code) ?? []),
    [positions]
  )

  const exitMonitoringCodes = useMemo(() =>
    new Set(positions?.filter(p => p.exit_monitoring_enabled).map(p => p.stock_code) ?? []),
    [positions]
  )

  return (
    <StockDataTable
      codes={codes}
      holdingCodes={holdingCodes}
      exitMonitoringCodes={exitMonitoringCodes}
      onDelete={onDelete}
    />
  )
}
```

---

## 사용 예시

### 최소 코드로 종목 리스트 표시

```tsx
// ✅ 이것만 있으면 실시간 가격, 로고, 녹색점/빨간점 모두 자동!
const codes = ['195990', '073570', '005930']

<StockListTable codes={codes} />
```

### 삭제 기능 추가

```tsx
<StockListTable
  codes={codes}
  onDelete={(code) => removeFromList(code)}
/>
```

---

## 가격 데이터 흐름

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Frontend   │────▶│  Backend    │────▶│  External   │
│  (React)    │◀────│  (Go API)   │◀────│  (KIS/Naver)│
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │                    │
       │ useRealtimePrices │ GET /api/prices    │
       │ (1초 polling)     │                    │
       │                   │ 1. WS Cache ✓      │ KIS WebSocket
       │                   │ 2. KIS REST        │ (실시간)
       │                   │ 3. Naver Backup    │
       │                   │                    │ Naver 크롤링
       ▼                   ▼                    │ (백업)
   PriceCell          priceCache               │
   ChangeCell         (메모리)                  │
```

### 스타일 가이드

#### 색상 (한국 주식 시장 기준)

| 상태 | 색상 | CSS Variable | 값 |
|------|------|--------------|-----|
| 상승 (▲) | 빨간색 | `text-positive` | `#EA5455` |
| 하락 (▼) | 파란색 | `text-negative` | `#2196F3` |
| 보합 | 회색 | `text-neutral` | `#82868B` |
| 배경 | - | `bg-background` | Light: `oklch(0.97 0 0)`, Dark: `oklch(0.145 0 0)` |
| 카드 | - | `bg-card` | Light: `oklch(1 0 0)`, Dark: `oklch(0.205 0 0)` |

> ⚠️ **중요**: 한국 주식 시장은 미국과 반대로 빨간색=상승, 파란색=하락입니다.

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
/* globals.css - 한국 주식 시장 기준 */
:root {
  --background: oklch(0.97 0 0);      /* 연한 회색 */
  --card: oklch(1 0 0);               /* 흰색 */
  --positive: #EA5455;                /* 빨간색 - 상승 */
  --positive-light: #EA54551A;
  --negative: #2196F3;                /* 파란색 - 하락 */
  --negative-light: #2196F31A;
  --neutral: #82868B;
}

.dark {
  --background: oklch(0.145 0 0);     /* 진한 검정 */
  --card: oklch(0.205 0 0);           /* 밝은 검정 */
  --positive: #EA5455;                /* 빨간색 - 상승 */
  --positive-light: #EA54551A;
  --negative: #2196F3;                /* 파란색 - 하락 */
  --negative-light: #2196F31A;
  --neutral: #82868B;
}
```

```tsx
// Tailwind v4: @theme inline 사용
@theme inline {
  --color-positive: var(--positive);
  --color-positive-light: var(--positive-light);
  --color-negative: var(--negative);
  --color-negative-light: var(--negative-light);
  --color-neutral: var(--neutral);
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
