---
sidebar_position: 4
title: Selection Layer
description: S2-S3 스크리닝/랭킹
---

# Selection Layer

> S2: Screener + S3: Ranker

---

## 파이프라인 구조 (S0~S4)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Aegis v13 Pipeline                                │
└─────────────────────────────────────────────────────────────────────────┘

S0: Naver Ranking    → 실시간 종목 데이터 수집
        ↓
S1: Signals          → 6개 팩터 계산 (Momentum, Technical, Value, Quality, Flow, Event)
        ↓
S2: Screener         → Hard Cut 필터링 (PER/PBR/ROE 재무 지표)
        ↓
S3: Ranking          → 종합 점수 산출 + 순위 부여
        ↓
S4: Portfolio        → 포트폴리오 구성
```

---

## 책임

| Stage | 역할 | 목적 |
|-------|------|------|
| S2 | Hard Cut 필터링 (재무) | 부실 종목 제거 |
| S3 | 종합 점수 산출 | Top N 선별 |

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
│         S2: Screener              │       │           S3: Ranker              │
│       (Hard Cut 필터)              │──────▶│        (가중치 랭킹)              │
├───────────────────────────────────┤       ├───────────────────────────────────┤
│ 입력: S1 Signal 통과 종목         │       │ 입력: S2 필터 통과 종목           │
│                                   │       │                                   │
│ Hard Cut 조건 (재무 지표만):       │       │ 가중치 (YAML SSOT):               │
│                                   │       │ • Momentum: 25%                   │
│ • PER > 0 AND <= 50              │       │ • Flow: 20%                       │
│   (적자/고평가 제외)              │       │ • Technical: 15%                  │
│                                   │       │ • Event: 15%                      │
│ • PBR >= 0.2                     │       │ • Value: 15%                      │
│   (자산가치 필터)                 │       │ • Quality: 10%                    │
│                                   │       │                                   │
│ • ROE >= 5                       │       │ 출력: RankedStock[] (점수순)       │
│   (수익성 필터)                   │       │                                   │
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

## S2: Screener

### 목적

**재무 지표 기반 Hard Cut**으로 부실 종목 제거

```
S1 통과 종목 (1,000개) → S2 Screener → 필터 통과 (300개)
                         (PER/PBR/ROE)
```

### Hard Cut 조건 (재무 지표만)

| 지표 | 조건 | 목적 |
|------|------|------|
| **PER** | > 0 AND <= 50 | 적자/고평가 기업 제외 |
| **PBR** | >= 0.2 | 자산가치 부실 기업 제외 |
| **ROE** | >= 5 | 수익성 낮은 기업 제외 |

:::info 팩터 조건 제외
팩터 조건(momentum, technical, flow)은 S3 Ranker에서 가중치로 반영됩니다.
S2 Screener는 **재무 건전성 필터링만** 담당합니다.
:::

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
    // 재무 Hard Cut 조건
    MaxPER float64 `yaml:"max_per"`  // 50 (고평가 제외)
    MinPER float64 `yaml:"min_per"`  // 0 (적자 제외)
    MinPBR float64 `yaml:"min_pbr"`  // 0.2 (자산가치 필터)
    MinROE float64 `yaml:"min_roe"`  // 5 (수익성 필터)
}

func (s *screener) passAllConditions(fundamentals Fundamentals) bool {
    // PER 필터: 적자/고평가 제외
    if fundamentals.PER <= s.config.MinPER || fundamentals.PER > s.config.MaxPER {
        return false
    }

    // PBR 필터: 자산가치 부실 제외
    if fundamentals.PBR < s.config.MinPBR {
        return false
    }

    // ROE 필터: 수익성 부족 제외
    if fundamentals.ROE < s.config.MinROE {
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
    "count": 320,
    "hardCutConditions": {
      "per": "> 0 AND <= 50 (적자/고평가 제외)",
      "pbr": ">= 0.2 (자산가치 필터)",
      "roe": ">= 5 (수익성 필터)"
    },
    "items": [
      {
        "stockCode": "005930",
        "stockName": "삼성전자",
        "market": "KOSPI",
        "per": 12.5,
        "pbr": 1.2,
        "roe": 8.5,
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

## S3: Ranker

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
    Value     float64 `yaml:"value"`      // 0.15
    Quality   float64 `yaml:"quality"`    // 0.10
    Flow      float64 `yaml:"flow"`       // 0.20 ⭐ 수급
    Event     float64 `yaml:"event"`      // 0.15
}

func (r *ranker) Rank(ctx context.Context, codes []string, signals *contracts.SignalSet) ([]contracts.RankedStock, error) {
    ranked := make([]contracts.RankedStock, 0, len(codes))

    for _, code := range codes {
        signal := signals.Signals[code]

        // 가중 합산 (SSOT: data-flow.md 기준)
        totalScore :=
            signal.Momentum * r.weights.Momentum +      // 25%
            signal.Technical * r.weights.Technical +    // 15%
            signal.Value * r.weights.Value +            // 15%
            signal.Quality * r.weights.Quality +        // 10%
            signal.Flow * r.weights.Flow +              // 20% ⭐
            signal.Event * r.weights.Event              // 15%

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
  # 재무 Hard Cut 조건 (재무 지표만)
  max_per: 50            # PER <= 50 (고평가 제외)
  min_per: 0             # PER > 0 (적자기업 제외)
  min_pbr: 0.2           # PBR >= 0.2 (자산가치 필터)
  min_roe: 5             # ROE >= 5 (수익성 필터)

ranker:
  weights:
    momentum: 0.25
    technical: 0.15
    value: 0.15
    quality: 0.10
    flow: 0.20       # 수급 ⭐
    event: 0.15

  # Top N 선택
  top_n: 30
```

---

## 스크리닝 단계별 효과

```
┌─────────────────────────────────────────────┐
│ S0: Naver Ranking (실시간)          2,500개  │
├─────────────────────────────────────────────┤
│ S1: Signals (6개 팩터 계산)         2,000개  │
├─────────────────────────────────────────────┤
│ S2: Screener (재무 Hard Cut)               │
│   - PER > 0 AND <= 50               800개  │
│   - PBR >= 0.2                      600개  │
│   - ROE >= 5                        400개  │
├─────────────────────────────────────────────┤
│ S3: Ranker (Top 30)                   30개  │
├─────────────────────────────────────────────┤
│ S4: Portfolio                         30개  │
└─────────────────────────────────────────────┘
```

**종목 필터링**: 2,500개 → 30개 = **99% 감소**

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
