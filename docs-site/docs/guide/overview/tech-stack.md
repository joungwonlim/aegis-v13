---
sidebar_position: 2
title: Tech Stack
description: 기술 스택 소개
---

# Tech Stack

> Aegis v13 기술 스택

---

## Overview

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Frontend** | Next.js 14+ | React 기반 UI, shadcn/ui 컴포넌트 |
| **Backend (API)** | Go 1.21+ | BFF API 서버, 인증/권한, 데이터 계약 |
| **Backend (Worker)** | Go 1.21+ | 수집/지표/백테스트 등 장시간 작업 |
| **Database** | PostgreSQL 15+ | 시계열/포지션/주문/스냅샷 저장 |
| **Cache/Queue** | Redis | 캐시/레이트리밋/실시간 pub/sub |

### 도입 검토 후 보류된 기술

| 기술 | 보류 이유 |
|------|----------|
| **TimescaleDB** | 현재 데이터 규모(~수백만 rows)는 PostgreSQL로 충분. 1억 rows 이상 시 재검토 |
| **OpenTelemetry** | zerolog 구조화 로깅으로 충분. 분산 트레이싱 필요 시 재검토 |

---

## Backend (Go)

### 핵심 라이브러리

```go
// Database
github.com/jackc/pgx/v5       // PostgreSQL 드라이버
github.com/redis/go-redis/v9  // Redis 클라이언트

// Utilities
github.com/rs/zerolog         // 구조화 로깅
```

### 아키텍처

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Next.js   │────▶│   Go API    │────▶│ PostgreSQL  │
│  (Client)   │     │   (BFF)     │     │    (DB)     │
└─────────────┘     └─────────────┘     └─────────────┘
                           │                   ▲
                           ▼                   │
                    ┌─────────────┐     ┌─────────────┐
                    │    Redis    │     │  Go Worker  │
                    │(Cache/Queue)│     │ (장시간작업) │
                    └─────────────┘     └─────────────┘
                           │                   │
                           └───────┬───────────┘
                                   ▼
                            ┌─────────────┐
                            │ External API│
                            │ (KIS, DART) │
                            └─────────────┘
```

### CLI 명령어

#### 기본 실행 순서

```bash
# 1. 데이터 수집 (KIS, DART, Naver 전체)
go run ./cmd/quant fetcher collect all

# 2. 파이프라인 실행
go run ./cmd/quant brain run
```

#### 추가 옵션

**Brain 옵션**

```bash
# 특정 날짜로 실행
go run ./cmd/quant brain run --date 2026-01-10

# 자본금 지정 (기본: 1억원)
go run ./cmd/quant brain run --capital 200000000

# 실제 주문 없이 계획만 생성
go run ./cmd/quant brain run --dry-run
```

**Fetcher 옵션**

```bash
# 특정 소스만 수집
go run ./cmd/quant fetcher collect kis
go run ./cmd/quant fetcher collect dart
go run ./cmd/quant fetcher collect naver

# 비동기 수집 (큐에 작업 추가)
go run ./cmd/quant fetcher collect all --async
```

**서버 실행**

```bash
# Backend (8089 포트 kill 후 재시작)
go run ./cmd/quant backend start

# Frontend (3000 포트 kill 후 재시작)
go run ./cmd/quant frontend start
```

**Database 관리**

```bash
# 마이그레이션 실행 (새 테이블/스키마 생성)
go run ./cmd/quant migrate up

# 마이그레이션 롤백 (최신 1개)
go run ./cmd/quant migrate down

# 마이그레이션 상태 확인
go run ./cmd/quant migrate status

# 특정 버전까지 마이그레이션
go run ./cmd/quant migrate goto 20
```

**Forecast 모듈**

```bash
# 전체 실행 (detect → fill-forward → aggregate)
go run ./cmd/quant forecast run --from 2024-01-01

# 이벤트 감지만
go run ./cmd/quant forecast detect --from 2024-01-01 --to 2024-12-31

# 전방 성과 채우기
go run ./cmd/quant forecast fill-forward

# 통계 집계
go run ./cmd/quant forecast aggregate

# 특정 종목 예측 조회
go run ./cmd/quant forecast predict --code 005930
```

#### 참고사항

현재 `brain run`은 **재무 데이터 없이도** 작동하도록 설정되어 있습니다. 완전한 필터링을 위해서는:
1. DART에서 재무제표 수집 필요 (PER/PBR/ROE 필터용)
2. 매일 market_cap 데이터 수집 권장

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

## Redis

### 용도

| 기능 | 설명 |
|------|------|
| **레이트 리밋** | 외부 API 호출 제한 (KIS 초당 5회 등) |
| **캐시** | 시세/종목정보 캐싱으로 DB 부하 감소 |
| **Pub/Sub** | 실시간 시세 WebSocket fanout |
| **Job Queue** | 백그라운드 작업 큐 (선택적) |

### 설치 (macOS)

```bash
# 설치
brew install redis

# 백그라운드 실행 (시작 시 자동 실행)
brew services start redis

# 상태 확인
redis-cli ping  # PONG 응답

# 메모리 사용량: ~2MB (idle)
```

---

## Development Tools

| Tool | Purpose |
|------|---------|
| `make` | 빌드/실행 자동화 |
| `golangci-lint` | Go 린트 |
| `pnpm` | 패키지 매니저 |
| `brew services` | PostgreSQL/Redis 로컬 실행 |

### 로컬 환경 설정 (Docker 없이)

```bash
# PostgreSQL 설치 및 실행
brew install postgresql@15
brew services start postgresql@15

# Redis 설치 및 실행
brew install redis
brew services start redis

# 서비스 상태 확인
brew services list
```

---

**Prev**: [Introduction](./introduction.md)
**Next**: [Getting Started](./getting-started.md)
