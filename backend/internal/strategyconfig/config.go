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
