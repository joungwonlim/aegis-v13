---
sidebar_position: 1
title: Foundation
description: UI 기초 요소 - 색상, 타이포그래피, 그림자, 아이콘
---

# Foundation

> UI 디자인 시스템의 기초 요소

---

## Colors

### Primary Colors

Aegis v13의 메인 컬러는 **보라색 계열**입니다.

| 이름 | HEX | 용도 |
|------|-----|------|
| **Primary** | `#7367F0` | 주요 액션, 링크, 강조 |
| Primary Dark | `#5E50EE` | 호버 상태 |
| Primary Light | `#887EF2` | 배경, 뱃지 |

### Semantic Colors

| 이름 | HEX | 용도 |
|------|-----|------|
| **Success** | `#28C76F` | 성공, 상승, 매수 |
| **Danger** | `#EA5455` | 에러, 하락, 매도 |
| **Warning** | `#FF9F43` | 경고, 주의 |
| **Info** | `#00CFE8` | 정보, 안내 |

### Trading Colors

주식 거래 시스템에 특화된 색상:

```css
/* 상승/매수 */
--color-positive: #28C76F;
--color-buy: #28C76F;

/* 하락/매도 */
--color-negative: #EA5455;
--color-sell: #EA5455;

/* 보합 */
--color-neutral: #82868B;
```

### 사용 예시

```tsx
// ✅ 올바른 사용
<span className="text-positive">+3.25%</span>
<span className="text-negative">-2.10%</span>

// ❌ 금지
<span className="text-green-500">+3.25%</span>
<span style={{ color: '#28C76F' }}>+3.25%</span>
```

### Dark Mode

| 요소 | Light | Dark |
|------|-------|------|
| Background | `#F8F8F8` | `#161D31` |
| Card | `#FFFFFF` | `#283046` |
| Text Primary | `#5E5873` | `#B4B7BD` |
| Text Secondary | `#82868B` | `#676D7D` |
| Border | `#EBE9F1` | `#3B4253` |

---

## Typography

### Font Family

```css
--font-sans: 'Public Sans', system-ui, -apple-system, sans-serif;
--font-mono: 'JetBrains Mono', 'Fira Code', monospace;
```

### Headings

| Element | Size | Weight | Line Height |
|---------|------|--------|-------------|
| h1 | 2.5rem (40px) | 700 | 1.2 |
| h2 | 2rem (32px) | 600 | 1.3 |
| h3 | 1.5rem (24px) | 600 | 1.4 |
| h4 | 1.25rem (20px) | 600 | 1.4 |
| h5 | 1rem (16px) | 600 | 1.5 |
| h6 | 0.875rem (14px) | 600 | 1.5 |

### Body Text

| Type | Size | Weight | 용도 |
|------|------|--------|------|
| Body 1 | 1rem (16px) | 400 | 기본 본문 |
| Body 2 | 0.875rem (14px) | 400 | 보조 텍스트 |
| Caption | 0.75rem (12px) | 400 | 캡션, 레이블 |
| Overline | 0.625rem (10px) | 600 | 오버라인 |

### 숫자 표시 (필수)

**모든 숫자는 `font-mono` 필수!**

```tsx
// ✅ 올바른 사용
<span className="font-mono">72,300</span>
<span className="font-mono tabular-nums">+3.25%</span>

// ❌ 금지
<span>72,300</span>
```

`tabular-nums`를 사용하면 숫자 너비가 균일해져 정렬이 깔끔해집니다.

---

## Shadows

### Elevation Levels

| Level | Shadow | 용도 |
|-------|--------|------|
| **0** | none | 기본 상태 |
| **1** | `0 2px 4px rgba(0,0,0,0.05)` | 카드, 버튼 |
| **2** | `0 4px 8px rgba(0,0,0,0.08)` | 드롭다운, 팝오버 |
| **3** | `0 8px 16px rgba(0,0,0,0.1)` | 모달, 다이얼로그 |
| **4** | `0 16px 32px rgba(0,0,0,0.12)` | 토스트, 알림 |

### Tailwind Classes

```tsx
<div className="shadow-sm">Level 1</div>
<div className="shadow">Level 2</div>
<div className="shadow-md">Level 3</div>
<div className="shadow-lg">Level 4</div>
```

### Dark Mode Shadows

다크 모드에서는 그림자 대신 **border**나 **배경색 차이**로 elevation 표현:

```tsx
// Light mode
<Card className="shadow-sm" />

// Dark mode
<Card className="dark:shadow-none dark:border dark:border-white/10" />
```

---

## Icons

### Icon Library

[Tabler Icons](https://tabler.io/icons) 사용 (MIT License)

```bash
pnpm add @tabler/icons-react
```

### 사용법

```tsx
import { IconTrendingUp, IconTrendingDown } from '@tabler/icons-react'

// 기본 사용
<IconTrendingUp size={20} />

// 색상 적용
<IconTrendingUp className="text-positive" />
<IconTrendingDown className="text-negative" />
```

### 주요 아이콘

| 용도 | 아이콘 | 컴포넌트 |
|------|--------|----------|
| 상승 | ↑ | `IconTrendingUp` |
| 하락 | ↓ | `IconTrendingDown` |
| 매수 | + | `IconPlus` |
| 매도 | - | `IconMinus` |
| 설정 | ⚙ | `IconSettings` |
| 검색 | 🔍 | `IconSearch` |
| 새로고침 | ↻ | `IconRefresh` |
| 차트 | 📊 | `IconChartLine` |
| 포트폴리오 | 💼 | `IconBriefcase` |
| 알림 | 🔔 | `IconBell` |

### Icon Sizes

| Size | Value | 용도 |
|------|-------|------|
| xs | 16px | 인라인, 뱃지 |
| sm | 20px | 버튼, 메뉴 |
| md | 24px | 기본 |
| lg | 32px | 헤더, 강조 |
| xl | 48px | 빈 상태, 히어로 |

---

## Spacing

### Scale

```css
--space-1: 0.25rem;  /* 4px */
--space-2: 0.5rem;   /* 8px */
--space-3: 0.75rem;  /* 12px */
--space-4: 1rem;     /* 16px */
--space-5: 1.25rem;  /* 20px */
--space-6: 1.5rem;   /* 24px */
--space-8: 2rem;     /* 32px */
--space-10: 2.5rem;  /* 40px */
--space-12: 3rem;    /* 48px */
--space-16: 4rem;    /* 64px */
```

### 사용 예시

```tsx
// Tailwind
<div className="p-4 m-2 gap-3">

// CSS
padding: var(--space-4);
margin: var(--space-2);
gap: var(--space-3);
```

---

## Border Radius

| Name | Value | 용도 |
|------|-------|------|
| none | 0 | - |
| sm | 4px | 작은 요소, 뱃지 |
| md | 6px | 버튼, 입력 |
| lg | 8px | 카드 |
| xl | 12px | 모달, 큰 카드 |
| full | 9999px | 아바타, 원형 버튼 |

```tsx
<Button className="rounded-md" />
<Card className="rounded-lg" />
<Avatar className="rounded-full" />
```

---

## Z-Index

| Layer | Z-Index | 용도 |
|-------|---------|------|
| Base | 0 | 기본 |
| Dropdown | 10 | 드롭다운 메뉴 |
| Sticky | 20 | 고정 헤더 |
| Fixed | 30 | 고정 요소 |
| Modal Backdrop | 40 | 모달 배경 |
| Modal | 50 | 모달 컨텐츠 |
| Popover | 60 | 팝오버 |
| Tooltip | 70 | 툴팁 |
| Toast | 80 | 토스트 알림 |

---

**Next**: [Components](./components.md)
