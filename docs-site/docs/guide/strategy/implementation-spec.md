# strategyconfig 구현 명세서

> Sonnet 개발용 - 이 문서대로 구현할 것

---

## 1. 개요

### 1.1 목적
- 종목 선정 전략 설정을 SSOT YAML 파일로 관리
- Go 코드에서 로드, 검증, 해시 생성
- DecisionSnapshot에 저장하여 재현성 보장

### 1.2 파일 구조
```
backend/
├── config/strategy/
│   └── korea_equity_v13.yaml    ← SSOT 설정 파일 (수정)
└── internal/strategyconfig/      ← 신규 패키지
    ├── config.go                 ← 타입 정의
    ├── loader.go                 ← Load, Hash 함수
    ├── validate.go               ← Validate, Warn 함수
    └── config_test.go            ← 테스트
```

---

## 2. YAML 수정 사항

### 2.1 현재 오류 목록

| # | 항목 | 현재값 | 수정값 |
|---|------|--------|--------|
| 1 | `universe.filters.adtv20_min_krw` | `20_000_000_000` | `2_000_000_000` |
| 2 | `universe.exclude_flags` | Boolean 구조 | 열거형 배열 |
| 3 | `universe.filters.spread` | `max_pct`만 | `formula` 추가 |
| 4 | `execution.avoid_*` | 존재 | 삭제 |
| 5 | `portfolio.weighting.tiers` | 합 110% | 합 90% (현금 10%) |

### 2.2 수정된 YAML 전문

```yaml
# backend/config/strategy/korea_equity_v13.yaml
# SSOT: 종목 선정 전략 설정 파일
# 문서: docs-site/docs/guide/strategy/stock-selection.md

meta:
  strategy_id: "korea_equity_v13"
  version: "1.0.1"
  timezone: "Asia/Seoul"
  decision_time_local: "17:00"
  execution_window:
    start: "09:05"
    end: "15:10"

universe:
  # 제외할 KRX 플래그 (열거형)
  exclude_krx_flags:
    - "TRADING_HALT"
    - "ADMIN_ISSUE"
    - "MANAGEMENT"
    - "INVESTMENT_WARNING"
    - "INVESTMENT_DANGER"

  filters:
    marketcap_min_krw: 200_000_000_000   # 2,000억
    adtv20_min_krw: 2_000_000_000        # 20억 (수정됨)
    price_min_krw: 1_000
    ipo_days_min: 180
    spread:
      max_pct: 0.006
      formula: "((ask1-bid1)/((ask1+bid1)/2))"

signals:
  normalization:
    winsorize_pct: 0.015
    zscore_clip: 3.0
    score_range_min: 0
    score_range_max: 100
    missing_policy: "NEUTRAL"

  momentum:
    lookbacks_days: [20, 60, 120]
    weights: [0.40, 0.35, 0.25]

  technical:
    ma_short: 20
    ma_long: 60
    slope_window: 10
    high52w_window: 252

  value:
    metrics: ["PBR", "PER"]
    per_score_cap: 80

  quality:
    metrics: ["ROE_TTM", "OP_MARGIN_TTM", "DEBT_RATIO"]

  flow:
    actors: ["FOREIGN", "INSTITUTION"]
    lookbacks_days: [5, 20]
    weights: [0.55, 0.45]
    normalize_by: "ADTV20"

  event:
    ttl_days: 10
    types: ["GAP_UP", "SURGE", "DISCLOSURE", "EARNINGS_SURPRISE"]

screening:
  drawdown:
    day1_return_min: -0.09
    day5_return_min: -0.18

  overheat:
    enable: true
    day5_return_max: 0.35

  fundamentals:
    net_income_ttm_min: 0
    per_max: 60

  volatility:
    enable: true
    vol20_exclude_top_pct: 0.10

ranking:
  weights_pct:
    momentum: 25
    flow: 20
    technical: 15
    event: 15
    value: 15
    quality: 10

  constraints:
    momentum_plus_technical_max_pct: 50
    monthly_weight_change_max_pctpt: 5

portfolio:
  holdings:
    target: 20
    min: 15
    max: 20

  allocation:
    cash_target_pct: 0.10
    position_min_pct: 0.04
    position_max_pct: 0.10
    sector_max_pct: 0.25
    turnover_daily_max_pct: 0.20

  weighting:
    method: "TIERED"
    tiers:
      # 합: 5×5% + 10×4.5% + 5×4% = 25% + 45% + 20% = 90%
      # 주식 90% + 현금 10% = 100%
      # 모든 tier가 position_min_pct(4%) 이상 충족
      - { count: 5,  weight_each_pct: 0.05 }
      - { count: 10, weight_each_pct: 0.045 }
      - { count: 5,  weight_each_pct: 0.04 }

  liquidity_caps:
    max_order_to_adtv20_pct: 0.02
    max_participation_per_minute_pct: 0.10

execution:
  order_type: "LIMIT"
  limit_policy:
    buy: "ASK1"
    sell: "BID1"

  # avoid_open_minutes, avoid_close_minutes 삭제됨
  # execution_window.start/end로 통합

  splitting:
    enable: true
    trigger_order_to_adtv20_pct_ge: 0.02
    min_slices: 3
    max_slices: 8
    interval_seconds: 90

  slippage_model:
    segments:
      - { adtv20_min_krw: 5_000_000_000, slippage_pct: 0.0025 }
      - { adtv20_min_krw: 2_000_000_000, slippage_pct: 0.0045 }
      - { adtv20_min_krw: 0,             slippage_pct: 0.0070 }

exit:
  mode: "ATR"

  fixed:
    stop_loss_pct: -0.07
    trail_activate_pct: 0.06
    trail_drawdown_pct: -0.05
    time_stop_days: 10
    partial_take_profit:
      enable: true
      trigger_ret_pct: 0.18
      sell_fraction_pct: 0.50

  atr:
    window_days: 20
    stop_mult: 2.0
    trail_mult: 1.5
    min_stop_pct: 0.07
    min_trail_pct: 0.05

risk_overlay:
  nasdaq_adjust:
    enable: true
    run_time_local: "08:00"
    triggers:
      - { nasdaq_ret_le: -0.030, scale_equity_pct: -0.30 }
      - { nasdaq_ret_le: -0.015, scale_equity_pct: -0.15 }
      - { nasdaq_ret_ge:  0.015, scale_equity_pct:  0.10 }
    clamp:
      min_equity_exposure_pct: 0.60
      max_equity_exposure_pct: 1.00

backtest_costs:
  commission_bps: 1.5
  tax_bps: 23.0
  apply_slippage_model: true
```

