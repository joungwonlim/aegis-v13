package risk

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Monte Carlo Simulator
// =============================================================================

// MonteCarloSimulator Monte Carlo 시뮬레이터
// ⭐ SSOT: Monte Carlo 시뮬레이션은 여기서만
type MonteCarloSimulator struct {
	config MonteCarloConfig
	rng    *rand.Rand
}

// NewMonteCarloSimulator 새 시뮬레이터 생성
func NewMonteCarloSimulator(config MonteCarloConfig) *MonteCarloSimulator {
	// Seed 설정
	var rng *rand.Rand
	if config.Seed == 0 {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	} else {
		rng = rand.New(rand.NewSource(config.Seed))
	}

	return &MonteCarloSimulator{
		config: config,
		rng:    rng,
	}
}

// SimulateReturns 포트폴리오 수익률 시계열로 Monte Carlo 시뮬레이션
// returns: 포트폴리오 일별 수익률 (상위 레이어에서 조립해서 전달)
// 반환: 시뮬레이션 결과 (VaR, CVaR, 분포 등)
func (s *MonteCarloSimulator) SimulateReturns(returns []float64) (*MonteCarloResult, error) {
	if len(returns) == 0 {
		return nil, fmt.Errorf("no returns provided")
	}

	// 최소 샘플 체크
	if len(returns) < s.config.MinSamples {
		return nil, fmt.Errorf("insufficient samples: got %d, need %d", len(returns), s.config.MinSamples)
	}

	// 시뮬레이션 실행
	var simReturns []float64
	switch s.config.Method {
	case MethodHistoricalBootstrap:
		simReturns = s.bootstrapSimulation(returns)
	case MethodParametricNormal:
		simReturns = s.parametricNormalSimulation(returns)
	case MethodParametricT:
		simReturns = s.parametricTSimulation(returns)
	default:
		simReturns = s.bootstrapSimulation(returns)
	}

	// 결과 계산
	result := s.calculateResults(simReturns)
	result.RunID = uuid.New().String()[:8]
	result.RunDate = time.Now()
	result.Config = s.config
	result.CreatedAt = time.Now()

	return result, nil
}

// bootstrapSimulation Historical Bootstrap 시뮬레이션
// 과거 수익률을 랜덤 추출하여 미래 수익률 시뮬레이션
func (s *MonteCarloSimulator) bootstrapSimulation(returns []float64) []float64 {
	n := len(returns)
	if n == 0 {
		return nil
	}

	simReturns := make([]float64, s.config.NumSimulations)

	for i := 0; i < s.config.NumSimulations; i++ {
		// 보유 기간 동안의 누적 수익률
		cumReturn := 1.0
		for d := 0; d < s.config.HoldingPeriod; d++ {
			// 랜덤 추출 (복원추출)
			idx := s.rng.Intn(n)
			cumReturn *= (1 + returns[idx])
		}
		simReturns[i] = cumReturn - 1 // 수익률로 변환
	}

	return simReturns
}

// parametricNormalSimulation 정규분포 가정 시뮬레이션
func (s *MonteCarloSimulator) parametricNormalSimulation(returns []float64) []float64 {
	if len(returns) == 0 {
		return nil
	}

	// 평균, 표준편차 계산
	mean := Mean(returns)
	stdDev := StdDev(returns)

	simReturns := make([]float64, s.config.NumSimulations)

	// 보유 기간 스케일링
	// 일별 → 기간 수익률: mean * days, stdDev * sqrt(days)
	periodMean := mean * float64(s.config.HoldingPeriod)
	periodStd := stdDev * math.Sqrt(float64(s.config.HoldingPeriod))

	for i := 0; i < s.config.NumSimulations; i++ {
		// Box-Muller transform for normal random
		u1, u2 := s.rng.Float64(), s.rng.Float64()
		z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)

		simReturns[i] = periodMean + periodStd*z
	}

	return simReturns
}

