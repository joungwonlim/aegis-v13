---
sidebar_position: 1
title: Folder Structure
description: Go BFF 폴더 구조와 SSOT
---

# Backend Folder Structure

> Go BFF 아키텍처와 SSOT 원칙

---

## BFF (Backend For Frontend) 개념

```
┌─────────────┐
│   Next.js   │  ← 프론트엔드는 Go BFF만 호출
└──────┬──────┘
       │ HTTP
       ▼
┌─────────────┐
│   Go BFF    │  ← 모든 비즈니스 로직은 여기서
├─────────────┤
│ • API 집계  │
│ • 데이터 가공│
│ • 인증/인가 │
│ • 캐싱      │
└──────┬──────┘
       │
       ▼
┌─────────────┬─────────────┬─────────────┐
│ PostgreSQL  │  KIS API    │  DART API   │
└─────────────┴─────────────┴─────────────┘
```

### BFF가 해결하는 문제

| 문제 | BFF 해결책 |
|------|-----------|
| 프론트에서 여러 API 호출 | BFF가 집계해서 1개 응답 |
| API 키 노출 위험 | BFF에서만 외부 API 호출 |
| 데이터 가공 중복 | BFF에서 한번만 가공 |
| 타입 불일치 | BFF에서 표준 타입으로 변환 |

---

## SSOT (Single Source of Truth) 원칙

### ❌ 금지 패턴

```go
// handler에서 직접 env 읽기 - 금지!
func (h *Handler) GetStocks(w http.ResponseWriter, r *http.Request) {
    apiKey := os.Getenv("KIS_API_KEY")  // ❌ SSOT 위반
}

// service에서 직접 DB 연결 - 금지!
func NewService() *Service {
    db, _ := pgx.Connect(ctx, os.Getenv("DB_URL"))  // ❌ SSOT 위반
}
```

### ✅ 올바른 패턴

```go
// config에서만 env 읽기
type Config struct {
    KISApiKey string `env:"KIS_API_KEY"`
    DBUrl     string `env:"DB_URL"`
}

// 의존성 주입
func NewService(cfg *Config, db *pgxpool.Pool) *Service {
    return &Service{cfg: cfg, db: db}  // ✅ SSOT 준수
}
```

---

## SSOT 책임 매핑

| 책임 | 허용 위치 | 금지 |
|------|----------|------|
| **환경변수** | `pkg/config/` | 다른 곳에서 `os.Getenv()` |
| **DB 연결** | `pkg/database/` | 다른 곳에서 `pgx.Connect()` |
| **HTTP 클라이언트** | `pkg/httputil/` | 다른 곳에서 `http.Client{}` |
| **로깅** | `pkg/logger/` | 다른 곳에서 `log.Println()` |
| **외부 API** | `internal/external/` | 다른 레이어에서 직접 호출 |
| **타입 정의** | `internal/contracts/` | 레이어마다 중복 정의 |

---

## 폴더 구조