---

## 3. Go 타입 정의

### 3.1 파일: `backend/internal/strategyconfig/config.go`

```go
package strategyconfig

import "time"

// Config는 종목 선정 전략의 전체 설정
type Config struct {
	Meta         Meta         `yaml:"meta" json:"meta"`
	Universe     Universe     `yaml:"universe" json:"universe"`
	Signals      Signals      `yaml:"signals" json:"signals"`
	Screening    Screening    `yaml:"screening" json:"screening"`
	Ranking      Ranking      `yaml:"ranking" json:"ranking"`
	Portfolio    Portfolio    `yaml:"portfolio" json:"portfolio"`
	Execution    Execution    `yaml:"execution" json:"execution"`
	Exit         Exit         `yaml:"exit" json:"exit"`
	RiskOverlay  RiskOverlay  `yaml:"risk_overlay" json:"risk_overlay"`
	BacktestCost BacktestCost `yaml:"backtest_costs" json:"backtest_costs"`
}

// Meta 메타 정보
type Meta struct {
	StrategyID        string `yaml:"strategy_id" json:"strategy_id"`
	Version           string `yaml:"version" json:"version"`
	Timezone          string `yaml:"timezone" json:"timezone"`
	DecisionTimeLocal string `yaml:"decision_time_local" json:"decision_time_local"`
	ExecutionWindow   Window `yaml:"execution_window" json:"execution_window"`
}

type Window struct {
	Start string `yaml:"start" json:"start"` // HH:MM
	End   string `yaml:"end" json:"end"`     // HH:MM
}

// Universe S1: 투자 가능 풀
type Universe struct {
	ExcludeKRXFlags []string        `yaml:"exclude_krx_flags" json:"exclude_krx_flags"`
	Filters         UniverseFilters `yaml:"filters" json:"filters"`
}

type UniverseFilters struct {
	MarketcapMinKRW int64  `yaml:"marketcap_min_krw" json:"marketcap_min_krw"`
	ADTV20MinKRW    int64  `yaml:"adtv20_min_krw" json:"adtv20_min_krw"`
	PriceMinKRW     int64  `yaml:"price_min_krw" json:"price_min_krw"`
	IPODaysMin      int    `yaml:"ipo_days_min" json:"ipo_days_min"`
	Spread          Spread `yaml:"spread" json:"spread"`
}

type Spread struct {
	MaxPct  float64 `yaml:"max_pct" json:"max_pct"`
	Formula string  `yaml:"formula" json:"formula"` // 고정: "((ask1-bid1)/((ask1+bid1)/2))"
}

// Signals S2: 6팩터 점수화
type Signals struct {
	Normalization Normalization `yaml:"normalization" json:"normalization"`
	Momentum      Momentum      `yaml:"momentum" json:"momentum"`
	Technical     Technical     `yaml:"technical" json:"technical"`
	Value         Value         `yaml:"value" json:"value"`
	Quality       Quality       `yaml:"quality" json:"quality"`
	Flow          Flow          `yaml:"flow" json:"flow"`
	Event         Event         `yaml:"event" json:"event"`
}

type Normalization struct {
	WinsorizePct  float64 `yaml:"winsorize_pct" json:"winsorize_pct"`
	ZScoreClip    float64 `yaml:"zscore_clip" json:"zscore_clip"`
	ScoreRangeMin int     `yaml:"score_range_min" json:"score_range_min"`
	ScoreRangeMax int     `yaml:"score_range_max" json:"score_range_max"`
	MissingPolicy string  `yaml:"missing_policy" json:"missing_policy"` // NEUTRAL
}

type Momentum struct {
	LookbacksDays []int     `yaml:"lookbacks_days" json:"lookbacks_days"`
	Weights       []float64 `yaml:"weights" json:"weights"` // 합 = 1.0
}

type Technical struct {
	MAShort       int `yaml:"ma_short" json:"ma_short"`
	MALong        int `yaml:"ma_long" json:"ma_long"`
	SlopeWindow   int `yaml:"slope_window" json:"slope_window"`
	High52WWindow int `yaml:"high52w_window" json:"high52w_window"`
}

type Value struct {
	Metrics     []string `yaml:"metrics" json:"metrics"`
	PERScoreCap int      `yaml:"per_score_cap" json:"per_score_cap"`
}

type Quality struct {
	Metrics []string `yaml:"metrics" json:"metrics"`
}

type Flow struct {
	Actors        []string  `yaml:"actors" json:"actors"`
	LookbacksDays []int     `yaml:"lookbacks_days" json:"lookbacks_days"`
	Weights       []float64 `yaml:"weights" json:"weights"` // 합 = 1.0
	NormalizeBy   string    `yaml:"normalize_by" json:"normalize_by"`
}

type Event struct {
	TTLDays int      `yaml:"ttl_days" json:"ttl_days"`
	Types   []string `yaml:"types" json:"types"`
}

// Screening S3: Hard Cut
type Screening struct {
	Drawdown     Drawdown     `yaml:"drawdown" json:"drawdown"`
	Overheat     Overheat     `yaml:"overheat" json:"overheat"`
	Fundamentals Fundamentals `yaml:"fundamentals" json:"fundamentals"`
	Volatility   Volatility   `yaml:"volatility" json:"volatility"`
}

type Drawdown struct {
	Day1ReturnMin float64 `yaml:"day1_return_min" json:"day1_return_min"`
	Day5ReturnMin float64 `yaml:"day5_return_min" json:"day5_return_min"`
}

type Overheat struct {
	Enable        bool    `yaml:"enable" json:"enable"`
	Day5ReturnMax float64 `yaml:"day5_return_max" json:"day5_return_max"`
}

type Fundamentals struct {
	NetIncomeTTMMin int64   `yaml:"net_income_ttm_min" json:"net_income_ttm_min"`
	PERMax          float64 `yaml:"per_max" json:"per_max"`
}

type Volatility struct {
	Enable             bool    `yaml:"enable" json:"enable"`
	Vol20ExcludeTopPct float64 `yaml:"vol20_exclude_top_pct" json:"vol20_exclude_top_pct"`
}

// Ranking S4: 가중치
type Ranking struct {
	WeightsPct  RankingWeights  `yaml:"weights_pct" json:"weights_pct"`
	Constraints RankConstraints `yaml:"constraints" json:"constraints"`
}

type RankingWeights struct {
	Momentum  int `yaml:"momentum" json:"momentum"`
	Flow      int `yaml:"flow" json:"flow"`
	Technical int `yaml:"technical" json:"technical"`
	Event     int `yaml:"event" json:"event"`
	Value     int `yaml:"value" json:"value"`
	Quality   int `yaml:"quality" json:"quality"`
}

// Sum returns the sum of all weights
func (w RankingWeights) Sum() int {
	return w.Momentum + w.Flow + w.Technical + w.Event + w.Value + w.Quality
}

type RankConstraints struct {
	MomentumPlusTechnicalMaxPct int `yaml:"momentum_plus_technical_max_pct" json:"momentum_plus_technical_max_pct"`
	MonthlyWeightChangeMaxPctPt int `yaml:"monthly_weight_change_max_pctpt" json:"monthly_weight_change_max_pctpt"`
}

// Portfolio S5: 포트폴리오 구성
type Portfolio struct {
	Holdings      Holdings      `yaml:"holdings" json:"holdings"`
	Allocation    Allocation    `yaml:"allocation" json:"allocation"`
	Weighting     Weighting     `yaml:"weighting" json:"weighting"`
	LiquidityCaps LiquidityCaps `yaml:"liquidity_caps" json:"liquidity_caps"`
}

type Holdings struct {
	Target int `yaml:"target" json:"target"`
	Min    int `yaml:"min" json:"min"`
	Max    int `yaml:"max" json:"max"`
}

type Allocation struct {
	CashTargetPct       float64 `yaml:"cash_target_pct" json:"cash_target_pct"`
	PositionMinPct      float64 `yaml:"position_min_pct" json:"position_min_pct"`
	PositionMaxPct      float64 `yaml:"position_max_pct" json:"position_max_pct"`
	SectorMaxPct        float64 `yaml:"sector_max_pct" json:"sector_max_pct"`
	TurnoverDailyMaxPct float64 `yaml:"turnover_daily_max_pct" json:"turnover_daily_max_pct"`
}

type Weighting struct {
	Method string `yaml:"method" json:"method"` // TIERED
	Tiers  []Tier `yaml:"tiers" json:"tiers"`
}

type Tier struct {
	Count         int     `yaml:"count" json:"count"`
	WeightEachPct float64 `yaml:"weight_each_pct" json:"weight_each_pct"`
}

// TotalCount returns sum of all tier counts
func (w Weighting) TotalCount() int {
	total := 0
	for _, t := range w.Tiers {
		total += t.Count
	}
	return total
}

// TotalWeightPct returns sum of all tier weights (excluding cash)
func (w Weighting) TotalWeightPct() float64 {
	total := 0.0
	for _, t := range w.Tiers {
		total += float64(t.Count) * t.WeightEachPct
	}
	return total
}

type LiquidityCaps struct {
	MaxOrderToADTV20Pct          float64 `yaml:"max_order_to_adtv20_pct" json:"max_order_to_adtv20_pct"`
	MaxParticipationPerMinutePct float64 `yaml:"max_participation_per_minute_pct" json:"max_participation_per_minute_pct"`
}

// Execution 주문 집행
type Execution struct {
	OrderType     string        `yaml:"order_type" json:"order_type"`
	LimitPolicy   LimitPolicy   `yaml:"limit_policy" json:"limit_policy"`
	Splitting     Splitting     `yaml:"splitting" json:"splitting"`
	SlippageModel SlippageModel `yaml:"slippage_model" json:"slippage_model"`
}

type LimitPolicy struct {
	Buy  string `yaml:"buy" json:"buy"`
	Sell string `yaml:"sell" json:"sell"`
}

type Splitting struct {
	Enable                    bool    `yaml:"enable" json:"enable"`
	TriggerOrderToADTV20PctGe float64 `yaml:"trigger_order_to_adtv20_pct_ge" json:"trigger_order_to_adtv20_pct_ge"`
	MinSlices                 int     `yaml:"min_slices" json:"min_slices"`
	MaxSlices                 int     `yaml:"max_slices" json:"max_slices"`
	IntervalSeconds           int     `yaml:"interval_seconds" json:"interval_seconds"`
}

type SlippageModel struct {
	Segments []SlippageSegment `yaml:"segments" json:"segments"`
}

type SlippageSegment struct {
	ADTV20MinKRW int64   `yaml:"adtv20_min_krw" json:"adtv20_min_krw"`
	SlippagePct  float64 `yaml:"slippage_pct" json:"slippage_pct"`
}

// Exit 청산 전략
type Exit struct {
	Mode  string    `yaml:"mode" json:"mode"` // FIXED | ATR
	Fixed ExitFixed `yaml:"fixed" json:"fixed"`
	ATR   ExitATR   `yaml:"atr" json:"atr"`
}

type ExitFixed struct {
	StopLossPct       float64           `yaml:"stop_loss_pct" json:"stop_loss_pct"`
	TrailActivatePct  float64           `yaml:"trail_activate_pct" json:"trail_activate_pct"`
	TrailDrawdownPct  float64           `yaml:"trail_drawdown_pct" json:"trail_drawdown_pct"`
	TimeStopDays      int               `yaml:"time_stop_days" json:"time_stop_days"`
	PartialTakeProfit PartialTakeProfit `yaml:"partial_take_profit" json:"partial_take_profit"`
}

type PartialTakeProfit struct {
	Enable          bool    `yaml:"enable" json:"enable"`
	TriggerRetPct   float64 `yaml:"trigger_ret_pct" json:"trigger_ret_pct"`
	SellFractionPct float64 `yaml:"sell_fraction_pct" json:"sell_fraction_pct"`
}

type ExitATR struct {
	WindowDays  int     `yaml:"window_days" json:"window_days"`
	StopMult    float64 `yaml:"stop_mult" json:"stop_mult"`
	TrailMult   float64 `yaml:"trail_mult" json:"trail_mult"`
	MinStopPct  float64 `yaml:"min_stop_pct" json:"min_stop_pct"`
	MinTrailPct float64 `yaml:"min_trail_pct" json:"min_trail_pct"`
}

// RiskOverlay 리스크 조정
type RiskOverlay struct {
	NasdaqAdjust NasdaqAdjust `yaml:"nasdaq_adjust" json:"nasdaq_adjust"`
}

type NasdaqAdjust struct {
	Enable       bool            `yaml:"enable" json:"enable"`
	RunTimeLocal string          `yaml:"run_time_local" json:"run_time_local"`
	Triggers     []NasdaqTrigger `yaml:"triggers" json:"triggers"`
	Clamp        NasdaqClamp     `yaml:"clamp" json:"clamp"`
}

type NasdaqTrigger struct {
	NasdaqRetLe    *float64 `yaml:"nasdaq_ret_le,omitempty" json:"nasdaq_ret_le,omitempty"`
	NasdaqRetGe    *float64 `yaml:"nasdaq_ret_ge,omitempty" json:"nasdaq_ret_ge,omitempty"`
	ScaleEquityPct float64  `yaml:"scale_equity_pct" json:"scale_equity_pct"`
}

type NasdaqClamp struct {
	MinEquityExposurePct float64 `yaml:"min_equity_exposure_pct" json:"min_equity_exposure_pct"`
	MaxEquityExposurePct float64 `yaml:"max_equity_exposure_pct" json:"max_equity_exposure_pct"`
}

// BacktestCost 백테스트 비용
type BacktestCost struct {
	CommissionBps      float64 `yaml:"commission_bps" json:"commission_bps"`
	TaxBps             float64 `yaml:"tax_bps" json:"tax_bps"`
	ApplySlippageModel bool    `yaml:"apply_slippage_model" json:"apply_slippage_model"`
}

// DecisionSnapshot 의사결정 스냅샷 (재현성용)
type DecisionSnapshot struct {
	ConfigHash     string    `json:"config_hash"`
	ConfigYAML     string    `json:"config_yaml"`
	StrategyID     string    `json:"strategy_id"`
	GitCommit      string    `json:"git_commit"`
	DataSnapshotID string    `json:"data_snapshot_id"`
	CreatedAt      time.Time `json:"created_at"`
}
```

