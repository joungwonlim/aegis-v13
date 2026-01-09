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
│   └── api/
│       └── main.go           # 엔트리포인트
│
├── internal/                  # 비공개 비즈니스 로직
│   ├── contracts/            # ⭐ 타입/인터페이스 SSOT
│   │   ├── data.go
│   │   ├── universe.go
│   │   ├── signals.go
│   │   ├── portfolio.go
│   │   └── interfaces.go
│   │
│   ├── brain/                # 오케스트레이터 (로직 없음)
│   │   └── orchestrator.go
│   │
│   ├── data/                 # S0-S1: 데이터 레이어
│   │   ├── quality.go        # 품질 검증
│   │   ├── universe.go       # 유니버스 생성
│   │   └── repository.go     # DB 접근
│   │
│   ├── signals/              # S2: 시그널 레이어
│   │   ├── momentum.go
│   │   ├── value.go
│   │   ├── event.go
│   │   └── builder.go        # SignalBuilder 구현
│   │
│   ├── selection/            # S3-S4: 선택 레이어
│   │   ├── screener.go       # 1차 필터링
│   │   └── ranker.go         # 순위 산출
│   │
│   ├── portfolio/            # S5: 포트폴리오 레이어
│   │   ├── constructor.go
│   │   └── rebalancer.go
│   │
│   ├── execution/            # S6: 실행 레이어
│   │   ├── planner.go
│   │   └── broker.go         # KIS 연동
│   │
│   ├── audit/                # S7: 감사 레이어
│   │   ├── performance.go
│   │   └── attribution.go
│   │
│   ├── external/             # ⭐ 외부 API SSOT
│   │   ├── kis/
│   │   ├── dart/
│   │   └── naver/
│   │
│   └── api/                  # HTTP 핸들러
│       ├── router.go
│       ├── handlers/
│       └── middleware/
│
├── pkg/                      # ⭐ 공용 패키지 SSOT
│   ├── config/               # 환경변수
│   ├── database/             # DB 연결
│   ├── logger/               # 로깅
│   └── httputil/             # HTTP 유틸
│
├── migrations/               # DB 마이그레이션
│
└── Makefile
```

---

## 레이어별 파일 수

| 레이어 | 파일 수 | 역할 |
|--------|---------|------|
| contracts | 5 | 타입/인터페이스 정의 |
| brain | 1 | 오케스트레이터 |
| data | 3 | 데이터 수집/유니버스 |
| signals | 4 | 시그널 생성 |
| selection | 2 | 스크리닝/랭킹 |
| portfolio | 2 | 포트폴리오 구성 |
| execution | 2 | 주문 실행 |
| audit | 2 | 성과 분석 |
| **Total** | **~21** | v10 brain 42개 → 21개 |

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

**Prev**: [Contracts](../architecture/contracts.md)
**Next**: [Data Layer](./data-layer.md)
