# CLAUDE.md

> Aegis v13 - 기관급 퀀트 트레이딩 시스템

---

## v13 설계 철학 ⭐

> **v10은 너무 복잡하고 비효율적이었다. v13은 명확·간결·효율을 추구한다.**

### v10 vs v13

| 관점 | v10 (문제점) | v13 (목표) |
|------|-------------|-----------|
| 구조 | 분산된 바이너리, 중복 로직 | 통합 CLI, 단일 책임 |
| 코드 | 과도한 추상화, 복잡한 상태머신 | 직관적 흐름, 최소 추상화 |
| 설정 | 수십 개 파라미터, 하드코딩 | 핵심만 설정, 합리적 기본값 |
| 문서 | 코드와 불일치, 분산 | SSOT, 문서-코드 동기화 |

### 핵심 원칙

1. **명확함 (Clarity)**: 코드를 읽으면 의도가 바로 파악되어야 함
2. **간결함 (Simplicity)**: 필요한 것만 구현, 과잉 설계 금지
3. **효율성 (Efficiency)**: 중복 제거, 단일 경로, 빠른 실행

### v10 참고 시 주의

v10의 검증된 로직(청산 전략, 스코어링 등)은 **참고만** 할 것.
그대로 복사하지 말고, v13 철학에 맞게 **단순화**하여 재구현.

```
❌ v10 코드 복사 붙여넣기
❌ v10의 복잡한 상태머신 그대로 사용
❌ 과도한 설정 파라미터 추가

✅ v10 핵심 로직만 추출
✅ v13 구조에 맞게 단순화
✅ 불필요한 복잡성 제거
```

---

## 절대 규칙

### 1. 한국어 응답 필수
모든 응답은 한국어로 작성 (코드/커밋 메시지 제외)

### 2. 새 파일 생성 금지
기존 파일 수정 우선. 새 파일은 꼭 필요할 때만.

### 3. SSOT 준수
정해진 위치에서만 해당 책임 수행.

### 4. 문서-코드 동기화 필수 ⚠️
**코드 변경 시 관련 문서도 반드시 함께 업데이트.**
- 폴더 구조 변경 → `development-schedule.md`, `folder-structure.md` 수정
- DB 스키마 변경 → `schema-design.md` 수정
- API 변경 → 관련 레이어 문서 수정
- **문서와 코드가 불일치하면 안 됨**

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
│   ├── cmd/quant/         # 통합 CLI (Cobra)
│   │   ├── main.go
│   │   └── commands/      # 서브커맨드
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
│   ├── pkg/               # 공용 패키지
│   └── migrations/        # DB 마이그레이션
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

### Documentation

| 책임 | 허용 위치 | 금지 |
|------|----------|------|
| 모든 문서 | `docs-site/docs/guide/` | `backend/`, `frontend/` 내 .md 생성 |
| DB 스키마 | `docs-site/docs/guide/database/` | `backend/migrations/*.md` |
| 아키텍처 | `docs-site/docs/guide/architecture/` | 프로젝트 루트 .md |
| API 문서 | `docs-site/docs/guide/api/` | 코드 내 별도 문서 |

> ⚠️ **문서는 반드시 `docs-site/` 에만 생성**. README.md, CLAUDE.md 제외.

---

## Build & Run

### Backend
```bash
cd backend
make deps       # 의존성
make build      # 빌드 (bin/quant)
make test       # 테스트
make lint       # 린트

# 실행 (통합 CLI)
go run ./cmd/quant [command]
go run ./cmd/quant fetcher collect all   # 데이터 수집
go run ./cmd/quant api                   # API 서버 (예정)
go run ./cmd/quant test-db               # DB 테스트
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

## 작업 순서 (강제 실행)

### 1. 문서 확인 (BLOCKER)
- `docs-site/docs/guide/` 관련 문서 읽기
- 구현할 기능이 이미 문서화되어 있는지 확인

### 2. 파일 생성 전 SSOT 검증 (BLOCKER)
- **강제 규칙**: `.claude/rules/ssot-auto-check.md`
- 파일을 생성하기 **전에** 올바른 SSOT 위치인지 확인
- 기존 파일이 있는지 검색 (`find`, `ls`, `rg`)
- 위반 감지 시 즉시 중단

### 3. 최소 변경
- 필요한 것만 수정
- 과도한 추상화 금지

### 4. 코드 완료 후 문서 동기화 (BLOCKER)
- **강제 규칙**: `.claude/rules/docs-sync-mandatory.md`
- 코드 변경 시 관련 문서 **반드시** 업데이트
- Phase 완료 시 `development-schedule.md` 필수 업데이트
- 문서 없이 커밋 금지

### 5. 커밋 전 필수 체크 (BLOCKER)
- **강제 규칙**: `.claude/rules/pre-commit-mandatory.md`
- 4단계 체크리스트 실행:
  1. ✅ SSOT 준수 확인
  2. ✅ 관련 문서 업데이트 확인
  3. ✅ 빌드 및 테스트 통과
  4. ✅ 원자적 커밋 준비 (코드+문서 함께)
- **하나라도 실패하면 커밋 불가**

### 6. 커밋 실행
- 규칙에 맞게 커밋 메시지 작성
- 코드와 문서를 함께 커밋

---

## ⚠️ 강제 실행 규칙 (MANDATORY)

이 규칙들은 **선택사항이 아닙니다**. 위반 시 작업이 자동으로 중단됩니다.

| 규칙 파일 | 실행 시점 | 설명 |
|-----------|----------|------|
| `.claude/rules/ssot-auto-check.md` | 파일 생성/수정 전 | SSOT 위치 자동 검증 |
| `.claude/rules/docs-sync-mandatory.md` | 코드 완료 후 | 문서 동기화 강제 |
| `.claude/rules/pre-commit-mandatory.md` | 커밋 전 | 4단계 체크리스트 실행 |

**Claude Code는 이 규칙들을 자동으로 실행하며, 사용자가 명시적으로 "강제로 진행해"라고 하지 않는 한 위반 시 중단합니다.**

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