---

## 4. 로더 및 해시

### 4.1 파일: `backend/internal/strategyconfig/loader.go`

```go
package strategyconfig

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Load reads YAML file and returns Config with raw bytes
func Load(path string) (*Config, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, err
	}

	if err := Validate(&cfg); err != nil {
		return nil, data, err
	}

	return &cfg, data, nil
}

// Hash generates SHA256 hash from Config (canonical JSON)
// 주의: map 대신 struct 사용으로 해시 재현성 보장
func Hash(cfg *Config) (string, error) {
	// Struct → JSON (결정적 순서)
	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(sum[:]), nil
}

// NewDecisionSnapshot creates a snapshot for audit
func NewDecisionSnapshot(cfg *Config, yamlData []byte, gitCommit, dataSnapshotID string) (*DecisionSnapshot, error) {
	hash, err := Hash(cfg)
	if err != nil {
		return nil, err
	}

	return &DecisionSnapshot{
		ConfigHash:     hash,
		ConfigYAML:     string(yamlData),
		StrategyID:     cfg.Meta.StrategyID,
		GitCommit:      gitCommit,
		DataSnapshotID: dataSnapshotID,
		CreatedAt:      time.Now(),
	}, nil
}
```

---

## 5. Validate 및 Warn

### 5.1 파일: `backend/internal/strategyconfig/validate.go`

