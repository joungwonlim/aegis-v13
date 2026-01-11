package risk

import (
	"math"
	"sort"
)

// CalculateVaR 주어진 수익률 배열에서 VaR와 CVaR 계산
// returns: 일간 수익률 배열 (예: [-0.02, 0.01, -0.03, ...])
// confidence: 신뢰 수준 (0.95 또는 0.99)
func CalculateVaR(returns []float64, confidence float64) VaRResult {
	if len(returns) == 0 {
		return VaRResult{Confidence: confidence, VaR: 0, CVaR: 0}
	}

	// 수익률 복사 및 정렬 (오름차순 - 손실이 앞에)
	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	// VaR 인덱스 계산
	// 95% confidence → 하위 5% 손실 (왼쪽 꼬리)
	alpha := 1.0 - confidence
	varIndex := int(math.Floor(alpha * float64(len(sorted))))
	if varIndex >= len(sorted) {
		varIndex = len(sorted) - 1
	}

	// VaR: 해당 분위수의 손실 (음수를 양수로 변환)
	var_ := -sorted[varIndex]
	if var_ < 0 {
		var_ = 0 // 이익인 경우 VaR은 0
	}

	// CVaR (Expected Shortfall): VaR보다 나쁜 손실의 평균
	cvar := calculateCVaR(sorted, varIndex)

	return VaRResult{
		Confidence: confidence,
		VaR:        var_,
		CVaR:       cvar,
	}
}

// calculateCVaR CVaR (Expected Shortfall) 계산
// sorted: 정렬된 수익률 배열 (오름차순)
// varIndex: VaR에 해당하는 인덱스
func calculateCVaR(sorted []float64, varIndex int) float64 {
	if varIndex <= 0 {
		return -sorted[0]
	}

	// VaR 이하의 모든 손실의 평균
	sum := 0.0
	count := 0
	for i := 0; i <= varIndex; i++ {
		sum += sorted[i]
		count++
	}

	if count == 0 {
		return 0
	}

	// 음수를 양수로 변환 (손실을 양수로 표현)
	cvar := -sum / float64(count)
	if cvar < 0 {
		cvar = 0
	}
	return cvar
}

// CalculateVolatility 변동성 (표준편차) 계산
func CalculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	// 평균 계산
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	// 분산 계산
	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1) // 표본 분산

	return math.Sqrt(variance)
}

// CalculateMean 평균 수익률 계산
func CalculateMean(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	sum := 0.0
	for _, r := range returns {
		sum += r
	}
	return sum / float64(len(returns))
}

// CalculatePercentiles 백분위수 계산
func CalculatePercentiles(returns []float64, percentiles []int) map[int]float64 {
	result := make(map[int]float64)
	if len(returns) == 0 {
		return result
	}

	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	for _, p := range percentiles {
		idx := int(float64(p) / 100.0 * float64(len(sorted)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		result[p] = sorted[idx]
	}

	return result
}

// AnnualizeVolatility 일간 변동성을 연간화
func AnnualizeVolatility(dailyVol float64) float64 {
	return dailyVol * math.Sqrt(252) // 거래일 252일 기준
}

// ScaleVaR VaR를 다른 기간으로 스케일링 (제곱근 법칙)
// 예: 1일 VaR → 5일 VaR
func ScaleVaR(var1Day float64, days int) float64 {
	return var1Day * math.Sqrt(float64(days))
}
