# 종목 선정 전략 설계

> v13 핵심: **명확·간결·효율** - 모든 파라미터는 수치로 정의하고, SSOT 설정 파일로 관리

---

## 1. 설계 원칙

### 1.1 v13 철학 적용

| 원칙 | 적용 |
|------|------|
| **명확함** | 모든 조건을 수치로 정의 (문장 금지) |
| **간결함** | 핵심 파라미터만 노출, 내부 복잡성 숨김 |
| **효율성** | 비용/유동성 먼저 → 신호 → 비중 → 청산 순 |

### 1.2 핵심 제약

```
⚠️ 백테스트 성과 ≠ 실전 성과
   → 비용/유동성 가정이 현실적이어야 재현 가능
```

**튜닝 순서 (절대 규칙)**:
1. 비용/유동성 캘리브레이션 (S1)
2. Hard Cut 조건 (S3)
3. 랭킹 가중치 (S4)
4. 포트폴리오 제약 (S5)
5. 청산 전략 (Exit)

---

## 2. 파이프라인별 파라미터

### 2.1 S1 Universe (투자 가능 풀)

> **가장 중요**: 여기서 비용/유동성이 결정됨

| 파라미터 | 시작값 | 튜닝 범위 | 설명 |
|---------|--------|----------|------|
| `marketcap_min` | 2,000억 | 1,000~5,000억 | 시총 하한 |
| `adtv20_min` | 20억 | 10~50억 | 20일 평균 거래대금 하한 |
| `price_min` | 1,000원 | 500~5,000원 | 종가 하한 (저가주 제외) |
| `ipo_days_min` | 180일 | 60~365일 | 상장 후 기간 |
| `spread_max` | 0.6% | 0.3~1.0% | 스프레드 상한 |

**제외 조건 (Boolean)**:
- `trading_halt`: 거래정지
- `admin_issue`: 관리종목
- `managed_stock`: 투자경고/위험

```yaml
# 핵심 3개 먼저 고정
universe:
  marketcap_min_krw: 200_000_000_000  # 2,000억
  adtv20_min_krw: 20_000_000_000      # 20억
  spread_max_pct: 0.006               # 0.6%
```

---

### 2.2 S2 Signals (6팩터 점수화)

> **핵심**: 가중치보다 **산출 기간(lookback)과 정규화**가 더 중요

#### 정규화 (필수)

```
원시값 → Winsorize(1.5%) → Z-score → Clip(±3) → 0~100 맵핑
```

#### 팩터별 산출 정의

| 팩터 | 산출 방식 | Lookback | 비고 |
|------|----------|----------|------|
| **모멘텀** | 20D/60D/120D 수익률 가중평균 | 20/60/120일 | 단기만 쓰면 과열 위험 |
| **기술적** | MA(20/60) 기울기 + 52주 신고가 근접도 | 20/60/252일 | 기울기로 횡보장 필터링 |
| **밸류** | PBR + PER (EV/EBITDA 권장) | TTM | PER 단독은 왜곡 큼 |
| **퀄리티** | ROE, 영업이익률, 부채비율 | TTM | 완만하게 반영 |
| **수급** | 외인/기관 순매수 (거래대금 대비) | 5/20일 | **비율로 정규화** |
| **이벤트** | 급등/공시/실적 서프라이즈 | TTL 10일 | 유효기간이 성과 좌우 |

```yaml
signals:
  normalization:
    winsorize_pct: 0.015
    zscore_clip: 3.0

  momentum:
    lookbacks: [20, 60, 120]
    weights: [0.40, 0.35, 0.25]

  flow:
    lookbacks: [5, 20]
    normalize_by: "adtv20"  # 핵심: 절대금액 아닌 비율
```

---

### 2.3 S3 Screening (Hard Cut)

> **수치로 정의**: "급락 종목 제외" → "1일 -9% 이하 제외"

| 조건 | 시작값 | 튜닝 범위 | 목적 |
|------|--------|----------|------|
| `day1_return_min` | -9% | -7~-12% | 급락 제외 |
| `day5_return_min` | -18% | -12~-25% | 급락 제외 |
| `day5_return_max` | +35% | +25~+60% | 과열 제외 |
| `net_income_min` | 0 | - | 적자 제외 |
| `per_max` | 60 | 40~100 | 고PER 제외 |
| `vol20_top_pct` | 10% | 5~20% | 고변동성 제외 |

