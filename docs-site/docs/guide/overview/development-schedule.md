---
sidebar_position: 4
title: Development Schedule
description: Aegis v13 개발 스케줄
---

# Development Schedule

> Aegis v13 단계별 개발 스케줄 및 SSOT 체크포인트

---

## 개발 원칙

### 1. 문서 우선 (Documentation First)
```
문서 작성 → 코드 구현 → 테스트 → 문서 업데이트
```

### 2. SSOT 철저 준수
- 모든 책임은 정해진 위치에서만 수행
- 중복 구현 절대 금지
- 레이어 간 의존성 단방향

### 3. 점진적 구현
- Phase별 독립 실행 가능
- 각 Phase 완료 후 통합 테스트
- 다음 Phase 시작 전 SSOT 검증

---

## 전체 로드맵

```
Phase 0: 프로젝트 셋업             [1-2일]
  └─> Backend 기본 구조
  └─> SSOT 기반 구축

Phase 1: 데이터 레이어 (S0-S1)     [5-7일]
  └─> 데이터 수집 & 품질 검증
  └─> Universe 생성
  └─> 실시간 가격 피드 (KIS WebSocket + REST)

Phase 2: 시그널 레이어 (S2)        [7-10일]
  └─> Momentum, Technical, Value, Quality
  └─> Flow (수급), Event 시그널

Phase 3: 선택 레이어 (S3-S4)       [3-4일]
  └─> Screener, Ranker

Phase 4: 포트폴리오 (S5)          [4-5일]
  └─> Portfolio 구성 & 최적화

Phase 5: 실행 레이어 (S6)         [3-4일]
  └─> 주문 생성 & 실행

Phase 6: 감사 레이어 (S7)         [3-4일]
  └─> 성과 분석 & Attribution

Phase 7: Brain Orchestrator       [2-3일]
  └─> 파이프라인 조율
  └─> 백테스팅 프레임워크

Phase 8: 프론트엔드              [10-14일]
  └─> Next.js 앱 구축

Phase 9: 통합 & 최적화           [5-7일]
  └─> End-to-End 테스트
  └─> 성능 최적화
  └─> 모니터링 & 알림
```

**총 예상 기간**: 45-60일 (약 9-12주)

---

## Phase 0: 프로젝트 셋업

### 목표
Go BFF 기본 구조 및 SSOT 레이어 구축

### 작업 목록

#### 0.1 Backend 폴더 구조
```bash
backend/
├── cmd/
│   └── quant/             # 통합 CLI (Cobra 기반)
│       ├── main.go
│       └── commands/
│           ├── root.go
│           ├── fetcher.go
│           ├── worker.go
│           ├── status.go
│           └── test_*.go
├── go.mod
├── go.sum
├── Makefile
└── .env.example
```

**산출물**:
- [x] `go.mod` (Go 1.21+)
- [x] `Makefile` (build, run, test, lint)
- [x] `.env.example`
- [x] `cmd/quant/main.go` - 통합 CLI 진입점
- [x] `cmd/quant/commands/` - Cobra 서브커맨드

**Makefile 예시**:
```makefile
# 빌드
build:
    go build -o bin/quant ./cmd/quant

# 테스트
test:
    go test -v -race ./...

test-db:
    go run ./cmd/quant test-db

test-logger:
    go run ./cmd/quant test-logger

lint:
    golangci-lint run
```

**실행 방식**:
```bash
# 통합 CLI 사용법
go run ./cmd/quant [command]

# 데이터 수집 (fetcher 서브커맨드)
go run ./cmd/quant fetcher collect all
go run ./cmd/quant fetcher collect prices
go run ./cmd/quant fetcher collect investor

# API 서버 실행 (구현 예정)
go run ./cmd/quant api

# 테스트
go run ./cmd/quant test-db
go run ./cmd/quant test-logger
```

**SSOT 체크**: N/A (기본 구조)

---

#### 0.2 pkg/ SSOT 레이어

```bash
pkg/
├── config/      # 환경변수 SSOT
├── database/    # DB 연결 SSOT
├── logger/      # 로깅 SSOT
└── httputil/    # HTTP 클라이언트 SSOT
```

**산출물**:
- [ ] `pkg/config/config.go` - 환경변수 로드
- [ ] `pkg/database/postgres.go` - PostgreSQL 연결
- [ ] `pkg/logger/logger.go` - Structured logging
- [ ] `pkg/httputil/client.go` - HTTP 클라이언트

**SSOT 체크포인트**:
```go
✅ os.Getenv()는 config/ 안에서만 사용
✅ pgxpool.New()는 database/ 안에서만 사용
✅ http.Client{}는 httputil/ 안에서만 생성
```

---

#### 0.3 contracts/ 타입 정의