```
backend/
├── cmd/
│   └── quant/                # ⭐ 통합 CLI 진입점
│       ├── main.go           # 메인 엔트리포인트
│       └── commands/         # 서브커맨드
│           ├── root.go       # Root command
│           ├── format.go     # 공통 포맷팅 유틸리티 (SSOT)
│           ├── api.go        # API 서버
│           ├── fetcher.go    # 데이터 수집
│           ├── worker.go     # 백그라운드 워커
│           └── scheduler.go  # 스케줄러
│
├── config/
│   └── strategy/
│       └── korea_equity_v13.yaml  # ⭐ 전략 설정 SSOT
│
├── internal/                  # 비공개 비즈니스 로직
│   ├── contracts/            # ⭐ 타입/인터페이스 SSOT
│   │   ├── data.go           # DataQualitySnapshot (S0→S1)
│   │   ├── universe.go       # Universe (S1→S2)
│   │   ├── signals.go        # SignalSet, StockSignals (S2→S3/S4)
│   │   ├── ranked.go         # RankedStock (S4→S5)
│   │   ├── portfolio.go      # TargetPortfolio (S5→S6)
│   │   ├── order.go          # Order (S6→Broker)
│   │   ├── audit.go          # PerformanceReport (S7)
│   │   └── interfaces.go     # 7단계 인터페이스
│   │
│   ├── brain/                # ⭐ 오케스트레이터
│   │   └── orchestrator.go   # S0→S7 파이프라인 조율
│   │
│   ├── s0_data/              # S0: 데이터 품질
│   │   ├── quality/
│   │   │   ├── validator.go  # 품질 검증
│   │   │   └── validator_test.go
│   │   ├── collector/
│   │   │   └── collector.go  # 데이터 수집
│   │   └── repository.go
│   │
│   ├── s1_universe/          # S1: 유니버스
│   │   ├── builder.go        # 투자 가능 종목 추출
│   │   ├── builder_test.go
│   │   └── repository.go
│   │
│   ├── s2_signals/           # S2: 시그널 레이어
│   │   ├── momentum.go       # 모멘텀 (1M, 3M, 거래량)
│   │   ├── technical.go      # 기술적 (RSI, MACD, MA)
│   │   ├── value.go          # 밸류 (PER, PBR, PSR)
│   │   ├── quality.go        # 퀄리티 (ROE, 부채비율)
│   │   ├── flow.go           # 수급 (외인/기관 순매수)
│   │   ├── event.go          # 이벤트 (공시, 배당)
│   │   ├── builder.go        # SignalBuilder 통합
│   │   └── repository.go
│   │
│   ├── selection/            # S3-S4: 스크리닝/랭킹
│   │   ├── screener.go       # S3: 하드컷 필터링
│   │   ├── ranker.go         # S4: 가중치 기반 랭킹
│   │   └── repository.go
│   │
│   ├── portfolio/            # S5: 포트폴리오 구성
│   │   ├── constructor.go    # 포트폴리오 생성
│   │   ├── constraints.go    # 제약조건 적용
│   │   └── repository.go
│   │
│   ├── execution/            # S6: 주문 실행
│   │   ├── planner.go        # 주문 계획
│   │   ├── broker.go         # 브로커 연동
│   │   ├── monitor.go        # 실행 모니터링
│   │   └── repository.go
│   │
│   ├── audit/                # S7: 성과 분석
│   │   ├── performance.go    # 수익률/리스크 분석
│   │   ├── attribution.go    # 팩터별 기여도
│   │   ├── snapshot.go       # 스냅샷 저장
│   │   └── repository.go
│   │
│   ├── backtest/             # ⭐ 백테스트 프레임워크
│   │   ├── engine.go         # 백테스트 엔진
│   │   └── simulator.go      # 주문 시뮬레이션
│   │
│   ├── scheduler/            # 스케줄러
│   │   ├── scheduler.go
│   │   └── jobs/
│   │
│   ├── realtime/             # 실시간 데이터 (미완성)
│   │   ├── broker/
│   │   ├── cache/
│   │   ├── feed/
│   │   └── queue/
│   │
│   ├── external/             # ⭐ 외부 API SSOT
│   │   ├── dart/             # DART API (공시)
│   │   ├── naver/            # Naver (시세/수급)
│   │   └── krx/              # KRX (시장 데이터)
│   │
│   ├── api/                  # HTTP 핸들러
│   │   ├── router.go
│   │   └── handlers/
│   │
│   └── data/                 # 데이터 유틸리티
│
├── pkg/                      # ⭐ 공용 패키지 SSOT
│   ├── config/               # 환경변수 (SSOT)
│   ├── database/             # DB 연결 (SSOT)
│   ├── logger/               # 로깅 (SSOT)
│   └── httputil/             # HTTP 클라이언트 (SSOT)
│
├── migrations/               # DB 마이그레이션
│
└── Makefile
```

