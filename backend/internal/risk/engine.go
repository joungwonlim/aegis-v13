package risk

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Engine 리스크 엔진 (S6/S7 공용)
// SSOT: 리스크 계산 로직의 단일 진실원천
type Engine struct {
	limits    RiskLimits
	mcConfig  MonteCarloConfig
	simulator *MonteCarloSimulator
	logger    *logger.Logger
}

// NewEngine 새 리스크 엔진 생성
func NewEngine(limits RiskLimits, mcConfig MonteCarloConfig, logger *logger.Logger) *Engine {
	return &Engine{
		limits:    limits,
		mcConfig:  mcConfig,
		simulator: NewMonteCarloSimulator(mcConfig),
		logger:    logger,
	}
}

// NewDefaultEngine 기본 설정으로 리스크 엔진 생성
func NewDefaultEngine(logger *logger.Logger) *Engine {
	return NewEngine(
		DefaultRiskLimits(),
		DefaultMonteCarloConfig(),
		logger,
	)
}

// CalculateVaR 수익률 배열에서 VaR 계산 (S6/S7 공용)
func (e *Engine) CalculateVaR(returns []float64, confidence float64) VaRResult {
	return CalculateVaR(returns, confidence)
}

// SimulatePortfolio Monte Carlo 시뮬레이션 실행 (S7용)
func (e *Engine) SimulatePortfolio(
	ctx context.Context,
	historicalReturns map[string][]float64,
	weights map[string]float64,
) (*MonteCarloResult, error) {
	return e.simulator.Simulate(ctx, historicalReturns, weights)
}

// SimulateSimple 단순 포트폴리오 시뮬레이션 (S7용)
func (e *Engine) SimulateSimple(
	ctx context.Context,
	portfolioReturns []float64,
) (*MonteCarloResult, error) {
	return e.simulator.SimulateSimple(ctx, portfolioReturns)
}

// CheckRiskLimits 리스크 한도 체크 (S6 게이트용)
func (e *Engine) CheckRiskLimits(
	ctx context.Context,
	holdings []Holding,
	historicalReturns map[string][]float64,
) (*RiskCheckResult, error) {
	violations := make([]RiskViolation, 0)

	// 1. 비중 맵 생성
	weights := make(map[string]float64)
	for _, h := range holdings {
		weights[h.Code] = h.Weight
	}

	// 2. 포트폴리오 VaR 계산
	portfolioVaR := e.calculatePortfolioVaR(historicalReturns, weights)

	// 3. 익스포저 계산
	metrics := e.calculateMetrics(holdings)
	metrics.PortfolioVaR95 = portfolioVaR.VaR95
	metrics.PortfolioVaR99 = portfolioVaR.VaR99

	// 4. 한도 체크
	// VaR 95 한도
	if metrics.PortfolioVaR95 > e.limits.MaxVaR95 {
		violations = append(violations, RiskViolation{
			Type:     "VAR_95_LIMIT",
			Limit:    e.limits.MaxVaR95,
			Actual:   metrics.PortfolioVaR95,
			Severity: "BLOCK",
			Message:  fmt.Sprintf("95%% VaR %.2f%% exceeds limit %.2f%%", metrics.PortfolioVaR95*100, e.limits.MaxVaR95*100),
		})
	}

	// VaR 99 한도
	if metrics.PortfolioVaR99 > e.limits.MaxVaR99 {
		violations = append(violations, RiskViolation{
			Type:     "VAR_99_LIMIT",
			Limit:    e.limits.MaxVaR99,
			Actual:   metrics.PortfolioVaR99,
			Severity: "BLOCK",
			Message:  fmt.Sprintf("99%% VaR %.2f%% exceeds limit %.2f%%", metrics.PortfolioVaR99*100, e.limits.MaxVaR99*100),
		})
	}

	// 단일 종목 비중 한도
	if metrics.MaxSingleExposure > e.limits.MaxSingleExposure {
		violations = append(violations, RiskViolation{
			Type:     "SINGLE_EXPOSURE_LIMIT",
			Limit:    e.limits.MaxSingleExposure,
			Actual:   metrics.MaxSingleExposure,
			Severity: "BLOCK",
			Message:  fmt.Sprintf("Max single exposure %.2f%% exceeds limit %.2f%%", metrics.MaxSingleExposure*100, e.limits.MaxSingleExposure*100),
		})
	}

	// 집중도 한도
	if metrics.ConcentrationRatio > e.limits.MaxConcentration {
		violations = append(violations, RiskViolation{
			Type:     "CONCENTRATION_LIMIT",
			Limit:    e.limits.MaxConcentration,
			Actual:   metrics.ConcentrationRatio,
			Severity: "WARNING",
			Message:  fmt.Sprintf("Top 5 concentration %.2f%% exceeds limit %.2f%%", metrics.ConcentrationRatio*100, e.limits.MaxConcentration*100),
		})
	}

	// 유동성 점수
	if metrics.LiquidityScore < e.limits.MinLiquidityScore {
		violations = append(violations, RiskViolation{
			Type:     "LIQUIDITY_LIMIT",
			Limit:    e.limits.MinLiquidityScore,
			Actual:   metrics.LiquidityScore,
			Severity: "WARNING",
			Message:  fmt.Sprintf("Liquidity score %.2f below minimum %.2f", metrics.LiquidityScore, e.limits.MinLiquidityScore),
		})
	}

	// BLOCK 위반이 있는지 확인
	passed := true
	for _, v := range violations {
		if v.Severity == "BLOCK" {
			passed = false
			break
		}
	}

	result := &RiskCheckResult{
		Passed:     passed,
		Violations: violations,
		Metrics:    metrics,
		CheckedAt:  time.Now(),
	}

	// 로깅
	if !passed {
		e.logger.WithFields(map[string]interface{}{
			"violations": len(violations),
			"var_95":     metrics.PortfolioVaR95,
			"var_99":     metrics.PortfolioVaR99,
		}).Warn("Risk check failed")
	}

	return result, nil
}

