package audit

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Analyzer implements S7: Performance analysis
// ⭐ SSOT: S7 성과 분석 로직은 여기서만
type Analyzer struct {
	repository *Repository
	logger     *logger.Logger
}

// NewAnalyzer creates a new performance analyzer
func NewAnalyzer(repository *Repository, logger *logger.Logger) *Analyzer {
	return &Analyzer{
		repository: repository,
		logger:     logger,
	}
}

// PerformanceReport represents performance analysis report
type PerformanceReport struct {
	Period      string    `json:"period"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`

	// 수익률
	TotalReturn  float64 `json:"total_return"`
	AnnualReturn float64 `json:"annual_return"`

	// 리스크 지표
	Volatility  float64 `json:"volatility"`
	Sharpe      float64 `json:"sharpe"`
	Sortino     float64 `json:"sortino"`
	MaxDrawdown float64 `json:"max_drawdown"`

	// 트레이딩 지표
	WinRate      float64 `json:"win_rate"`
	AvgWin       float64 `json:"avg_win"`
	AvgLoss      float64 `json:"avg_loss"`
	ProfitFactor float64 `json:"profit_factor"`

	// 비교
	Benchmark float64 `json:"benchmark"` // KOSPI 수익률
	Alpha     float64 `json:"alpha"`
	Beta      float64 `json:"beta"`
}

// Analyze performs performance analysis for a period
func (a *Analyzer) Analyze(ctx context.Context, period string) (*PerformanceReport, error) {
	report := &PerformanceReport{Period: period}

	// 기간 파싱
	startDate, endDate := a.parsePeriod(period)
	report.StartDate = startDate
	report.EndDate = endDate

	// 일별 수익률 조회
	dailyReturns, err := a.repository.GetDailyReturns(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily returns: %w", err)
	}

	if len(dailyReturns) == 0 {
		return nil, fmt.Errorf("no data for period %s", period)
	}

	// 수익률 계산
	report.TotalReturn = a.calculateTotalReturn(dailyReturns)
	report.AnnualReturn = a.annualize(report.TotalReturn, len(dailyReturns))

	// 리스크 지표
	report.Volatility = a.calculateVolatility(dailyReturns)
	report.Sharpe = a.calculateSharpe(report.AnnualReturn, report.Volatility)
	report.Sortino = a.calculateSortino(dailyReturns)
	report.MaxDrawdown = a.calculateMaxDrawdown(dailyReturns)

	// 트레이딩 지표
	trades, err := a.repository.GetTrades(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	report.WinRate = a.calculateWinRate(trades)
	report.AvgWin, report.AvgLoss = a.calculateAvgWinLoss(trades)
	report.ProfitFactor = a.calculateProfitFactor(trades)

	// 벤치마크 비교
	report.Benchmark = a.getBenchmarkReturn(ctx, startDate, endDate)
	report.Alpha = report.TotalReturn - report.Benchmark
	report.Beta = a.calculateBeta(dailyReturns, startDate, endDate)

	a.logger.WithFields(map[string]interface{}{
		"period":        period,
		"total_return":  report.TotalReturn,
		"sharpe":        report.Sharpe,
		"max_drawdown":  report.MaxDrawdown,
		"win_rate":      report.WinRate,
	}).Info("Performance analysis completed")

	return report, nil
}

// parsePeriod parses period string to date range
func (a *Analyzer) parsePeriod(period string) (time.Time, time.Time) {
	endDate := time.Now()

	switch period {
	case "1M":
		return endDate.AddDate(0, -1, 0), endDate
	case "3M":
		return endDate.AddDate(0, -3, 0), endDate
	case "6M":
		return endDate.AddDate(0, -6, 0), endDate
	case "1Y":
		return endDate.AddDate(-1, 0, 0), endDate
	case "YTD":
		return time.Date(endDate.Year(), 1, 1, 0, 0, 0, 0, endDate.Location()), endDate
	default:
		// Default to 1 month
		return endDate.AddDate(0, -1, 0), endDate
	}
}

// calculateTotalReturn calculates cumulative return
func (a *Analyzer) calculateTotalReturn(dailyReturns []float64) float64 {
	cumReturn := 1.0
	for _, r := range dailyReturns {
		cumReturn *= (1.0 + r)
	}
	return cumReturn - 1.0
}

// annualize converts return to annualized return
func (a *Analyzer) annualize(totalReturn float64, days int) float64 {
	if days == 0 {
		return 0
	}
	return math.Pow(1.0+totalReturn, 252.0/float64(days)) - 1.0
}

// calculateVolatility calculates annualized volatility
func (a *Analyzer) calculateVolatility(dailyReturns []float64) float64 {
	if len(dailyReturns) < 2 {
		return 0
	}

	// Mean
	var sum float64
	for _, r := range dailyReturns {
		sum += r
	}
	mean := sum / float64(len(dailyReturns))

	// Variance
	var variance float64
	for _, r := range dailyReturns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(dailyReturns) - 1)

	// Annualized volatility
	return math.Sqrt(variance) * math.Sqrt(252)
}