```go
package strategyconfig

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"
)

// ValidationError 검증 실패 (프로그램 중단)
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Warning 권장 위반 (경고만)
type Warning struct {
	Code    string
	Message string
}

// Validate checks all required constraints
// 실패 시 error 반환 (프로그램 중단)
func Validate(cfg *Config) error {
	// === Meta ===
	if cfg.Meta.StrategyID == "" {
		return ValidationError{"meta.strategy_id", "required"}
	}
	if err := validateHHMM(cfg.Meta.DecisionTimeLocal); err != nil {
		return ValidationError{"meta.decision_time_local", err.Error()}
	}
	if err := validateHHMM(cfg.Meta.ExecutionWindow.Start); err != nil {
		return ValidationError{"meta.execution_window.start", err.Error()}
	}
	if err := validateHHMM(cfg.Meta.ExecutionWindow.End); err != nil {
		return ValidationError{"meta.execution_window.end", err.Error()}
	}

	// execution_window: start < end
	startTime, _ := time.Parse("15:04", cfg.Meta.ExecutionWindow.Start)
	endTime, _ := time.Parse("15:04", cfg.Meta.ExecutionWindow.End)
	if !startTime.Before(endTime) {
		return ValidationError{"meta.execution_window", "start must be before end"}
	}

	// === Universe ===
	if cfg.Universe.Filters.MarketcapMinKRW <= 0 {
		return ValidationError{"universe.filters.marketcap_min_krw", "must be > 0"}
	}
	if cfg.Universe.Filters.ADTV20MinKRW <= 0 {
		return ValidationError{"universe.filters.adtv20_min_krw", "must be > 0"}
	}
	if cfg.Universe.Filters.Spread.MaxPct <= 0 || cfg.Universe.Filters.Spread.MaxPct > 0.05 {
		return ValidationError{"universe.filters.spread.max_pct", "must be in (0, 0.05]"}
	}
	expectedFormula := "((ask1-bid1)/((ask1+bid1)/2))"
	if cfg.Universe.Filters.Spread.Formula != expectedFormula {
		return ValidationError{"universe.filters.spread.formula", fmt.Sprintf("must be '%s'", expectedFormula)}
	}

	// === Signals ===
	// lookbacks_days와 weights 배열 길이 일치 확인
	if len(cfg.Signals.Momentum.LookbacksDays) != len(cfg.Signals.Momentum.Weights) {
		return ValidationError{"signals.momentum", "lookbacks_days length must match weights length"}
	}
	if len(cfg.Signals.Flow.LookbacksDays) != len(cfg.Signals.Flow.Weights) {
		return ValidationError{"signals.flow", "lookbacks_days length must match weights length"}
	}

	if err := validateWeightsSum(cfg.Signals.Momentum.Weights, 1.0, 1e-6); err != nil {
		return ValidationError{"signals.momentum.weights", err.Error()}
	}
	if err := validateWeightsSum(cfg.Signals.Flow.Weights, 1.0, 1e-6); err != nil {
		return ValidationError{"signals.flow.weights", err.Error()}
	}

	// === Ranking ===
	if cfg.Ranking.WeightsPct.Sum() != 100 {
		return ValidationError{"ranking.weights_pct", fmt.Sprintf("must sum to 100, got %d", cfg.Ranking.WeightsPct.Sum())}
	}
	momTech := cfg.Ranking.WeightsPct.Momentum + cfg.Ranking.WeightsPct.Technical
	if momTech > cfg.Ranking.Constraints.MomentumPlusTechnicalMaxPct {
		return ValidationError{"ranking.constraints", fmt.Sprintf("momentum+technical=%d exceeds max=%d", momTech, cfg.Ranking.Constraints.MomentumPlusTechnicalMaxPct)}
	}

	// === Portfolio ===
	h := cfg.Portfolio.Holdings
	if h.Min > h.Target || h.Target > h.Max {
		return ValidationError{"portfolio.holdings", "must satisfy min <= target <= max"}
	}

	a := cfg.Portfolio.Allocation
	if a.PositionMinPct > a.PositionMaxPct {
		return ValidationError{"portfolio.allocation", "position_min_pct must be <= position_max_pct"}
	}
	if a.SectorMaxPct < a.PositionMaxPct {
		return ValidationError{"portfolio.allocation", "sector_max_pct must be >= position_max_pct"}
	}

	// Tier count == holdings.target
	w := cfg.Portfolio.Weighting
	if w.TotalCount() != h.Target {
		return ValidationError{"portfolio.weighting.tiers", fmt.Sprintf("count sum must equal holdings.target=%d, got %d", h.Target, w.TotalCount())}
	}

	// Tier weights + cash ≈ 1.0 (±0.5%)
	totalAlloc := w.TotalWeightPct() + a.CashTargetPct
	if math.Abs(totalAlloc-1.0) > 0.005 {
		return ValidationError{"portfolio", fmt.Sprintf("tiers + cash must equal 1.0±0.005, got %.4f", totalAlloc)}
	}

	// === Execution ===
	if len(cfg.Execution.SlippageModel.Segments) == 0 {
		return ValidationError{"execution.slippage_model.segments", "required"}
	}

	// splitting 제약 조건
	if cfg.Execution.Splitting.Enable {
		if cfg.Execution.Splitting.MinSlices > cfg.Execution.Splitting.MaxSlices {
			return ValidationError{"execution.splitting", "min_slices must be <= max_slices"}
		}
		if cfg.Execution.Splitting.IntervalSeconds <= 0 {
			return ValidationError{"execution.splitting.interval_seconds", "must be > 0"}
		}
	}

	// === Exit ===
	if cfg.Exit.Mode != "FIXED" && cfg.Exit.Mode != "ATR" {
		return ValidationError{"exit.mode", "must be FIXED or ATR"}
	}

	// === RiskOverlay ===
	if cfg.RiskOverlay.NasdaqAdjust.Enable {
		// clamp: min <= max
		clamp := cfg.RiskOverlay.NasdaqAdjust.Clamp
		if clamp.MinEquityExposurePct > clamp.MaxEquityExposurePct {
			return ValidationError{"risk_overlay.nasdaq_adjust.clamp", "min must be <= max"}
		}

		// trigger: 각 트리거에 ret_le 또는 ret_ge 중 하나는 반드시 존재
		for i, trigger := range cfg.RiskOverlay.NasdaqAdjust.Triggers {
			if trigger.NasdaqRetLe == nil && trigger.NasdaqRetGe == nil {
				return ValidationError{
					Field:   fmt.Sprintf("risk_overlay.nasdaq_adjust.triggers[%d]", i),
					Message: "must have nasdaq_ret_le or nasdaq_ret_ge",
				}
			}
		}
	}

	return nil
}

// Warn checks recommended constraints (non-fatal)
func Warn(cfg *Config) []Warning {
	var warnings []Warning

	// ADTV < 10억 경고
	if cfg.Universe.Filters.ADTV20MinKRW < 1_000_000_000 {
		warnings = append(warnings, Warning{
			Code:    "LOW_ADTV",
			Message: "ADTV20 < 10억: 체결/슬리피지 리스크 높음",
		})
	}

	// 슬리피지 낙관적 가정 경고
	for _, seg := range cfg.Execution.SlippageModel.Segments {
		if seg.ADTV20MinKRW == 2_000_000_000 && seg.SlippagePct < 0.0035 {
			warnings = append(warnings, Warning{
				Code:    "OPTIMISTIC_SLIPPAGE",
				Message: "ADTV20 20억 구간 슬리피지 < 0.35%: 낙관적일 수 있음",
			})
		}
	}

	// 과도한 회전율 경고
	if cfg.Portfolio.Allocation.TurnoverDailyMaxPct > 0.25 {
		warnings = append(warnings, Warning{
			Code:    "HIGH_TURNOVER",
			Message: "일 회전율 > 25%: 거래비용 증가 우려",
		})
	}

	return warnings
}

// === Helper Functions ===

func validateHHMM(s string) error {
	re := regexp.MustCompile(`^\d{2}:\d{2}$`)
	if !re.MatchString(s) {
		return errors.New("must be HH:MM format")
	}
	_, err := time.Parse("15:04", s)
	return err
}

func validateWeightsSum(weights []float64, target float64, epsilon float64) error {
	if len(weights) == 0 {
		return errors.New("must not be empty")
	}
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if math.Abs(sum-target) > epsilon {
		return fmt.Errorf("must sum to %.2f, got %.4f", target, sum)
	}
	return nil
}
```

