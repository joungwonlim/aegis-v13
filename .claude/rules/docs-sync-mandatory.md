# Documentation Sync Mandatory (문서 동기화 강제)

**CRITICAL**: 코드를 변경한 후, 커밋하기 전에 반드시 관련 문서를 업데이트해야 합니다.

---

## 작동 방식

Claude Code가 다음 작업을 완료했을 때 자동으로 실행:
- 새 레이어/모듈 구현 완료
- API 엔드포인트 추가
- 데이터베이스 스키마 변경
- Phase 완료
- 아키텍처 변경

---

## 코드 변경 유형별 필수 문서 업데이트

### 1. 새 레이어/모듈 구현

**코드 변경**:
```bash
# 예: S2 Signals 레이어 구현
internal/s2_signals/repository.go
internal/s2_signals/momentum.go
internal/s2_signals/technical.go
```

**필수 문서 업데이트**:
```bash
docs-site/docs/guide/backend/signals-layer.md
```

**업데이트 내용**:
- 레이어 개요
- 주요 함수 설명
- 사용 예시 코드
- 데이터 흐름

#### 검증 명령어

```bash
# 레이어 파일 변경 확인
git diff --name-only HEAD | grep "internal/s2_signals/"

# 관련 문서 변경 확인
git diff --name-only HEAD | grep "docs-site/docs/guide/backend/signals-layer.md"
```

**기준**: 코드 변경 있음 + 문서 변경 없음 = ❌ BLOCKER

---

### 2. API 엔드포인트 추가/변경

**코드 변경**:
```bash
# 예: 새 API 추가
internal/api/handlers/signals.go
```

**필수 문서 업데이트**:
```bash
docs-site/docs/guide/api/signals-api.md
```

**업데이트 내용**:
- 엔드포인트 경로
- Request/Response 스키마
- 예시 cURL 명령어
- 에러 코드

#### 검증 명령어

```bash
# API 핸들러 변경 확인
git diff --name-only HEAD | grep "internal/api/"

# 관련 문서 변경 확인
git diff --name-only HEAD | grep "docs-site/docs/guide/api/"
```

---

### 3. 데이터베이스 스키마 변경

**코드 변경**:
```bash
# 예: 새 테이블 추가
db/migrations/000006_create_signals_table.sql
```

**필수 문서 업데이트**:
```bash
docs-site/docs/guide/database/schema.md
```

**업데이트 내용**:
- ERD 다이어그램 업데이트
- 테이블 정의
- 인덱스 정의
- 관계 설명

#### 검증 명령어

```bash
# 마이그레이션 파일 변경 확인
git diff --name-only HEAD | grep "db/migrations/"

# 관련 문서 변경 확인
git diff --name-only HEAD | grep "docs-site/docs/guide/database/"
```

---

### 4. Phase 완료 ⭐ (가장 자주 누락됨)

**코드 변경**:
```bash
# 예: Phase 2 (S2 Signals) 완료
internal/s2_signals/repository.go
internal/s2_signals/momentum.go
internal/s2_signals/technical.go
```

**필수 문서 업데이트**:
```bash
docs-site/docs/guide/overview/development-schedule.md
```

**업데이트 내용**:

```markdown
## Phase 2: S2 - Signals Layer

**상태**: ✅ 완료 (2024-01-15)

**구현 내역**:
- [x] Momentum factor calculation
- [x] Technical factor calculation
- [x] Value factor calculation
- [x] Quality factor calculation
- [x] Flow (수급) factor calculation
- [x] Event factor calculation
- [x] Repository layer
```

#### 검증 명령어

```bash
# Phase 관련 파일 변경 확인
git diff --name-only HEAD | grep "internal/s2_signals/"

# development-schedule.md 변경 확인
git diff --name-only HEAD | grep "development-schedule.md"
```

**기준**: Phase 완료 코드 커밋 시 development-schedule.md 미업데이트 = ❌ BLOCKER

---

### 5. 아키텍처 변경

**코드 변경**:
```bash
# 예: 새 패턴 도입
internal/contracts/  # 새 인터페이스 정의
pkg/database/        # 새 DB 풀 관리 방식
```

**필수 문서 업데이트**:
```bash
docs-site/docs/guide/architecture/system-architecture.md
docs-site/docs/guide/architecture/data-flow.md
```

**업데이트 내용**:
- 아키텍처 다이어그램
- 레이어 간 의존성
- 데이터 흐름
- 설계 결정 사항

---

