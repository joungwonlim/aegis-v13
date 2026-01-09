package backtest

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/brain"
	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Engine runs backtesting simulations
// ⭐ SSOT: 백테스팅 실행은 여기서만
type Engine struct {
	orchestrator *brain.Orchestrator
	simulator    *Simulator
	logger       *logger.Logger
}

// Config holds backtest configuration
type Config struct {
	StartDate      time.Time
	EndDate        time.Time
	InitialCapital int64
	RebalanceDays  int  // Rebalancing frequency (e.g., 7 for weekly)
	Commission     float64 // Commission rate (e.g., 0.0015 for 0.15%)
	Slippage       float64 // Slippage rate (e.g., 0.001 for 0.1%)
}

// Result holds backtest results
type Result struct {
	Config         Config
	StartDate      time.Time
	EndDate        time.Time
	Duration       time.Duration
	TotalDays      int
	TradingDays    int
	RebalanceCount int

	// Performance metrics
	InitialCapital   int64
	FinalCapital     int64
	TotalReturn      float64
	AnnualizedReturn float64
	CAGR             float64
	Volatility       float64
	SharpeRatio      float64
	SortinoRatio     float64
	MaxDrawdown      float64
	WinRate          float64

	// Trading metrics
	TotalTrades      int
	WinningTrades    int
	LosingTrades     int
	TotalCommission  int64
	TotalSlippage    int64

	// Equity curve
	EquityCurve []EquityPoint

	// Factor attribution
	Attributions []contracts.FactorAttribution

	// Daily runs
	DailyRuns []*brain.RunResult
}

// EquityPoint represents a point in the equity curve
type EquityPoint struct {
	Date   time.Time
	Equity int64
	Return float64
}

// NewEngine creates a new backtest engine
func NewEngine(
	orchestrator *brain.Orchestrator,
	simulator *Simulator,
	logger *logger.Logger,
) *Engine {
	return &Engine{
		orchestrator: orchestrator,
		simulator:    simulator,
		logger:       logger,
	}
}