---

## 6. 테스트

### 6.1 파일: `backend/internal/strategyconfig/config_test.go`

```go
package strategyconfig

import (
	"math"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// 테스트용 YAML 경로
	path := "../../../config/strategy/korea_equity_v13.yaml"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("config file not found")
	}

	cfg, yamlData, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 기본 검증
	if cfg.Meta.StrategyID != "korea_equity_v13" {
		t.Errorf("expected strategy_id=korea_equity_v13, got %s", cfg.Meta.StrategyID)
	}

	// ADTV20 수정 확인
	if cfg.Universe.Filters.ADTV20MinKRW != 2_000_000_000 {
		t.Errorf("expected ADTV20=2_000_000_000, got %d", cfg.Universe.Filters.ADTV20MinKRW)
	}

	// 해시 생성
	hash, err := Hash(cfg)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(hash))
	}

	// 동일 설정 → 동일 해시
	hash2, _ := Hash(cfg)
	if hash != hash2 {
		t.Error("hash not deterministic")
	}

	t.Logf("config hash: %s", hash)
	t.Logf("yaml size: %d bytes", len(yamlData))
}

func TestValidateWeights(t *testing.T) {
	// 가중치 합 검증
	cfg := &Config{}
	cfg.Ranking.WeightsPct = RankingWeights{
		Momentum:  25,
		Flow:      20,
		Technical: 15,
		Event:     15,
		Value:     15,
		Quality:   10,
	}

	if cfg.Ranking.WeightsPct.Sum() != 100 {
		t.Errorf("expected 100, got %d", cfg.Ranking.WeightsPct.Sum())
	}
}

func TestValidateTiers(t *testing.T) {
	w := Weighting{
		Method: "TIERED",
		Tiers: []Tier{
			{Count: 5, WeightEachPct: 0.05},
			{Count: 10, WeightEachPct: 0.045},
			{Count: 5, WeightEachPct: 0.04},
		},
	}

	// Count 합 = 20
	if w.TotalCount() != 20 {
		t.Errorf("expected count=20, got %d", w.TotalCount())
	}

	// Weight 합 = 0.25 + 0.45 + 0.20 = 0.90 (현금 0.10 별도)
	expectedWeight := 0.90
	if math.Abs(w.TotalWeightPct()-expectedWeight) > 1e-9 {
		t.Errorf("expected weight=%.2f, got %.4f", expectedWeight, w.TotalWeightPct())
	}
}

func TestWarn(t *testing.T) {
	cfg := &Config{}
	cfg.Universe.Filters.ADTV20MinKRW = 500_000_000 // 5억 (10억 미만)
	cfg.Execution.SlippageModel.Segments = []SlippageSegment{
		{ADTV20MinKRW: 2_000_000_000, SlippagePct: 0.002}, // 0.2% (낙관적)
	}

	warnings := Warn(cfg)
	if len(warnings) < 2 {
		t.Errorf("expected at least 2 warnings, got %d", len(warnings))
	}
}
```