```bash
internal/contracts/
├── data.go          # S0-S1 타입
├── signals.go       # S2 타입
├── selection.go     # S3-S4 타입
├── portfolio.go     # S5 타입
├── execution.go     # S6 타입
├── audit.go         # S7 타입
└── interfaces.go    # 레이어 인터페이스 + Brain 인터페이스
```

**산출물**:
- [ ] 7단계 파이프라인 타입 정의
- [ ] 레이어 간 인터페이스 정의
- [ ] **Brain Orchestrator 인터페이스** (구현은 Phase 7)
- [ ] 공통 에러 타입

**SSOT 체크포인트**:
```go
✅ 모든 레이어가 contracts/만 import
✅ 타입 중복 정의 없음
✅ 순환 참조 없음
```

---

#### 0.4 Database 마이그레이션

```bash
migrations/
├── 001_create_schemas.sql
├── 002_data_schema.sql
├── 003_signals_schema.sql
├── 004_selection_schema.sql
├── 005_portfolio_schema.sql
├── 006_execution_schema.sql
├── 007_audit_schema.sql
└── 008_realtime_schema.sql    # 실시간 가격 피드
```

**산출물**:
- [ ] PostgreSQL 스키마 생성 스크립트
- [ ] 테이블 생성 (schema-design.md 기준)
- [ ] 인덱스 생성
- [ ] **실시간 스키마 (realtime.sync_jobs, realtime.price_ticks)** ⭐

**데이터 체크**:
- [ ] `data.investor_flow` 테이블 포함 (수급 데이터)
- [ ] `signals.flow_details` 테이블 포함
- [ ] `realtime.sync_jobs` 테이블 포함 (동기화 큐) ⭐
- [ ] `realtime.price_ticks` 테이블 포함 (가격 히스토리) ⭐
- [ ] 모든 시계열 데이터 `(stock_code, date)` PK

---

### Phase 0 완료 조건

- [ ] `make build` 성공
- [ ] `make test` 통과
- [ ] `make lint` 통과
- [ ] DB 마이그레이션 실행 성공
- [ ] SSOT 위반 없음

---

## Phase 1: 데이터 레이어 (S0-S1)

### 🔄 현재 상태 (2026-01-10)

| 작업 | 상태 | 완료일 |
|------|------|--------|
| **S0: Quality Gate** | ✅ 완료 | 2026-01-10 |
| **S1: Universe Builder** | ✅ 완료 | 2026-01-10 |
| Data Sources (Naver/DART/KRX) | ❌ TODO | - |
| Scheduler | ❌ TODO | - |
| Real-time Price Feeds | ❌ TODO | - |

**완료된 것:**
- ✅ `internal/s0_data/quality/validator.go` - 품질 검증
- ✅ `internal/s0_data/repository.go` - DB 저장
- ✅ `internal/s1_universe/builder.go` - Universe 생성
- ✅ `internal/s1_universe/repository.go` - DB 저장
- ✅ `config/data.yaml` - 설정 파일
- ✅ 테스트 작성 및 실행 (922종목, 911 유니버스)

**TODO:**
- ❌ Data Sources 구현 (Naver, DART, KRX)
- ❌ Scheduler 구현
- ❌ KIS WebSocket (실시간 가격)
- ❌ CLI/API 인터페이스

### 목표
외부 API에서 데이터 수집 및 품질 검증, Universe 생성

### 작업 목록

#### 1.1 external/ API 클라이언트

```bash
internal/external/
├── kis/          # KIS API (체결/계좌)
├── dart/         # DART (공시)
└── naver/        # Naver Finance (가격, 수급)
```

**산출물**:
- [ ] `external/naver/price.go` - 가격 데이터
- [ ] `external/naver/investor.go` - **투자자 수급 데이터** ⭐
- [ ] `external/dart/disclosure.go` - 공시 데이터
- [ ] `external/kis/account.go` - 계좌 정보 (나중에)

**SSOT 체크포인트**:
```go
✅ HTTP 요청은 pkg/httputil/ 사용
✅ 다른 레이어에서 external/ 직접 호출 금지
✅ 모든 응답은 contracts/ 타입으로 변환
```

---

#### 1.2 data/ 레이어 (공통 로직)

```bash
internal/data/
├── collector.go     # 데이터 수집 오케스트레이터 (공통 로직)
├── quality.go       # 품질 검증 (S0)
├── universe.go      # Universe 생성 (S1)
└── repository.go    # DB 접근
```

**산출물**:
- [ ] S0: `DataQualitySnapshot` 생성
- [ ] S1: `Universe` 생성 (필터 조건 적용)
- [ ] DB 저장 로직

