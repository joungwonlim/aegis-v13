---
sidebar_position: 0
title: Design System
description: Aegis v13 퀀트 시스템 디자인 가이드
---

# Design System

> 퀀트 트레이딩 시스템을 위한 통일된 디자인 규정

---

## Tech Stack (2025 Latest)

### Core

| 기술 | 버전 | 용도 |
|------|------|------|
| **Next.js** | 15.x | App Router, Server Components |
| **React** | 19.x | UI 라이브러리 |
| **TypeScript** | 5.x | 타입 안정성 |

### Styling

| 기술 | 버전 | 용도 |
|------|------|------|
| **Tailwind CSS** | 4.x | 유틸리티 CSS |
| **shadcn/ui** | latest | 기본 컴포넌트 |
| **Radix UI** | latest | Headless 컴포넌트 |
| **tailwind-animate** | latest | 애니메이션 |

### Data & State

| 기술 | 버전 | 용도 |
|------|------|------|
| **TanStack Query** | 5.x | 서버 상태 관리 |
| **Zustand** | 5.x | 클라이언트 상태 |
| **nuqs** | 2.x | URL 상태 관리 |

### Charts

| 기술 | 용도 | 선택 이유 |
|------|------|----------|
| **Lightweight Charts** | 캔들 차트, 주가 차트 | TradingView 오픈소스, 금융 특화 |
| **Recharts** | 일반 차트 (파이, 바, 라인) | shadcn 호환, 간단함 |
| **Tremor** | 대시보드 차트 | 퀀트 대시보드 최적화 |

### Icons & Assets

| 기술 | 용도 |
|------|------|
| **Tabler Icons** | 메인 아이콘 (4000+) |
| **Lucide Icons** | shadcn 기본 아이콘 |

### Fonts

| 폰트 | 용도 |
|------|------|
| **Pretendard** | 한글 기본 |
| **Inter** | 영문 기본 |
| **JetBrains Mono** | 숫자, 코드 |

---

## Color System

### Brand Colors

```css
:root {
  /* Primary - Aegis Purple */
  --primary: #7367F0;
  --primary-foreground: #FFFFFF;

  /* Secondary */
  --secondary: #82868B;
  --secondary-foreground: #FFFFFF;
}
```

### Trading Colors (필수 준수)

```css
:root {
  /* 상승 - Green */
  --positive: #28C76F;
  --positive-light: #28C76F1A;  /* 10% opacity */

  /* 하락 - Red */
  --negative: #EA5455;
  --negative-light: #EA54551A;  /* 10% opacity */

  /* 보합 - Gray */
  --neutral: #82868B;
  --neutral-light: #82868B1A;

  /* 매수 */
  --buy: #28C76F;
  --buy-light: #28C76F1A;

  /* 매도 */
  --sell: #EA5455;
  --sell-light: #EA54551A;
}
```

### 사용 규칙

```tsx
// ✅ 올바른 사용
<span className="text-positive">+3.25%</span>
<span className="text-negative">-2.10%</span>
<span className="text-neutral">0.00%</span>

// 배경색
<div className="bg-positive-light text-positive">상승</div>
<div className="bg-negative-light text-negative">하락</div>

// ❌ 금지
<span className="text-green-500">+3.25%</span>
<span className="text-red-500">-2.10%</span>
<span style={{ color: '#28C76F' }}>+3.25%</span>
```

### Tailwind Config

```ts
// tailwind.config.ts
export default {
  theme: {
    extend: {
      colors: {
        positive: {
          DEFAULT: '#28C76F',
          light: '#28C76F1A',
        },
        negative: {
          DEFAULT: '#EA5455',
          light: '#EA54551A',
        },
        neutral: {
          DEFAULT: '#82868B',
          light: '#82868B1A',
        },
      },
    },
  },
}
```

---

## 종목 표시 규정

### 종목 정보 구조

```ts
interface Stock {
  code: string       // "005930"
  name: string       // "삼성전자"
  market: string     // "KOSPI" | "KOSDAQ"
  logoUrl?: string   // 종목 이미지 URL
}
```

