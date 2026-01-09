# Tech Stack

> Aegis v13 기술 스택

---

## Overview

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Frontend** | Next.js 14+ | React 기반 UI |
| **Backend** | Go 1.21+ | BFF API 서버 |
| **Database** | PostgreSQL 15+ | 메인 데이터 저장소 |
| **Cache** | Redis (optional) | 세션/캐시 |

---

## Backend (Go)

### 핵심 라이브러리

```go
// HTTP
github.com/go-chi/chi/v5      // 라우터

// Database
github.com/jackc/pgx/v5       // PostgreSQL 드라이버

// Utilities
github.com/rs/zerolog         // 구조화 로깅
github.com/go-playground/validator/v10  // 검증
```

### BFF 패턴

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Next.js   │────▶│   Go BFF    │────▶│ PostgreSQL  │
│  (Client)   │     │  (Server)   │     │    (DB)     │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │ External API│
                    │ (KIS, DART) │
                    └─────────────┘
```

---

## Frontend (Next.js)

### 핵심 라이브러리

```json
{
  "next": "14.x",
  "react": "18.x",
  "typescript": "5.x",
  "tailwindcss": "3.x",
  "shadcn/ui": "latest",
  "@tanstack/react-query": "5.x"
}
```

### App Router 구조

```
app/
├── (dashboard)/      # 대시보드 레이아웃
├── (auth)/           # 인증 페이지
├── api/              # API Routes (필요시)
└── layout.tsx        # 루트 레이아웃
```

---

## Database (PostgreSQL)

### 스키마 구조

```sql
-- 레이어별 스키마
CREATE SCHEMA data;       -- 원천 데이터
CREATE SCHEMA signals;    -- 시그널 결과
CREATE SCHEMA selection;  -- 스크리닝/랭킹
CREATE SCHEMA portfolio;  -- 포트폴리오
CREATE SCHEMA execution;  -- 주문/체결
CREATE SCHEMA audit;      -- 성과 분석
```

---

## Development Tools

| Tool | Purpose |
|------|---------|
| `make` | 빌드/실행 자동화 |
| `golangci-lint` | Go 린트 |
| `pnpm` | 패키지 매니저 |
| `docker-compose` | 로컬 DB 실행 |

---

**Prev**: [Introduction](./introduction.md)
**Next**: [Getting Started](./getting-started.md)