// parametricTSimulation t-분포 시뮬레이션 (Fat Tail 반영)
func (s *MonteCarloSimulator) parametricTSimulation(returns []float64) []float64 {
	if len(returns) == 0 {
		return nil
	}

	// 평균, 표준편차 계산
	mean := Mean(returns)
	stdDev := StdDev(returns)

	// t-분포 자유도 (일반적으로 5-10 사용)
	df := 5.0

	simReturns := make([]float64, s.config.NumSimulations)

	// 보유 기간 스케일링
	periodMean := mean * float64(s.config.HoldingPeriod)
	periodStd := stdDev * math.Sqrt(float64(s.config.HoldingPeriod))

	for i := 0; i < s.config.NumSimulations; i++ {
		// t-분포 랜덤: z / sqrt(chi^2/df)
		// Box-Muller for normal
		u1, u2 := s.rng.Float64(), s.rng.Float64()
		z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)

		// Chi-squared approximation (sum of squared normals)
		var chi2 float64
		for j := 0; j < int(df); j++ {
			u3, u4 := s.rng.Float64(), s.rng.Float64()
			n := math.Sqrt(-2*math.Log(u3)) * math.Cos(2*math.Pi*u4)
			chi2 += n * n
		}

		t := z / math.Sqrt(chi2/df)
		simReturns[i] = periodMean + periodStd*t
	}

	return simReturns
}

// calculateResults 시뮬레이션 결과 계산
func (s *MonteCarloSimulator) calculateResults(simReturns []float64) *MonteCarloResult {
	if len(simReturns) == 0 {
		return &MonteCarloResult{}
	}

	// 정렬
	sorted := make([]float64, len(simReturns))
	copy(sorted, simReturns)
	sort.Float64s(sorted)

	// 기본 통계
	result := &MonteCarloResult{
		MeanReturn:  Mean(simReturns),
		StdDev:      StdDev(simReturns),
		Percentiles: make(map[int]float64),
	}

	// 백분위수
	percentiles := []int{1, 5, 10, 25, 50, 75, 90, 95, 99}
	for _, p := range percentiles {
		result.Percentiles[p] = Percentile(sorted, float64(p))
	}

	// VaR/CVaR 계산
	var95Result := CalculateVaR(simReturns, 0.95)
	var99Result := CalculateVaR(simReturns, 0.99)

	result.VaR95 = var95Result.VaR
	result.CVaR95 = var95Result.CVaR
	result.VaR99 = var99Result.VaR
	result.CVaR99 = var99Result.CVaR

	return result
}

// =============================================================================
// Scenario Simulation
// =============================================================================

// Scenario 스트레스 시나리오
type Scenario struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Shocks      map[string]float64 `json:"shocks"` // 종목별 충격 (예: {"005930": -0.10})
}

// PredefinedScenarios 사전 정의 시나리오
func PredefinedScenarios() []Scenario {
	return []Scenario{
		{
			Name:        "market_crash",
			Description: "시장 급락 (-10%)",
			Shocks:      map[string]float64{"*": -0.10},
		},
		{
			Name:        "sector_rotation",
			Description: "섹터 회전 (기술 -5%, 금융 +3%)",
			Shocks:      map[string]float64{"TECH": -0.05, "FINANCE": 0.03},
		},
		{
			Name:        "black_swan",
			Description: "블랙스완 (-20%)",
			Shocks:      map[string]float64{"*": -0.20},
		},
	}
}

// =============================================================================
// Block Bootstrap (자기상관 유지)
// =============================================================================

// BlockBootstrapSimulation 블록 부트스트랩 시뮬레이션
// 연속된 수익률 블록을 추출하여 자기상관 유지
func (s *MonteCarloSimulator) BlockBootstrapSimulation(returns []float64, blockSize int) []float64 {
	n := len(returns)
	if n == 0 || blockSize <= 0 {
		return nil
	}

	// 블록 수 계산
	numBlocks := (s.config.HoldingPeriod + blockSize - 1) / blockSize

	simReturns := make([]float64, s.config.NumSimulations)

	for i := 0; i < s.config.NumSimulations; i++ {
		var cumReturn float64
		daysUsed := 0

		for b := 0; b < numBlocks && daysUsed < s.config.HoldingPeriod; b++ {
			// 블록 시작점 랜덤 선택
			maxStart := n - blockSize
			if maxStart < 0 {
				maxStart = 0
			}
			startIdx := s.rng.Intn(maxStart + 1)

			// 블록 내 수익률 합산
			for d := 0; d < blockSize && daysUsed < s.config.HoldingPeriod; d++ {
				if startIdx+d < n {
					cumReturn += returns[startIdx+d]
					daysUsed++
				}
			}
		}

		simReturns[i] = cumReturn
	}

	return simReturns
}
