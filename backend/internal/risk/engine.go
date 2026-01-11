package risk

import (
	"errors"
	"fmt"
	"time"
)

// =============================================================================
// RiskEngine Interface - 순수 계산기
// =============================================================================

// Engine 리스크 엔진 (순수 계산기)
// ⭐ SSOT: 데이터 수집/포트폴리오 구성/한도정책은 상위 레이어(S6/S7)에서 조립
// internal/risk는 순수 계산만 담당
type Engine struct{}

// NewEngine 새 리스크 엔진 생성
func NewEngine() *Engine {
	return &Engine{}
}

// =============================================================================
// VaR/CVaR Calculation (Pure)
// =============================================================================

// VaR Historical VaR 계산
// returns: 일별 수익률 (양수=이익, 음수=손실)
// confidence: 신뢰수준 (예: 0.95, 0.99)
// 반환: VaRResult (손실을 양수로 표현)
func (e *Engine) VaR(returns []float64, confidence float64) VaRResult {
	return CalculateVaR(returns, confidence)
}

// CVaR Conditional VaR (Expected Shortfall) 계산
func (e *Engine) CVaR(returns []float64, confidence float64) VaRResult {
	return CalculateVaR(returns, confidence) // CVaR도 함께 계산됨
}

// ParametricVaR 정규분포 가정 VaR 계산
func (e *Engine) ParametricVaR(mean, stdDev, confidence float64) VaRResult {
	return CalculateParametricVaR(mean, stdDev, confidence)
}

// =============================================================================
// Monte Carlo Simulation (Pure)
// =============================================================================

var (
	ErrInsufficientData = errors.New("insufficient data for simulation")
	ErrInvalidConfig    = errors.New("invalid configuration")
)

// MonteCarlo 포트폴리오 Monte Carlo 시뮬레이션
// input: 포트폴리오 수익률 시계열 (상위 레이어에서 조립해서 전달)
// config: 시뮬레이션 설정
// 반환: MonteCarloResult
func (e *Engine) MonteCarlo(input PortfolioReturns, config MonteCarloConfig) (*MonteCarloResult, error) {
	// Fail-closed: 최소 샘플 수 체크
	if len(input.Returns) < config.MinSamples {
		return nil, fmt.Errorf("%w: got %d, need %d",
			ErrInsufficientData, len(input.Returns), config.MinSamples)
	}

	// 시뮬레이터 생성 및 실행
	simulator := NewMonteCarloSimulator(config)
	result, err := simulator.SimulateReturns(input.Returns)
	if err != nil {
		return nil, err
	}

	result.InputSampleCount = len(input.Returns)
	return result, nil
}

// =============================================================================
// Risk Check (S6 Gate용 - 순수 계산)
// =============================================================================

// CheckLimits 리스크 한도 체크 (순수 계산)
// input: 포트폴리오 수익률 (상위 레이어에서 조립)
// limits: 리스크 한도
// 반환: RiskCheckResult
func (e *Engine) CheckLimits(input RiskCheckInput, limits RiskLimits) *RiskCheckResult {
	result := &RiskCheckResult{
		Passed:       true,
		MaxVaRLimit:  limits.MaxVaR95,
		MaxCVaRLimit: limits.MaxCVaR95,
		Violations:   make([]string, 0),
		CheckedAt:    time.Now(),
	}

	// Historical VaR 계산
	varResult := CalculateVaR(input.PortfolioReturns, 0.95)
	result.VaR95 = varResult.VaR
	result.CVaR95 = varResult.CVaR

	// VaR 한도 체크
	if varResult.VaR > limits.MaxVaR95 {
		result.Passed = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("VaR95 %.4f exceeds limit %.4f", varResult.VaR, limits.MaxVaR95))
	}

	// CVaR 한도 체크
	if varResult.CVaR > limits.MaxCVaR95 {
		result.Passed = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("CVaR95 %.4f exceeds limit %.4f", varResult.CVaR, limits.MaxCVaR95))
	}

	return result
}

// =============================================================================
// Stress Test (순수 계산)
// =============================================================================

// StressTest 스트레스 시나리오 테스트
// weights: 종목별 비중 map[code]weight
// scenarios: 스트레스 시나리오
// 반환: 시나리오별 포트폴리오 손실
func (e *Engine) StressTest(weights map[string]float64, scenarios []Scenario) map[string]float64 {
	results := make(map[string]float64)

	for _, scenario := range scenarios {
		var portfolioLoss float64

		for code, weight := range weights {
			shock, exists := scenario.Shocks[code]
			if !exists {
				// 전체 시장 충격 확인
				shock, exists = scenario.Shocks["*"]
				if !exists {
					continue
				}
			}
			portfolioLoss += weight * shock
		}

		results[scenario.Name] = portfolioLoss
	}

	return results
}

// =============================================================================
// Utility Functions
// =============================================================================

// CalculatePortfolioReturns 종목별 수익률에서 포트폴리오 수익률 계산
// weights: 종목별 비중 map[code]weight
// assetReturns: 종목별 수익률 시계열 map[code][]returns
// 반환: 포트폴리오 수익률 시계열
func CalculatePortfolioReturns(weights map[string]float64, assetReturns map[string][]float64) []float64 {
	// 최소 데이터 길이 찾기
	minLen := -1
	for _, returns := range assetReturns {
		if minLen == -1 || len(returns) < minLen {
			minLen = len(returns)
		}
	}

	if minLen <= 0 {
		return nil
	}

	// 비중 가중 수익률
	portfolioReturns := make([]float64, minLen)
	for i := 0; i < minLen; i++ {
		var dayReturn float64
		for code, weight := range weights {
			if returns, ok := assetReturns[code]; ok && i < len(returns) {
				dayReturn += weight * returns[i]
			}
		}
		portfolioReturns[i] = dayReturn
	}

	return portfolioReturns
}

// ValidateConfig 설정 유효성 검사
func ValidateConfig(config MonteCarloConfig) error {
	if config.NumSimulations <= 0 {
		return fmt.Errorf("%w: NumSimulations must be > 0", ErrInvalidConfig)
	}
	if config.HoldingPeriod <= 0 {
		return fmt.Errorf("%w: HoldingPeriod must be > 0", ErrInvalidConfig)
	}
	if config.MinSamples <= 0 {
		return fmt.Errorf("%w: MinSamples must be > 0", ErrInvalidConfig)
	}
	if len(config.ConfidenceLevels) == 0 {
		return fmt.Errorf("%w: ConfidenceLevels cannot be empty", ErrInvalidConfig)
	}
	for _, cl := range config.ConfidenceLevels {
		if cl <= 0 || cl >= 1 {
			return fmt.Errorf("%w: ConfidenceLevel must be between 0 and 1", ErrInvalidConfig)
		}
	}
	return nil
}
