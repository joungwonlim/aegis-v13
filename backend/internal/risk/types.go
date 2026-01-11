package risk

import "time"

// =============================================================================
// Return Type & Convention
// =============================================================================

// ReturnType 수익률 계산 방식
type ReturnType string

const (
	ReturnSimple ReturnType = "simple" // (P1 - P0) / P0
	ReturnLog    ReturnType = "log"    // ln(P1 / P0)
)

// VaRConvention VaR 부호 규약
// ⭐ SSOT: Loss를 양수로 표현 (VaR=0.05 → 5% 손실 가능)
// 전체 시스템에서 이 규약을 일관되게 사용
const VaRConvention = "loss_positive"

// =============================================================================
// VaR/CVaR Types
// =============================================================================

// VaRResult VaR 계산 결과
// ⭐ SSOT: VaR/CVaR는 손실을 양수로 표현
// - VaR=0.05 → 95% 신뢰수준에서 최대 5% 손실 가능
// - CVaR=0.07 → 5% tail에서 평균 7% 손실 예상
type VaRResult struct {
	Confidence float64 `json:"confidence"` // 신뢰수준 (예: 0.95, 0.99)
	VaR        float64 `json:"var"`        // Value at Risk (손실, 양수)
	CVaR       float64 `json:"cvar"`       // Conditional VaR (Expected Shortfall, 양수)
}

// =============================================================================
// Monte Carlo Types
// =============================================================================

// SimulationMode 시뮬레이션 모드
type SimulationMode string

const (
	// ModePortfolioUnivariate 포트폴리오 단변량 (빠름, S6/S7 공용)
	// 포트폴리오 수익률 = sum(weight_i * return_i)을 단일 시계열로 만들어 MC
	ModePortfolioUnivariate SimulationMode = "portfolio_univariate"
	// ModeAssetMultivariate 자산별 다변량 (정교함, S7 권장)
	// 종목별 수익률 매트릭스로 공분산/상관 반영
	ModeAssetMultivariate SimulationMode = "asset_multivariate"
)

// MonteCarloMethod 시뮬레이션 방법
type MonteCarloMethod string

const (
	MethodHistoricalBootstrap MonteCarloMethod = "historical_bootstrap" // 과거 수익률 Bootstrap
	MethodParametricNormal    MonteCarloMethod = "parametric_normal"    // 정규분포 가정
	MethodParametricT         MonteCarloMethod = "parametric_t"         // t-분포 가정 (fat tail)
)

// MonteCarloConfig Monte Carlo 시뮬레이션 설정
// ⭐ SSOT: 재현성을 위해 모든 설정을 명시적으로 기록
type MonteCarloConfig struct {
	Mode             SimulationMode   `json:"mode"`              // 단변량/다변량
	ReturnType       ReturnType       `json:"return_type"`       // simple/log
	NumSimulations   int              `json:"num_simulations"`   // 시뮬레이션 횟수 (기본: 10000)
	HoldingPeriod    int              `json:"holding_period"`    // 보유 기간 (일, 기본: 5)
	ConfidenceLevels []float64        `json:"confidence_levels"` // 신뢰수준 [0.95, 0.99]
	Method           MonteCarloMethod `json:"method"`            // bootstrap/normal/t
	LookbackDays     int              `json:"lookback_days"`     // 과거 데이터 조회 기간 (기본: 200)
	Seed             int64            `json:"seed"`              // 재현성용 시드 (0=랜덤)
	MinSamples       int              `json:"min_samples"`       // 최소 샘플 수 (fail-closed, 기본: 30)
}

// DefaultMonteCarloConfig 기본 Monte Carlo 설정
func DefaultMonteCarloConfig() MonteCarloConfig {
	return MonteCarloConfig{
		Mode:             ModePortfolioUnivariate,
		ReturnType:       ReturnSimple,
		NumSimulations:   10000,
		HoldingPeriod:    5,
		ConfidenceLevels: []float64{0.95, 0.99},
		Method:           MethodHistoricalBootstrap,
		LookbackDays:     200,
		Seed:             0,  // 랜덤
		MinSamples:       30, // fail-closed: 30개 미만이면 실패
	}
}

// MonteCarloResult Monte Carlo 시뮬레이션 결과
// ⭐ SSOT: 재현성을 위해 Config 포함, 추적을 위해 snapshot_id 포함
type MonteCarloResult struct {
	RunID              string           `json:"run_id"`               // 실행 고유 ID
	RunDate            time.Time        `json:"run_date"`             // 실행 날짜
	DecisionSnapshotID *int64           `json:"decision_snapshot_id"` // 의사결정 스냅샷 ID (추적용)
	Config             MonteCarloConfig `json:"config"`               // 재현성용 설정 기록
	InputSampleCount   int              `json:"input_sample_count"`   // 입력 샘플 수
	MeanReturn         float64          `json:"mean_return"`          // 평균 수익률
	StdDev             float64          `json:"std_dev"`              // 표준편차
	VaR95              float64          `json:"var_95"`               // 95% VaR (손실, 양수)
	VaR99              float64          `json:"var_99"`               // 99% VaR (손실, 양수)
	CVaR95             float64          `json:"cvar_95"`              // 95% CVaR (손실, 양수)
	CVaR99             float64          `json:"cvar_99"`              // 99% CVaR (손실, 양수)
	Percentiles        map[int]float64  `json:"percentiles"`          // 백분위수 (1, 5, 10, 25, 50, 75, 90, 95, 99)
	CreatedAt          time.Time        `json:"created_at"`
}

