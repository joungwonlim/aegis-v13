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
    // Hard Cut 조건
    MinMomentum  float64 `yaml:"min_momentum"`   // -1.0 이상
    MinVolume    int64   `yaml:"min_volume"`     // 5억 이상
    MaxPER       float64 `yaml:"max_per"`        // 50 이하
    MinROE       float64 `yaml:"min_roe"`        // 5% 이상

    // Negative 필터
    ExcludeNegativeEarnings bool `yaml:"exclude_negative_earnings"`
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
    // 모멘텀 필터
    if signal.Factors["momentum"] < s.config.MinMomentum {
        return false
    }

    // PER 필터
    if per := signal.Factors["per"]; per > s.config.MaxPER || per < 0 {
        return false
    }

    // ROE 필터
    if signal.Factors["roe"] < s.config.MinROE {
        return false
    }

    return true
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
    Momentum  float64 `yaml:"momentum"`   // 0.30
    Value     float64 `yaml:"value"`      // 0.25
    Quality   float64 `yaml:"quality"`    // 0.20
    Event     float64 `yaml:"event"`      // 0.15
    Technical float64 `yaml:"technical"`  // 0.10
}

func (r *ranker) Rank(ctx context.Context, codes []string, signals *contracts.SignalSet) ([]contracts.RankedStock, error) {
    ranked := make([]contracts.RankedStock, 0, len(codes))

    for _, code := range codes {
        signal := signals.Signals[code]

        // 가중 합산
        totalScore :=
            signal.Factors["momentum"] * r.weights.Momentum +
            signal.Factors["value"] * r.weights.Value +
            signal.Factors["quality"] * r.weights.Quality +
            signal.Factors["event"] * r.weights.Event +
            signal.Factors["technical"] * r.weights.Technical

        ranked = append(ranked, contracts.RankedStock{
            Code:       code,
            TotalScore: totalScore,
            Factors:    signal.Factors,
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
  min_momentum: -1.0
  min_volume: 500000000    # 5억
  max_per: 50
  min_roe: 0.05
  exclude_negative_earnings: true

ranker:
  weights:
    momentum: 0.30
    value: 0.25
    quality: 0.20
    event: 0.15
    technical: 0.10

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
