# Frontend 컴포넌트 강제 규칙

**CRITICAL: 이 규칙은 모든 frontend 작업에서 필수로 지켜져야 합니다.**

---

## 필수 규칙 (Non-negotiable)

### 1. 컴포넌트 모듈만 사용 (SSOT)

모든 페이지와 컴포넌트는 **반드시** 정해진 위치의 컴포넌트만 사용:

```tsx
✅ 허용:
import { Button } from '@/shared/components/ui/button'
import { Card } from '@/shared/components/ui/card'
import { StockCard } from '@/modules/stocks/components/stock-card'

❌ 금지:
<button className="px-4 py-2 bg-blue-500">...</button>
<div className="rounded-lg border p-4">...</div>
// 직접 스타일링 금지!
```

### 2. 숫자 표시는 font-mono 필수

모든 숫자(가격, 수량, 퍼센트)는 **반드시** monospace 폰트 사용:

```tsx
✅ 허용:
<span className="font-mono">72,300원</span>

❌ 금지:
<span>72,300원</span>  // font-mono 없음
```

### 3. 디자인 토큰 사용

하드코딩 금지:

```tsx
❌ 금지:
style={{ color: '#ef4444', padding: '16px' }}
```

---

## 컴포넌트 위치 규칙

| 타입 | 위치 | 용도 |
|------|------|------|
| UI | `shared/components/ui/` | Button, Card, Badge 등 |
| Domain | `modules/*/components/` | StockChart, SignalBadge 등 |

**새 컴포넌트 생성 시: 적절한 폴더에 배치 필수!**

---

## 작업 전 체크리스트

Claude Code가 작업 시작 전 확인:

- [ ] 기존 컴포넌트 검색 (`ls shared/components/ui/`)
- [ ] 재사용 가능한 컴포넌트 있는지 확인
- [ ] 없으면 적절한 위치에 생성
- [ ] 있으면 기존 컴포넌트 사용

---

## 작업 완료 후 체크리스트

- [ ] shared/modules 컴포넌트만 import
- [ ] 숫자에 font-mono 적용
- [ ] 인라인 스타일 하드코딩 없음

---

## 참고 문서

- `/.claude/rules/frontend-ssot.md` - SSOT 정책