**실행 구조** (SSOT):
```go
// internal/data/collector.go - 공통 로직 (한 곳에만 구현)
type Collector struct {
    naverClient  *naver.Client
    dartClient   *dart.Client
    repository   *Repository
}

func (c *Collector) CollectAll(ctx context.Context) error { ... }
func (c *Collector) CollectPrices(ctx context.Context) error { ... }
func (c *Collector) CollectInvestor(ctx context.Context) error { ... }

// cmd/quant/commands/fetcher.go - CLI에서 Collector 호출
collector := data.NewCollector(...)
collector.CollectAll(ctx)

// cmd/quant/commands/api.go - API에서 Collector 호출 (구현 예정)
collector := data.NewCollector(...)
collector.CollectAll(ctx)
```

**핵심**: 로직은 `collector.go`에만, CLI/API는 호출만

**데이터 요구사항 체크**:
```yaml
✅ 가격 (OHLCV): 100% 커버리지
✅ 거래량: 100% 커버리지
✅ 시가총액: 95%+ 커버리지
✅ 재무제표: 80%+ 커버리지
✅ 투자자 수급: 80%+ 커버리지  # ⭐ 수급 데이터
✅ 공시: 70%+ 커버리지 (선택)
```

---

#### 1.3 에러 복구 전략

```bash
internal/data/
└── retry.go         # 재시도 및 복구 로직
```

**산출물**:
- [ ] API 호출 실패 시 재시도 로직 (exponential backoff)
- [ ] 부분 실패 허용 (일부 종목 데이터 누락 시 계속 진행)
- [ ] 대체 데이터 소스 전환 (primary 실패 시 fallback)
- [ ] 실패 기록 및 알림 (로그 + DB 저장)

**복구 시나리오**:
```yaml
시나리오 1: 외부 API 일시 장애
  → 3회 재시도 (1초, 2초, 4초 간격)
  → 실패 시 이전 데이터 사용 (stale data)

시나리오 2: 특정 종목 데이터 누락
  → 해당 종목 제외하고 계속 진행
  → Universe에서 자동 필터링

시나리오 3: 품질 게이트 실패
  → 관리자 알림
  → 수동 승인 대기 또는 자동 롤백
```

---

#### 1.4 CLI & API 인터페이스

**CLI 도구** (`cmd/quant/commands/fetcher.go`):
```bash
# 통합 CLI를 통한 데이터 수집
go run ./cmd/quant fetcher collect all

# 개별 수집
go run ./cmd/quant fetcher collect prices    # 가격 데이터
go run ./cmd/quant fetcher collect investor  # 수급 데이터
go run ./cmd/quant fetcher collect disclosure # 공시 데이터

# 빌드 후 실행
./bin/quant fetcher collect all

# cron 스케줄 예시 (매일 오후 4시)
0 16 * * * cd /path/to/backend && ./bin/quant fetcher collect all
```

**API 엔드포인트** (구현 예정):
```bash
# API 서버 실행
go run ./cmd/quant api

# 엔드포인트
GET  /api/data/quality          # 품질 스냅샷 조회
GET  /api/data/universe         # Universe 조회
POST /api/data/collect          # 데이터 수집 트리거
     Body: {"type": "all|prices|investor|disclosure"}
```

**산출물**:
- [x] `cmd/quant/commands/fetcher.go` - CLI fetcher 구현
- [ ] `cmd/quant/commands/api.go` - API 서버 구현 (예정)
- [ ] 둘 다 `internal/data/collector.go` 호출

**SSOT 체크**:
```
✅ 수집 로직은 internal/data/collector.go에만 존재
✅ CLI와 API는 collector 호출만 담당
✅ 중복 구현 절대 금지
```

---

#### 1.5 실시간 가격 피드 인프라

v10 수준의 실시간 가격 데이터 수집 시스템 구축

```bash
internal/realtime/
├── feed/
│   ├── manager.go       # 피드 관리자 (소스 통합)
│   ├── kis_ws.go        # KIS WebSocket 연결
│   ├── kis_rest.go      # KIS REST 폴링
│   └── naver.go         # Naver 백업 소스
├── cache/
│   ├── price_cache.go   # 인메모리 캐시
│   └── ttl.go           # TTL 관리
├── broker/
│   └── pubsub.go        # 구독자 브로드캐스트
└── queue/
    └── sync_job.go      # PostgreSQL 작업 큐
```

**산출물**:
- [ ] KIS WebSocket 연결 관리 (40 심볼 제한)
- [ ] Tiered REST 폴링 전략
- [ ] 가격 캐시 및 브로커
- [ ] PostgreSQL 동기화 큐
- [ ] 가격 검증 및 이상 탐지

---

##### 1.5.1 KIS WebSocket 연결

```go
// 40 심볼 제한 관리
type WSManager struct {
    maxSymbols     int           // 40
    activeSymbols  map[string]bool
    priorityQueue  *PriorityQueue
    conn           *websocket.Conn
}

// 우선순위 계산
type SymbolPriority struct {
    Code           string
    Score          float64  // 높을수록 WS 할당
    LastTradeTime  time.Time
    Volatility     float64
    UserWatching   bool
}
```