### 종목명 표시

| 상황 | 표시 | 예시 |
|------|------|------|
| 기본 | 종목명만 | 삼성전자 |
| 상세 | 종목명 + 코드 | 삼성전자 (005930) |
| 축약 | 코드만 | 005930 |
| 리스트 | 종목명 + 마켓 | 삼성전자 `KOSPI` |

### 종목 이미지 (로고)

```tsx
// 컴포넌트
interface StockLogoProps {
  code: string
  name: string
  size?: 'sm' | 'md' | 'lg'
}

// 크기 규정
const sizes = {
  sm: 24,   // 리스트, 테이블
  md: 32,   // 카드
  lg: 48,   // 상세 페이지
}

// 이미지 소스
const logoUrl = `https://file.alphasquare.co.kr/media/images/stock_logo/kr/${code}.png`

// Fallback (이미지 없을 때)
<div className="bg-muted rounded-full flex items-center justify-center">
  <span className="font-semibold text-muted-foreground">
    {name.charAt(0)}  // 첫 글자
  </span>
</div>
```

### 종목 표시 컴포넌트

```tsx
// StockDisplay.tsx
<div className="flex items-center gap-3">
  {/* 로고 */}
  <StockLogo code="005930" name="삼성전자" size="md" />

  {/* 정보 */}
  <div>
    <p className="font-semibold">삼성전자</p>
    <p className="text-sm text-muted-foreground">005930 · KOSPI</p>
  </div>
</div>
```

---

## 가격 표시 규정

### 기본 규칙

| 규칙 | 설명 |
|------|------|
| **font-mono 필수** | 모든 숫자는 고정폭 폰트 |
| **tabular-nums** | 숫자 정렬용 |
| **천 단위 콤마** | 1000 → 1,000 |
| **원 단위 생략** | 72,300 (원 생략) |

### 가격 포맷

```ts
// utils/format.ts
export function formatPrice(price: number): string {
  return price.toLocaleString('ko-KR')
}

// 72300 → "72,300"
```

### 등락률 표시

```tsx
// 등락률 컴포넌트
interface ChangeRateProps {
  value: number  // 0.0325 = +3.25%
}

function ChangeRate({ value }: ChangeRateProps) {
  const formatted = (value * 100).toFixed(2)
  const isPositive = value > 0
  const isNegative = value < 0

  return (
    <span className={cn(
      "font-mono tabular-nums",
      isPositive && "text-positive",
      isNegative && "text-negative",
      !isPositive && !isNegative && "text-neutral"
    )}>
      {isPositive && '+'}{formatted}%
    </span>
  )
}
```

### 등락폭 표시

```tsx
// 등락폭 (가격 차이)
function ChangeAmount({ value, price }: { value: number; price: number }) {
  const isPositive = value > 0
  const sign = isPositive ? '▲' : value < 0 ? '▼' : ''

  return (
    <span className={cn(
      "font-mono tabular-nums",
      isPositive && "text-positive",
      value < 0 && "text-negative"
    )}>
      {sign} {Math.abs(value).toLocaleString()}
    </span>
  )
}

// 출력: ▲ 1,700 또는 ▼ 1,200
```

### 가격 + 등락 조합

```tsx
// 표준 가격 표시
<div className="text-right">
  <p className="font-mono text-lg font-semibold">72,300</p>
  <div className="flex items-center justify-end gap-2 text-sm">
    <span className="font-mono text-positive">▲ 1,700</span>
    <span className="font-mono text-positive">+2.41%</span>
  </div>