// calculatePortfolioVaR 간이 포트폴리오 VaR 계산
func (e *Engine) calculatePortfolioVaR(
	historicalReturns map[string][]float64,
	weights map[string]float64,
) struct {
	VaR95 float64
	VaR99 float64
} {
	// 포트폴리오 수익률 시계열 구성
	// 가장 짧은 시계열 길이 찾기
	minLen := 0
	for _, returns := range historicalReturns {
		if minLen == 0 || len(returns) < minLen {
			minLen = len(returns)
		}
	}

	if minLen == 0 {
		return struct {
			VaR95 float64
			VaR99 float64
		}{0, 0}
	}

	// 날짜별 포트폴리오 수익률 계산
	portfolioReturns := make([]float64, minLen)
	for i := 0; i < minLen; i++ {
		dayReturn := 0.0
		for code, weight := range weights {
			if returns, ok := historicalReturns[code]; ok && i < len(returns) {
				dayReturn += weight * returns[i]
			}
		}
		portfolioReturns[i] = dayReturn
	}

	// VaR 계산
	var95 := CalculateVaR(portfolioReturns, 0.95)
	var99 := CalculateVaR(portfolioReturns, 0.99)

	return struct {
		VaR95 float64
		VaR99 float64
	}{var95.VaR, var99.VaR}
}

// calculateMetrics 포트폴리오 메트릭 계산
func (e *Engine) calculateMetrics(holdings []Holding) RiskMetrics {
	metrics := RiskMetrics{}

	if len(holdings) == 0 {
		return metrics
	}

	// 비중 정렬 (내림차순)
	sorted := make([]Holding, len(holdings))
	copy(sorted, holdings)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Weight > sorted[j].Weight
	})

	// 최대 단일 종목 비중
	metrics.MaxSingleExposure = sorted[0].Weight

	// 상위 5종목 집중도
	top5Weight := 0.0
	for i := 0; i < 5 && i < len(sorted); i++ {
		top5Weight += sorted[i].Weight
	}
	metrics.ConcentrationRatio = top5Weight

	// 유동성 점수 (TODO: ADTV 기반 계산)
	// 현재는 더미 값
	metrics.LiquidityScore = 0.8

	return metrics
}

// GetLimits 현재 리스크 한도 반환
func (e *Engine) GetLimits() RiskLimits {
	return e.limits
}

// SetLimits 리스크 한도 업데이트
func (e *Engine) SetLimits(limits RiskLimits) {
	e.limits = limits
}