---

## 7. DB 마이그레이션

### 7.1 파일: `backend/migrations/013_create_decision_snapshots.sql`

```sql
-- =====================================================
-- 013_create_decision_snapshots.sql
-- audit 스키마: 의사결정 스냅샷 테이블
-- =====================================================

CREATE TABLE audit.decision_snapshots (
    id              SERIAL PRIMARY KEY,
    config_hash     VARCHAR(64) NOT NULL,
    config_yaml     TEXT NOT NULL,
    strategy_id     VARCHAR(50) NOT NULL,
    git_commit      VARCHAR(40),
    data_snapshot_id VARCHAR(50),
    decision_date   DATE NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT uq_decision_snapshot UNIQUE (strategy_id, decision_date)
);

CREATE INDEX idx_decision_snapshots_hash ON audit.decision_snapshots(config_hash);
CREATE INDEX idx_decision_snapshots_date ON audit.decision_snapshots(decision_date);

COMMENT ON TABLE audit.decision_snapshots IS '의사결정 스냅샷 (설정 재현성 보장)';
COMMENT ON COLUMN audit.decision_snapshots.config_hash IS 'SHA256(canonical JSON)';
COMMENT ON COLUMN audit.decision_snapshots.config_yaml IS '원본 YAML 전문';

-- =====================================================
-- 검증: SELECT * FROM audit.decision_snapshots LIMIT 1;
-- =====================================================
```

