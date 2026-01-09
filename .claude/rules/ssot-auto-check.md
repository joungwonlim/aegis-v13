# SSOT Auto-Check (파일 생성/수정 시 자동 검증)

**CRITICAL**: 파일을 생성하거나 수정하기 **전에** 반드시 이 체크를 실행해야 합니다.

---

## 작동 방식

Claude Code가 다음 작업을 하려고 할 때 자동으로 실행:
- 새 Go 파일 생성
- 새 TypeScript/React 파일 생성
- 기존 파일의 책임 범위 변경

---

## Backend SSOT 검증

### 1단계: 파일 생성 전 검색

**질문**: 이미 같은 책임을 가진 파일이 있는가?

```bash
# Repository 파일 검색
find backend/internal -name "repository.go" -o -name "*_repo.go"

# Service 파일 검색
find backend/internal -name "service.go" -o -name "*_service.go"
```

**기준**:
- 같은 책임의 파일이 **다른 위치**에 있으면 → SSOT 위반!
- 기존 파일을 수정하거나 올바른 위치로 이동해야 함

#### ❌ 위반 예시

```bash
# 발견됨:
internal/data/repos/signal_repo.go       # ❌ 잘못된 위치
internal/s2_signals/repository.go        # ✅ 올바른 위치

# 조치: signal_repo.go 삭제, repository.go 사용
```

---

### 2단계: 레이어별 SSOT 위치 확인

#### 규칙 테이블

| 파일 유형 | 허용 위치 | 금지 위치 |
|-----------|-----------|-----------|
| **Repository (데이터 영속성)** | `internal/<layer>/repository.go` | `internal/data/repos/`, `internal/common/` |
| **Service (비즈니스 로직)** | `internal/<layer>/service.go` | `pkg/`, `internal/common/` |
| **타입 정의 (레이어 간 공유)** | `internal/contracts/` | 각 레이어 내부 중복 정의 |
| **외부 API 호출** | `internal/external/<service>/` | 레이어에서 직접 호출 |
| **환경 변수** | `pkg/config/` | `os.Getenv()` 외부 사용 |
| **DB 연결 풀** | `pkg/database/` | `pgxpool.New()` 외부 사용 |
| **HTTP 클라이언트** | `pkg/httpclient/` | `http.Client{}` 외부 생성 |

#### 검증 명령어

**Repository 파일 생성 시**:

```bash
# 올바른 위치 확인
ls internal/s2_signals/repository.go  # ✅ 존재해야 함

# 잘못된 위치 확인
ls internal/data/repos/signal_repo.go  # ❌ 존재하면 안 됨
```

**타입 정의 시**:

```bash
# 레이어 간 공유 타입은 contracts에
cat internal/contracts/stock.go  # ✅

# 레이어 내부 타입은 레이어에
cat internal/s2_signals/momentum.go  # ✅ (내부 사용만)
```

---

### 3단계: 파일 내용 SSOT 검증

**질문**: 이 파일이 다른 SSOT를 침범하는가?

#### ❌ 금지 패턴

```go
// ❌ 레이어에서 직접 환경변수 읽기
apiKey := os.Getenv("API_KEY")

// ✅ config에서 주입받기
func NewService(cfg *config.Config) *Service {
    apiKey := cfg.APIKey
}
```

```go
// ❌ 레이어에서 직접 DB 연결
pool, _ := pgxpool.New(ctx, connString)

// ✅ database 패키지에서 주입받기
func NewRepository(pool *pgxpool.Pool) *Repository {
    return &Repository{pool: pool}
}
```

```go
// ❌ 레이어에서 외부 API 직접 호출
resp, _ := http.Get("https://api.stock.com/prices")

// ✅ external 패키지 사용
func NewService(stockAPI *external.StockAPI) *Service {
    prices := stockAPI.GetPrices()
}
```

#### 검증 명령어

```bash
# 금지된 패턴 검색
cd backend

# os.Getenv() 외부 사용 검색 (pkg/config/ 제외)
rg "os\.Getenv" internal/

# pgxpool.New() 외부 사용 검색 (pkg/database/ 제외)
rg "pgxpool\.New" internal/

# http.Client{} 외부 생성 검색 (pkg/httpclient/ 제외)
rg "http\.Client\{" internal/
```