---

## 레이어별 파일 수

| 레이어 | 파일 수 | 역할 | 상태 |
|--------|---------|------|------|
| contracts | 8 | 타입/인터페이스 정의 | ✅ 완성 |
| brain | 1 | 오케스트레이터 | ✅ 완성 |
| s0_data | 4 | 데이터 품질 검증 | ⚠️ 부분 완성 |
| s1_universe | 3 | 유니버스 생성 | ✅ 완성 |
| s2_signals | 8 | 6팩터 시그널 생성 | ⚠️ 부분 완성 |
| selection | 3 | 스크리닝/랭킹 | ✅ 완성 |
| portfolio | 3 | 포트폴리오 구성 | ⚠️ 부분 완성 |
| execution | 4 | 주문 실행 | ⚠️ 부분 완성 |
| audit | 4 | 성과 분석 | ✅ 완성 |
| backtest | 2 | 백테스트 프레임워크 | ⚠️ 부분 완성 |
| external | 6 | 외부 API (DART, Naver, KRX) | ✅ 완성 |
| **Total** | **~47** | 7,500+ 라인 | |

---

## 7단계 파이프라인 흐름

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Brain Orchestrator                                 │
│                     (internal/brain/orchestrator.go)                        │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        ▼                           ▼                           ▼
┌───────────────┐           ┌───────────────┐           ┌───────────────┐
│   S0: Data    │──────────▶│  S1: Universe │──────────▶│  S2: Signals  │
│   Quality     │           │   Builder     │           │   Builder     │
├───────────────┤           ├───────────────┤           ├───────────────┤
│ • 커버리지 검증│           │ • 시총 필터   │           │ • Momentum    │
│ • 품질 점수   │           │ • 거래대금    │           │ • Technical   │
│ • 필수 데이터 │           │ • 관리종목 제외│           │ • Value       │
│               │           │ • 거래정지 제외│           │ • Quality     │
│               │           │               │           │ • Flow        │
│               │           │               │           │ • Event       │
└───────────────┘           └───────────────┘           └───────────────┘
        │                           │                           │
        │ DataQualitySnapshot       │ Universe                  │ SignalSet
        ▼                           ▼                           ▼
┌───────────────┐           ┌───────────────┐           ┌───────────────┐
│  S3: Screener │──────────▶│  S4: Ranker   │──────────▶│ S5: Portfolio │
├───────────────┤           ├───────────────┤           ├───────────────┤
│ • 모멘텀 컷   │           │ • 가중치 적용 │           │ • Top-N 선정  │
│ • PER/PBR 컷  │           │ • 종합 점수   │           │ • 비중 계산   │
│ • ROE 컷      │           │ • 순위 정렬   │           │ • 제약조건    │
│ • 수급 컷     │           │               │           │ • 현금 비중   │
└───────────────┘           └───────────────┘           └───────────────┘
        │                           │                           │
        │ screened[]                │ RankedStock[]             │ TargetPortfolio
        ▼                           ▼                           ▼
┌───────────────┐           ┌───────────────┐
│ S6: Execution │──────────▶│  S7: Audit    │
├───────────────┤           ├───────────────┤
│ • 주문 생성   │           │ • 수익률 분석 │
│ • 매도 우선   │           │ • 리스크 지표 │
│ • 슬리피지    │           │ • 팩터 기여도 │
│ • 분할 주문   │           │ • 벤치마크    │
└───────────────┘           └───────────────┘
        │                           │
        │ Order[]                   │ PerformanceReport
        ▼                           ▼
    [Broker]                   [Dashboard]