**우선순위 기준**:
```yaml
Tier-1 (WebSocket 우선):
  - 포트폴리오 보유 종목
  - 활성 주문 대기 종목
  - 사용자 실시간 모니터링 종목
  - 고변동성 종목

Tier-2 (REST 5-10초):
  - 관심 종목 (watchlist)
  - Universe 상위 100

Tier-3 (REST 30-60초):
  - Universe 나머지
```

---

##### 1.5.2 Tiered REST 폴링

KIS API Rate Limit 대응: 초당 10 요청

```go
type TieredPoller struct {
    tier1Interval  time.Duration  // 2-5초
    tier2Interval  time.Duration  // 10-15초
    tier3Interval  time.Duration  // 30-60초
    rateLimiter    *rate.Limiter
}

// 티어별 폴링 루프
func (p *TieredPoller) Start(ctx context.Context) {
    go p.pollTier1(ctx)  // 고빈도
    go p.pollTier2(ctx)  // 중빈도
    go p.pollTier3(ctx)  // 저빈도
}
```

**Rate Limit 전략**:
```yaml
KIS 제한: 10 req/sec
할당:
  - Tier-1: 6 req/sec (60%)
  - Tier-2: 3 req/sec (30%)
  - Tier-3: 1 req/sec (10%)
```

---

##### 1.5.3 피드 병합 및 캐시

```go
type PriceCache struct {
    mu       sync.RWMutex
    prices   map[string]*PriceTick
    ttl      time.Duration  // 60초
}

type PriceTick struct {
    Code        string
    Price       int64
    Change      int64
    ChangeRate  float64
    Volume      int64
    Timestamp   time.Time
    Source      string   // "KIS_WS", "KIS_REST", "NAVER"
    IsStale     bool
}

// 소스 우선순위: KIS_WS > KIS_REST > NAVER
func (c *PriceCache) Update(tick *PriceTick) {
    // 더 신선한 데이터만 적용
    // 소스 우선순위 고려
}
```

---

##### 1.5.4 PostgreSQL 동기화 큐

안정적인 가격 저장을 위한 작업 큐

```go
type SyncJob struct {
    ID        int64
    StockCode string
    Price     int64
    Volume    int64
    Timestamp time.Time
    Status    string   // "pending", "processing", "done", "failed"
    Retries   int
    CreatedAt time.Time
}

// DB 테이블
// CREATE TABLE realtime.sync_jobs (
//     id SERIAL PRIMARY KEY,
//     stock_code VARCHAR(20),
//     price BIGINT,
//     volume BIGINT,
//     timestamp TIMESTAMPTZ,
//     status VARCHAR(20) DEFAULT 'pending',
//     retries INT DEFAULT 0,
//     created_at TIMESTAMPTZ DEFAULT NOW()
// );
```

**배치 처리**:
```yaml
배치 크기: 100건
처리 주기: 1초
재시도: 3회 (exponential backoff)
실패 처리: dead letter queue
```

---

##### 1.5.5 가격 검증 및 이상 탐지

```go
type PriceVerifier struct {
    maxChangeRate  float64  // 30% (상한가 기준)
    staleTTL       time.Duration
}

func (v *PriceVerifier) Verify(tick *PriceTick, prev *PriceTick) error {
    // 1. 가격 범위 검증 (상/하한가 내)
    // 2. 급격한 변동 탐지
    // 3. 소스 간 가격 비교 (KIS vs Naver)
    // 4. 타임스탬프 유효성
}

// 이상 탐지 시
type PriceAnomaly struct {
    Code       string
    Expected   int64
    Actual     int64
    Source     string
    Severity   string  // "warning", "critical"
    DetectedAt time.Time
}
```

**검증 규칙**:
```yaml
변동폭 검증:
  - 일반: ±15% (경고)
  - 급등락: ±30% (거부 + 알림)

소스 비교:
  - KIS vs Naver 차이 > 1%: 경고
  - 차이 > 3%: 데이터 보류

Stale 데이터:
  - 60초 이상: 경고
  - 300초 이상: 무효화
```

---

##### 1.5.6 Circuit Breaker

```go
type CircuitBreaker struct {
    state        string  // "closed", "open", "half-open"
    failures     int
    threshold    int     // 5
    resetTimeout time.Duration
}

// 외부 API 장애 시 자동 전환
func (cb *CircuitBreaker) Execute(fn func() error) error {
    if cb.state == "open" {
        return ErrCircuitOpen
    }
    err := fn()
    if err != nil {
        cb.failures++
        if cb.failures >= cb.threshold {
            cb.state = "open"
            go cb.scheduleReset()
        }
    }
    return err
}
```

