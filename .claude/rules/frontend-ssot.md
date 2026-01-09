# Frontend SSOT 규칙 (Claude Code 강제)

## 핵심 원칙

**"새로 만들지 말고, 있는 것을 사용하라"**

---

## 작업 전 필수 확인

### 1. 컴포넌트 작성 전

```bash
# 기존 컴포넌트 확인
ls frontend/src/shared/components/ui/
ls frontend/src/modules/*/components/
```

**없으면** → 적절한 SSOT 위치에 추가 후 사용

### 2. 훅 작성 전

```bash
ls frontend/src/shared/hooks/
```

### 3. 타입 정의 전

```bash
ls frontend/src/modules/*/types/
```

---

## SSOT 위치 (필수 준수)

| 종류 | 위치 | 금지 |
|------|------|------|
| **UI 컴포넌트** | `shared/components/ui/` | 페이지에 직접 정의 |
| **도메인 컴포넌트** | `modules/*/components/` | 페이지에 직접 정의 |
| **API 호출** | `modules/*/api.ts` | 직접 fetch 호출 |
| **타입 정의** | `modules/*/types/` | 인라인 interface |
| **훅** | `shared/hooks/` | 페이지에 직접 정의 |

---

## 금지 패턴

### ❌ 페이지에 컴포넌트 직접 정의

```tsx
// app/stocks/page.tsx
function StockCard({ stock }) { ... }  // 금지!
```

### ❌ 직접 fetch 호출

```tsx
fetch('/api/stocks')  // 금지!
```

### ❌ 인라인 타입 정의

```tsx
interface Stock { ... }  // 금지! modules/*/types/에 정의
```

---

## 필수 import 패턴

```tsx
// 컴포넌트
import { Button } from '@/shared/components/ui/button'
import { StockCard } from '@/modules/stocks/components/stock-card'

// API
import { stocksApi } from '@/modules/stocks/api'

// 타입
import type { Stock } from '@/modules/stocks/types'
```

---

## 새 컴포넌트 추가 시

1. 적절한 SSOT 위치에 파일 생성
2. Props 인터페이스 정의
3. className 전달 지원
4. 로딩 상태 지원 (필요시)
5. 접근성 고려

---

## 참고 문서

- `docs/guide/frontend/` - 프론트엔드 가이드
- `CLAUDE.md` - 프로젝트 규칙