```yaml
screening:
  drawdown:
    day1_return_min: -0.09
    day5_return_min: -0.18
  overheat:
    day5_return_max: 0.35
  fundamentals:
    net_income_ttm_min: 0
    per_max: 60
  volatility:
    vol20_exclude_top_pct: 0.10
```

---

### 2.4 S4 Ranking (가중치)

> **제약 조건 필수**: 모멘텀+기술 합 ≤ 50%

#### 시작 가중치 (합 = 100)

| 팩터 | 시작값 | 튜닝 범위 |
|------|--------|----------|
| 모멘텀 | 25% | 20~35% |
| 수급 | 20% | 10~30% |
| 기술적 | 15% | 10~25% |
| 이벤트 | 15% | 5~20% |
| 밸류 | 15% | 10~30% |
| 퀄리티 | 10% | 5~15% |

```yaml
ranking:
  weights:
    momentum: 25
    flow: 20
    technical: 15
    event: 15
    value: 15
    quality: 10

  constraints:
    momentum_plus_technical_max: 50  # 상관 높은 팩터 동시 과대 방지
    monthly_change_max: 5            # 월 단위 ±5p 제한
```

---

### 2.5 S5 Portfolio (포트폴리오 구성)

#### 핵심 파라미터

| 파라미터 | 시작값 | 튜닝 범위 | 설명 |
|---------|--------|----------|------|
| `holdings_target` | 20 | 15~25 | 목표 종목 수 |
| `position_min` | 4% | 2~5% | 종목당 최소 비중 |
| `position_max` | 10% | 8~15% | 종목당 최대 비중 |
| `cash_target` | 10% | 5~20% | 현금 비중 |
| `sector_max` | 25% | 15~35% | 섹터당 상한 |
| `turnover_daily_max` | 20% | 10~35% | 일 회전율 상한 |

#### 비중 산정 (Tiered)

```yaml
portfolio:
  holdings_target: 20
  cash_target_pct: 0.10

  weighting:
    type: "tiered"
    tiers:
      - { rank: "1-5",   weight: 9% }   # 상위 5개
      - { rank: "6-15",  weight: 5% }   # 중간 10개
      - { rank: "16-20", weight: 3% }   # 하위 5개

  constraints:
    position_min_pct: 0.04
    position_max_pct: 0.10
    sector_max_pct: 0.25
```

#### 유동성 캡 (필수)

```yaml
liquidity_caps:
  max_order_to_adtv20: 0.02   # 주문/ADTV ≤ 2%
  max_participation_per_min: 0.10  # 분당 참여율 ≤ 10%
```

---

### 2.6 Execution (주문 집행)

> **0.1% 슬리피지는 과소평가**: 현실은 0.25~0.45%

#### 슬리피지 모델 (2단계)

| ADTV20 | 슬리피지 가정 |
|--------|-------------|
| 50억 이상 | 0.25% |
| 20억 이상 | 0.45% |
| 20억 미만 | 0.70% (Universe 제외 권장) |

#### 주문 규칙

```yaml
execution:
  order_type: "LIMIT"
  limit_policy:
    buy: "ASK1"   # 매수: 매도1호가
    sell: "BID1"  # 매도: 매수1호가

  time_window:
    start: "09:05"  # 시초가 회피
    end: "15:10"    # 동시호가 회피

  splitting:
    trigger: "order_to_adtv20 >= 2%"
    slices: 3~8
    interval_sec: 90

  slippage:
    large_cap: 0.0025   # ADTV 50억+
    mid_cap: 0.0045     # ADTV 20억+
```

---

### 2.7 Exit (청산 전략)

> **변동성(ATR) 연동 권장**: 고정 %보다 시장 적응력 높음

#### Option A: 고정형 (단순)

| 파라미터 | 시작값 | 튜닝 범위 |
|---------|--------|----------|
| 초기 손절 | -7% | -5~-12% |
| 트레일링 활성화 | +6% | +4~+10% |
| 트레일링 폭 | -5% | -3~-8% |
| 타임스탑 | 10일 | 7~15일 |
| 부분익절 트리거 | +18% | +12~+25% |
| 부분익절 비율 | 50% | 30~70% |

