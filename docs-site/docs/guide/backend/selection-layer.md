---
sidebar_position: 4
title: Selection Layer
description: S3-S4 스크리닝/랭킹
---

# Selection Layer

> S3: Screener + S4: Ranker

---

## 책임

| Stage | 역할 | 목적 |
|-------|------|------|
| S3 | Hard Cut 필터링 | AI 비용 절감 (90% 제거) |
| S4 | 종합 점수 산출 | Top N 선별 |

---

## 구현 상태 (2026-01-11)

| 컴포넌트 | 상태 | 파일 |
|---------|------|------|
| **Screener** | ✅ 완료 | `internal/selection/screener.go` |
| **Ranker** | ✅ 완료 | `internal/selection/ranker.go` |
| **Repository** | ✅ 완료 | `internal/selection/repository.go` |

:::tip YAML SSOT
스크리닝 조건과 랭킹 가중치는 `backend/config/strategy/korea_equity_v13.yaml`의 `screening` 및 `ranking` 섹션에서 관리됩니다.
:::

---

## 프로세스 흐름

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Selection Pipeline                                │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┴───────────────────────────┐
        ▼                                                       ▼
┌───────────────────────────────────┐       ┌───────────────────────────────────┐
│         S3: Screener              │       │           S4: Ranker              │
│       (Hard Cut 필터)              │──────▶│        (가중치 랭킹)              │
├───────────────────────────────────┤       ├───────────────────────────────────┤
│ 입력: SignalSet (전체 종목)        │       │ 입력: 필터 통과 종목 + SignalSet   │
│                                   │       │                                   │
│ Hard Cut 조건:                    │       │ 가중치 (YAML SSOT):               │
│                                   │       │ • Momentum: 25%                   │
│ [팩터 조건]                        │       │ • Flow: 20%                       │
│ • momentum >= 0 (상승 모멘텀)     │       │ • Technical: 15%                  │
│ • technical >= -0.5 (과매도 제외)  │       │ • Event: 15%                      │
│ • flow >= -0.3 (수급 악화 제외)    │       │ • Value: 15%                      │
│                                   │       │ • Quality: 10%                    │
│ [재무 조건]                        │       │                                   │
│ • PER > 0 AND <= 50 (고평가 제외) │       │ 출력: RankedStock[] (점수순)       │
│ • PBR >= 0.2 (자산가치 필터)       │       │                                   │
│                                   │       │                                   │
│ 출력: 통과 종목 리스트             │       │                                   │
└───────────────────────────────────┘       └───────────────────────────────────┘
```

---

## 폴더 구조

```
internal/selection/
├── screener.go     # Screener 구현
├── ranker.go       # Ranker 구현
└── config.go       # 설정 로더
```

---

## S3: Screener

### 목적

**AI 분석 전 빠른 필터링**으로 비용 절감

```
1000개 종목 → Screener → 100개 → AI 분석
                (90% 제거)
```

### 인터페이스

```go
type Screener interface {
    Screen(ctx context.Context, signals *SignalSet) ([]string, error)
}
```

### 구현

```go
// internal/selection/screener.go

type screener struct {
    config ScreenerConfig
}

type ScreenerConfig struct {
    // Hard Cut 조건 (팩터 점수 기반)
    MinMomentum  float64 `yaml:"min_momentum"`   // 0 이상 (상승 모멘텀)
    MinTechnical float64 `yaml:"min_technical"`  // -0.5 이상 (과매도 제외)
    MinFlow      float64 `yaml:"min_flow"`       // -0.3 이상 (수급 악화 제외)
}

func (s *screener) Screen(ctx context.Context, signals *contracts.SignalSet) ([]string, error) {
    passed := make([]string, 0)

    for code, signal := range signals.Signals {
        if s.passAllConditions(signal) {
            passed = append(passed, code)
        }
    }

    return passed, nil
}

func (s *screener) passAllConditions(signal contracts.StockSignal) bool {
    // 모멘텀 필터: 상승 추세만
    if signal.Momentum < s.config.MinMomentum {
        return false
    }

    // 기술적 필터: 과매도 종목 제외
    if signal.Technical < s.config.MinTechnical {
        return false
    }

    // 수급 필터: 수급 악화 종목 제외
    if signal.Flow < s.config.MinFlow {
        return false
    }

    return true
}
```

### API 엔드포인트

```
GET /api/v1/pipeline/screened?market={ALL|KOSPI|KOSDAQ}
```

**응답 예시:**
```json
{
  "success": true,
  "data": {
    "date": "2026-01-11",
    "market": "ALL",
    "count": 450,
    "hardCutConditions": {
      "momentum": ">= 0",
      "technical": ">= -0.5",
      "flow": ">= -0.3"
    },
    "items": [
      {
        "stockCode": "005930",
        "stockName": "삼성전자",
        "market": "KOSPI",
        "momentum": 0.25,
        "technical": 0.10,
        "flow": 0.15,
        "totalScore": 0.85,
        "passedAll": true
      }
    ]
  }
}
```

---

## S4: Ranker

### 인터페이스

```go
type Ranker interface {
    Rank(ctx context.Context, codes []string, signals *SignalSet) ([]RankedStock, error)
}
```

### 구현

```go
// internal/selection/ranker.go

