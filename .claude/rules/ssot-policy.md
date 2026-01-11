# SSOT Policy (Single Source of Truth)

프로젝트 전반에 걸친 단일 진실원천 규칙입니다.

---

## Backend SSOT 경로

각 책임은 지정된 위치에서만 처리합니다.

| 책임 | 허용 위치 | 금지 패턴 |
|------|----------|----------|
| env 파싱 | `pkg/config` | `os.Getenv()`, `godotenv.Load()` 외부 사용 |
| DB pool | `pkg/database` | `pgxpool.New()`, `pgxpool.ParseConfig()` 외부 사용 |
| HTTP client | `pkg/httputil` | `http.Client{}`, `http.NewRequest()` 외부 사용 |
| 타입 정의 | `internal/contracts` | 레이어에서 중복 정의 |
| 외부 API | `internal/external` | 레이어에서 직접 호출 |

---

## ⚠️ Claude Code DB 접속 규칙 (BLOCKER)

**Claude Code가 데이터베이스에 접속해야 할 때 반드시 따라야 하는 규칙입니다.**

### 절대 금지

```bash
# ❌ 절대 금지: 임의의 credentials 사용
psql -U postgres -d aegis ...
psql -U root -d database ...
PGPASSWORD=password psql ...
```

### 필수 절차

**Step 1**: `pkg/config/config.go` 읽기 (MANDATORY)
```bash
# DB 접속 전 반드시 설정 파일 확인
cat backend/pkg/config/config.go | grep -A5 "DatabaseConfig"
```

**Step 2**: 기본값 확인
```go
// pkg/config/config.go에서 확인
DB_HOST     = getEnv("DB_HOST", "localhost")
DB_PORT     = getEnv("DB_PORT", "5432")
DB_NAME     = getEnv("DB_NAME", "aegis_v13")      // ⭐ 기본값
DB_USER     = getEnv("DB_USER", "aegis_v13")      // ⭐ 기본값
DB_PASSWORD = getEnv("DB_PASSWORD", "")
```

**Step 3**: SSOT 기본값으로 접속
```bash
# ✅ 올바른 접속 방법
psql -h localhost -p 5432 -U aegis_v13 -d aegis_v13
```

### 위반 시 조치

1. **즉시 중단**: 임의 credentials 사용 시도 시 작업 중단
2. **SSOT 확인**: `pkg/config/config.go` 읽기
3. **재시도**: 올바른 credentials로 재접속

### 이 규칙이 필요한 이유

- DB 연결 정보는 환경마다 다를 수 있음
- 임의 추측은 보안 위험 + 시간 낭비
- SSOT 원칙: 설정 정보는 `pkg/config/`에서만 관리

### 위반 예시

```go
// ❌ 금지: service 레이어에서 직접 env 읽기
func NewService() *Service {
    apiKey := os.Getenv("API_KEY")  // 위반!
}

// ✅ 허용: config에서 주입받기
func NewService(cfg *config.Config) *Service {
    apiKey := cfg.APIKey  // OK
}
```

---

## Frontend SSOT 경로

| 책임 | 허용 위치 | 금지 |
|------|----------|------|
| API 호출 | `modules/*/api.ts` | 직접 `fetch()`, `axios` 사용 |
| 타입 정의 | `modules/*/types/` | 컴포넌트/페이지에서 인라인 타입 정의 |
| 컴포넌트 | `modules/*/components/` | 페이지에서 정의 |
| UI 기본 | `shared/components/ui/` | 직접 스타일링 |

### 위반 예시

```typescript
// ❌ 금지: 컴포넌트에서 직접 fetch
const data = await fetch('/api/prices')  // 위반!

// ✅ 허용: api.ts 사용
import { api } from '@/modules/stocks/api'
const data = await api.getPrices()  // OK
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

## 재현성 (Reproducibility)

모든 실행/의사결정에 다음 필드를 기록합니다.

| 필드 | 용도 | 예시 |
|------|------|------|
| `run_id` | 실행 고유 ID | `run_20240115_143052` |
| `job_runs` | 작업 실행 기록 | job_runs 테이블 참조 |
| `feature_version` | 피처 버전 | `v1.2.0` |
| `prompt_hash` | AI 프롬프트 해시 | `sha256:abc123...` |
| `git_sha` | 코드 버전 | `a1b2c3d` |

---

## SSOT 검증

### 검증 항목

| 검사 | 위반 시 |
|------|--------|
| env 외부 읽기 | CI 실패 |
| DB pool 외부 생성 | CI 실패 |
| HTTP client 외부 생성 | CI 실패 |
| 외부 API 직접 호출 | CI 실패 |

---

## 문서 우선 규칙

모듈 추가/변경 시:

```
1. docs-site/docs/guide/* 먼저 업데이트
2. 그 다음 구현
```

### 이유

- API 계약이 문서에 먼저 정의되어야 함
- 구현이 문서를 따라가야 함 (역순 금지)
- 문서와 코드의 불일치 방지

---

## 체크리스트

### 구현 전

- [ ] 해당 책임의 SSOT 위치 확인했나?
- [ ] 기존 구현 검색했나? (`rg`)
- [ ] docs-site/docs/guide/* 먼저 업데이트했나?

### 구현 후

- [ ] `make lint && make test` 통과했나?
- [ ] SSOT 위치 외부에서 직접 호출 없나?
- [ ] 재현성 필드 기록했나? (해당 시)
