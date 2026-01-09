# Getting Started

> 프로젝트 설정 및 실행 가이드

---

## 요구사항

| 도구 | 버전 | 확인 명령 |
|------|------|----------|
| Go | 1.21+ | `go version` |
| Node.js | 18+ | `node -v` |
| pnpm | 8+ | `pnpm -v` |
| PostgreSQL | 15+ | `psql --version` |
| Make | - | `make --version` |

---

## 1. 저장소 클론

```bash
git clone https://github.com/your/aegis-v13.git
cd aegis-v13
```

---

## 2. 환경 설정

### Backend

```bash
cd backend
cp .env.example .env
```

`.env` 파일 편집:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=aegis
DB_PASSWORD=your_password
DB_NAME=aegis_v13

# KIS API (한국투자증권)
KIS_APP_KEY=your_app_key
KIS_APP_SECRET=your_app_secret
KIS_ACCOUNT_NO=your_account

# DART API (공시정보)
DART_API_KEY=your_dart_key
```

### Frontend

```bash
cd frontend
cp .env.example .env.local
```

`.env.local` 파일 편집:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

---

## 3. 데이터베이스 설정

### PostgreSQL 설치 (Docker)

```bash
docker run -d \
  --name aegis-postgres \
  -e POSTGRES_USER=aegis \
  -e POSTGRES_PASSWORD=your_password \
  -e POSTGRES_DB=aegis_v13 \
  -p 5432:5432 \
  postgres:15
```

### 스키마 생성

```bash
cd backend
make migrate-up
```

---

## 4. 의존성 설치

### Backend

```bash
cd backend
make deps
# 또는
go mod download
```

### Frontend

```bash
cd frontend
pnpm install
```

---

## 5. 실행

### 개발 모드

터미널 1 - Backend:
```bash
cd backend
make run
# Server running on http://localhost:8080
```

터미널 2 - Frontend:
```bash
cd frontend
pnpm dev
# Ready on http://localhost:3000
```

### 프로덕션 빌드

```bash
# Backend
cd backend
make build
./bin/aegis

# Frontend
cd frontend
pnpm build
pnpm start
```

---

## 6. 확인

### API 헬스체크

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### 프론트엔드 접속

브라우저에서 http://localhost:3000 접속

---

## 프로젝트 구조

```
aegis-v13/
├── backend/
│   ├── cmd/api/          # 엔트리포인트
│   ├── internal/         # 비즈니스 로직
│   │   ├── contracts/    # 타입/인터페이스
│   │   ├── brain/        # 오케스트레이터
│   │   ├── data/         # S0-S1
│   │   ├── signals/      # S2
│   │   ├── selection/    # S3-S4
│   │   ├── portfolio/    # S5
│   │   ├── execution/    # S6
│   │   └── audit/        # S7
│   ├── pkg/              # 공용 패키지
│   └── migrations/       # DB 마이그레이션
│
├── frontend/
│   ├── src/
│   │   ├── app/          # Next.js App Router
│   │   ├── modules/      # 도메인 모듈
│   │   └── shared/       # 공용 코드
│   └── public/
│
└── docs/                 # 문서
```

---

## 자주 사용하는 명령어

### Backend

```bash
make run          # 개발 서버 실행
make build        # 빌드
make test         # 테스트
make lint         # 린트
make migrate-up   # 마이그레이션 적용
make migrate-down # 마이그레이션 롤백
```

### Frontend

```bash
pnpm dev          # 개발 서버
pnpm build        # 프로덕션 빌드
pnpm test         # 테스트
pnpm lint         # 린트
pnpm typecheck    # 타입 체크
```

---

## 다음 단계

1. [System Overview](../architecture/system-overview.md) - 전체 아키텍처 이해
2. [Data Flow](../architecture/data-flow.md) - 7단계 파이프라인
3. [Contracts](../architecture/contracts.md) - 핵심 타입/인터페이스

---

**Prev**: [Tech Stack](./tech-stack.md)
**Next**: [System Overview](../architecture/system-overview.md)