#### Option B: ATR 연동 (권장)

| 파라미터 | 산출 |
|---------|------|
| 초기 손절 | max(7%, 2.0 × ATR20) |
| 트레일링 폭 | max(5%, 1.5 × ATR20) |

```yaml
exit:
  mode: "atr"  # "fixed" | "atr"

  fixed:
    stop_loss: -0.07
    trail_activate: 0.06
    trail_drawdown: -0.05
    time_stop_days: 10
    partial_tp:
      trigger: 0.18
      sell_fraction: 0.50

  atr:
    window: 20
    stop_mult: 2.0
    trail_mult: 1.5
    min_stop: 0.07
    min_trail: 0.05
```

---

### 2.8 NASDAQ 2차 조정 (08시)

> **한국 시장의 NASDAQ 의존성 반영**

| NASDAQ 변동 | 조치 |
|------------|------|
| ≤ -3.0% | 주식 비중 -30% (현금 ↑) |
| ≤ -1.5% | 주식 비중 -15% |
| ≥ +1.5% | 주식 비중 +10% |

```yaml
risk_overlay:
  nasdaq_adjust:
    enable: true
    run_time: "08:00"
    triggers:
      - { nasdaq_ret_le: -0.030, scale: -0.30 }
      - { nasdaq_ret_le: -0.015, scale: -0.15 }
      - { nasdaq_ret_ge:  0.015, scale:  0.10 }
    clamp:
      min_equity: 0.60
      max_equity: 1.00
```

---

## 3. SSOT 설정 파일 구조

### 3.1 파일 위치

```
backend/
├── config/
│   └── strategy/
│       └── korea_equity_v13.yaml  ← SSOT
```

### 3.2 전체 스키마

```yaml
# backend/config/strategy/korea_equity_v13.yaml
meta:
  strategy_id: "korea_equity_v13"
  version: "1.0.0"
  timezone: "Asia/Seoul"
  decision_time: "17:00"

universe:
  # S1 파라미터

signals:
  # S2 파라미터

screening:
  # S3 파라미터

ranking:
  # S4 파라미터

portfolio:
  # S5 파라미터

execution:
  # 집행 파라미터

exit:
  # 청산 파라미터

risk_overlay:
  # NASDAQ 조정 등

backtest:
  commission_bps: 1.5
  tax_bps: 23.0
```

### 3.3 재현성 보장

```go
// 모든 의사결정에 config_hash 저장
type DecisionSnapshot struct {
    ConfigHash    string    // sha256(yaml)
    StrategyID    string
    GitCommit     string
    CreatedAt     time.Time
}
```

---

## 4. Validation 규칙

### 4.1 런타임 검증 (필수)

```go
// 잘못된 설정 방지
func ValidateConfig(cfg *StrategyConfig) error {
    // 가중치 합 = 100
    if sum(cfg.Ranking.Weights) != 100 {
        return ErrInvalidWeights
    }

    // 비중 합 + 현금 = 100%
    totalWeight := sumTiers(cfg.Portfolio.Tiers) + cfg.Portfolio.CashTarget
    if abs(totalWeight - 1.0) > 0.005 {
        return ErrInvalidAllocation
    }

    // position_min <= position_max
    if cfg.Portfolio.PositionMin > cfg.Portfolio.PositionMax {
        return ErrInvalidPositionBounds
    }

    // sector_max >= position_max
    if cfg.Portfolio.SectorMax < cfg.Portfolio.PositionMax {
        return ErrSectorCapTooLow
    }

    return nil
}
```

### 4.2 경고 규칙

```go
// 권장 범위 체크
func WarnIfSuboptimal(cfg *StrategyConfig) []Warning {
    var warnings []Warning

    if cfg.Universe.ADTV20Min < 10_000_000_000 {
        warnings = append(warnings, "ADTV < 10억: 체결 리스크 높음")
    }

    if cfg.Execution.Slippage.MidCap < 0.003 {
        warnings = append(warnings, "슬리피지 가정이 낙관적")
    }

    return warnings
}
```

---

## 5. 민감도 실험 가이드

