---
sidebar_position: 8
title: Infrastructure
description: Redis, Database, HTTP Client 설정
---

# Infrastructure

> 인프라 계층: Redis, Database, HTTP Client

---

## Overview

| 패키지 | 역할 | 위치 |
|--------|------|------|
| **pkg/redis** | 캐시, 레이트 리밋, Pub/Sub | `pkg/redis/` |
| **pkg/database** | PostgreSQL 연결 풀 | `pkg/database/` |
| **pkg/httputil** | HTTP 클라이언트 (재시도, 로깅) | `pkg/httputil/` |
| **pkg/config** | 환경변수 관리 (SSOT) | `pkg/config/` |

---

## Redis

### 용도

| 기능 | 설명 | TTL |
|------|------|-----|
| **레이트 리밋** | 외부 API 호출 제한 | 자동 만료 |
| **캐시** | 시세/종목정보 캐싱 | 1분 ~ 24시간 |
| **Pub/Sub** | 실시간 시세 브로드캐스트 | - |

### 파일 구조

```
pkg/redis/
├── client.go      # 연결 관리
├── ratelimit.go   # 레이트 리미터
├── cache.go       # 캐시 헬퍼
└── redis_test.go  # 테스트
```

### 연결 설정

```go
import "github.com/wonny/aegis/v13/backend/pkg/redis"

// 클라이언트 생성
client, err := redis.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Redis 비활성화 시 graceful fallback
if !client.Enabled() {
    // 모든 기능이 no-op으로 동작
}
```

### 레이트 리미터

```go
// 생성
limiter := redis.NewRateLimiter(client, "aegis")

// 사전 정의된 설정
redis.KISRateLimit   // 5 req/sec (한국투자증권)
redis.DARTRateLimit  // 100 req/min (전자공시)
redis.NaverRateLimit // 10 req/sec (네이버 금융)

// 사용
allowed, remaining, err := limiter.Allow(ctx, redis.KISRateLimit)
if !allowed {
    // 요청 거부 또는 대기
}

// 블로킹 대기 (허용될 때까지)
err := limiter.Wait(ctx, redis.KISRateLimit)
```

### 캐시 헬퍼

```go
cache := redis.NewCache(client, "aegis")

// 기본 사용
var stock StockInfo
found, err := cache.Get(ctx, "stock:005930", &stock)

// TTL과 함께 저장
err := cache.Set(ctx, "stock:005930", stock, redis.TTLMedium)

// GetOrSet 패턴 (캐시 미스 시 함수 호출)
err := cache.GetOrSet(ctx, "stock:005930", &stock, redis.TTLMedium, func() (interface{}, error) {
    return fetchStockFromDB("005930")
})

// 사전 정의된 TTL
redis.TTLShort  // 1분 (실시간 시세)
redis.TTLMedium // 10분 (종목 정보)
redis.TTLLong   // 1시간 (마스터 데이터)
redis.TTLDaily  // 24시간 (일별 데이터)
```

### HTTP 클라이언트 통합

```go
// 레이트 리밋이 적용된 HTTP 클라이언트
httpClient := httputil.New(cfg, log).
    WithRateLimiter(limiter, redis.KISRateLimit)

// 요청 시 자동으로 레이트 리밋 대기
resp, err := httpClient.Get(ctx, "https://api.kis.com/...")
```

---

## 환경변수

```bash
# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_ENABLED=true  # false로 설정 시 모든 기능 비활성화
```

---

## 로컬 설정

```bash
# macOS (Homebrew)
brew install redis
brew services start redis

# 상태 확인
redis-cli ping  # → PONG

# 메모리 사용량 확인
redis-cli info memory | grep used_memory_human
# → used_memory_human:2.5M (idle 상태)
```

---

## 테스트

```bash
# Redis 패키지 테스트
cd backend
go test ./pkg/redis/...

# Redis 없이 테스트 (graceful fallback 확인)
REDIS_ENABLED=false go test ./...
```

---

## Scheduler

### Overview

스케줄러는 주기적인 작업들을 관리합니다.

```
internal/scheduler/
├── scheduler.go      # 스케줄러 코어
└── jobs/
    ├── data_collection.go  # 데이터 수집
    ├── maintenance.go      # 캐시 정리
    ├── universe.go         # Universe 생성
    └── forecast.go         # Forecast 파이프라인
```

### 등록된 작업

| 작업 | 스케줄 | 설명 |
|------|--------|------|
| `data_collection` | 매일 16:00 | 전체 데이터 수집 |
| `price_collection` | 평일 9-15시 매시간 | 가격 데이터 |
| `investor_flow` | 매일 17:00 | 투자자 수급 |
| `disclosure_collection` | 6시간마다 | 공시 데이터 |
| `universe_generation` | 매일 18:00 | Universe 생성 |
| `forecast_pipeline` | 매일 18:30 | 이벤트 감지/예측 |
| `cache_cleanup` | 5분마다 | 캐시 정리 |

### CLI 명령어

```bash
# 스케줄러 시작
go run ./cmd/quant scheduler start

# 작업 목록
go run ./cmd/quant scheduler list

# 특정 작업 즉시 실행
go run ./cmd/quant scheduler run forecast_pipeline

# 상태 확인
go run ./cmd/quant scheduler status
```

### Job 인터페이스

```go
type Job interface {
    Name() string                      // 작업 이름
    Schedule() string                  // Cron 스케줄 (6자리)
    Run(ctx context.Context) error     // 실행 로직
}
```

### Cron 스케줄 형식

```
┌────────────── 초 (0-59)
│ ┌──────────── 분 (0-59)
│ │ ┌────────── 시 (0-23)
│ │ │ ┌──────── 일 (1-31)
│ │ │ │ ┌────── 월 (1-12)
│ │ │ │ │ ┌──── 요일 (0-6, 0=일요일)
│ │ │ │ │ │
0 30 18 * * *   → 매일 18:30:00
0 0 16 * * 1-5  → 평일 16:00:00
0 */5 * * * *   → 5분마다
```

---

## Cleanup (정리 도구)

### Overview

데이터 정리 도구는 잘못된 데이터를 정리하는 CLI 명령어를 제공합니다.

```
cmd/quant/commands/
└── cleanup.go      # 데이터 정리 명령어
```

### CLI 명령어

```bash
# 투자자 매매동향 데이터 정리
go run ./cmd/quant cleanup investor-flow
```

### investor-flow 정리

2025-12-24 이전의 투자자 매매동향 데이터는 금액(원) 단위로 저장되어 있어서,
주식수 단위 데이터와 혼재되어 차트 표시에 문제가 있습니다.

이 명령어는 2025-12-24 이전 데이터를 삭제합니다.

```bash
# 실행 예시
$ go run ./cmd/quant cleanup investor-flow

=== Investor Flow Data Cleanup ===
📊 Found 12345 records with wrong units (before 2025-12-24)
🗑️ Deleting old data...
✅ Deleted 12345 records
📊 Remaining records: 67890

✅ Cleanup complete!
```

> **주의**: 이 명령어는 데이터를 **영구 삭제**합니다. 실행 전 백업을 권장합니다.

---

**Prev**: [Audit Layer](./audit-layer.md)
**Next**: [Frontend Folder Structure](../frontend/folder-structure.md)
