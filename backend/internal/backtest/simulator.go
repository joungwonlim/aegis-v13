package backtest

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Simulator simulates order execution in backtesting
// ⭐ SSOT: 백테스팅 시뮬레이션은 여기서만
type Simulator struct {
	db     *pgxpool.Pool
	logger *logger.Logger

	// Current state
	cash      int64
	positions map[string]*Position
	trades    []Trade

	// Statistics
	totalTrades     int
	winningTrades   int
	losingTrades    int
	totalCommission int64
	totalSlippage   int64
}

// Position represents a stock position
type Position struct {
	Code      string
	Shares    int64
	AvgPrice  int64 // Average entry price
	CostBasis int64 // Total cost including commission
}

// Trade represents a completed trade
type Trade struct {
	Code        string
	Direction   string // "buy" or "sell"
	Shares      int64
	Price       int64
	Value       int64
	Commission  int64
	Slippage    int64
	PnL         int64 // For sell orders
	ReturnPct   float64
}

// Stats holds simulation statistics
type Stats struct {
	TotalTrades     int
	WinningTrades   int
	LosingTrades    int
	TotalCommission int64
	TotalSlippage   int64
}

// NewSimulator creates a new trading simulator
func NewSimulator(db *pgxpool.Pool, logger *logger.Logger) *Simulator {
	return &Simulator{
		db:        db,
		logger:    logger,
		positions: make(map[string]*Position),
		trades:    make([]Trade, 0),
	}
}

// Initialize resets the simulator with initial capital
func (s *Simulator) Initialize(capital int64) {
	s.cash = capital
	s.positions = make(map[string]*Position)
	s.trades = make([]Trade, 0)
	s.totalTrades = 0
	s.winningTrades = 0
	s.losingTrades = 0
	s.totalCommission = 0
	s.totalSlippage = 0
}

// ExecutePlan executes an execution plan in simulation
func (s *Simulator) ExecutePlan(ctx context.Context, plan *contracts.ExecutionPlan, commission, slippage float64) error {
	for _, order := range plan.Orders {
		if err := s.executeOrder(ctx, order, commission, slippage); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"code":  order.Code,
				"error": err.Error(),
			}).Warn("Order execution failed in simulation")
			continue
		}
	}

	return nil
}

// executeOrder executes a single order
func (s *Simulator) executeOrder(ctx context.Context, order contracts.Order, commissionRate, slippageRate float64) error {
	// Get current price
	price, err := s.getCurrentPrice(ctx, order.Code, order.CreatedAt)
	if err != nil {
		return fmt.Errorf("get current price: %w", err)
	}

	// Apply slippage
	actualPrice := price
	if order.Side == contracts.OrderSideBuy {
		actualPrice = int64(float64(price) * (1.0 + slippageRate))
	} else {
		actualPrice = int64(float64(price) * (1.0 - slippageRate))
	}

	// Calculate values
	qty := int64(order.Qty)
	totalValue := actualPrice * qty
	commission := int64(math.Ceil(float64(totalValue) * commissionRate))
	slippageCost := int64(math.Abs(float64(actualPrice-price))) * qty

	trade := Trade{
		Code:       order.Code,
		Direction:  string(order.Side),
		Shares:     qty,
		Price:      actualPrice,
		Value:      totalValue,
		Commission: commission,
		Slippage:   slippageCost,
	}

	if order.Side == contracts.OrderSideBuy {
		// Buy order
		totalCost := totalValue + commission
		if s.cash < totalCost {
			return fmt.Errorf("insufficient cash: need %d, have %d", totalCost, s.cash)
		}

		s.cash -= totalCost

		// Update or create position
		if pos, exists := s.positions[order.Code]; exists {
			// Average down/up
			newShares := pos.Shares + qty
			newCost := pos.CostBasis + totalCost
			pos.Shares = newShares
			pos.CostBasis = newCost
			pos.AvgPrice = newCost / newShares
		} else {
			s.positions[order.Code] = &Position{
				Code:      order.Code,
				Shares:    qty,
				AvgPrice:  actualPrice,
				CostBasis: totalCost,
			}
		}

		s.totalTrades++

	} else {
		// Sell order
		pos, exists := s.positions[order.Code]
		if !exists {
			return fmt.Errorf("no position to sell: %s", order.Code)
		}

		if pos.Shares < qty {
			return fmt.Errorf("insufficient shares: need %d, have %d", qty, pos.Shares)
		}

		// Calculate P&L
		proceeds := totalValue - commission
		costBasis := (pos.CostBasis * qty) / pos.Shares
		pnl := proceeds - costBasis
		returnPct := float64(pnl) / float64(costBasis)

		trade.PnL = pnl
		trade.ReturnPct = returnPct

		// Update cash
		s.cash += proceeds

		// Update position
		pos.Shares -= qty
		pos.CostBasis -= costBasis

		// Remove position if fully closed
		if pos.Shares == 0 {
			delete(s.positions, order.Code)
		}

		// Update statistics
		s.totalTrades++
		if pnl > 0 {
			s.winningTrades++
		} else if pnl < 0 {
			s.losingTrades++
		}
	}

	// Record trade
	s.trades = append(s.trades, trade)
	s.totalCommission += commission
	s.totalSlippage += slippageCost

	return nil
}

// GetEquity returns current total equity (cash + positions)
func (s *Simulator) GetEquity() int64 {
	// TODO: Mark positions to market
	// For now, use cost basis as approximation
	positionValue := int64(0)
	for _, pos := range s.positions {
		positionValue += pos.CostBasis
	}

	return s.cash + positionValue
}

// GetStats returns simulation statistics
func (s *Simulator) GetStats() Stats {
	return Stats{
		TotalTrades:     s.totalTrades,
		WinningTrades:   s.winningTrades,
		LosingTrades:    s.losingTrades,
		TotalCommission: s.totalCommission,
		TotalSlippage:   s.totalSlippage,
	}
}

// getCurrentPrice retrieves the closing price for a stock on a given date
func (s *Simulator) getCurrentPrice(ctx context.Context, code string, date time.Time) (int64, error) {
	query := `
		SELECT close
		FROM data.daily_prices
		WHERE stock_code = $1
		  AND trade_date = $2
		LIMIT 1
	`

	var price int64
	err := s.db.QueryRow(ctx, query, code, date).Scan(&price)
	if err != nil {
		return 0, fmt.Errorf("query price: %w", err)
	}

	return price, nil
}
