package risk

import (
	"math"
	"sort"
)

// =============================================================================
// VaR (Value at Risk) Calculation
// =============================================================================

// CalculateVaR 과거 수익률 기반 VaR 계산 (Historical Simulation)
// returns: 일별 수익률 배열 (양수=이익, 음수=손실)
// confidence: 신뢰수준 (예: 0.95, 0.99)
// 반환값: VaR는 손실을 양수로 표현 (예: 0.05 = 5% 손실 가능)
func CalculateVaR(returns []float64, confidence float64) VaRResult {
	if len(returns) == 0 {
		return VaRResult{Confidence: confidence, VaR: 0, CVaR: 0}
	}

	// 수익률 정렬 (오름차순: 손실이 앞에)
	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	// VaR: (1-confidence) 백분위수
	// 예: 95% VaR = 하위 5% 백분위수
	percentile := 1.0 - confidence
	idx := int(math.Floor(percentile * float64(len(sorted))))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	// VaR = 손실을 양수로 표현
	var varValue float64
	if sorted[idx] < 0 {
		varValue = -sorted[idx]
	} else {
		varValue = 0 // 손실 없음
	}

	// CVaR (Expected Shortfall): VaR 이하의 평균 손실
	cvar := CalculateCVaR(sorted, idx)

	return VaRResult{
		Confidence: confidence,
		VaR:        varValue,
		CVaR:       cvar,
	}
}

// CalculateCVaR Conditional VaR (Expected Shortfall) 계산
// sorted: 오름차순 정렬된 수익률
// varIdx: VaR 인덱스 (이 인덱스 이하의 수익률이 tail)
func CalculateCVaR(sorted []float64, varIdx int) float64 {
	if len(sorted) == 0 || varIdx < 0 {
		return 0
	}

	// VaR 인덱스까지의 수익률 평균 (tail 평균)
	var sum float64
	count := varIdx + 1
	for i := 0; i <= varIdx && i < len(sorted); i++ {
		sum += sorted[i]
	}

	if count == 0 {
		return 0
	}

	avgTailReturn := sum / float64(count)

	// CVaR = 손실을 양수로 표현
	if avgTailReturn < 0 {
		return -avgTailReturn
	}
	return 0
}

// =============================================================================
// Parametric VaR (정규분포 가정)
// =============================================================================

// CalculateParametricVaR 정규분포 가정 VaR 계산
// mean: 평균 수익률
// stdDev: 표준편차
// confidence: 신뢰수준
func CalculateParametricVaR(mean, stdDev, confidence float64) VaRResult {
	// Z-score for confidence level
	// 95%: 1.645, 99%: 2.326
	z := NormInv(confidence)

	// Parametric VaR = -mean + z * stdDev
	// 단순화: mean은 작으므로 무시하고 z * stdDev 사용
	varValue := z * stdDev
	if varValue < 0 {
		varValue = 0
	}

	// Parametric CVaR (Cornish-Fisher approximation)
	// CVaR ≈ VaR + stdDev * φ(z) / (1-confidence)
	// φ(z) = 정규분포 pdf at z
	phi := NormPDF(z)
	cvar := varValue + (stdDev * phi / (1 - confidence))

	return VaRResult{
		Confidence: confidence,
		VaR:        varValue,
		CVaR:       cvar,
	}
}


// =============================================================================
// 통계 유틸리티
// =============================================================================

// NormInv 정규분포 역함수 (Quantile Function)
// Beasley-Springer-Moro approximation
func NormInv(p float64) float64 {
	if p <= 0 || p >= 1 {
		return 0
	}

	// 일반적인 신뢰수준에 대한 빠른 반환
	switch {
	case p == 0.99:
		return 2.326
	case p == 0.95:
		return 1.645
	case p == 0.90:
		return 1.282
	case p == 0.975:
		return 1.96
	}

	// Beasley-Springer-Moro approximation
	a := []float64{
		-3.969683028665376e+01,
		2.209460984245205e+02,
		-2.759285104469687e+02,
		1.383577518672690e+02,
		-3.066479806614716e+01,
		2.506628277459239e+00,
	}
	b := []float64{
		-5.447609879822406e+01,
		1.615858368580409e+02,
		-1.556989798598866e+02,
		6.680131188771972e+01,
		-1.328068155288572e+01,
	}
	c := []float64{
		-7.784894002430293e-03,
		-3.223964580411365e-01,
		-2.400758277161838e+00,
		-2.549732539343734e+00,
		4.374664141464968e+00,
		2.938163982698783e+00,
	}
	d := []float64{
		7.784695709041462e-03,
		3.224671290700398e-01,
		2.445134137142996e+00,
		3.754408661907416e+00,
	}

	pLow := 0.02425
	pHigh := 1 - pLow

	var q, r float64

	if p < pLow {
		q = math.Sqrt(-2 * math.Log(p))
		return (((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) /
			((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	} else if p <= pHigh {
		q = p - 0.5
		r = q * q
		return (((((a[0]*r+a[1])*r+a[2])*r+a[3])*r+a[4])*r + a[5]) * q /
			(((((b[0]*r+b[1])*r+b[2])*r+b[3])*r+b[4])*r + 1)
	} else {
		q = math.Sqrt(-2 * math.Log(1-p))
		return -(((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) /
			((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	}
}

// NormPDF 정규분포 확률밀도함수
func NormPDF(x float64) float64 {
	return math.Exp(-x*x/2) / math.Sqrt(2*math.Pi)
}

// Mean 평균 계산
func Mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// StdDev 표준편차 계산
func StdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := Mean(values)
	var sumSq float64
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}

// Percentile 백분위수 계산
func Percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}

	idx := p / 100.0 * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	// 선형 보간
	weight := idx - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