**기준**: 위 명령어가 결과를 반환하면 → SSOT 위반!

---

## Frontend SSOT 검증

### 1단계: 컴포넌트 생성 전 검색

**질문**: 이미 비슷한 컴포넌트가 있는가?

```bash
# UI 컴포넌트 검색
ls frontend/src/shared/components/ui/

# 도메인 컴포넌트 검색
ls frontend/src/modules/*/components/
```

**기준**:
- 비슷한 컴포넌트가 있으면 → 재사용 또는 확장
- 새로 만들 필요가 있으면 → 올바른 SSOT 위치에 생성

---

### 2단계: 파일 유형별 SSOT 위치

| 파일 유형 | 허용 위치 | 금지 위치 |
|-----------|-----------|-----------|
| **UI 컴포넌트** | `shared/components/ui/` | 페이지, 모듈 컴포넌트 |
| **도메인 컴포넌트** | `modules/*/components/` | 페이지, shared |
| **API 호출** | `modules/*/api.ts` | 컴포넌트, 페이지 |
| **타입 정의** | `modules/*/types/` | 인라인 interface |
| **훅** | `shared/hooks/` | 컴포넌트 내부 |

---

### 3단계: 파일 내용 SSOT 검증

#### ❌ 금지 패턴

```typescript
// ❌ 컴포넌트에서 직접 fetch
const data = await fetch('/api/stocks')

// ✅ api.ts 사용
import { stocksApi } from '@/modules/stocks/api'
const data = await stocksApi.getList()
```

```tsx
// ❌ 페이지에서 컴포넌트 정의
function StockCard() { ... }
export default function Page() { ... }

// ✅ 별도 파일로 분리
// modules/stocks/components/stock-card.tsx
export function StockCard() { ... }
```

```typescript
// ❌ 인라인 타입 정의
interface Stock { code: string; ... }

// ✅ types에 정의
// modules/stocks/types/stock.ts
export interface Stock { code: string; ... }
```

#### 검증 명령어

```bash
cd frontend

# 직접 fetch 호출 검색 (api.ts 제외)
rg "fetch\(" src/modules --glob '!**/api.ts'

# 페이지 내 컴포넌트 정의 검색
rg "^function [A-Z].*\(" src/app

# 인라인 interface 검색 (types/ 제외)
rg "^interface [A-Z]" src/modules --glob '!**/types/**'
```

**기준**: 위 명령어가 결과를 반환하면 → SSOT 위반!

---

## 위반 시 조치

### 자동 중단

Claude Code는 다음 상황에서 작업을 중단하고 사용자에게 보고:

1. **파일이 잘못된 위치에 생성되려 함**
   - 예: `internal/data/repos/signal_repo.go` 대신 `internal/s2_signals/repository.go`

2. **기존 SSOT 파일이 있는데 중복 생성하려 함**
   - 예: `repository.go`가 이미 있는데 `*_repo.go` 생성

3. **금지된 패턴이 코드에 포함됨**
   - 예: 레이어에서 `os.Getenv()` 사용

### 보고 형식

```markdown
⚠️ SSOT 위반 감지!

**문제**: signal_repo.go를 internal/data/repos/에 생성하려 했습니다.

**올바른 위치**: internal/s2_signals/repository.go

**조치**: 파일을 올바른 위치에 생성하시겠습니까? (Y/n)
```

---

## Claude Code 실행 원칙

1. **파일 생성 전**: 항상 이 체크를 실행
2. **기존 파일 확인**: `find`, `ls`, `rg`로 검색
3. **위반 감지 시**: 즉시 중단, 사용자에게 보고
4. **사용자 승인 후**: 올바른 위치에 파일 생성

**이 체크를 건너뛰면 안 되는 이유**:
- SSOT 위반은 나중에 찾기 매우 어려움
- 리팩토링 비용이 기하급수적으로 증가
- 팀원들이 "어디에 코드를 추가해야 하는지" 혼란스러워함
