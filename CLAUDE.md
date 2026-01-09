# CLAUDE.md

> Aegis v13 - 기관급 퀀트 트레이딩 시스템

---

## 절대 규칙

### 1. 한국어 응답 필수
모든 응답은 한국어로 작성 (코드/커밋 메시지 제외)

### 2. 새 파일 생성 금지
기존 파일 수정 우선. 새 파일은 꼭 필요할 때만.

### 3. SSOT 준수
정해진 위치에서만 해당 책임 수행.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | Next.js 14+ (App Router) |
| Backend | Go 1.21+ (BFF) |
| Database | PostgreSQL 15+ |

---

## 프로젝트 구조

```
aegis-v13/
├── backend/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── contracts/     # 타입/인터페이스 SSOT
│   │   ├── brain/         # 오케스트레이터
│   │   ├── data/          # S0-S1: 데이터/유니버스
│   │   ├── signals/       # S2: 시그널
│   │   ├── selection/     # S3-S4: 스크리닝/랭킹
│   │   ├── portfolio/     # S5: 포트폴리오
│   │   ├── execution/     # S6: 주문
│   │   ├── audit/         # S7: 성과분석
│   │   ├── external/      # 외부 API (KIS, DART)
│   │   └── api/           # HTTP 핸들러
│   └── pkg/               # 공용 패키지
├── frontend/
│   └── src/
│       ├── app/           # Next.js App Router
│       ├── modules/       # 도메인별 모듈
│       └── shared/        # 공용 컴포넌트/훅
└── docs-site/             # Docusaurus 문서 사이트
    └── docs/guide/        # 가이드 문서
```

---

## 7단계 파이프라인

```
S0: Data Quality  → 데이터 수집/검증
S1: Universe      → 투자 가능 종목
S2: Signals       → 팩터/이벤트 시그널
S3: Screener      → 1차 필터링 (Hard Cut)
S4: Ranking       → 종합 점수
S5: Portfolio     → 포트폴리오 구성
S6: Execution     → 주문 실행
S7: Audit         → 성과 분석
```

---

## SSOT 규칙

### Backend

| 책임 | 허용 위치 | 금지 |
|------|----------|------|
| 환경변수 | `pkg/config/` | `os.Getenv()` 외부 사용 |
| DB 연결 | `pkg/database/` | `pgx.Connect()` 외부 사용 |
| 타입 정의 | `internal/contracts/` | 레이어에서 중복 정의 |
| 외부 API | `internal/external/` | 레이어에서 직접 호출 |

### Frontend

| 책임 | 허용 위치 | 금지 |
|------|----------|------|
| API 호출 | `modules/*/api.ts` | 직접 `fetch()` |
| 타입 | `modules/*/types/` | 인라인 interface |
| 컴포넌트 | `modules/*/components/` | 페이지에서 정의 |
| UI 기본 | `shared/components/ui/` | 직접 스타일링 |

---

## Build & Run

### Backend
```bash
cd backend
make deps       # 의존성
make run        # 실행
make build      # 빌드
make test       # 테스트
make lint       # 린트
```

### Frontend
```bash
cd frontend
pnpm install    # 의존성
pnpm dev        # 개발 서버
pnpm build      # 빌드
pnpm test       # 테스트
pnpm lint       # 린트
pnpm typecheck  # 타입 체크
```

---

## Quality Gates (커밋 전 필수)

```bash
# Backend
cd backend && make lint && make test

# Frontend
cd frontend && pnpm lint && pnpm typecheck
```

---

## 커밋 규칙

```
feat|fix|perf|refactor|test|docs|chore(scope): summary
tidy(scope): summary   # 동작 변경 없는 정리
```

---

## 작업 순서

1. **문서 확인**: `docs-site/docs/guide/` 관련 문서 읽기
2. **SSOT 위치 확인**: 어디서 작업해야 하는지 확인
3. **최소 변경**: 필요한 것만 수정
4. **품질 게이트**: lint, test 통과
5. **커밋**: 규칙에 맞게

---

## 금지 패턴

### Go
```go
// ❌ 전역 상태
var globalDB *sql.DB

// ❌ 외부에서 env 읽기
apiKey := os.Getenv("API_KEY")

// ❌ 순환 참조
import "internal/brain"  // selection에서
```

### TypeScript
```tsx
// ❌ 직접 fetch
const data = await fetch('/api/stocks')

// ❌ 페이지에서 컴포넌트 정의
function Card() { ... }
export default function Page() { ... }

// ❌ 인라인 타입
interface Stock { ... }
```

---

## 문서 위치

| 문서 | 위치 |
|------|------|
| 시스템 구조 | `docs-site/docs/guide/architecture/` |
| 백엔드 레이어 | `docs-site/docs/guide/backend/` |
| 프론트엔드 | `docs-site/docs/guide/frontend/` |
| DB 스키마 | `docs-site/docs/guide/database/` |
| UI 디자인 | `docs-site/docs/guide/ui/` |