**Failover 전략**:
```yaml
KIS WebSocket 장애:
  → KIS REST 폴링으로 전환
  → 폴링 간격 단축 (2초)

KIS 전체 장애:
  → Naver 백업 소스 활성화
  → 거래 기능 일시 중지 (선택)

복구 시:
  → Half-open 상태에서 테스트
  → 성공 시 정상 운영 복귀
```

---

##### 1.5.7 API 엔드포인트 (실시간)

```bash
# REST API
GET  /api/prices/{code}           # 단일 종목 가격
GET  /api/prices?codes=A,B,C      # 다중 종목 가격
GET  /api/prices/portfolio        # 포트폴리오 종목 가격

# WebSocket API
WS   /ws/prices                   # 실시간 가격 스트림
     Subscribe: {"action": "subscribe", "codes": ["005930", "000660"]}
     Unsubscribe: {"action": "unsubscribe", "codes": ["005930"]}
```

**WebSocket 메시지 형식**:
```json
{
  "type": "price",
  "data": {
    "code": "005930",
    "name": "삼성전자",
    "price": 71000,
    "change": 500,
    "changeRate": 0.71,
    "volume": 12345678,
    "timestamp": "2025-01-09T15:30:00+09:00"
  }
}
```

---

### Phase 1 완료 조건

- [ ] 데이터 수집 성공 (CLI, API 둘 다)
- [ ] `DataQualitySnapshot` 생성 성공
- [ ] `Universe` 생성 성공 (필터링 적용)
- [ ] DB 저장 성공 (`data.investor_flow` 포함)
- [ ] **실시간 가격 피드 동작 확인** ⭐
- [ ] **KIS WebSocket 연결 성공 (40 심볼)** ⭐
- [ ] **Tiered REST 폴링 동작 확인** ⭐
- [ ] **가격 검증 및 이상 탐지 동작** ⭐
- [ ] 품질 게이트 통과
- [ ] 단위 테스트 통과

---

## Phase 2: 시그널 레이어 (S2)

### 🔄 현재 상태 (2026-01-10)

| 작업 | 상태 | 완료일 |
|------|------|--------|
| **S2: Signal Builder** | ✅ 완료 | 2026-01-10 |
| 6가지 시그널 계산기 | ✅ 완료 | 2026-01-10 |
| Signal Repository | ✅ 완료 | 2026-01-10 |

**완료된 것:**
- ✅ `internal/s2_signals/builder.go` - SignalBuilder
- ✅ `internal/s2_signals/momentum.go` - 모멘텀 시그널
- ✅ `internal/s2_signals/technical.go` - 기술적 시그널
- ✅ `internal/s2_signals/value.go` - 가치 시그널
- ✅ `internal/s2_signals/quality.go` - 퀄리티 시그널
- ✅ `internal/s2_signals/flow.go` - 수급 시그널 ⭐
- ✅ `internal/s2_signals/event.go` - 이벤트 시그널
- ✅ `internal/s2_signals/repository.go` - DB 저장

### 목표
6가지 시그널 생성 (Momentum, Technical, Value, Quality, Flow, Event)

### 작업 목록

#### 2.1 signals/ 레이어

```bash
internal/signals/
├── momentum.go      # 모멘텀 시그널
├── technical.go     # 기술적 지표
├── value.go         # 가치 시그널
├── quality.go       # 퀄리티 시그널
├── flow.go          # 수급 시그널 ⭐
├── event.go         # 이벤트 시그널
└── builder.go       # SignalSet 생성
```

**산출물**:
- [ ] Momentum: 수익률, 거래량 증가율
- [ ] Technical: RSI, MACD, 이평선
- [ ] Value: PER, PBR, PSR
- [ ] Quality: ROE, 부채비율, 성장률
- [ ] **Flow: 외국인/기관 순매수, 연속 순매수일** ⭐
- [ ] Event: 실적, 공시 이벤트

**수급 시그널 상세**:
```go
type FlowSignal struct {
    ForeignNet5D   int64   // 외국인 5일 순매수
    ForeignNet20D  int64   // 외국인 20일 순매수
    InstNet5D      int64   // 기관 5일 순매수
    InstNet20D     int64   // 기관 20일 순매수
    ForeignStreak  int     // 연속 순매수일
    InstStreak     int     // 연속 순매수일
    Score          float64 // -1.0 ~ 1.0
}
```

---

#### 2.2 SignalSet 생성

**산출물**:
- [ ] `SignalSet` 생성 (종목별 6가지 시그널)
- [ ] DB 저장 (`signals.factor_scores`, `signals.flow_details`)

**SSOT 체크포인트**:
```go
✅ Universe는 data/ 레이어에서 전달받음
✅ 원본 데이터는 data.repository를 통해 읽기
✅ 모든 시그널은 -1.0 ~ 1.0 정규화
```

---