```

---

## 데이터 흐름 상세

| 단계 | 입력 | 출력 | 핵심 로직 |
|------|------|------|----------|
| **S0** | Raw Data | `DataQualitySnapshot` | 커버리지 ≥ 70% 검증 |
| **S1** | S0 결과 | `Universe` (종목 리스트) | 시총/거래대금/관리종목 필터 |
| **S2** | Universe | `SignalSet` (6팩터) | 각 팩터 -1.0~1.0 정규화 |
| **S3** | SignalSet | `screened[]` | 하드컷 조건 적용 |
| **S4** | screened + SignalSet | `RankedStock[]` | 가중합 점수 계산, 정렬 |
| **S5** | RankedStock[] | `TargetPortfolio` | Top-20, 비중 배분 |
| **S6** | TargetPortfolio | `Order[]` | Diff 계산, 주문 생성 |
| **S7** | 실행 결과 | `PerformanceReport` | 수익률/리스크/기여도 |

---

## Import 규칙

```go
// ✅ 허용: 상위 → 하위
import "internal/contracts"     // 모든 레이어에서 OK
import "pkg/config"             // 모든 곳에서 OK

// ✅ 허용: 같은 레벨
import "internal/data"          // brain에서 OK

// ❌ 금지: 하위 → 상위
import "internal/brain"         // data에서 금지!

// ❌ 금지: 순환 참조
// signals ↔ selection 서로 import 금지
```

---

## 통합 CLI 사용법

### 실행 방식 (v10과 동일)

```bash
# 개발 모드
go run ./cmd/quant [command] [args...]

# 빌드 후
make build
./bin/quant [command] [args...]
```

### 커맨드 목록

| 커맨드 | 용도 | 예시 |
|--------|------|------|
| `api` | API 서버 실행 | `go run ./cmd/quant api --port 8080` |
| `fetcher` | 데이터 수집 (KIS, DART, Naver) | `go run ./cmd/quant fetcher collect all` |
| `worker` | 백그라운드 워커 시작 | `go run ./cmd/quant worker start` |
| `status` | 큐 상태 모니터링 | `go run ./cmd/quant status start` |
| `test-db` | DB 연결 테스트 | `go run ./cmd/quant test-db` |
| `test-logger` | Logger 테스트 | `go run ./cmd/quant test-logger` |

### 예시

```bash
# API 서버 실행
go run ./cmd/quant api
go run ./cmd/quant api --port 8080 --env production

# 데이터 수집
go run ./cmd/quant fetcher collect --source kis
go run ./cmd/quant fetcher collect --source dart
go run ./cmd/quant fetcher collect --source naver
go run ./cmd/quant fetcher collect all

# 백그라운드 워커 (큐 기반 작업 처리)
go run ./cmd/quant worker start
go run ./cmd/quant worker start --concurrency 5

# 큐 상태 모니터링
go run ./cmd/quant status start
go run ./cmd/quant status start --refresh 2s

# 테스트
go run ./cmd/quant test-db
go run ./cmd/quant test-logger

# 빌드 후 실행
make build
./bin/quant api
./bin/quant fetcher collect all
./bin/quant worker start
./bin/quant status start
```

### 장점

1. **통일성**: 모든 명령어가 `go run ./cmd/quant ...` 패턴
2. **확장성**: 새 커맨드 추가 쉬움 (`commands/new.go`)
3. **플래그 공유**: 공통 플래그 (`--env`, `--config`) 일관성
4. **빌드 단순화**: `make build` 하나로 전체 빌드

---

## 공통 포맷팅 (format.go)

모든 커맨드가 **동일한 출력 형식**을 사용하도록 통일된 포맷팅 유틸리티를 제공합니다.

### Job 실행 포맷

```
═══════════════════════════════════════════════════════════
  Fetch Ranking Data
───────────────────────────────────────────────────────────
  Job ID    : #653
  Period    : 2025-12-09 ~ 2026-01-09
  Symbols   : all
