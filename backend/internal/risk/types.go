package risk

import "time"

// MonteCarloConfig Monte Carlo 시뮬레이션 설정
// SSOT: config/strategy/korea_equity_v13.yaml (향후 추가 예정)
type MonteCarloConfig struct {
	NumSimulations   int       // 시뮬레이션 횟수 (기본: 10000)
	HoldingPeriod    int       // 보유 기간 (일), 기본: 5
	ConfidenceLevels []float64 // 신뢰 수준 [0.95, 0.99]
	Method           string    // "historical" | "parametric"
	LookbackDays     int       // 과거 데이터 일수 (기본: 200)
	Seed             int64     // 재현성용 시드 (0=랜덤)
}

// DefaultMonteCarloConfig 기본 Monte Carlo 설정
func DefaultMonteCarloConfig() MonteCarloConfig {
	return MonteCarloConfig{
		NumSimulations:   10000,
		HoldingPeriod:    5,
		ConfidenceLevels: []float64{0.95, 0.99},
		Method:           "historical",
		LookbackDays:     200,
		Seed:             0,
	}
}

// MonteCarloResult Monte Carlo 시뮬레이션 결과
type MonteCarloResult struct {
	RunID       string           `json:"run_id"`
	RunDate     time.Time        `json:"run_date"`
	Config      MonteCarloConfig `json:"config"` // 재현성용
	MeanReturn  float64          `json:"mean_return"`
	StdDev      float64          `json:"std_dev"`
	VaR95       float64          `json:"var_95"`  // 95% VaR (손실, 양수)
	VaR99       float64          `json:"var_99"`  // 99% VaR
	CVaR95      float64          `json:"cvar_95"` // 95% CVaR (Expected Shortfall)
	CVaR99      float64          `json:"cvar_99"` // 99% CVaR
	Percentiles map[int]float64  `json:"percentiles"` // 1, 5, 10, 25, 50, 75, 90, 95, 99
	CreatedAt   time.Time        `json:"created_at"`
}

// VaRResult VaR 계산 결과
type VaRResult struct {
	Confidence float64 `json:"confidence"` // 신뢰 수준 (0.95, 0.99)
	VaR        float64 `json:"var"`        // Value at Risk (손실, 양수)
	CVaR       float64 `json:"cvar"`       // Conditional VaR (Expected Shortfall)
}