#### 2.3 API 엔드포인트

```bash
internal/api/handlers/
└── signals.go
```

**산출물**:
- [ ] `GET /api/signals/{date}` - 시그널 조회
- [ ] `GET /api/signals/{code}/{date}` - 종목별 시그널
- [ ] `POST /api/signals/generate` - 시그널 생성 트리거

---

### Phase 2 완료 조건

- [ ] 6가지 시그널 생성 성공
- [ ] 수급 시그널 정상 동작 확인
- [ ] `SignalSet` DB 저장 성공
- [ ] API 엔드포인트 동작 확인
- [ ] 단위 테스트 + 통합 테스트 통과

---

## Phase 3: 선택 레이어 (S3-S4)

### 🔄 현재 상태 (2026-01-10)

| 작업 | 상태 | 완료일 |
|------|------|--------|
| **S3: Screener** | ✅ 완료 | 2026-01-10 |
| **S4: Ranker** | ✅ 완료 | 2026-01-10 |
| Selection Repository | ✅ 완료 | 2026-01-10 |

**완료된 것:**
- ✅ `internal/selection/screener.go` - Hard Cut 필터링
- ✅ `internal/selection/ranker.go` - 종합 점수 랭킹
- ✅ `internal/selection/repository.go` - DB 저장
- ✅ 가중치 설정: Flow 25%, Momentum 20%, Technical 20%, Value 15%, Quality 15%, Event 5%

### 목표
Screening (Hard Cut) + Ranking

### 작업 목록

#### 3.1 selection/ 레이어

```bash
internal/selection/
├── screener.go      # S3: Hard Cut 필터링
└── ranker.go        # S4: 종합 점수 산출
```

**산출물**:
- [ ] Screener: 통과 종목 필터링
- [ ] Ranker: 가중치 적용 종합 점수
- [ ] `[]RankedStock` 생성

**가중치 예시**:
```yaml
momentum: 0.25
technical: 0.15
value: 0.20
quality: 0.15
flow: 0.20       # 수급
event: 0.05
```

---

#### 3.2 API 엔드포인트

**산출물**:
- [ ] `GET /api/selection/screened` - 스크리닝 결과
- [ ] `GET /api/selection/ranked` - 랭킹 결과
- [ ] `POST /api/selection/run` - 선택 프로세스 실행

---

### Phase 3 완료 조건

- [ ] Screening 성공 (Hard Cut 적용)
- [ ] Ranking 성공 (가중치 적용)
- [ ] DB 저장 성공
- [ ] 단위 테스트 통과

---

## Phase 4: 포트폴리오 (S5)

### 🔄 현재 상태 (2026-01-10)

| 작업 | 상태 | 완료일 |
|------|------|--------|
| **S5: Portfolio Constructor** | ✅ 완료 | 2026-01-10 |
| Constraints | ✅ 완료 | 2026-01-10 |
| Portfolio Repository | ✅ 완료 | 2026-01-10 |

**완료된 것:**
- ✅ `internal/portfolio/constructor.go` - 포트폴리오 구성
- ✅ `internal/portfolio/constraints.go` - 제약조건 검증
- ✅ `internal/portfolio/repository.go` - DB 저장
- ✅ 3가지 비중 모드: equal, score_based, risk_parity(TODO)
- ✅ 기본값: 최대 20종목, 최대 15%, 최소 3%, 5% 현금

### 목표
목표 포트폴리오 생성 및 리밸런싱

### 작업 목록

#### 4.1 portfolio/ 레이어

```bash
internal/portfolio/
├── constructor.go   # 포트폴리오 구성
└── rebalancer.go    # 리밸런싱 로직
```

**산출물**:
- [ ] 제약 조건 적용 (최대 종목 수, 비중 제한)
- [ ] `TargetPortfolio` 생성
- [ ] 리밸런싱 로직 (회전율 제한)

---

### Phase 4 완료 조건

- [ ] TargetPortfolio 생성 성공
- [ ] 제약 조건 준수 확인
- [ ] DB 저장 성공

---

## Phase 5: 실행 레이어 (S6)

### 🔄 현재 상태 (2026-01-10)

| 작업 | 상태 | 완료일 |
|------|------|--------|
| **S6: Execution Planner** | ✅ 완료 | 2026-01-10 |
| Broker Interface | ✅ 완료 | 2026-01-10 |
| Monitor | ✅ 완료 | 2026-01-10 |
| Execution Repository | ✅ 완료 | 2026-01-10 |

**완료된 것:**
- ✅ `internal/execution/planner.go` - 주문 계획 (슬리피지, 분할)
- ✅ `internal/execution/broker.go` - Broker 인터페이스 + MockBroker
- ✅ `internal/execution/monitor.go` - 체결 모니터링
- ✅ `internal/execution/repository.go` - DB 저장
- ✅ 기본값: 지정가, 0.1% 슬리피지, 5천만원 최대 주문