## 자동 문서 체크 프로세스

### Step 1: 코드 변경 감지

```bash
# 최근 변경된 파일 목록
git diff --name-only HEAD
```

### Step 2: 변경 유형 분류

```bash
# Backend 레이어 변경
git diff --name-only HEAD | grep "internal/s[0-7]_"

# API 변경
git diff --name-only HEAD | grep "internal/api/"

# DB 스키마 변경
git diff --name-only HEAD | grep "db/migrations/"
```

### Step 3: 필수 문서 확인

각 변경 유형에 대해 대응하는 문서가 업데이트되었는지 확인:

```bash
# 예: Signals 레이어 변경 시
if git diff --name-only HEAD | grep -q "internal/s2_signals/"; then
  if ! git diff --name-only HEAD | grep -q "docs-site/docs/guide/backend/signals-layer.md"; then
    echo "❌ BLOCKER: signals-layer.md 업데이트 필요"
    exit 1
  fi
fi
```

### Step 4: Phase 완료 특수 체크

```bash
# Phase 구현 파일이 있는지 확인
PHASE_FILES=$(git diff --name-only HEAD | grep "internal/s[0-7]_" | wc -l)

if [ "$PHASE_FILES" -gt 0 ]; then
  # development-schedule.md 업데이트 확인
  if ! git diff --name-only HEAD | grep -q "development-schedule.md"; then
    echo "❌ BLOCKER: development-schedule.md 업데이트 필요"
    echo "Phase 구현 완료 시 반드시 개발 일정 문서를 업데이트하세요."
    exit 1
  fi
fi
```

---

## 위반 시 조치

### 자동 중단 및 보고

Claude Code는 다음 상황에서 커밋을 중단하고 사용자에게 보고:

#### 예시 1: Phase 완료 시 development-schedule.md 미업데이트

```markdown
⚠️ 문서 동기화 필요!

**문제**: S2 Signals 레이어를 구현했지만 development-schedule.md가 업데이트되지 않았습니다.

**필수 조치**:
1. docs-site/docs/guide/overview/development-schedule.md 열기
2. Phase 2 상태를 "✅ 완료"로 변경
3. 완료 날짜 추가
4. 구현 내역 체크

**계속하시겠습니까?** (문서 먼저 업데이트 후 Y 입력)
```

#### 예시 2: 새 레이어 구현 시 레이어 문서 미작성

```markdown
⚠️ 문서 동기화 필요!

**문제**: internal/audit/ 레이어를 구현했지만 audit-layer.md가 없습니다.

**필수 조치**:
1. docs-site/docs/guide/backend/audit-layer.md 생성
2. 레이어 개요 작성
3. 주요 함수 설명 추가
4. 사용 예시 코드 추가

**자동 생성하시겠습니까?** (Y/n)
```

---

## Frontend 문서 동기화

### UI 컴포넌트 추가 시

**코드 변경**:
```bash
frontend/src/shared/components/ui/new-component.tsx
```

**필수 문서 업데이트**:
```bash
docs-site/docs/guide/ui/components.md
```

### 새 페이지 추가 시

**코드 변경**:
```bash
frontend/src/app/dashboard/page.tsx
```

**필수 문서 업데이트**:
```bash
docs-site/docs/guide/frontend/pages.md
```

---

## Claude Code 실행 원칙

### 코드 작업 완료 시

1. **자동 체크 실행**: 위의 검증 명령어 실행
2. **필수 문서 확인**: 업데이트 필요 여부 판단
3. **문서 누락 감지**: 사용자에게 보고 및 중단
4. **사용자 승인 대기**: 문서 업데이트 후 진행

### 예외 케이스

다음 경우에만 문서 업데이트 생략 가능:
- 버그 수정 (기능 변경 없음)
- 내부 리팩토링 (인터페이스 변경 없음)
- 테스트 코드 추가
- 린트 수정

**단, 사용자가 명시적으로 "문서 업데이트 생략"이라고 지시한 경우만 해당**

---

## 이 규칙을 지켜야 하는 이유

1. **코드-문서 불일치 방지**: 문서가 오래되면 쓸모없어짐
2. **팀 협업 효율**: 다른 개발자가 문서를 믿고 작업할 수 있음
3. **온보딩 비용 감소**: 새 팀원이 문서만 보고 이해 가능
4. **미래의 나 자신**: 6개월 후 코드를 다시 볼 때 문서가 필수

**문서 없는 코드는 기술 부채입니다.**
