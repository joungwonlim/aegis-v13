package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Planner implements S6: Execution planning
// ⭐ SSOT: S6 주문 계획 로직은 여기서만
type Planner struct {
	broker Broker
	config ExecutionConfig
	logger *logger.Logger
}

// ExecutionConfig defines execution parameters
type ExecutionConfig struct {
	OrderType      contracts.OrderType // LIMIT or MARKET
	SlippageBps    int                 // 슬리피지 (10 = 0.1%)
	MaxOrderSize   int64               // 최대 주문 금액
	SplitThreshold int64               // 분할 주문 기준
}

// NewPlanner creates a new execution planner
func NewPlanner(broker Broker, config ExecutionConfig, logger *logger.Logger) *Planner {
	return &Planner{
		broker: broker,
		config: config,
		logger: logger,
	}
}

// Plan creates execution orders from target portfolio
func (p *Planner) Plan(ctx context.Context, target *contracts.TargetPortfolio) ([]contracts.Order, error) {
	orders := make([]contracts.Order, 0)

	// 1. 매도 주문 먼저 (자금 확보)
	for _, pos := range target.Positions {
		if pos.Action == contracts.ActionSell {
			order, err := p.createSellOrder(ctx, pos)
			if err != nil {
				p.logger.WithFields(map[string]interface{}{
					"code":  pos.Code,
					"error": err,
				}).Warn("Failed to create sell order")
				continue
			}
			orders = append(orders, order)
		}
	}

	// 2. 매수 주문
	for _, pos := range target.Positions {
		if pos.Action == contracts.ActionBuy {
			order, err := p.createBuyOrder(ctx, pos)
			if err != nil {
				p.logger.WithFields(map[string]interface{}{
					"code":  pos.Code,
					"error": err,
				}).Warn("Failed to create buy order")
				continue
			}
			orders = append(orders, order)
		}
	}

	p.logger.WithFields(map[string]interface{}{
		"total_orders": len(orders),
		"sell_orders":  p.countOrders(orders, contracts.OrderSideSell),
		"buy_orders":   p.countOrders(orders, contracts.OrderSideBuy),
	}).Info("Execution plan created")

	return orders, nil
}

// createBuyOrder creates a buy order from target position
func (p *Planner) createBuyOrder(ctx context.Context, pos contracts.TargetPosition) (contracts.Order, error) {
	price, err := p.getTargetPrice(ctx, pos.Code, contracts.OrderSideBuy)
	if err != nil {
		return contracts.Order{}, fmt.Errorf("failed to get target price: %w", err)
	}

	return contracts.Order{
		Code:      pos.Code,
		Name:      pos.Name,
		Side:      contracts.OrderSideBuy,
		Qty:       pos.TargetQty,
		Price:     price,
		OrderType: p.config.OrderType,
		Status:    contracts.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// createSellOrder creates a sell order from target position
func (p *Planner) createSellOrder(ctx context.Context, pos contracts.TargetPosition) (contracts.Order, error) {
	price, err := p.getTargetPrice(ctx, pos.Code, contracts.OrderSideSell)
	if err != nil {
		return contracts.Order{}, fmt.Errorf("failed to get target price: %w", err)
	}

	return contracts.Order{
		Code:      pos.Code,
		Name:      pos.Name,
		Side:      contracts.OrderSideSell,
		Qty:       pos.TargetQty,
		Price:     price,
		OrderType: p.config.OrderType,
		Status:    contracts.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// getTargetPrice calculates target price with slippage
func (p *Planner) getTargetPrice(ctx context.Context, code string, side contracts.OrderSide) (float64, error) {
	if p.config.OrderType == contracts.OrderTypeMarket {
		return 0, nil // 시장가
	}

	currentPrice, err := p.broker.GetCurrentPrice(ctx, code)
	if err != nil {
		return 0, fmt.Errorf("failed to get current price: %w", err)
	}

	// 지정가: 슬리피지 적용
	slippage := float64(p.config.SlippageBps) / 10000

	if side == contracts.OrderSideBuy {
		return currentPrice * (1 + slippage), nil
	}
	return currentPrice * (1 - slippage), nil
}

// splitOrder splits large order into smaller chunks
func (p *Planner) splitOrder(order contracts.Order) []contracts.Order {
	orderValue := int64(order.Qty) * int64(order.Price)

	if orderValue < p.config.SplitThreshold {
		return []contracts.Order{order}
	}

	// 분할
	chunks := make([]contracts.Order, 0)
	remaining := order.Qty
	chunkSize := int(p.config.MaxOrderSize / int64(order.Price))

	for remaining > 0 {
		qty := remaining
		if qty > chunkSize {
			qty = chunkSize
		}

		chunk := order
		chunk.Qty = qty
		chunks = append(chunks, chunk)
		remaining -= qty
	}

	return chunks
}

// countOrders counts orders by side
func (p *Planner) countOrders(orders []contracts.Order, side contracts.OrderSide) int {
	count := 0
	for _, order := range orders {
		if order.Side == side {
			count++
		}
	}
	return count
}

// DefaultExecutionConfig returns default configuration
func DefaultExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		OrderType:      contracts.OrderTypeLimit, // 기본: 지정가
		SlippageBps:    10,                       // 0.1% 슬리피지
		MaxOrderSize:   50_000_000,               // 5천만원
		SplitThreshold: 100_000_000,              // 1억원 이상 분할
	}
}
