# Pre-Commit Mandatory Checklist

**CRITICAL**: 이 체크리스트는 모든 커밋 전에 반드시 실행되어야 합니다.
하나라도 실패하면 커밋을 진행할 수 없습니다.

---

## 커밋 전 필수 단계 (BLOCKER)

### ✅ Step 1: SSOT 준수 확인

**질문**: 생성/수정한 파일이 올바른 SSOT 위치에 있는가?

#### Backend 파일 위치 검증

```bash
# 각 레이어의 repository.go 위치 확인
ls internal/*/repository.go

# 예상 결과:
# internal/s0_data/repository.go
# internal/s1_universe/repository.go
# internal/s2_signals/repository.go
# internal/selection/repository.go
# internal/portfolio/repository.go
# internal/execution/repository.go
# internal/audit/repository.go
```

#### ❌ 금지된 위치
- `internal/data/repos/` - 중앙 집중식 금지!
- `internal/common/` - 공통 레이어에 비즈니스 로직 금지!

#### ✅ SSOT 준수 기준
- [ ] 각 레이어의 데이터 로직은 `<layer>/repository.go`에만 존재
- [ ] 타입 정의는 `internal/contracts/`에만 존재 (레이어 간 공유 타입)
- [ ] 환경변수는 `pkg/config/`에서만 읽기
- [ ] DB 연결은 `pkg/database/`에서만 생성

**실패 시**: 파일을 올바른 SSOT 위치로 이동 후 재시도

---

### ✅ Step 2: 관련 문서 업데이트 확인

**질문**: 코드 변경에 따라 업데이트해야 할 문서가 있는가?

#### 체크 항목

| 코드 변경 유형 | 업데이트 필수 문서 |
|----------------|-------------------|
| 새 레이어/모듈 추가 | `docs-site/docs/guide/backend/<layer>.md` |
| API 엔드포인트 추가 | `docs-site/docs/guide/api/` |
| 데이터베이스 스키마 변경 | `docs-site/docs/guide/database/` |
| Phase 완료 | `docs-site/docs/guide/overview/development-schedule.md` |
| 아키텍처 변경 | `docs-site/docs/guide/architecture/` |

#### 실행 명령어

```bash
# 최근 변경된 코드 파일 확인
git diff --name-only HEAD

# 관련 문서가 업데이트되었는지 확인
git diff --name-only HEAD | grep "docs-site/docs/"
```

**기준**:
- [ ] 새 기능 추가 → 해당 레이어 문서에 설명 추가
- [ ] Phase 완료 → development-schedule.md 상태 업데이트
- [ ] API 변경 → API 문서 업데이트

**실패 시**: 관련 문서를 먼저 업데이트한 후 코드와 함께 커밋

---

### ✅ Step 3: 빌드 및 테스트 통과

**질문**: 코드가 빌드되고 테스트가 통과하는가?

#### Backend

```bash
cd backend

# 1. 린트 검사
make lint

# 2. 빌드 검사
make build

# 3. 테스트 실행
make test
```

#### Frontend

```bash
cd frontend

# 1. 린트 검사
pnpm lint

# 2. 타입 체크
pnpm typecheck

# 3. 빌드 검사
pnpm build
```

**기준**:
- [ ] 모든 명령어가 에러 없이 통과
- [ ] Warning은 수정 권장 (BLOCKER 아님)

**실패 시**: 에러를 수정한 후 재시도

---

### ✅ Step 4: 원자적 커밋 준비

**질문**: 관련된 모든 변경사항을 함께 커밋하는가?

#### 함께 커밋해야 하는 파일 패턴

```bash
# 예시 1: 새 레이어 구현
git add internal/s2_signals/repository.go
git add internal/s2_signals/momentum.go
git add docs-site/docs/guide/backend/signals-layer.md
git commit -m "feat(signals): implement S2 signals layer with momentum factor"

# 예시 2: Phase 완료
git add internal/audit/performance.go
git add internal/audit/repository.go
git add docs-site/docs/guide/overview/development-schedule.md
git commit -m "feat(audit): complete S7 audit layer (Phase 6)"
```

**원칙**:
- 코드 변경 + 관련 문서 = 하나의 커밋
- 여러 레이어에 걸친 변경 = 여러 커밋으로 분리 (레이어별)

**기준**:
- [ ] 코드와 문서가 함께 스테이징됨 (`git status`로 확인)
- [ ] 커밋 메시지가 Conventional Commits 규칙 준수

**실패 시**: 누락된 파일을 `git add`한 후 재시도

---

## 커밋 실행

위 4단계를 모두 통과했다면:

```bash
git commit -m "type(scope): summary"
```

### 커밋 메시지 규칙

```
feat(signals): implement momentum factor calculation
fix(portfolio): correct weight calculation logic
docs(overview): update Phase 2 completion status
refactor(execution): extract order validation logic
tidy(brain): rename variables for clarity
```

---

## 이 체크리스트를 건너뛰면 안 되는 이유

1. **SSOT 위반**: 나중에 찾기 어려운 버그의 원인
2. **문서 누락**: 팀원/미래의 나 자신이 코드를 이해하지 못함
3. **테스트 실패**: 프로덕션 배포 시 장애 발생
4. **분리된 커밋**: Git 히스토리가 의미 없어짐

---

## Claude Code 실행 원칙

**이 체크리스트는 선택사항이 아닙니다.**

- 모든 커밋 전에 자동으로 실행
- 하나라도 실패하면 사용자에게 보고 후 중단
- 사용자가 명시적으로 "강제로 커밋해"라고 하지 않는 한 진행 금지