// =============================================================================
// Input Types (for pure calculation)
// =============================================================================

// ReturnSeries 수익률 시계열 (순수 입력)
type ReturnSeries struct {
	Returns    []float64  `json:"returns"`     // 일별 수익률
	ReturnType ReturnType `json:"return_type"` // simple/log
}

// PortfolioReturns 포트폴리오 수익률 입력
type PortfolioReturns struct {
	Returns    []float64  `json:"returns"`     // 포트폴리오 일별 수익률 (가중합)
	ReturnType ReturnType `json:"return_type"` // simple/log
	SampleSize int        `json:"sample_size"` // 샘플 크기
}

// =============================================================================
// Risk Check Types (S6 Gate용 - 데이터는 S6에서 조립)
// =============================================================================

// RiskCheckInput S6 리스크 체크 입력 (S6에서 조립해서 전달)
type RiskCheckInput struct {
	PortfolioReturns []float64 `json:"portfolio_returns"` // 포트폴리오 수익률 시계열
	ReturnType       ReturnType `json:"return_type"`
}

// RiskCheckResult 리스크 체크 결과
type RiskCheckResult struct {
	Passed       bool      `json:"passed"`         // 통과 여부
	VaR95        float64   `json:"var_95"`         // 95% VaR (손실, 양수)
	CVaR95       float64   `json:"cvar_95"`        // 95% CVaR (손실, 양수)
	MaxVaRLimit  float64   `json:"max_var_limit"`  // VaR 한도
	MaxCVaRLimit float64   `json:"max_cvar_limit"` // CVaR 한도
	Violations   []string  `json:"violations"`     // 위반 항목
	CheckedAt    time.Time `json:"checked_at"`
}

// RiskLimits 리스크 한도 설정
type RiskLimits struct {
	MaxVaR95    float64 `json:"max_var_95"`    // 최대 95% VaR (예: 0.05 = 5%)
	MaxCVaR95   float64 `json:"max_cvar_95"`   // 최대 95% CVaR
	MaxDrawdown float64 `json:"max_drawdown"`  // 최대 MDD
}

// DefaultRiskLimits 기본 리스크 한도
func DefaultRiskLimits() RiskLimits {
	return RiskLimits{
		MaxVaR95:    0.05, // 5% VaR
		MaxCVaR95:   0.07, // 7% CVaR
		MaxDrawdown: 0.15, // 15% MDD
	}
}

// =============================================================================
// Forecast Validation Types
// =============================================================================

// ValidationResult Forecast 검증 결과
// ⭐ 모델 버전을 포함하여 다중 모델 비교 가능
type ValidationResult struct {
	EventID      int64     `json:"event_id"`
	ModelVersion string    `json:"model_version"` // 모델/파라미터 버전
	Code         string    `json:"code"`
	EventType    string    `json:"event_type"`
	PredictedRet float64   `json:"predicted_ret"` // 예측 수익률
	ActualRet    float64   `json:"actual_ret"`    // 실제 수익률
	Error        float64   `json:"error"`         // 오차 (실제 - 예측)
	AbsError     float64   `json:"abs_error"`     // 절대 오차
	DirectionHit bool      `json:"direction_hit"` // 방향성 적중
	ValidatedAt  time.Time `json:"validated_at"`
}

// AccuracyReport 정확도 리포트
type AccuracyReport struct {
	ModelVersion string    `json:"model_version"` // 모델 버전
	Level        string    `json:"level"`         // SYMBOL/SECTOR/BUCKET/MARKET
	Key          string    `json:"key"`           // 키 값
	EventType    string    `json:"event_type"`    // 이벤트 타입
	SampleCount  int       `json:"sample_count"`  // 샘플 수
	MAE          float64   `json:"mae"`           // Mean Absolute Error
	RMSE         float64   `json:"rmse"`          // Root Mean Squared Error
	HitRate      float64   `json:"hit_rate"`      // 방향성 적중률
	MeanError    float64   `json:"mean_error"`    // 평균 오차 (편향)
	UpdatedAt    time.Time `json:"updated_at"`
}

// CalibrationBin 캘리브레이션 빈 (신뢰도 다이어그램용)
type CalibrationBin struct {
	ModelVersion string  `json:"model_version"`
	HorizonDays  int     `json:"horizon_days"` // 1, 5 등
	Bin          int     `json:"bin"`          // 0-9 (10개 빈)
	SampleCount  int     `json:"sample_count"`
	AvgPredicted float64 `json:"avg_predicted"` // 평균 예측값
	AvgActual    float64 `json:"avg_actual"`    // 평균 실측값
	HitRate      float64 `json:"hit_rate"`      // 적중률
}
