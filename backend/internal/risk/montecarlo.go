package risk

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// MonteCarloSimulator Monte Carlo 시뮬레이터
type MonteCarloSimulator struct {
	config MonteCarloConfig
	rng    *rand.Rand
}

// NewMonteCarloSimulator 새 시뮬레이터 생성
func NewMonteCarloSimulator(config MonteCarloConfig) *MonteCarloSimulator {
	var rng *rand.Rand
	if config.Seed != 0 {
		rng = rand.New(rand.NewSource(config.Seed))
	} else {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	return &MonteCarloSimulator{
		config: config,
		rng:    rng,
	}
}

// Simulate 포트폴리오 Monte Carlo 시뮬레이션 실행
// historicalReturns: 각 종목별 과거 수익률 [code][]float64
// weights: 각 종목별 비중 [code]float64
func (mc *MonteCarloSimulator) Simulate(
	ctx context.Context,
	historicalReturns map[string][]float64,
	weights map[string]float64,
) (*MonteCarloResult, error) {
	// 입력 검증
	if len(historicalReturns) == 0 || len(weights) == 0 {
		return nil, fmt.Errorf("empty historical returns or weights")
	}

	// 시뮬레이션 결과 저장
	portfolioReturns := make([]float64, mc.config.NumSimulations)

	// Historical Simulation 방식
	if mc.config.Method == "historical" {
		portfolioReturns = mc.historicalSimulation(historicalReturns, weights)
	} else {
		// Parametric 방식 (정규분포 가정)
		portfolioReturns = mc.parametricSimulation(historicalReturns, weights)
	}

	// 결과 계산
	result := mc.calculateResult(portfolioReturns)

	return result, nil
}

// historicalSimulation Historical Simulation 방식
// 과거 수익률을 랜덤하게 재샘플링
func (mc *MonteCarloSimulator) historicalSimulation(
	historicalReturns map[string][]float64,
	weights map[string]float64,
) []float64 {
	results := make([]float64, mc.config.NumSimulations)

	// 모든 종목의 과거 데이터 길이 확인
	minLen := math.MaxInt32
	for _, returns := range historicalReturns {
		if len(returns) < minLen {
			minLen = len(returns)
		}
	}

	if minLen == 0 {
		return results
	}

	// 보유 기간 동안의 누적 수익률 시뮬레이션
	for i := 0; i < mc.config.NumSimulations; i++ {
		// 랜덤 시작 인덱스 선택
		portfolioReturn := 0.0

		for code, weight := range weights {
			returns := historicalReturns[code]
			if len(returns) == 0 {
				continue
			}

			// 보유 기간 동안의 누적 수익률
			cumReturn := 1.0
			for d := 0; d < mc.config.HoldingPeriod; d++ {
				// 랜덤하게 과거 수익률 선택
				idx := mc.rng.Intn(len(returns))
				cumReturn *= (1 + returns[idx])
			}
			stockReturn := cumReturn - 1

			// 포트폴리오 가중 수익률
			portfolioReturn += weight * stockReturn
		}

		results[i] = portfolioReturn
	}

	return results
}

// parametricSimulation Parametric Simulation 방식
// 정규분포 가정 하에 시뮬레이션
func (mc *MonteCarloSimulator) parametricSimulation(
	historicalReturns map[string][]float64,
	weights map[string]float64,
) []float64 {
	results := make([]float64, mc.config.NumSimulations)

	// 각 종목의 평균과 표준편차 계산
	means := make(map[string]float64)
	stds := make(map[string]float64)

	for code, returns := range historicalReturns {
		means[code] = CalculateMean(returns)
		stds[code] = CalculateVolatility(returns)
	}

	// 보유 기간 스케일링
	sqrtT := math.Sqrt(float64(mc.config.HoldingPeriod))

	for i := 0; i < mc.config.NumSimulations; i++ {
		portfolioReturn := 0.0

		for code, weight := range weights {
			mean := means[code] * float64(mc.config.HoldingPeriod)
			std := stds[code] * sqrtT

			// 정규분포에서 랜덤 샘플링
			z := mc.rng.NormFloat64()
			stockReturn := mean + std*z

			portfolioReturn += weight * stockReturn
		}

		results[i] = portfolioReturn
	}

	return results
}

// calculateResult 시뮬레이션 결과 통계 계산
func (mc *MonteCarloSimulator) calculateResult(portfolioReturns []float64) *MonteCarloResult {
	// 기본 통계
	mean := CalculateMean(portfolioReturns)
	stdDev := CalculateVolatility(portfolioReturns)

	// VaR/CVaR 계산
	var95 := CalculateVaR(portfolioReturns, 0.95)
	var99 := CalculateVaR(portfolioReturns, 0.99)

	// 백분위수 계산
	percentiles := CalculatePercentiles(portfolioReturns, []int{1, 5, 10, 25, 50, 75, 90, 95, 99})

	return &MonteCarloResult{
		RunID:       uuid.New().String(),
		RunDate:     time.Now(),
		Config:      mc.config,
		MeanReturn:  mean,
		StdDev:      stdDev,
		VaR95:       var95.VaR,
		VaR99:       var99.VaR,
		CVaR95:      var95.CVaR,
		CVaR99:      var99.CVaR,
		Percentiles: percentiles,
		CreatedAt:   time.Now(),
	}
}

// SimulateSimple 단일 종목 또는 단순 포트폴리오 시뮬레이션
// returns: 포트폴리오 수익률 시계열
func (mc *MonteCarloSimulator) SimulateSimple(
	ctx context.Context,
	portfolioReturns []float64,
) (*MonteCarloResult, error) {
	if len(portfolioReturns) == 0 {
		return nil, fmt.Errorf("empty portfolio returns")
	}

	// Historical Simulation
	results := make([]float64, mc.config.NumSimulations)

	for i := 0; i < mc.config.NumSimulations; i++ {
		// 보유 기간 동안의 누적 수익률
		cumReturn := 1.0
		for d := 0; d < mc.config.HoldingPeriod; d++ {
			idx := mc.rng.Intn(len(portfolioReturns))
			cumReturn *= (1 + portfolioReturns[idx])
		}
		results[i] = cumReturn - 1
	}

	return mc.calculateResult(results), nil
}
