package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/execution"
)

// DailySnapshot represents daily portfolio snapshot
type DailySnapshot struct {
	Date        time.Time           `json:"date"`
	TotalValue  float64             `json:"total_value"`
	Cash        float64             `json:"cash"`
	Positions   []PositionSnapshot  `json:"positions"`
	DailyReturn float64             `json:"daily_return"`
	CumReturn   float64             `json:"cum_return"`
}

// PositionSnapshot represents position snapshot
type PositionSnapshot struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
	Value    float64 `json:"value"`
	Weight   float64 `json:"weight"`
	DailyPnL float64 `json:"daily_pnl"`
}

// SaveSnapshot saves daily portfolio snapshot
func (a *Analyzer) SaveSnapshot(ctx context.Context, broker execution.Broker) error {
	snapshot := &DailySnapshot{
		Date:      time.Now(),
		Positions: make([]PositionSnapshot, 0),
	}

	// 잔고 조회
	balance, err := broker.GetBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	snapshot.TotalValue = balance.TotalValue
	snapshot.Cash = balance.Cash

	// 보유 종목 조회
	holdings, err := broker.GetHoldings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get holdings: %w", err)
	}

	// 포지션 스냅샷
	for _, h := range holdings {
		snapshot.Positions = append(snapshot.Positions, PositionSnapshot{
			Code:     h.Code,
			Name:     h.Name,
			Quantity: h.Qty,
			Price:    h.CurrentPrice,
			Value:    h.MarketValue,
			Weight:   h.MarketValue / snapshot.TotalValue,
			DailyPnL: 0, // TODO: Calculate daily PnL
		})
	}

	// 일일 수익률 계산
	prevSnapshot, err := a.repository.GetPreviousSnapshot(ctx, snapshot.Date)
	if err == nil && prevSnapshot != nil {
		if prevSnapshot.TotalValue > 0 {
			snapshot.DailyReturn = (snapshot.TotalValue - prevSnapshot.TotalValue) / prevSnapshot.TotalValue
			snapshot.CumReturn = prevSnapshot.CumReturn * (1 + snapshot.DailyReturn)
		}
	} else {
		// 첫 스냅샷
		snapshot.DailyReturn = 0
		snapshot.CumReturn = 1.0
	}

	// 저장
	if err := a.repository.SaveSnapshot(ctx, snapshot); err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	a.logger.WithFields(map[string]interface{}{
		"date":         snapshot.Date.Format("2006-01-02"),
		"total_value":  snapshot.TotalValue,
		"daily_return": snapshot.DailyReturn,
		"positions":    len(snapshot.Positions),
	}).Info("Snapshot saved")

	return nil
}

// GetSnapshot retrieves snapshot for a date
func (a *Analyzer) GetSnapshot(ctx context.Context, date time.Time) (*DailySnapshot, error) {
	return a.repository.GetSnapshot(ctx, date)
}

// GetSnapshotHistory retrieves snapshot history for a period
func (a *Analyzer) GetSnapshotHistory(ctx context.Context, startDate, endDate time.Time) ([]DailySnapshot, error) {
	return a.repository.GetSnapshotHistory(ctx, startDate, endDate)
}

// GetEquityCurve generates equity curve from snapshots
func (a *Analyzer) GetEquityCurve(ctx context.Context, startDate, endDate time.Time) ([]EquityPoint, error) {
	snapshots, err := a.repository.GetSnapshotHistory(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot history: %w", err)
	}

	curve := make([]EquityPoint, 0, len(snapshots))
	for _, s := range snapshots {
		curve = append(curve, EquityPoint{
			Date:       s.Date,
			Value:      s.TotalValue,
			Return:     s.DailyReturn,
			CumReturn:  s.CumReturn,
		})
	}

	return curve, nil
}

// EquityPoint represents a point on equity curve
type EquityPoint struct {
	Date      time.Time `json:"date"`
	Value     float64   `json:"value"`
	Return    float64   `json:"return"`
	CumReturn float64   `json:"cum_return"`
}

// CompareWithBenchmark compares portfolio performance with benchmark
func (a *Analyzer) CompareWithBenchmark(ctx context.Context, startDate, endDate time.Time) (*BenchmarkComparison, error) {
	// Portfolio equity curve
	portfolioCurve, err := a.GetEquityCurve(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio curve: %w", err)
	}

	if len(portfolioCurve) == 0 {
		return nil, fmt.Errorf("no portfolio data for period")
	}

	// Portfolio return
	portfolioReturn := portfolioCurve[len(portfolioCurve)-1].CumReturn - 1.0

	// Benchmark return (TODO: Get actual benchmark data)
	benchmarkReturn := 0.05 // Mock KOSPI return

	comparison := &BenchmarkComparison{
		StartDate:       startDate,
		EndDate:         endDate,
		PortfolioReturn: portfolioReturn,
		BenchmarkReturn: benchmarkReturn,
		Alpha:           portfolioReturn - benchmarkReturn,
		OutperformDays:  0, // TODO: Calculate
		TotalDays:       len(portfolioCurve),
	}

	return comparison, nil
}

// BenchmarkComparison represents benchmark comparison result
type BenchmarkComparison struct {
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	PortfolioReturn float64   `json:"portfolio_return"`
	BenchmarkReturn float64   `json:"benchmark_return"`
	Alpha           float64   `json:"alpha"`
	OutperformDays  int       `json:"outperform_days"`
	TotalDays       int       `json:"total_days"`
}