</div>
```

---

## 테이블 규정

### 정렬 규칙

| 데이터 타입 | 정렬 | 클래스 |
|------------|------|--------|
| 종목명 | 좌측 | `text-left` |
| 가격/수량 | 우측 | `text-right font-mono` |
| 등락률 | 우측 | `text-right font-mono` |
| 상태 | 중앙 | `text-center` |
| 날짜 | 중앙 | `text-center` |
| 액션 | 우측 | `text-right` |

### 표준 주식 테이블

```tsx
<Table>
  <TableHeader>
    <TableRow>
      <TableHead className="w-[200px]">종목</TableHead>
      <TableHead className="text-right">현재가</TableHead>
      <TableHead className="text-right">등락률</TableHead>
      <TableHead className="text-right">거래량</TableHead>
      <TableHead className="text-center">시장</TableHead>
    </TableRow>
  </TableHeader>
  <TableBody>
    <TableRow>
      <TableCell>
        <div className="flex items-center gap-2">
          <StockLogo code="005930" name="삼성전자" size="sm" />
          <span className="font-medium">삼성전자</span>
        </div>
      </TableCell>
      <TableCell className="text-right font-mono">72,300</TableCell>
      <TableCell className="text-right font-mono text-positive">+2.41%</TableCell>
      <TableCell className="text-right font-mono">12,345,678</TableCell>
      <TableCell className="text-center">
        <Badge variant="outline">KOSPI</Badge>
      </TableCell>
    </TableRow>
  </TableBody>
</Table>
```

---

## 차트 규정

### Lightweight Charts (주가 차트)

```tsx
import { createChart, ColorType } from 'lightweight-charts'

// 테마 설정
const chartOptions = {
  layout: {
    background: { type: ColorType.Solid, color: 'transparent' },
    textColor: '#82868B',
  },
  grid: {
    vertLines: { color: '#EBE9F1' },
    horzLines: { color: '#EBE9F1' },
  },
  // 상승/하락 색상
  upColor: '#28C76F',
  downColor: '#EA5455',
  wickUpColor: '#28C76F',
  wickDownColor: '#EA5455',
}
```

### Recharts (일반 차트)

```tsx
// 색상 통일
const CHART_COLORS = {
  primary: '#7367F0',
  positive: '#28C76F',
  negative: '#EA5455',
  neutral: '#82868B',
  grid: '#EBE9F1',
}

<LineChart>
  <CartesianGrid stroke={CHART_COLORS.grid} />
  <Line stroke={CHART_COLORS.primary} />
</LineChart>
```

### 차트 높이 규정

| 용도 | 높이 |
|------|------|
| 미니 차트 (리스트) | 40px |
| 카드 차트 | 120px |
| 메인 차트 | 400px |
| 전체 화면 | calc(100vh - 200px) |

---

## 레이아웃 규정

### 간격 (Spacing)

| 이름 | 값 | 용도 |
|------|-----|------|
| `gap-1` | 4px | 아이콘-텍스트 |
| `gap-2` | 8px | 인라인 요소 |
| `gap-3` | 12px | 리스트 아이템 |
| `gap-4` | 16px | 카드 내부 |
| `gap-6` | 24px | 섹션 |
| `gap-8` | 32px | 페이지 섹션 |

### 카드 규격

```tsx
// 표준 카드
<Card className="p-4">        {/* 내부 패딩 16px */}
<Card className="p-6">        {/* 내부 패딩 24px (대형) */}

// 카드 사이 간격
<div className="grid gap-4">  {/* 16px */}
<div className="grid gap-6">  {/* 24px (대시보드) */}
```

### 반응형 그리드

```tsx
// 대시보드 그리드
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
  <Card>...</Card>
  <Card>...</Card>
  <Card>...</Card>
  <Card>...</Card>
</div>
```

---

## 컴포넌트 체크리스트

모든 컴포넌트 개발 시 확인:

- [ ] 숫자에 `font-mono tabular-nums` 적용
- [ ] 상승/하락에 `text-positive` / `text-negative` 사용
- [ ] 직접 색상 코드 사용 안함 (`text-red-500` ❌)
- [ ] shadcn/ui 컴포넌트 사용
- [ ] 테이블 정렬 규칙 준수
- [ ] 다크 모드 테스트

---

**Next**: [Foundation](./foundation.md)