### 목표
주문 생성 및 실행

### 작업 목록

#### 5.1 execution/ 레이어

```bash
internal/execution/
├── planner.go       # 주문 계획
└── broker.go        # KIS 연동
```

**산출물**:
- [ ] 주문 생성 (`Order`)
- [ ] KIS API 연동 (주문 전송)
- [ ] 체결 확인

---

### Phase 5 완료 조건

- [ ] 주문 생성 성공
- [ ] KIS API 연동 확인
- [ ] DB 저장 성공

---

## Phase 6: 감사 레이어 (S7)

### 🔄 현재 상태 (2026-01-10)

| 작업 | 상태 | 완료일 |
|------|------|--------|
| **S7: Performance Analyzer** | ✅ 완료 | 2026-01-10 |
| Attribution | ✅ 완료 | 2026-01-10 |
| Snapshot | ✅ 완료 | 2026-01-10 |
| Audit Repository | ✅ 완료 | 2026-01-10 |

**완료된 것:**
- ✅ `internal/audit/performance.go` - 성과 분석 (Sharpe, Sortino, MDD)
- ✅ `internal/audit/attribution.go` - 팩터 기여도 분석
- ✅ `internal/audit/snapshot.go` - 일별 스냅샷, equity curve
- ✅ `internal/audit/repository.go` - DB 저장
- ✅ 6개 팩터 기여도 분석 (Flow 포함)

### 목표
성과 분석 및 시그널 기여도 분석

### 작업 목록

#### 6.1 audit/ 레이어

```bash
internal/audit/
├── performance.go   # 성과 분석
└── attribution.go   # 시그널 기여도
```

**산출물**:
- [ ] `PerformanceReport` 생성
- [ ] 시그널별 기여도 분석 (수급 시그널 포함)
- [ ] 리스크 지표 계산

---

### Phase 6 완료 조건

- [ ] 성과 분석 완료
- [ ] Attribution 분석 완료
- [ ] DB 저장 성공

---

## Phase 7: Brain Orchestrator

### 목표
전체 파이프라인 조율 (로직 없음) + 백테스팅 프레임워크

### 작업 목록

#### 7.1 Brain Orchestrator

```bash
internal/brain/
└── orchestrator.go
```

**산출물**:
- [ ] S0 → S1 → S2 → S3 → S4 → S5 → S6 → S7 실행
- [ ] 에러 처리 및 로깅
- [ ] 재현성 기록 (`run_id`, `git_sha`, etc.)
- [ ] 파이프라인 중단/재개 지원

---

#### 7.2 백테스팅 프레임워크

```bash
internal/backtest/
├── engine.go        # 백테스트 엔진
├── simulator.go     # 주문 시뮬레이션
└── report.go        # 백테스트 리포트
```

**산출물**:
- [ ] 과거 데이터 기반 파이프라인 실행
- [ ] 가상 주문 체결 시뮬레이션
- [ ] 성과 분석 리포트 생성
- [ ] 시그널별 기여도 분석

**백테스트 시나리오**:
```yaml
기본 백테스트:
  - 기간: 최근 1년
  - 초기 자본: 1억원
  - 리밸런싱: 주간

검증 항목:
  - 수익률 vs KOSPI
  - MDD (Maximum Drawdown)
  - Sharpe Ratio
  - 승률
  - 회전율
```

---

### Phase 7 완료 조건

- [ ] End-to-End 파이프라인 실행 성공
- [ ] 재현성 필드 기록 확인
- [ ] 백테스트 실행 성공 (최근 1년)
- [ ] 백테스트 리포트 생성 완료
- [ ] 통합 테스트 통과

---

## Phase 8: 프론트엔드

### 목표
Next.js 앱 구축

### 작업 목록

#### 8.1 기본 구조

```bash
frontend/
├── src/
│   ├── app/           # Next.js App Router
│   ├── modules/       # 도메인별 모듈
│   └── shared/        # 공용 컴포넌트
└── package.json
```

**산출물**:
- [ ] Next.js 14+ 셋업
- [ ] 기본 레이아웃
- [ ] API 클라이언트

---

#### 8.2 주요 페이지

- [ ] Dashboard (포트폴리오 현황)
- [ ] Stocks (종목 리스트)
- [ ] Signals (시그널 시각화)
- [ ] Performance (성과 분석)

---

### Phase 8 완료 조건

- [ ] 모든 페이지 구현 완료
- [ ] API 연동 성공
- [ ] SSOT 준수 (frontend-ssot.md)

---

## Phase 9: 통합 & 최적화

### 목표
End-to-End 테스트, 성능 최적화, 모니터링 구축

### 작업 목록

#### 9.1 End-to-End 테스트