// Holding 포트폴리오 보유 종목
type Holding struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`       // 비중 (0.0 ~ 1.0)
	MarketValue int64   `json:"market_value"` // 평가금액 (원)
}

// RiskCheckResult S6 리스크 게이트 체크 결과
type RiskCheckResult struct {
	Passed     bool            `json:"passed"`
	Violations []RiskViolation `json:"violations"`
	Metrics    RiskMetrics     `json:"metrics"`
	CheckedAt  time.Time       `json:"checked_at"`
}

// RiskViolation 리스크 위반 항목
type RiskViolation struct {
	Type     string  `json:"type"`     // "VAR_LIMIT", "EXPOSURE_LIMIT", "LIQUIDITY_LIMIT"
	Limit    float64 `json:"limit"`    // 한도
	Actual   float64 `json:"actual"`   // 실제값
	Severity string  `json:"severity"` // "WARNING", "BLOCK"
	Message  string  `json:"message"`
}

// RiskMetrics 리스크 지표
type RiskMetrics struct {
	PortfolioVaR95     float64 `json:"portfolio_var_95"`
	PortfolioVaR99     float64 `json:"portfolio_var_99"`
	MaxSingleExposure  float64 `json:"max_single_exposure"`  // 최대 단일 종목 비중
	MaxSectorExposure  float64 `json:"max_sector_exposure"`  // 최대 섹터 비중
	ConcentrationRatio float64 `json:"concentration_ratio"`  // 상위 5종목 비중 합
	LiquidityScore     float64 `json:"liquidity_score"`      // 유동성 점수 (0~1)
}

// RiskLimits 리스크 한도 설정
type RiskLimits struct {
	MaxVaR95          float64 `yaml:"max_var_95"`          // 최대 95% VaR (예: 0.05 = 5%)
	MaxVaR99          float64 `yaml:"max_var_99"`          // 최대 99% VaR
	MaxSingleExposure float64 `yaml:"max_single_exposure"` // 단일 종목 최대 비중
	MaxSectorExposure float64 `yaml:"max_sector_exposure"` // 섹터 최대 비중
	MaxConcentration  float64 `yaml:"max_concentration"`   // 상위 5종목 최대 비중 합
	MinLiquidityScore float64 `yaml:"min_liquidity_score"` // 최소 유동성 점수
}

// DefaultRiskLimits 기본 리스크 한도
func DefaultRiskLimits() RiskLimits {
	return RiskLimits{
		MaxVaR95:          0.05, // 5%
		MaxVaR99:          0.08, // 8%
		MaxSingleExposure: 0.15, // 15%
		MaxSectorExposure: 0.30, // 30%
		MaxConcentration:  0.50, // 50%
		MinLiquidityScore: 0.60, // 60%
	}
}

// =============================================================================
// Forecast Validation Types (S7 Audit)
// =============================================================================

// ValidationResult 예측 검증 결과
type ValidationResult struct {
	EventID      int64     `json:"event_id"`
	ModelVersion string    `json:"model_version"`
	Code         string    `json:"code"`
	EventType    string    `json:"event_type"`
	PredictedRet float64   `json:"predicted_ret"` // 예측 수익률
	ActualRet    float64   `json:"actual_ret"`    // 실제 수익률
	Error        float64   `json:"error"`         // 예측 - 실제
	AbsError     float64   `json:"abs_error"`     // |예측 - 실제|
	DirectionHit bool      `json:"direction_hit"` // 방향성 적중
	ValidatedAt  time.Time `json:"validated_at"`
}

// AccuracyReport 정확도 리포트
type AccuracyReport struct {
	ModelVersion string    `json:"model_version"`
	Level        string    `json:"level"`       // "ALL", "EVENT_TYPE", "CODE"
	Key          string    `json:"key"`         // 레벨별 키 값
	EventType    string    `json:"event_type"`  // 이벤트 유형 (레벨이 EVENT_TYPE일 때)
	SampleCount  int       `json:"sample_count"`
	MAE          float64   `json:"mae"`       // Mean Absolute Error
	RMSE         float64   `json:"rmse"`      // Root Mean Squared Error
	HitRate      float64   `json:"hit_rate"`  // 방향성 적중률
	MeanError    float64   `json:"mean_error"` // 편향 (bias)
	UpdatedAt    time.Time `json:"updated_at"`
}

// CalibrationBin 캘리브레이션 빈 (신뢰도 다이어그램용)
type CalibrationBin struct {
	ModelVersion string  `json:"model_version"`
	HorizonDays  int     `json:"horizon_days"` // 예측 기간 (일)
	Bin          int     `json:"bin"`          // 빈 인덱스 (0~N)
	SampleCount  int     `json:"sample_count"`
	AvgPredicted float64 `json:"avg_predicted"` // 평균 예측값
	AvgActual    float64 `json:"avg_actual"`    // 평균 실제값
	HitRate      float64 `json:"hit_rate"`      // 빈 내 적중률
}

// =============================================================================
// Backward Compatibility (Legacy API)
// =============================================================================

// MonteCarloMethod 시뮬레이션 방법 (legacy)
type MonteCarloMethod string

const (
	MethodHistoricalBootstrap MonteCarloMethod = "historical"
	MethodParametricNormal    MonteCarloMethod = "normal"
	MethodParametricT         MonteCarloMethod = "t"
)

// CalculatePortfolioReturns 종목별 수익률과 비중에서 포트폴리오 수익률 계산
func CalculatePortfolioReturns(weights map[string]float64, assetReturns map[string][]float64) []float64 {
	if len(weights) == 0 || len(assetReturns) == 0 {
		return nil
	}

	// 최소 길이 찾기
	minLen := 0
	for code, weight := range weights {
		if returns, ok := assetReturns[code]; ok && weight > 0 {
			if minLen == 0 || len(returns) < minLen {
				minLen = len(returns)
			}
		}
	}

	if minLen == 0 {
		return nil
	}

	// 포트폴리오 수익률 계산
	portfolioReturns := make([]float64, minLen)
	for i := 0; i < minLen; i++ {
		dayReturn := 0.0
		for code, weight := range weights {
			if returns, ok := assetReturns[code]; ok && i < len(returns) {
				dayReturn += weight * returns[i]
			}
		}
		portfolioReturns[i] = dayReturn
	}

	return portfolioReturns
}