// Run executes a backtest simulation
func (e *Engine) Run(ctx context.Context, config Config) (*Result, error) {
	e.logger.WithFields(map[string]interface{}{
		"start_date":      config.StartDate.Format("2006-01-02"),
		"end_date":        config.EndDate.Format("2006-01-02"),
		"initial_capital": config.InitialCapital,
		"rebalance_days":  config.RebalanceDays,
	}).Info("Starting backtest")

	startTime := time.Now()

	result := &Result{
		Config:         config,
		StartDate:      config.StartDate,
		EndDate:        config.EndDate,
		InitialCapital: config.InitialCapital,
		DailyRuns:      make([]*brain.RunResult, 0),
		EquityCurve:    make([]EquityPoint, 0),
	}

	// Initialize simulator
	e.simulator.Initialize(config.InitialCapital)

	// Run pipeline for each trading day
	currentDate := config.StartDate
	daysSinceRebalance := 0
	tradingDays := 0

	for currentDate.Before(config.EndDate) || currentDate.Equal(config.EndDate) {
		// Skip weekends
		if currentDate.Weekday() == time.Saturday || currentDate.Weekday() == time.Sunday {
			currentDate = currentDate.AddDate(0, 0, 1)
			continue
		}

		tradingDays++

		// Check if it's a rebalancing day
		shouldRebalance := daysSinceRebalance >= config.RebalanceDays
		if shouldRebalance {
			// Run pipeline
			runConfig := brain.RunConfig{
				Date:           currentDate,
				RunID:          brain.GenerateRunID(),
				GitSHA:         "backtest",
				FeatureVersion: "v1.0.0",
				Capital:        e.simulator.GetEquity(),
				DryRun:         false, // Actually execute in simulation
			}

			runResult, err := e.orchestrator.Run(ctx, runConfig)
			if err != nil {
				e.logger.WithFields(map[string]interface{}{
					"date":  currentDate.Format("2006-01-02"),
					"error": err.Error(),
				}).Warn("Pipeline run failed")
			} else {
				result.DailyRuns = append(result.DailyRuns, runResult)

				// Execute trades in simulation
				if runResult.ExecutionPlan != nil {
					e.simulator.ExecutePlan(ctx, runResult.ExecutionPlan, config.Commission, config.Slippage)
					result.RebalanceCount++
				}
			}

			daysSinceRebalance = 0
		} else {
			daysSinceRebalance++
		}

		// Update portfolio value (mark to market)
		equity := e.simulator.GetEquity()
		returnPct := float64(equity-result.InitialCapital) / float64(result.InitialCapital)

		result.EquityCurve = append(result.EquityCurve, EquityPoint{
			Date:   currentDate,
			Equity: equity,
			Return: returnPct,
		})

		// Move to next day
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// Calculate final metrics
	result.Duration = time.Since(startTime)
	result.TotalDays = int(config.EndDate.Sub(config.StartDate).Hours() / 24)
	result.TradingDays = tradingDays
	result.FinalCapital = e.simulator.GetEquity()

	e.calculateMetrics(result)

	e.logger.WithFields(map[string]interface{}{
		"duration":       result.Duration.Seconds(),
		"trading_days":   result.TradingDays,
		"rebalances":     result.RebalanceCount,
		"total_return":   fmt.Sprintf("%.2f%%", result.TotalReturn*100),
		"sharpe_ratio":   fmt.Sprintf("%.2f", result.SharpeRatio),
		"max_drawdown":   fmt.Sprintf("%.2f%%", result.MaxDrawdown*100),
	}).Info("Backtest completed")

	return result, nil
}

// calculateMetrics calculates performance metrics from equity curve
func (e *Engine) calculateMetrics(result *Result) {
	if len(result.EquityCurve) == 0 {
		return
	}

	// Total return
	result.TotalReturn = float64(result.FinalCapital-result.InitialCapital) / float64(result.InitialCapital)

	// Annualized return
	years := float64(result.TotalDays) / 365.25
	result.AnnualizedReturn = result.TotalReturn / years

	// CAGR
	if years > 0 {
		result.CAGR = (math.Pow(float64(result.FinalCapital)/float64(result.InitialCapital), 1.0/years) - 1.0)
	}

	// Calculate daily returns
	dailyReturns := make([]float64, 0, len(result.EquityCurve)-1)
	for i := 1; i < len(result.EquityCurve); i++ {
		prevEquity := result.EquityCurve[i-1].Equity
		currEquity := result.EquityCurve[i].Equity
		dailyReturn := float64(currEquity-prevEquity) / float64(prevEquity)
		dailyReturns = append(dailyReturns, dailyReturn)
	}

	// Volatility (annualized)
	result.Volatility = e.calculateVolatility(dailyReturns) * math.Sqrt(252)

	// Sharpe Ratio (assuming 0% risk-free rate)
	if result.Volatility > 0 {
		result.SharpeRatio = result.AnnualizedReturn / result.Volatility
	}

	// Sortino Ratio (downside deviation)
	downsideReturns := make([]float64, 0)
	for _, r := range dailyReturns {
		if r < 0 {
			downsideReturns = append(downsideReturns, r)
		}
	}
	downsideDeviation := e.calculateVolatility(downsideReturns) * math.Sqrt(252)
	if downsideDeviation > 0 {
		result.SortinoRatio = result.AnnualizedReturn / downsideDeviation
	}

	// Maximum Drawdown
	result.MaxDrawdown = e.calculateMaxDrawdown(result.EquityCurve)

	// Win rate from simulator
	stats := e.simulator.GetStats()
	result.TotalTrades = stats.TotalTrades
	result.WinningTrades = stats.WinningTrades
	result.LosingTrades = stats.LosingTrades
	result.TotalCommission = stats.TotalCommission
	result.TotalSlippage = stats.TotalSlippage
	if result.TotalTrades > 0 {
		result.WinRate = float64(result.WinningTrades) / float64(result.TotalTrades)
	}
}

// calculateVolatility calculates standard deviation
func (e *Engine) calculateVolatility(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Mean
	sum := 0.0
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	// Variance
	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))

	// Standard deviation
	return math.Sqrt(variance)
}

// calculateMaxDrawdown calculates maximum drawdown from equity curve
func (e *Engine) calculateMaxDrawdown(curve []EquityPoint) float64 {
	if len(curve) == 0 {
		return 0
	}

	maxDrawdown := 0.0
	peak := curve[0].Equity

	for _, point := range curve {
		if point.Equity > peak {
			peak = point.Equity
		}

		drawdown := float64(peak-point.Equity) / float64(peak)
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}
