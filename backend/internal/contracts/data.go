package contracts

import "time"

// DataQualitySnapshot represents data quality information passed from S0 to S1
// ⭐ SSOT: S0 → S1 데이터 품질 정보 전달
type DataQualitySnapshot struct {
	Date         time.Time          `json:"date"`
	TotalStocks  int                `json:"total_stocks"`
	ValidStocks  int                `json:"valid_stocks"`
	Coverage     map[string]float64 `json:"coverage"`      // 데이터별 커버리지
	QualityScore float64            `json:"quality_score"` // 0.0 ~ 1.0
	Passed       bool               `json:"passed"`        // 품질 검증 통과 여부
}

// IsValid checks if the data quality snapshot meets minimum requirements
func (d *DataQualitySnapshot) IsValid() bool {
	return d.QualityScore >= 0.7 && d.ValidStocks > 0
}

// CoverageRate returns the average coverage rate across all data types
func (d *DataQualitySnapshot) CoverageRate() float64 {
	if len(d.Coverage) == 0 {
		return 0.0
	}

	total := 0.0
	for _, rate := range d.Coverage {
		total += rate
	}

	return total / float64(len(d.Coverage))
}