---

## 8. 체크리스트

### 8.1 구현 순서

| # | 작업 | 파일 | 의존성 |
|---|------|------|--------|
| 1 | YAML 수정 | `config/strategy/korea_equity_v13.yaml` | - |
| 2 | 타입 정의 | `internal/strategyconfig/config.go` | - |
| 3 | 로더/해시 | `internal/strategyconfig/loader.go` | #2 |
| 4 | Validate/Warn | `internal/strategyconfig/validate.go` | #2 |
| 5 | 테스트 | `internal/strategyconfig/config_test.go` | #1~4 |
| 6 | DB 마이그레이션 | `migrations/013_create_decision_snapshots.sql` | - |
| 7 | 문서 업데이트 | `docs/guide/strategy/stock-selection.md` | #1 |

### 8.2 Validate 규칙 요약

| 필드 | 규칙 |
|------|------|
| `meta.strategy_id` | 필수 |
| `meta.*_time` | HH:MM 형식 |
| `meta.execution_window` | start < end |
| `universe.filters.adtv20_min_krw` | > 0 |
| `universe.filters.spread.formula` | 고정값 일치 |
| `signals.momentum` | lookbacks_days 길이 = weights 길이 |
| `signals.momentum.weights` | 합 = 1.0 |
| `signals.flow` | lookbacks_days 길이 = weights 길이 |
| `signals.flow.weights` | 합 = 1.0 |
| `ranking.weights_pct` | 합 = 100 |
| `ranking.constraints` | momentum+technical ≤ max |
| `portfolio.holdings` | min ≤ target ≤ max |
| `portfolio.allocation` | position_min ≤ position_max |
| `portfolio.allocation` | sector_max ≥ position_max |
| `portfolio.weighting.tiers` | count 합 = holdings.target |
| `portfolio` | tiers + cash = 1.0 (±0.5%) |
| `execution.slippage_model` | segments 필수 |
| `execution.splitting` | min_slices ≤ max_slices |
| `execution.splitting.interval_seconds` | > 0 |
| `exit.mode` | FIXED \| ATR |
| `risk_overlay.nasdaq_adjust.clamp` | min ≤ max |
| `risk_overlay.nasdaq_adjust.triggers[]` | ret_le 또는 ret_ge 필수 |

