package audit

import (
	"context"
	"fmt"
	"time"
)

// Attribution represents factor contribution analysis
type Attribution struct {
	Factor       string  `json:"factor"`
	Contribution float64 `json:"contribution"` // 수익 기여도
	Exposure     float64 `json:"exposure"`     // 평균 노출도
	ReturnPct    float64 `json:"return_pct"`   // 팩터 수익률
}

// AnalyzeAttribution performs factor attribution analysis
func (a *Analyzer) AnalyzeAttribution(ctx context.Context, period string) ([]Attribution, error) {
	startDate, endDate := a.parsePeriod(period)

	attrs := make([]Attribution, 0)

	// 팩터별 기여도 계산
	// ⭐ SSOT: data-flow.md 기준 팩터 가중치
	factors := []FactorInfo{
		{"momentum", 0.20},   // 20%
		{"technical", 0.20},  // 20%
		{"value", 0.15},      // 15%
		{"quality", 0.15},    // 15%
		{"flow", 0.25},       // 25% ⭐ 수급 (한국 시장 중요)
		{"event", 0.05},      // 5%
	}

	for _, factor := range factors {
		contrib, err := a.calculateFactorContribution(ctx, startDate, endDate, factor.Name)
		if err != nil {
			a.logger.WithFields(map[string]interface{}{
				"factor": factor.Name,
				"error":  err,
			}).Warn("Failed to calculate factor contribution")
			continue
		}

		exposure := a.getAverageExposure(ctx, startDate, endDate, factor.Name)

		attrs = append(attrs, Attribution{
			Factor:       factor.Name,
			Contribution: contrib,
			Exposure:     exposure,
			ReturnPct:    contrib / exposure * 100, // 기여도 / 노출도
		})
	}

	a.logger.WithFields(map[string]interface{}{
		"period":        period,
		"total_factors": len(attrs),
	}).Info("Attribution analysis completed")

	return attrs, nil
}

// FactorInfo represents factor information
type FactorInfo struct {
	Name   string
	Weight float64
}

// calculateFactorContribution calculates factor's contribution to return
func (a *Analyzer) calculateFactorContribution(ctx context.Context, startDate, endDate time.Time, factor string) (float64, error) {
	// 해당 팩터의 평균 점수가 높은 종목들의 평균 수익률
	query := `
		WITH factor_scores AS (
			SELECT
				h.stock_code,
				AVG(CASE
					WHEN $3 = 'momentum' THEN fs.momentum
					WHEN $3 = 'technical' THEN fs.technical
					WHEN $3 = 'value' THEN fs.value
					WHEN $3 = 'quality' THEN fs.quality
					WHEN $3 = 'flow' THEN fs.flow
					WHEN $3 = 'event' THEN fs.event
				END) as avg_score,
				SUM(h.unrealized_pnl_pct) as total_return
			FROM portfolio.holdings h
			JOIN signals.factor_scores fs ON h.stock_code = fs.code
			WHERE h.holding_date BETWEEN $1 AND $2
			GROUP BY h.stock_code
		)
		SELECT
			COALESCE(AVG(total_return) FILTER (WHERE avg_score > 0.5), 0) as contribution
		FROM factor_scores
	`

	var contribution float64
	err := a.repository.pool.QueryRow(ctx, query, startDate, endDate, factor).Scan(&contribution)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate factor contribution: %w", err)
	}

	return contribution, nil
}

// getAverageExposure calculates average exposure to a factor
func (a *Analyzer) getAverageExposure(ctx context.Context, startDate, endDate time.Time, factor string) float64 {
	// 해당 팩터 점수의 평균 절댓값
	query := `
		SELECT
			COALESCE(AVG(ABS(CASE
				WHEN $3 = 'momentum' THEN momentum
				WHEN $3 = 'technical' THEN technical
				WHEN $3 = 'value' THEN value
				WHEN $3 = 'quality' THEN quality
				WHEN $3 = 'flow' THEN flow
				WHEN $3 = 'event' THEN event
			END)), 0) as avg_exposure
		FROM signals.factor_scores fs
		JOIN portfolio.holdings h ON fs.code = h.stock_code
		WHERE h.holding_date BETWEEN $1 AND $2
	`

	var exposure float64
	err := a.repository.pool.QueryRow(ctx, query, startDate, endDate, factor).Scan(&exposure)
	if err != nil {
		a.logger.WithFields(map[string]interface{}{
			"factor": factor,
			"error":  err,
		}).Warn("Failed to get average exposure")
		return 0
	}

	return exposure
}

// GetTopContributors returns top contributing factors
func (a *Analyzer) GetTopContributors(attrs []Attribution, limit int) []Attribution {
	if len(attrs) == 0 {
		return attrs
	}

	// Sort by contribution (descending)
	sorted := make([]Attribution, len(attrs))
	copy(sorted, attrs)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Contribution > sorted[i].Contribution {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	if limit > len(sorted) {
		limit = len(sorted)
	}

	return sorted[:limit]
}

// GetBottomContributors returns worst contributing factors
func (a *Analyzer) GetBottomContributors(attrs []Attribution, limit int) []Attribution {
	if len(attrs) == 0 {
		return attrs
	}

	// Sort by contribution (ascending)
	sorted := make([]Attribution, len(attrs))
	copy(sorted, attrs)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Contribution < sorted[i].Contribution {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	if limit > len(sorted) {
		limit = len(sorted)
	}

	return sorted[:limit]
}