type ranker struct {
    weights WeightConfig
}

type WeightConfig struct {
    Momentum  float64 `yaml:"momentum"`   // 0.25
    Technical float64 `yaml:"technical"`  // 0.15
    Value     float64 `yaml:"value"`      // 0.20
    Quality   float64 `yaml:"quality"`    // 0.15
    Flow      float64 `yaml:"flow"`       // 0.20 ⭐ 수급
    Event     float64 `yaml:"event"`      // 0.05
}

func (r *ranker) Rank(ctx context.Context, codes []string, signals *contracts.SignalSet) ([]contracts.RankedStock, error) {
    ranked := make([]contracts.RankedStock, 0, len(codes))

    for _, code := range codes {
        signal := signals.Signals[code]

        // 가중 합산 (SSOT: data-flow.md 기준)
        totalScore :=
            signal.Momentum * r.weights.Momentum +      // 25%
            signal.Technical * r.weights.Technical +    // 15%
            signal.Value * r.weights.Value +            // 20%
            signal.Quality * r.weights.Quality +        // 15%
            signal.Flow * r.weights.Flow +              // 20% ⭐
            signal.Event * r.weights.Event              // 5%

        ranked = append(ranked, contracts.RankedStock{
            Code:       code,
            TotalScore: totalScore,
            Scores: contracts.ScoreDetail{
                Momentum:  signal.Momentum,
                Technical: signal.Technical,
                Value:     signal.Value,
                Quality:   signal.Quality,
                Flow:      signal.Flow,
                Event:     signal.Event,
            },
        })
    }

    // 점수 내림차순 정렬
    sort.Slice(ranked, func(i, j int) bool {
        return ranked[i].TotalScore > ranked[j].TotalScore
    })

    // 순위 부여
    for i := range ranked {
        ranked[i].Rank = i + 1
    }

    return ranked, nil
}
```

---

## 설정 예시 (YAML)

```yaml
# config/selection.yaml

screener:
  # 팩터 Hard Cut 조건
  min_momentum: 0.0      # 상승 모멘텀만
  min_technical: -0.5    # 과매도 제외
  min_flow: -0.3         # 수급 악화 제외

  # 재무 Hard Cut 조건
  max_per: 50            # PER <= 50 (고평가 제외)
  min_per: 0             # PER > 0 (적자기업 제외)
  min_pbr: 0.2           # PBR >= 0.2 (자산가치 필터)

ranker:
  weights:
    momentum: 0.25
    technical: 0.15
    value: 0.20
    quality: 0.15
    flow: 0.20       # 수급 ⭐
    event: 0.05

  # Top N 선택
  top_n: 30
```

---

## 스크리닝 단계별 효과

```
┌─────────────────────────────────────────────┐
│ 전체 종목                          2,500개  │
├─────────────────────────────────────────────┤
│ S1: Universe (거래정지/관리종목 제외)  2,000개  │
├─────────────────────────────────────────────┤
│ S3: Screener                               │
│   - 시총 필터                       1,500개  │
│   - 거래대금 필터                   1,000개  │
│   - 모멘텀 필터                      500개  │
│   - 재무 필터                        200개  │
├─────────────────────────────────────────────┤
│ S4: Ranker (Top 30)                   30개  │
└─────────────────────────────────────────────┘
```

**AI 비용 절감**: 2,500개 → 30개 = **99% 감소**

---

## DB 스키마

```sql
-- selection.screened: 스크리닝 결과
CREATE TABLE selection.screened (
    id          SERIAL PRIMARY KEY,
    date        DATE NOT NULL,
    passed      TEXT[],         -- 통과 종목
    filtered    JSONB,          -- 필터별 제외 수
    total_input INT,
    total_passed INT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- selection.ranked: 랭킹 결과
CREATE TABLE selection.ranked (
    id          SERIAL PRIMARY KEY,
    date        DATE NOT NULL,
    code        VARCHAR(10) NOT NULL,
    rank        INT NOT NULL,
    total_score DECIMAL(8,4),
    factors     JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, code)
);

CREATE INDEX idx_ranked_date_rank ON selection.ranked(date, rank);
```

---

**Prev**: [Signals Layer](./signals-layer.md)
**Next**: [Portfolio Layer](./portfolio-layer.md)