### 8.3 Warn 규칙 요약

| 코드 | 조건 | 메시지 |
|------|------|--------|
| `LOW_ADTV` | ADTV20 < 10억 | 체결/슬리피지 리스크 |
| `OPTIMISTIC_SLIPPAGE` | 20억 구간 < 0.35% | 낙관적 가정 |
| `HIGH_TURNOVER` | 회전율 > 25% | 거래비용 증가 |

---

## 9. 사용 예시

```go
package main

import (
	"log"
	"aegis-v13/backend/internal/strategyconfig"
)

func main() {
	// 1. 설정 로드
	cfg, yamlData, err := strategyconfig.Load("config/strategy/korea_equity_v13.yaml")
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	// 2. 경고 확인
	warnings := strategyconfig.Warn(cfg)
	for _, w := range warnings {
		log.Printf("[WARN] %s: %s", w.Code, w.Message)
	}

	// 3. 스냅샷 생성
	snapshot, err := strategyconfig.NewDecisionSnapshot(
		cfg,
		yamlData,
		"abc123def", // git commit
		"data_20240115",
	)
	if err != nil {
		log.Fatalf("snapshot creation failed: %v", err)
	}

	log.Printf("Config Hash: %s", snapshot.ConfigHash)
	log.Printf("Strategy ID: %s", snapshot.StrategyID)
}
```

---

이 문서대로 구현하면 됩니다.