- [ ] Backend API 통합 테스트
- [ ] Frontend E2E 테스트 (Playwright)
- [ ] 파이프라인 전체 흐름 테스트
- [ ] 실패 시나리오 테스트

---

#### 9.2 성능 최적화

- [ ] 성능 프로파일링 (Go pprof)
- [ ] DB 쿼리 최적화
- [ ] 캐싱 전략 (Redis)
- [ ] API 응답 시간 개선

**목표 성능**:
```yaml
데이터 수집: < 5분 (2,000 종목)
시그널 생성: < 3분
파이프라인 전체: < 10분
API 응답: < 100ms (95 percentile)
```

---

#### 9.3 모니터링 & 알림

```bash
internal/monitoring/
├── metrics.go       # Prometheus 메트릭
├── health.go        # 헬스체크
└── alerts.go        # 알림 시스템
```

**산출물**:
- [ ] Prometheus 메트릭 수집
- [ ] Grafana 대시보드
- [ ] 알림 시스템 (Slack/Email)
- [ ] 로그 수집 (ELK 또는 Loki)

**모니터링 항목**:
```yaml
시스템 메트릭:
  - CPU/메모리 사용률
  - DB 연결 풀 상태
  - API 요청/응답 시간

비즈니스 메트릭:
  - 데이터 수집 성공률
  - 품질 게이트 통과율
  - 파이프라인 실행 시간
  - 백테스트 수익률

알림 조건:
  - 데이터 수집 실패 (즉시)
  - 품질 게이트 실패 (즉시)
  - API 응답 시간 > 1초 (경고)
  - 파이프라인 실행 실패 (즉시)
```

---

#### 9.4 Production 준비

- [ ] 환경별 설정 (dev, staging, prod)
- [ ] Docker 컨테이너화
- [ ] CI/CD 파이프라인 (GitHub Actions)
- [ ] 문서 최종 검토 및 업데이트

---

### Phase 9 완료 조건

- [ ] 모든 테스트 통과 (단위, 통합, E2E)
- [ ] 성능 목표 달성
- [ ] 모니터링 대시보드 구축 완료
- [ ] 알림 시스템 동작 확인
- [ ] Production Ready

---

## SSOT 체크리스트 (전체)

각 Phase 완료 시 반드시 확인:

### Backend
- [ ] `os.Getenv()` 사용은 `pkg/config/`만
- [ ] `pgx.Connect()` 사용은 `pkg/database/`만
- [ ] HTTP 클라이언트는 `pkg/httputil/`만
- [ ] 외부 API 호출은 `internal/external/`만
- [ ] 타입 정의는 `internal/contracts/`만
- [ ] 순환 참조 없음

### Frontend
- [ ] API 호출은 `modules/*/api.ts`만
- [ ] 타입 정의는 `modules/*/types/`만
- [ ] 컴포넌트는 `modules/*/components/`만
- [ ] 인라인 타입 정의 없음

### Database
- [ ] 모든 시계열 데이터 `(stock_code, date)` PK
- [ ] 수급 데이터 테이블 존재 (`data.investor_flow`)
- [ ] 수급 시그널 테이블 존재 (`signals.flow_details`)
- [ ] **실시간 동기화 큐 테이블 존재 (`realtime.sync_jobs`)** ⭐
- [ ] **실시간 가격 테이블 존재 (`realtime.price_ticks`)** ⭐

### Realtime (실시간 가격 피드)
- [ ] KIS WebSocket 연결은 `internal/realtime/feed/kis_ws.go`만
- [ ] KIS REST 폴링은 `internal/realtime/feed/kis_rest.go`만
- [ ] 가격 캐시는 `internal/realtime/cache/`만
- [ ] 동기화 큐는 `internal/realtime/queue/`만
- [ ] Circuit Breaker 패턴 적용 확인

---

## 일일 작업 루틴

### 작업 시작 전
1. 해당 Phase 문서 읽기 (`docs-site/docs/guide/`)
2. SSOT 위치 확인
3. 최소 변경 범위 결정

### 작업 중
1. SSOT 원칙 준수
2. 테스트 작성 (TDD)
3. 코드 리뷰 (자가)

### 작업 완료 후
1. `make lint && make test` 통과
2. SSOT 체크리스트 확인
3. 커밋 (`feat|fix|refactor|docs(scope): message`)
4. 문서 업데이트 (필요시)

---

## 참고 문서

- [System Overview](./system-overview.md)
- [Data Flow](../architecture/data-flow.md)
- [Backend Folder Structure](../backend/folder-structure.md)
- [Database Schema](../database/schema-design.md)
- [SSOT Policy](/.claude/rules/ssot-policy.md)

---

**문서 버전**: v1.1
**최종 업데이트**: 2025-01-09
**변경 이력**:
- v1.1: 실시간 가격 피드 인프라 추가 (Phase 1.5)