// calculateSharpe calculates Sharpe ratio
func (a *Analyzer) calculateSharpe(annualReturn, volatility float64) float64 {
	if volatility == 0 {
		return 0
	}
	riskFreeRate := 0.03 // 3% 무위험 수익률
	return (annualReturn - riskFreeRate) / volatility
}

// calculateSortino calculates Sortino ratio
func (a *Analyzer) calculateSortino(dailyReturns []float64) float64 {
	if len(dailyReturns) < 2 {
		return 0
	}

	// Downside deviation (only negative returns)
	var sumSquaredNegative float64
	var countNegative int
	for _, r := range dailyReturns {
		if r < 0 {
			sumSquaredNegative += r * r
			countNegative++
		}
	}

	if countNegative == 0 {
		return 0
	}

	downsideVol := math.Sqrt(sumSquaredNegative/float64(countNegative)) * math.Sqrt(252)
	if downsideVol == 0 {
		return 0
	}

	totalReturn := a.calculateTotalReturn(dailyReturns)
	annualReturn := a.annualize(totalReturn, len(dailyReturns))
	riskFreeRate := 0.03

	return (annualReturn - riskFreeRate) / downsideVol
}

// calculateMaxDrawdown calculates maximum drawdown
func (a *Analyzer) calculateMaxDrawdown(dailyReturns []float64) float64 {
	if len(dailyReturns) == 0 {
		return 0
	}

	cumValue := 1.0
	peak := 1.0
	maxDD := 0.0

	for _, r := range dailyReturns {
		cumValue *= (1.0 + r)
		if cumValue > peak {
			peak = cumValue
		}
		dd := (cumValue - peak) / peak
		if dd < maxDD {
			maxDD = dd
		}
	}

	return maxDD
}

// calculateWinRate calculates win rate from trades
func (a *Analyzer) calculateWinRate(trades []Trade) float64 {
	if len(trades) == 0 {
		return 0
	}

	wins := 0
	for _, t := range trades {
		if t.PnL > 0 {
			wins++
		}
	}

	return float64(wins) / float64(len(trades))
}

// calculateAvgWinLoss calculates average win and loss
func (a *Analyzer) calculateAvgWinLoss(trades []Trade) (float64, float64) {
	if len(trades) == 0 {
		return 0, 0
	}

	var sumWin, sumLoss float64
	var countWin, countLoss int

	for _, t := range trades {
		if t.PnL > 0 {
			sumWin += t.PnL
			countWin++
		} else if t.PnL < 0 {
			sumLoss += t.PnL
			countLoss++
		}
	}

	avgWin := 0.0
	if countWin > 0 {
		avgWin = sumWin / float64(countWin)
	}

	avgLoss := 0.0
	if countLoss > 0 {
		avgLoss = sumLoss / float64(countLoss)
	}

	return avgWin, avgLoss
}

// calculateProfitFactor calculates profit factor
func (a *Analyzer) calculateProfitFactor(trades []Trade) float64 {
	var totalWin, totalLoss float64

	for _, t := range trades {
		if t.PnL > 0 {
			totalWin += t.PnL
		} else if t.PnL < 0 {
			totalLoss += math.Abs(t.PnL)
		}
	}

	if totalLoss == 0 {
		return 0
	}

	return totalWin / totalLoss
}

// getBenchmarkReturn retrieves benchmark return for period
func (a *Analyzer) getBenchmarkReturn(ctx context.Context, startDate, endDate time.Time) float64 {
	// TODO: Implement actual benchmark retrieval from DB
	// For now, return mock KOSPI return
	return 0.05 // 5%
}

// calculateBeta calculates beta against benchmark
func (a *Analyzer) calculateBeta(dailyReturns []float64, startDate, endDate time.Time) float64 {
	// TODO: Implement actual beta calculation
	// Requires benchmark daily returns
	return 1.0 // Market beta
}

// Trade represents a closed trade
type Trade struct {
	Code      string
	EntryDate time.Time
	ExitDate  time.Time
	EntryPrice float64
	ExitPrice  float64
	Quantity  int
	PnL       float64
}