### 5.1 실험 순서 (반드시 순서대로)

| 순서 | 실험 | 목표 |
|------|------|------|
| 0 | 비용/유동성 | 백테스트→실전 재현성 확보 |
| 1 | S3 Hard Cut | MDD 최소화 |
| 2 | S4 가중치 | 수익률/Sharpe 최적화 |
| 3 | S5 포트폴리오 | 변동성/비용 최적화 |
| 4 | Exit 청산 | 손익비/승률 최적화 |

### 5.2 실험 0: 비용/유동성

```yaml
# 그리드 탐색
adtv20_min: [10억, 20억, 30억, 50억]
spread_max: [0.4%, 0.6%, 0.8%]
slippage_mid: [0.35%, 0.45%, 0.60%]
max_order_to_adtv: [1%, 2%, 3%]
```

**통과 기준**: 슬리피지 증가해도 Sharpe 유지

### 5.3 실험 1: S3 Hard Cut

```yaml
day1_return_min: [-7%, -9%, -11%]
day5_return_min: [-12%, -18%, -25%]
per_max: [40, 60, 80]
vol20_exclude_top: [5%, 10%, 20%]
```

**목표**: MDD 감소 우선, 수익률은 그 다음

### 5.4 실험 2: 가중치

```yaml
# 5p 단위 그리드
momentum: [20, 25, 30, 35]
flow: [10, 15, 20, 25, 30]
# ... 합 100 유지
```

**제약**: momentum + technical ≤ 50

### 5.5 평가 지표 (모든 실험에서 기록)

| 분류 | 지표 |
|------|------|
| 성과 | CAGR, Sharpe, Sortino, MDD, Calmar |
| 거래 | Turnover, 평균보유일, 체결률, 주문/ADTV분포 |
| 손익구조 | 승률, 평균익절/손절, Payoff ratio, Tail loss |
| 용량 | order_to_adtv > 2% 빈도 |

```go
type ExperimentResult struct {
    ConfigHash     string
    GitCommit      string
    DataSnapshotID string

    CAGR           float64
    Sharpe         float64
    MDD            float64
    Turnover       float64
    AvgHoldingDays float64
    WinRate        float64
}
```

---

## 6. Top 12 우선 고정 파라미터

**이것들을 먼저 확정해야 나머지가 의미 있음**:

| # | 파라미터 | 시작값 | 튜닝 범위 |
|---|---------|--------|----------|
| 1 | 시총 하한 | 2,000억 | 1,000~5,000억 |
| 2 | ADTV20 하한 | 20억 | 10~50억 |
| 3 | 스프레드 상한 | 0.6% | 0.3~1.0% |
| 4 | 급락 컷 (1D) | -9% | -7~-12% |
| 5 | 급락 컷 (5D) | -18% | -12~-25% |
| 6 | PER 상한 | 60 | 40~100 |
| 7 | 변동성 컷 | 상위 10% | 5~20% |
| 8 | 종목 수 | 20 | 15~25 |
| 9 | 비중 범위 | 4~10% | 2~15% |
| 10 | 섹터 상한 | 25% | 15~35% |
| 11 | 일 회전율 | 20% | 10~35% |
| 12 | 슬리피지 | 0.25~0.45% | 0.15~0.80% |

---

## 7. 구현 로드맵

### Phase 1: 설정 인프라
- [ ] `backend/config/strategy/` 폴더 구조
- [ ] YAML 파싱 및 Validation
- [ ] ConfigHash 생성 로직

### Phase 2: S1~S3 구현
- [ ] Universe 필터링 (시총, ADTV, 스프레드)
- [ ] 6팩터 점수화 + 정규화
- [ ] Hard Cut 스크리닝

### Phase 3: S4~S5 구현
- [ ] 가중 평균 랭킹
- [ ] Tiered 비중 산정
- [ ] 유동성 캡 적용

### Phase 4: Execution + Exit
- [ ] 슬리피지 모델 적용
- [ ] ATR 기반 청산 로직
- [ ] NASDAQ 2차 조정

### Phase 5: 백테스트/튜닝
- [ ] 실험 프레임워크
- [ ] 결과 저장/비교
- [ ] 최적 파라미터 확정