───────────────────────────────────────────────────────────
[Ranking] Manual collection triggered at 21:57:45
[Ranking] Fetched trading/KOSPI: 100 items [1/8]
[Ranking] Fetched trading/KOSDAQ: 100 items [2/8]
[Ranking] Fetched quantHigh/KOSPI: 100 items [3/8]
...
✅ Job #653 completed in 1.62s (100%)
```

### 주요 함수

| 함수 | 용도 |
|------|------|
| `PrintJobHeader(meta)` | Job 실행 헤더 (ID, Period, Symbols) |
| `PrintProgress(tag, msg, current, total)` | 진행 상황 표시 [x/y] |
| `PrintJobCompletion(jobID, duration)` | 완료 메시지 |
| `PrintList(items)` | 항목 리스트 (• bullet) |
| `PrintNumberedList(items)` | 번호 리스트 (1., 2., ...) |
| `PrintTableHeader/Row(...)` | 테이블 형식 출력 |
| `PrintSeparator()` | 구분선 (─) |
| `PrintSuccess/Error/Warning(msg)` | 상태 메시지 |

### 사용 예시

```go
// Worker에서 사용
meta := JobMetadata{
    JobID:     653,
    JobType:   "Fetch Ranking Data",
    Tag:       "Ranking",
    Period:    GetCurrentPeriod(),
    Symbols:   "all",
}
PrintJobHeader(meta)

// 진행 상황 표시
steps := []string{"Fetch KOSPI", "Fetch KOSDAQ", ...}
for i, step := range steps {
    PrintProgress("Ranking", step, i+1, len(steps))
}

// 완료
PrintJobCompletion(jobID, duration)
```

### 혜택

- **일관성**: 모든 커맨드가 동일한 출력 형식
- **가독성**: 명확한 구분자와 진행 카운터 [x/y]
- **유지보수성**: 포맷 변경 시 `format.go`만 수정
- **확장성**: 새 커맨드 추가 시 재사용

---

## HTTP 클라이언트 (pkg/httputil)

모든 HTTP 요청은 **pkg/httputil**을 통해서만 수행합니다.

### SSOT 원칙

```go
// ❌ 금지: 직접 http.Client 생성
client := &http.Client{Timeout: 10 * time.Second}
resp, _ := client.Get(url)

// ✅ 허용: httputil 사용
client := httputil.New(cfg)
resp, _ := client.Get(ctx, url)
```

### 주요 기능

| 기능 | 설명 |
|------|------|
| **타임아웃** | 요청별/전체 타임아웃 설정 |
| **재시도** | 실패 시 자동 재시도 (지수 백오프) |
| **로깅** | 요청/응답 자동 로깅 |
| **컨텍스트** | context.Context 지원 |
| **Rate Limiting** | API 호출 제한 (선택적) |

### 사용 예시

```go
// 1. Client 생성
client := httputil.New(cfg)

// 2. GET 요청
resp, err := client.Get(ctx, "https://api.example.com/data")
if err != nil {
    return err
}
defer resp.Body.Close()

// 3. POST 요청 (JSON)
data := map[string]interface{}{"key": "value"}
resp, err := client.PostJSON(ctx, url, data)

// 4. 재시도 설정
client.WithRetry(3, 1*time.Second)
```

### 재시도 전략

```go
// 지수 백오프 (Exponential Backoff)
// 1st retry: 1s
// 2nd retry: 2s
// 3rd retry: 4s
// Max: 3 retries

client := httputil.NewWithRetry(cfg, httputil.RetryConfig{
    MaxRetries: 3,
    InitialDelay: 1 * time.Second,
    MaxDelay: 10 * time.Second,
})
```

### 혜택

- **SSOT**: HTTP Client 생성은 한 곳에서만
- **일관성**: 모든 외부 API 호출이 동일한 방식
- **관측성**: 자동 로깅으로 디버깅 쉬움
- **안정성**: 재시도 로직으로 일시적 오류 대응

---

**Prev**: [Contracts](../architecture/contracts.md)
**Next**: [Data Layer](./data-layer.md)
